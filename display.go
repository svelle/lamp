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
	defer func() { _ = file.Close() }()

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
		_, _ = fmt.Fprintln(writer, "No log entries found matching your criteria.")
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
		_, _ = fmt.Fprintf(writer, "%s [%s] %s%s%s",
			colorCyan+timestamp+colorReset,
			levelColored,
			colorBold+log.Source+colorReset,
			colorWhite+" → "+colorReset,
			log.Message,
		)

		// Print duplicate count if more than 1
		if log.DuplicateCount > 1 {
			_, _ = fmt.Fprintf(writer, " %s(repeated %d times)%s", colorYellow, log.DuplicateCount, colorReset)
		}
		_, _ = fmt.Fprintln(writer)

		// Print user if available
		if log.User != "" {
			_, _ = fmt.Fprintf(writer, "  %sUser:%s %s\n", colorPurple, colorReset, log.User)
		}

		// Print source if available
		if log.Source != "" {
			_, _ = fmt.Fprintf(writer, "  %sSource:%s %s\n", colorPurple, colorReset, log.Source)
		}
		
		// Print notification-specific fields if available
		if log.LogSource == "notifications" {
			_, _ = fmt.Fprintf(writer, "  %sLog Source:%s %s\n", colorPurple, colorReset, log.LogSource)
			
			if log.AckID != "" {
				_, _ = fmt.Fprintf(writer, "  %sAck ID:%s %s\n", colorPurple, colorReset, log.AckID)
			}
			
			if log.Type != "" {
				_, _ = fmt.Fprintf(writer, "  %sType:%s %s\n", colorPurple, colorReset, log.Type)
			}
			
			if log.Status != "" {
				_, _ = fmt.Fprintf(writer, "  %sStatus:%s %s\n", colorPurple, colorReset, log.Status)
			}
		}

		// Print extras if available
		for key, value := range log.Extras {
			_, _ = fmt.Fprintf(writer, "  %s%s:%s %s\n", colorPurple, key, colorReset, value)
		}

		// Add a separator between entries
		_, _ = fmt.Fprintln(writer, strings.Repeat("-", 80))
	}

	// Print summary
	_, _ = fmt.Fprintf(writer, "\nDisplayed %d log entries\n", len(logs))
}

// displayLogsJSON outputs logs in JSON format
func displayLogsJSON(logs []LogEntry, writer io.Writer) {
	if len(logs) == 0 {
		_, _ = fmt.Fprintln(writer, "[]")
		return
	}

	output, err := json.MarshalIndent(logs, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(writer, "Error formatting JSON: %v\n", err)
		return
	}

	_, _ = fmt.Fprintln(writer, string(output))
}

// exportToCSV exports log entries to a CSV file
func exportToCSV(logs []LogEntry, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Timestamp", "Level", "Source", "Message", "User", "LogSource", "AckID", "Type", "Status", "Extras"}
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
			log.LogSource,
			log.AckID,
			log.Type,
			log.Status,
			log.ExtrasToString(),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}
