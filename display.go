package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ANSI color codes for pretty output
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorBold   = "\033[1m"
)

// displayLogsPretty outputs logs in a human-readable colored format
func displayLogsPretty(logs []LogEntry) {
	if len(logs) == 0 {
		fmt.Println("No log entries found matching your criteria.")
		return
	}

	for _, log := range logs {
		// Format timestamp
		timestamp := log.Timestamp.Format("2006-01-02 15:04:05")
		
		// Color the log level
		var levelColored string
		switch strings.ToUpper(log.Level) {
		case "ERROR", "FATAL", "CRITICAL":
			levelColored = colorRed + log.Level + colorReset
		case "WARN", "WARNING":
			levelColored = colorYellow + log.Level + colorReset
		case "INFO":
			levelColored = colorGreen + log.Level + colorReset
		case "DEBUG":
			levelColored = colorBlue + log.Level + colorReset
		default:
			levelColored = log.Level
		}
		
		// Print the formatted log entry
		fmt.Printf("%s [%s] %s%s%s\n", 
			colorCyan + timestamp + colorReset,
			levelColored,
			colorBold + log.Source + colorReset,
			colorWhite + " → " + colorReset,
			log.Message,
		)
		
		// Print user if available
		if log.User != "" {
			fmt.Printf("  %sUser:%s %s\n", colorPurple, colorReset, log.User)
		}
		
		// Print caller if available
		if log.Caller != "" {
			fmt.Printf("  %sCaller:%s %s\n", colorPurple, colorReset, log.Caller)
		}
		
		// Print details if available
		if log.Details != "" {
			fmt.Printf("  %sDetails:%s %s\n", colorPurple, colorReset, log.Details)
		}
		
		// Add a separator between entries
		fmt.Println(strings.Repeat("-", 80))
	}
	
	// Print summary
	fmt.Printf("\nDisplayed %d log entries\n", len(logs))
}

// displayLogsJSON outputs logs in JSON format
func displayLogsJSON(logs []LogEntry) {
	if len(logs) == 0 {
		fmt.Println("[]")
		return
	}

	output, err := json.MarshalIndent(logs, "", "  ")
	if err != nil {
		fmt.Printf("Error formatting JSON: %v\n", err)
		return
	}
	
	fmt.Println(string(output))
}
