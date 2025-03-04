package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// LogEntry represents a parsed log entry from Mattermost logs
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
	User      string    `json:"user,omitempty"`
	Details   string    `json:"details,omitempty"`
}

// parseLogFile reads and parses a Mattermost log file, applying filters
func parseLogFile(filePath, searchTerm, levelFilter, userFilter string) ([]LogEntry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var logs []LogEntry
	scanner := bufio.NewScanner(file)
	
	// Use a larger buffer for potentially long log lines
	const maxCapacity = 512 * 1024 // 512KB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		line := scanner.Text()
		entry, err := parseLine(line)
		
		// Skip lines that couldn't be parsed
		if err != nil {
			continue
		}

		// Apply filters
		if shouldIncludeEntry(entry, searchTerm, levelFilter, userFilter) {
			logs = append(logs, entry)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return logs, nil
}

// parseLine attempts to parse a single log line into a LogEntry
func parseLine(line string) (LogEntry, error) {
	var entry LogEntry

	// Basic format detection and parsing
	// Example format: 2023-04-15T14:22:34.123Z [INFO] api.user.login.success
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 3 {
		return entry, fmt.Errorf("invalid log format")
	}

	// Parse timestamp
	timestamp, err := parseTimestamp(parts[0])
	if err != nil {
		return entry, err
	}
	entry.Timestamp = timestamp

	// Parse log level
	levelPart := parts[1]
	if len(levelPart) < 3 {
		return entry, fmt.Errorf("invalid level format")
	}
	entry.Level = strings.Trim(levelPart, "[]")

	// Parse message and additional details
	messagePart := parts[2]
	
	// Extract user if present
	if userStart := strings.Index(messagePart, "user_id="); userStart != -1 {
		userEnd := strings.Index(messagePart[userStart:], " ")
		if userEnd == -1 {
			userEnd = len(messagePart) - userStart
		}
		userInfo := messagePart[userStart : userStart+userEnd]
		entry.User = strings.TrimPrefix(userInfo, "user_id=")
	}

	// Extract source and message
	if sourceEnd := strings.Index(messagePart, " "); sourceEnd != -1 {
		entry.Source = messagePart[:sourceEnd]
		entry.Message = messagePart[sourceEnd+1:]
	} else {
		entry.Message = messagePart
	}

	return entry, nil
}

// parseTimestamp attempts to parse a timestamp string into a time.Time
func parseTimestamp(timestampStr string) (time.Time, error) {
	// Try common Mattermost timestamp formats
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05.000Z",
		"2006/01/02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestampStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", timestampStr)
}

// shouldIncludeEntry checks if a log entry matches all the specified filters
func shouldIncludeEntry(entry LogEntry, searchTerm, levelFilter, userFilter string) bool {
	// Apply level filter
	if levelFilter != "" && !strings.EqualFold(entry.Level, levelFilter) {
		return false
	}

	// Apply user filter
	if userFilter != "" && !strings.Contains(strings.ToLower(entry.User), strings.ToLower(userFilter)) {
		return false
	}

	// Apply search term filter
	if searchTerm != "" {
		searchLower := strings.ToLower(searchTerm)
		messageLower := strings.ToLower(entry.Message)
		sourceLower := strings.ToLower(entry.Source)
		
		if !strings.Contains(messageLower, searchLower) && 
		   !strings.Contains(sourceLower, searchLower) && 
		   !strings.Contains(strings.ToLower(entry.Details), searchLower) {
			return false
		}
	}

	return true
}
