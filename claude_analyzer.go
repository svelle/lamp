package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/atotto/clipboard"
)

const (
	claudeAPIURL         = "https://api.anthropic.com/v1/messages"
	defaultMaxLogEntries = 100 // Default limit for logs to send to Claude
)

// ClaudeRequest represents the request structure for Claude API
type ClaudeRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	Messages    []Message       `json:"messages"`
	System      string          `json:"system"`
	Temperature float64         `json:"temperature"`
	Thinking    *ThinkingConfig `json:"thinking,omitempty"`
}

// ThinkingConfig represents the configuration for Claude's thinking mode
type ThinkingConfig struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens"`
}

// Message represents a message in the Claude API request
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeResponse represents the response structure from Claude API
type ClaudeResponse struct {
	Content []ContentBlock `json:"content"`
	ID      string         `json:"id"`
	Model   string         `json:"model"`
	Type    string         `json:"type"`
	Error   *ClaudeError   `json:"error,omitempty"`
}

// ContentBlock represents a content block in the Claude API response
type ContentBlock struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

// ClaudeError represents an error from the Claude API
type ClaudeError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// analyzeWithClaude sends log data to Claude API for analysis
func analyzeWithClaude(logs []LogEntry, apiKey string, maxEntries int, problemStatement string, thinkingBudget int) {
	fmt.Println("Analyzing logs with Claude Sonnet API...")

	// If maxEntries is not set (0), use the default
	if maxEntries <= 0 {
		maxEntries = defaultMaxLogEntries
	}

	// Prepare logs for Claude
	logsToAnalyze := logs
	if len(logs) > maxEntries {
		fmt.Printf("Limiting analysis to %d most recent log entries (out of %d total)\n",
			maxEntries, len(logs))
		// Sort logs by timestamp (most recent first)
		// This is a simple approach - in a real implementation, you might want to
		// use a more sophisticated selection strategy
		logsToAnalyze = logs[len(logs)-maxEntries:]
	}

	// Format logs for Claude
	var logText strings.Builder
	for i, log := range logsToAnalyze {
		// Add count information for entries with duplicates
		if log.DuplicateCount > 1 {
			logText.WriteString(fmt.Sprintf("%d. [%s] [%s] %s: %s (repeated %d times)\n",
				i+1,
				log.Timestamp.Format("2006-01-02 15:04:05"),
				log.Level,
				log.Source,
				log.Message,
				log.DuplicateCount))
		} else {
			logText.WriteString(fmt.Sprintf("%d. [%s] [%s] %s: %s\n",
				i+1,
				log.Timestamp.Format("2006-01-02 15:04:05"),
				log.Level,
				log.Source,
				log.Message))
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

	// Create the system prompt
	systemPrompt := `You are an expert log analyzer for Mattermost server logs. 
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

	// Use a more concise prompt for Claude 3.7 Sonnet with thinking mode
	if thinkingBudget > 0 {
		systemPrompt = `You are an expert log analyzer for Mattermost server logs. Analyze these logs and identify issues, patterns, and solutions. Format your entire response in Markdown.`

		// Check if we have logs with duplicate counts
		hasDuplicates := false
		for _, log := range logsToAnalyze {
			if log.DuplicateCount > 1 {
				hasDuplicates = true
				break
			}
		}

		// Add information about duplicates in the prompt
		if hasDuplicates {
			systemPrompt += ` Some log entries may be marked with repetition counts, indicating they appeared multiple times.`
		}
	}

	// Create the user prompt
	userPrompt := ""

	// Count total entries including duplicates
	totalEntries := 0
	hasDuplicates := false
	for _, log := range logsToAnalyze {
		if log.DuplicateCount > 1 {
			hasDuplicates = true
			totalEntries += log.DuplicateCount
		} else {
			totalEntries += 1
		}
	}

	// Create appropriate preface based on duplication
	entryDescription := fmt.Sprintf("%d Mattermost server log entries", len(logsToAnalyze))
	if hasDuplicates {
		entryDescription = fmt.Sprintf("%d unique Mattermost server log entries representing %d total log entries",
			len(logsToAnalyze), totalEntries)
	}

	if problemStatement != "" {
		if thinkingBudget > 0 {
			userPrompt = fmt.Sprintf("I'm investigating this problem: %s\n\nHere are %s to analyze:\n\n%s",
				problemStatement, entryDescription, logText.String())
		} else {
			userPrompt = fmt.Sprintf("I'm investigating this problem: %s\n\nHere are %s to analyze:\n\n%s\n\nPlease provide a detailed analysis of these logs focusing on the problem I described.",
				problemStatement, entryDescription, logText.String())
		}
	} else {
		if thinkingBudget > 0 {
			userPrompt = fmt.Sprintf("Here are %s to analyze:\n\n%s",
				entryDescription, logText.String())
		} else {
			userPrompt = fmt.Sprintf("Here are %s to analyze:\n\n%s\n\nPlease provide a detailed analysis of these logs.",
				entryDescription, logText.String())
		}
	}

	// Create the request
	request := ClaudeRequest{
		Model:     "claude-3-5-haiku-latest",
		MaxTokens: 4000,
		Messages: []Message{
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
		System:      systemPrompt,
		Temperature: 0.3,
	}

	// Enable thinking mode if thinkingBudget is set
	if thinkingBudget > 0 {
		request.Model = "claude-3-7-sonnet-latest"

		// Ensure max_tokens is larger than thinking budget (Claude requirement)
		responseTokens := 4000 // Default tokens for actual response
		request.MaxTokens = thinkingBudget + responseTokens

		// Set temperature to 1 when thinking is enabled (Claude requirement)
		request.Temperature = 1.0

		request.Thinking = &ThinkingConfig{
			Type:         "enabled",
			BudgetTokens: thinkingBudget,
		}
		fmt.Printf("Extended thinking mode enabled with %d tokens budget (total max tokens: %d)\n",
			thinkingBudget, request.MaxTokens)
	}

	// Convert request to JSON
	requestJSON, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", claudeAPIURL, bytes.NewBuffer(requestJSON))
	if err != nil {
		fmt.Printf("Error creating HTTP request: %v\n", err)
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// Send request
	fmt.Println("Sending request to Claude API...")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request to Claude API: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	// Check if response is successful
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error from Claude API: %s\n", string(body))
		return
	}

	// Parse response
	var claudeResponse ClaudeResponse
	err = json.Unmarshal(body, &claudeResponse)
	if err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		return
	}

	// Check for API error
	if claudeResponse.Error != nil {
		fmt.Printf("Claude API error: %s - %s\n",
			claudeResponse.Error.Type,
			claudeResponse.Error.Message)
		return
	}

	// Capture analysis output in a buffer for potential clipboard copy
	var analysisBuffer strings.Builder

	// Add markdown header to buffer
	analysisBuffer.WriteString("# CLAUDE AI LOG ANALYSIS\n\n")

	// Check if we're using extended thinking mode
	if thinkingBudget > 0 {
		// Look for thinking content and final answer
		var thinkingOutput, finalAnswer string
		for _, content := range claudeResponse.Content {
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

		// Add thinking output to buffer if available
		if thinkingOutput != "" {
			analysisBuffer.WriteString("## CLAUDE THINKING PROCESS\n\n")
			analysisBuffer.WriteString(thinkingOutput)
			analysisBuffer.WriteString("\n\n## FINAL ANALYSIS\n\n")
		}

		// Add final answer to buffer
		if finalAnswer != "" {
			analysisBuffer.WriteString(finalAnswer)
		} else {
			// If there's no separate final answer, add all content
			for _, content := range claudeResponse.Content {
				if content.Type == "text" {
					analysisBuffer.WriteString(content.Text)
				}
			}
		}
	} else {
		// Standard mode - add all content to buffer
		for _, content := range claudeResponse.Content {
			if content.Type == "text" {
				analysisBuffer.WriteString(content.Text)
			}
		}
	}

	// Display the analysis
	fmt.Println("\n" + analysisBuffer.String())

	// Prompt the user to copy to clipboard
	fmt.Println("\n-------------------------------------------------")
	fmt.Println("The analysis above is formatted in Markdown.")
	fmt.Print("Would you like to copy it to your clipboard? (y/n): ")

	// Read user input
	var response string
	_, err = fmt.Scanln(&response)
	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}

	// Check if user wants to copy to clipboard
	if strings.ToLower(response) == "y" || strings.ToLower(response) == "yes" {
		err = clipboard.WriteAll(analysisBuffer.String())
		if err != nil {
			fmt.Println("Error copying to clipboard:", err)
		} else {
			fmt.Println("Analysis copied to clipboard!")
		}
	}
}
