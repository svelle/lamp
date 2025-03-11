package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// writeLogsToJSON writes log entries to a JSON file
func writeLogsToJSON(logs []LogEntry, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(logs); err != nil {
		return err
	}

	return nil
}

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
func displayLogsPretty(logs []LogEntry, writer io.Writer) {
	if len(logs) == 0 {
		fmt.Fprintln(writer, "No log entries found matching your criteria.")
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
		fmt.Fprintf(writer, "%s [%s] %s%s%s\n", 
			colorCyan + timestamp + colorReset,
			levelColored,
			colorBold + log.Source + colorReset,
			colorWhite + " → " + colorReset,
			log.Message,
		)
		
		// Print user if available
		if log.User != "" {
			fmt.Fprintf(writer, "  %sUser:%s %s\n", colorPurple, colorReset, log.User)
		}
		
		// Print caller if available
		if log.Caller != "" {
			fmt.Fprintf(writer, "  %sCaller:%s %s\n", colorPurple, colorReset, log.Caller)
		}
		
		// Print details if available
		if log.Details != "" {
			fmt.Fprintf(writer, "  %sDetails:%s %s\n", colorPurple, colorReset, log.Details)
		}
		
		// Add a separator between entries
		fmt.Fprintln(writer, strings.Repeat("-", 80))
	}
	
	// Print summary
	fmt.Fprintf(writer, "\nDisplayed %d log entries\n", len(logs))
}

// displayLogsJSON outputs logs in JSON format
func displayLogsJSON(logs []LogEntry, writer io.Writer) {
	if len(logs) == 0 {
		fmt.Fprintln(writer, "[]")
		return
	}

	output, err := json.MarshalIndent(logs, "", "  ")
	if err != nil {
		fmt.Fprintf(writer, "Error formatting JSON: %v\n", err)
		return
	}
	
	fmt.Fprintln(writer, string(output))
}

// exportToCSV exports log entries to a CSV file
func exportToCSV(logs []LogEntry, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Timestamp", "Level", "Source", "Message", "User", "Caller", "Details"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write data
	for _, log := range logs {
		row := []string{
			log.Timestamp.Format(time.RFC3339),
			log.Level,
			log.Source,
			log.Message,
			log.User,
			log.Caller,
			log.Details,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}
