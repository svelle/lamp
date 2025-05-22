package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
)

// OllamaHost stores the URL of the Ollama host (set from main.go)
var OllamaHost string = "http://localhost:11434"

// OllamaTimeout stores the timeout in seconds for Ollama requests (set from main.go)
var OllamaTimeout int = 120

// LLMProvider represents the different LLM providers available
type LLMProvider string

const (
	// ProviderAnthropic represents Anthropic's Claude models
	ProviderAnthropic LLMProvider = "anthropic"
	// ProviderOpenAI represents OpenAI's models
	ProviderOpenAI LLMProvider = "openai"
	// ProviderGemini represents Google's Gemini models
	ProviderGemini LLMProvider = "gemini"
	// ProviderOllama represents locally hosted models via Ollama
	ProviderOllama LLMProvider = "ollama"
	// Add more providers as needed

	// Default settings
	defaultMaxLogEntries = 100 // Default limit for logs to send to LLMs
)

// LLMConfig represents the configuration for an LLM-based analysis
type LLMConfig struct {
	Provider       LLMProvider
	Model          string
	APIKey         string
	MaxEntries     int
	Problem        string
	ThinkingBudget int
}

// AnalysisPrompt contains the prepared prompt data for LLM analysis
type AnalysisPrompt struct {
	SystemPrompt string
	UserPrompt   string
	LogText      string
	Description  string
	HasDuplicates bool
}

// analyzeWithLLM routes the log analysis to the appropriate LLM provider
func analyzeWithLLM(logs []LogEntry, config LLMConfig, configContent string) error {
	// If the API key is not provided and we're not using Ollama (which doesn't need a key), 
	// try to get it from the environment
	if config.APIKey == "" && config.Provider != ProviderOllama {
		envVar := getAPIKeyEnvVar(config.Provider)
		config.APIKey = getEnvAPIKey(envVar)
		if config.APIKey == "" {
			return fmt.Errorf("%s API key is required for AI analysis", config.Provider)
		}
	}

	// Route to the appropriate provider
	switch config.Provider {
	case ProviderAnthropic:
		return analyzeWithAnthropic(logs, config, configContent)
	case ProviderOpenAI:
		return analyzeWithOpenAI(logs, config, configContent)
	case ProviderGemini:
		return analyzeWithGemini(logs, config, configContent)
	case ProviderOllama:
		return analyzeWithOllama(logs, config, configContent)
	default:
		return fmt.Errorf("unsupported LLM provider: %s", config.Provider)
	}
}

// getAPIKeyEnvVar returns the environment variable name for the API key
func getAPIKeyEnvVar(provider LLMProvider) string {
	switch provider {
	case ProviderAnthropic:
		return "ANTHROPIC_API_KEY"
	case ProviderOpenAI:
		return "OPENAI_API_KEY"
	case ProviderGemini:
		return "GEMINI_API_KEY"
	case ProviderOllama:
		// Ollama is locally hosted and doesn't require an API key
		return ""
	default:
		return ""
	}
}

// getEnvAPIKey gets the API key from the environment variable
func getEnvAPIKey(envVar string) string {
	return os.Getenv(envVar)
}

// getDefaultModel returns the default model for a provider
func getDefaultModel(provider LLMProvider) string {
	return GetDefaultModel(provider)
}

// formatLogsForAnalysis formats log entries into a text representation for analysis
func formatLogsForAnalysis(logs []LogEntry) (string, int, bool) {
	var logText strings.Builder
	totalEntries := 0
	hasDuplicates := false

	for i, log := range logs {
		// Add count information for entries with duplicates
		if log.DuplicateCount > 1 {
			logText.WriteString(fmt.Sprintf("%d. [%s] [%s] %s: %s (repeated %d times)\n",
				i+1,
				log.Timestamp.Format("2006-01-02 15:04:05"),
				log.Level,
				log.Source,
				log.Message,
				log.DuplicateCount))
			hasDuplicates = true
			totalEntries += log.DuplicateCount
		} else {
			logText.WriteString(fmt.Sprintf("%d. [%s] [%s] %s: %s\n",
				i+1,
				log.Timestamp.Format("2006-01-02 15:04:05"),
				log.Level,
				log.Source,
				log.Message))
			totalEntries += 1
		}

		if log.User != "" {
			logText.WriteString(fmt.Sprintf("   User: %s\n", log.User))
		}
		if log.Source != "" {
			logText.WriteString(fmt.Sprintf("   Source: %s\n", log.Source))
		}
		if len(log.Extras) > 0 {
			logText.WriteString(fmt.Sprintf("   Extras: %s\n", log.ExtrasToString()))
		}
		logText.WriteString("\n")
	}

	return logText.String(), totalEntries, hasDuplicates
}

// prepareAnalysisPrompts generates system and user prompts for log analysis
func prepareAnalysisPrompts(logs []LogEntry, config LLMConfig, configContent string) (AnalysisPrompt, error) {
	var prompt AnalysisPrompt
	
	// If maxEntries is not set (0), use the default
	maxEntries := config.MaxEntries
	if maxEntries <= 0 {
		maxEntries = defaultMaxLogEntries
	}

	// Prepare logs
	logsToAnalyze := logs
	if len(logs) > maxEntries {
		fmt.Printf("Limiting analysis to %d most recent log entries (out of %d total)\n",
			maxEntries, len(logs))
		// Sort logs by timestamp (most recent first)
		logsToAnalyze = logs[len(logs)-maxEntries:]
	}

	// Format logs
	logText, totalEntries, hasDuplicates := formatLogsForAnalysis(logsToAnalyze)
	prompt.LogText = logText
	prompt.HasDuplicates = hasDuplicates

	// Include sanitized_config.json if available from support packet
	var configText string
	if configContent != "" {
		configText = configContent
		logger.Debug("Including sanitized_config.json in AI analysis", "size", len(configText))
	}

	// Create appropriate preface based on duplication and config inclusion
	entryDescription := fmt.Sprintf("%d Mattermost server log entries", len(logsToAnalyze))
	if hasDuplicates {
		entryDescription = fmt.Sprintf("%d unique Mattermost server log entries representing %d total log entries",
			len(logsToAnalyze), totalEntries)
	}
	if configText != "" {
		entryDescription += " and the sanitized Mattermost configuration"
	}
	prompt.Description = entryDescription

	// Create the system prompt
	systemPromptBase := `You are an expert log analyzer for Mattermost server logs. 
Analyze the provided logs and provide a comprehensive report including:

1. A high-level summary of what's happening in the logs
2. Identification of any errors, warnings, or critical issues
3. Patterns or trends you notice
 - Look for sudden spikes in errors within a short time
 - Look for network connectivity errors
 - Look for user_id or channel_id values that might be common across errors.
4. Potential root causes for any problems
5. Recommendations for further investigation or resolution

Format your entire response in Markdown for easy reading and sharing. Use appropriate headings, lists, code blocks, and other Markdown formatting to make your analysis clear and well-structured.

Focus on actionable insights and be specific about what you find.`

	if configText != "" {
		systemPromptBase += `

When configuration data is provided, also consider:
- Configuration settings that might be related to the issues in the logs
- Misconfigurations that could be causing problems
- Recommended configuration changes based on the log patterns`
	}

	prompt.SystemPrompt = systemPromptBase

	// Use a more concise prompt with thinking mode
	if config.ThinkingBudget > 0 {
		prompt.SystemPrompt = `You are an expert log analyzer for Mattermost server logs. Analyze these logs and identify issues, patterns, and solutions. Format your entire response in Markdown.`

		// Add information about duplicates in the prompt
		if hasDuplicates {
			prompt.SystemPrompt += ` Some log entries may be marked with repetition counts, indicating they appeared multiple times.`
		}
		// Add information about configuration if included
		if configText != "" {
			prompt.SystemPrompt += ` Configuration data is also provided - use it to identify misconfigurations and provide configuration recommendations.`
		}
	}

	// Create the user prompt
	var userPromptText string
	if config.Problem != "" {
		if config.ThinkingBudget > 0 {
			userPromptText = fmt.Sprintf("I'm investigating this problem: %s\n\nHere are %s to analyze:\n\n%s",
				config.Problem, entryDescription, logText)
		} else {
			userPromptText = fmt.Sprintf("I'm investigating this problem: %s\n\nHere are %s to analyze:\n\n%s\n\nPlease provide a detailed analysis of these logs focusing on the problem I described.",
				config.Problem, entryDescription, logText)
		}
	} else {
		if config.ThinkingBudget > 0 {
			userPromptText = fmt.Sprintf("Here are %s to analyze:\n\n%s",
				entryDescription, logText)
		} else {
			userPromptText = fmt.Sprintf("Here are %s to analyze:\n\n%s\n\nPlease provide a detailed analysis of these logs.",
				entryDescription, logText)
		}
	}

	// Add configuration data if available
	if configText != "" {
		userPromptText += "\n\n## Mattermost Configuration (sanitized_config.json)\n\n```json\n" + configText + "\n```"
	}

	prompt.UserPrompt = userPromptText

	return prompt, nil
}

// displayAndCopyAnalysis handles the common post-processing of analysis results
func displayAndCopyAnalysis(analysisText string) error {
	// Create buffer for the analysis with markdown header
	var analysisBuffer strings.Builder
	analysisBuffer.WriteString("# LLM LOG ANALYSIS\n\n")
	analysisBuffer.WriteString(analysisText)

	// Display the analysis
	fmt.Println("\n" + analysisBuffer.String())
	
	// Prompt the user to copy to clipboard
	fmt.Println("\n-------------------------------------------------")
	fmt.Println("The analysis above is formatted in Markdown.")
	fmt.Print("Would you like to copy it to your clipboard? (y/n): ")
	
	// Read user input
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		fmt.Println("Error reading input:", err)
		return nil // Non-fatal error
	} 
	
	if strings.ToLower(response) == "y" || strings.ToLower(response) == "yes" {
		err = clipboard.WriteAll(analysisBuffer.String())
		if err != nil {
			fmt.Println("Error copying to clipboard:", err)
			return nil // Non-fatal error
		} else {
			fmt.Println("Analysis copied to clipboard!")
		}
	}

	return nil
}

//
// Anthropic Claude Implementation
//

// AnthropicRequest represents the request structure for Anthropic API
type AnthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Messages    []AnthropicMessage `json:"messages"`
	System      string             `json:"system"`
	Temperature float64            `json:"temperature"`
	Thinking    *ThinkingConfig    `json:"thinking,omitempty"`
}

// ThinkingConfig represents the configuration for thinking mode
type ThinkingConfig struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens"`
}

// AnthropicMessage represents a message in the Anthropic API request
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicResponse represents the response structure from Anthropic API
type AnthropicResponse struct {
	Content []ContentBlock  `json:"content"`
	ID      string          `json:"id"`
	Model   string          `json:"model"`
	Type    string          `json:"type"`
	Error   *AnthropicError `json:"error,omitempty"`
}

// ContentBlock represents a content block in the Anthropic API response
type ContentBlock struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

// AnthropicError represents an error from the Anthropic API
type AnthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// analyzeWithAnthropic sends log data to Anthropic API for analysis
func analyzeWithAnthropic(logs []LogEntry, config LLMConfig, configContent string) error {
	// Get model info if available
	modelName := config.Model
	if modelName == "" {
		modelName = getDefaultModel(config.Provider)
	}
	
	// Try to get the human-friendly model name
	modelInfo, found := GetModelInfo(config.Provider, modelName)
	if found {
		fmt.Printf("Analyzing logs with %s API using %s (%s)...\n", 
			config.Provider, modelInfo.Name, modelName)
	} else {
		fmt.Printf("Analyzing logs with %s API using %s...\n", 
			config.Provider, modelName)
	}

	// Prepare prompts and logs
	prompt, err := prepareAnalysisPrompts(logs, config, configContent)
	if err != nil {
		return err
	}

	// Default model if not specified
	modelToUse := config.Model
	if modelToUse == "" {
		modelToUse = getDefaultModel(config.Provider)
	}

	// Create the request
	request := AnthropicRequest{
		Model:     modelToUse,
		MaxTokens: 4000,
		Messages: []AnthropicMessage{
			{
				Role:    "user",
				Content: prompt.UserPrompt,
			},
		},
		System:      prompt.SystemPrompt,
		Temperature: 0.3,
	}

	// Enable thinking mode if thinkingBudget is set
	if config.ThinkingBudget > 0 {
		// If thinking is enabled but the model isn't specified, default to Sonnet
		if config.Model == "" {
			request.Model = "claude-3-7-sonnet-latest"
		}

		// Ensure max_tokens is larger than thinking budget (Claude requirement)
		responseTokens := 4000 // Default tokens for actual response
		request.MaxTokens = config.ThinkingBudget + responseTokens

		// Set temperature to 1 when thinking is enabled (Claude requirement)
		request.Temperature = 1.0

		request.Thinking = &ThinkingConfig{
			Type:         "enabled",
			BudgetTokens: config.ThinkingBudget,
		}
		fmt.Printf("Extended thinking mode enabled with %d tokens budget (total max tokens: %d)\n",
			config.ThinkingBudget, request.MaxTokens)
	}

	// Convert request to JSON
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(requestJSON))
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// Send request
	fmt.Println("Sending request to Anthropic API...")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request to Anthropic API: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %v", err)
	}

	// Check if response is successful
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error from Anthropic API: %s", string(body))
	}

	// Parse response
	var anthropicResponse AnthropicResponse
	err = json.Unmarshal(body, &anthropicResponse)
	if err != nil {
		return fmt.Errorf("error parsing response: %v", err)
	}

	// Check for API error
	if anthropicResponse.Error != nil {
		return fmt.Errorf("anthropic API error: %s - %s",
			anthropicResponse.Error.Type,
			anthropicResponse.Error.Message)
	}

	// Extract analysis text from response
	var analysisText string
	
	// Check if we're using extended thinking mode
	if config.ThinkingBudget > 0 {
		// Look for thinking content and final answer
		var thinkingOutput, finalAnswer string
		for _, content := range anthropicResponse.Content {
			if content.Type == "text" {
				// If the content contains "[Thinking]", it's the thinking output
				if strings.Contains(content.Text, "[Thinking]") {
					thinkingOutput = content.Text
				} else {
					// Otherwise, it's the final answer
					finalAnswer = content.Text
				}
			}
		}

		// Format analysis with thinking section if available
		if thinkingOutput != "" {
			analysisText += "## LLM THINKING PROCESS\n\n"
			analysisText += thinkingOutput
			analysisText += "\n\n## FINAL ANALYSIS\n\n"
		}

		// Add final answer
		if finalAnswer != "" {
			analysisText += finalAnswer
		} else {
			// If there's no separate final answer, add all content
			for _, content := range anthropicResponse.Content {
				if content.Type == "text" {
					analysisText += content.Text
				}
			}
		}
	} else {
		// Standard mode - add all content
		for _, content := range anthropicResponse.Content {
			if content.Type == "text" {
				analysisText += content.Text
			}
		}
	}

	// Display the analysis and handle clipboard copy
	return displayAndCopyAnalysis(analysisText)
}

//
// OpenAI Implementation
//

// OpenAIRequest represents the request structure for OpenAI API
type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens"`
}

// OpenAIMessage represents a message in the OpenAI API request
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse represents the response structure from OpenAI API
type OpenAIResponse struct {
	ID      string            `json:"id"`
	Object  string            `json:"object"`
	Created int64             `json:"created"`
	Model   string            `json:"model"`
	Choices []OpenAIChoice    `json:"choices"`
	Usage   OpenAIUsage       `json:"usage"`
	Error   *OpenAIError      `json:"error,omitempty"`
}

// OpenAIChoice represents a completion choice in the OpenAI API response
type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// OpenAIUsage represents token usage information
type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAIError represents an error from the OpenAI API
type OpenAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

//
// Gemini Implementation
//

// GeminiRequest represents the request structure for Gemini API
type GeminiRequest struct {
	Contents         []GeminiContent `json:"contents"`
	GenerationConfig GeminiGenerationConfig `json:"generationConfig"`
	SafetySettings   []GeminiSafetySetting `json:"safetySettings,omitempty"`
}

// GeminiContent represents a content part in the Gemini API request
type GeminiContent struct {
	Role  string         `json:"role"`
	Parts []GeminiPart   `json:"parts"`
}

// GeminiPart represents a content part in a Gemini content message
type GeminiPart struct {
	Text string `json:"text"`
}

// GeminiGenerationConfig represents generation parameters for Gemini API
type GeminiGenerationConfig struct {
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
	TopP            float64 `json:"topP,omitempty"`
	TopK            int     `json:"topK,omitempty"`
}

// GeminiSafetySetting represents safety settings for Gemini API
type GeminiSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// GeminiResponse represents the response structure from Gemini API
type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
	PromptFeedback *GeminiPromptFeedback `json:"promptFeedback,omitempty"`
	Error *GeminiError `json:"error,omitempty"`
}

// GeminiCandidate represents a completion candidate in the Gemini API response
type GeminiCandidate struct {
	Content       GeminiContent       `json:"content"`
	FinishReason  string              `json:"finishReason"`
	SafetyRatings []GeminiSafetyRating `json:"safetyRatings,omitempty"`
}

// GeminiSafetyRating represents a safety rating in the Gemini API response
type GeminiSafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

// GeminiPromptFeedback represents feedback about the prompt
type GeminiPromptFeedback struct {
	SafetyRatings []GeminiSafetyRating `json:"safetyRatings"`
}

// GeminiError represents an error from the Gemini API
type GeminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

//
// Ollama Implementation
//

// OllamaRequest represents the request structure for Ollama API
type OllamaRequest struct {
	Model     string              `json:"model"`
	Messages  []OllamaMessage     `json:"messages"`
	Stream    bool                `json:"stream"`
	Options   OllamaOptions       `json:"options,omitempty"`
}

// OllamaMessage represents a message in the Ollama API request
type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OllamaOptions represents configuration options for the Ollama API
type OllamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

// OllamaResponse represents the response structure from Ollama API
type OllamaResponse struct {
	Model       string `json:"model"`
	CreatedAt   string `json:"created_at"`
	Message     OllamaMessage `json:"message"`
	Done        bool   `json:"done"`
	TotalDuration int64 `json:"total_duration"`
	LoadDuration  int64 `json:"load_duration"`
	PromptEvalCount int  `json:"prompt_eval_count"`
	PromptEvalDuration int64 `json:"prompt_eval_duration"`
	EvalCount      int  `json:"eval_count"`
	EvalDuration   int64 `json:"eval_duration"`
}

// analyzeWithGemini sends log data to Gemini API for analysis
func analyzeWithGemini(logs []LogEntry, config LLMConfig, configContent string) error {
	// Get model info if available
	modelName := config.Model
	if modelName == "" {
		modelName = getDefaultModel(config.Provider)
	}
	
	// Try to get the human-friendly model name
	modelInfo, found := GetModelInfo(config.Provider, modelName)
	if found {
		fmt.Printf("Analyzing logs with %s API using %s (%s)...\n", 
			config.Provider, modelInfo.Name, modelName)
	} else {
		fmt.Printf("Analyzing logs with %s API using %s...\n", 
			config.Provider, modelName)
	}

	// Prepare prompts and logs
	prompt, err := prepareAnalysisPrompts(logs, config, configContent)
	if err != nil {
		return err
	}

	// Default model if not specified
	modelToUse := config.Model
	if modelToUse == "" {
		modelToUse = getDefaultModel(config.Provider)
	}

	// Gemini doesn't support "system" role, so combine system and user prompts
	// into a single user message
	combinedPrompt := prompt.SystemPrompt + "\n\n" + prompt.UserPrompt
	
	userContent := GeminiContent{
		Role: "user",
		Parts: []GeminiPart{
			{Text: combinedPrompt},
		},
	}

	// Create the full request
	request := GeminiRequest{
		Contents: []GeminiContent{userContent},
		GenerationConfig: GeminiGenerationConfig{
			Temperature:     0.3,
			MaxOutputTokens: 4000,
			TopP:            0.95,
		},
	}

	// Convert request to JSON
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Create HTTP request
	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", 
		modelToUse, config.APIKey)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestJSON))
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// Send request
	fmt.Println("Sending request to Gemini API...")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request to Gemini API: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %v", err)
	}

	// Check if response is successful
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error from Gemini API (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var geminiResponse GeminiResponse
	err = json.Unmarshal(body, &geminiResponse)
	if err != nil {
		return fmt.Errorf("error parsing response: %v", err)
	}

	// Check for API error
	if geminiResponse.Error != nil {
		return fmt.Errorf("gemini API error (code %d): %s", 
			geminiResponse.Error.Code, geminiResponse.Error.Message)
	}

	// Extract the content from the response
	if len(geminiResponse.Candidates) == 0 {
		return fmt.Errorf("no completions returned from Gemini API")
	}

	// Get the analysis text from the response
	var analysisText string
	for _, part := range geminiResponse.Candidates[0].Content.Parts {
		analysisText += part.Text
	}

	// Display the analysis and handle clipboard copy
	return displayAndCopyAnalysis(analysisText)
}

// analyzeWithOllama sends log data to a local Ollama instance for analysis
func analyzeWithOllama(logs []LogEntry, config LLMConfig, configContent string) error {
	// Get model info if available
	modelName := config.Model
	if modelName == "" {
		modelName = getDefaultModel(config.Provider)
	}
	
	// Try to get the human-friendly model name
	modelInfo, found := GetModelInfo(config.Provider, modelName)
	if found {
		fmt.Printf("Analyzing logs with %s API using %s (%s)...\n", 
			config.Provider, modelInfo.Name, modelName)
	} else {
		fmt.Printf("Analyzing logs with %s API using %s...\n", 
			config.Provider, modelName)
	}

	// Prepare prompts and logs
	prompt, err := prepareAnalysisPrompts(logs, config, configContent)
	if err != nil {
		return err
	}

	// Combine system and user prompts for Ollama
	systemMessage := OllamaMessage{
		Role:    "system",
		Content: prompt.SystemPrompt,
	}
	
	userMessage := OllamaMessage{
		Role:    "user",
		Content: prompt.UserPrompt,
	}

	// Create the request
	request := OllamaRequest{
		Model:    modelName,
		Messages: []OllamaMessage{systemMessage, userMessage},
		Stream:   false,
		Options: OllamaOptions{
			Temperature: 0.3,
			NumPredict:  4000,
		},
	}

	// Convert request to JSON
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Create HTTP request using the configured Ollama host
	apiURL := OllamaHost
	if !strings.HasSuffix(apiURL, "/") {
		apiURL += "/"
	}
	apiURL += "api/chat"
	
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestJSON))
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Create HTTP client with the configured timeout
	client := &http.Client{
		Timeout: time.Duration(OllamaTimeout) * time.Second,
	}

	// Send request
	fmt.Printf("Sending request to local Ollama instance (timeout: %d seconds)...\n", OllamaTimeout)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request to Ollama: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %v", err)
	}

	// Check if response is successful
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error from Ollama (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var ollamaResponse OllamaResponse
	err = json.Unmarshal(body, &ollamaResponse)
	if err != nil {
		return fmt.Errorf("error parsing response: %v", err)
	}

	// Extract the analysis text from the response
	analysisText := ollamaResponse.Message.Content

	// Display timing information
	totalTimeSeconds := float64(ollamaResponse.TotalDuration) / 1e9
	fmt.Printf("Request completed in %.2f seconds\n", totalTimeSeconds)

	// Display the analysis and handle clipboard copy
	return displayAndCopyAnalysis(analysisText)
}

// analyzeWithOpenAI sends log data to OpenAI API for analysis
func analyzeWithOpenAI(logs []LogEntry, config LLMConfig, configContent string) error {
	// Get model info if available
	modelName := config.Model
	if modelName == "" {
		modelName = getDefaultModel(config.Provider)
	}
	
	// Try to get the human-friendly model name
	modelInfo, found := GetModelInfo(config.Provider, modelName)
	if found {
		fmt.Printf("Analyzing logs with %s API using %s (%s)...\n", 
			config.Provider, modelInfo.Name, modelName)
	} else {
		fmt.Printf("Analyzing logs with %s API using %s...\n", 
			config.Provider, modelName)
	}

	// Prepare prompts and logs
	prompt, err := prepareAnalysisPrompts(logs, config, configContent)
	if err != nil {
		return err
	}

	// Default model if not specified
	modelToUse := config.Model
	if modelToUse == "" {
		modelToUse = getDefaultModel(config.Provider)
	}

	// Create messages array for OpenAI (system message first, then user message)
	messages := []OpenAIMessage{
		{
			Role:    "system",
			Content: prompt.SystemPrompt,
		},
		{
			Role:    "user",
			Content: prompt.UserPrompt,
		},
	}

	// Create the request
	request := OpenAIRequest{
		Model:       modelToUse,
		Messages:    messages,
		Temperature: 0.3,
		MaxTokens:   4000,
	}

	// Convert request to JSON
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(requestJSON))
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// Send request
	fmt.Println("Sending request to OpenAI API...")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request to OpenAI API: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %v", err)
	}

	// Check if response is successful
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error from OpenAI API: %s", string(body))
	}

	// Parse response
	var openaiResponse OpenAIResponse
	err = json.Unmarshal(body, &openaiResponse)
	if err != nil {
		return fmt.Errorf("error parsing response: %v", err)
	}

	// Check for API error
	if openaiResponse.Error != nil {
		return fmt.Errorf("OpenAI API error: %s (type: %s, code: %s)",
			openaiResponse.Error.Message,
			openaiResponse.Error.Type,
			openaiResponse.Error.Code)
	}

	// Extract the content from the response
	if len(openaiResponse.Choices) == 0 {
		return fmt.Errorf("no completions returned from OpenAI API")
	}

	// Get the analysis text from the response
	analysisText := openaiResponse.Choices[0].Message.Content

	// Show token usage for OpenAI
	fmt.Printf("Token usage - Prompt: %d, Completion: %d, Total: %d\n",
		openaiResponse.Usage.PromptTokens,
		openaiResponse.Usage.CompletionTokens,
		openaiResponse.Usage.TotalTokens)

	// Display the analysis and handle clipboard copy
	return displayAndCopyAnalysis(analysisText)
}