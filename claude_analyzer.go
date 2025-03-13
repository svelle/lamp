package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	claudeAPIURL = "https://api.anthropic.com/v1/messages"
	defaultMaxLogEntries = 100 // Default limit for logs to send to Claude
)

// ClaudeRequest represents the request structure for Claude API
type ClaudeRequest struct {
	Model       string    `json:"model"`
	MaxTokens   int       `json:"max_tokens"`
	Messages    []Message `json:"messages"`
	System      string    `json:"system"`
	Temperature float64   `json:"temperature"`
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
func analyzeWithClaude(logs []LogEntry, apiKey string, maxEntries int, problemStatement string) {
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
		logText.WriteString(fmt.Sprintf("%d. [%s] [%s] %s: %s\n", 
			i+1,
			log.Timestamp.Format("2006-01-02 15:04:05"),
			log.Level,
			log.Source,
			log.Message))
		
		if log.User != "" {
			logText.WriteString(fmt.Sprintf("   User: %s\n", log.User))
		}
		if log.Caller != "" {
			logText.WriteString(fmt.Sprintf("   Caller: %s\n", log.Caller))
		}
		if log.Details != "" {
			logText.WriteString(fmt.Sprintf("   Details: %s\n", log.Details))
		}
		logText.WriteString("\n")
	}
	
	// Create the system prompt
	systemPrompt := `You are an expert log analyzer for Mattermost server logs. 
Analyze the provided logs and provide a comprehensive report including:

1. A high-level summary of what's happening in the logs
2. Identification of any errors, warnings, or critical issues
3. Patterns or trends you notice
4. Potential root causes for any problems
5. Recommendations for further investigation or resolution

Focus on actionable insights and be specific about what you find.`

	// Create the user prompt
	userPrompt := ""
	if problemStatement != "" {
		userPrompt = fmt.Sprintf("I'm investigating this problem: %s\n\nHere are %d Mattermost server log entries to analyze:\n\n%s\n\nPlease provide a detailed analysis of these logs focusing on the problem I described.", 
			problemStatement, len(logsToAnalyze), logText.String())
	} else {
		userPrompt = fmt.Sprintf("Here are %d Mattermost server log entries to analyze:\n\n%s\n\nPlease provide a detailed analysis of these logs.", 
			len(logsToAnalyze), logText.String())
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
	
	// Display analysis
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("CLAUDE AI LOG ANALYSIS")
	fmt.Println(strings.Repeat("=", 80))
	
	for _, content := range claudeResponse.Content {
		if content.Type == "text" {
			fmt.Println(content.Text)
		}
	}
	
	fmt.Println(strings.Repeat("=", 80))
}
