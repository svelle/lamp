package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
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
	Caller    string    `json:"caller,omitempty"`
}

// JSONLogEntry represents a JSON-formatted log entry
type JSONLogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Msg       string                 `json:"msg"`
	Caller    string                 `json:"caller,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	Extra     map[string]interface{} `json:"-"`
}

// parseLogFile reads and parses a Mattermost log file, applying filters
func parseLogFile(filePath, searchTerm, regexPattern, levelFilter, userFilter, startTimeStr, endTimeStr string) ([]LogEntry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Parse time range filters if provided
	var startTime, endTime time.Time
	if startTimeStr != "" {
		parsedTime, parseErr := time.Parse("2006-01-02T15:04:05", startTimeStr)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid start time format: %v", parseErr)
		}
		startTime = parsedTime
	}
	if endTimeStr != "" {
		parsedTime, parseErr := time.Parse("2006-01-02T15:04:05", endTimeStr)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid end time format: %v", parseErr)
		}
		endTime = parsedTime
	}

	// Compile regex if provided
	var regex *regexp.Regexp
	if regexPattern != "" {
		regex, err = regexp.Compile(regexPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %v", err)
		}
	}

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
		if shouldIncludeEntry(entry, searchTerm, regex, levelFilter, userFilter, startTime, endTime) {
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

	// Check if the line is in JSON format
	if strings.HasPrefix(strings.TrimSpace(line), "{") {
		return parseJSONLine(line)
	}

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

// parseJSONLine parses a JSON-formatted log line
func parseJSONLine(line string) (LogEntry, error) {
	var entry LogEntry
	var jsonEntry JSONLogEntry

	// Unmarshal the JSON log entry
	if err := json.Unmarshal([]byte(line), &jsonEntry); err != nil {
		return entry, fmt.Errorf("failed to parse JSON log: %v", err)
	}

	// Extract additional fields
	var extra map[string]interface{}
	if err := json.Unmarshal([]byte(line), &extra); err == nil {
		// Build details string from extra fields
		var details strings.Builder
		for k, v := range extra {
			// Skip fields we already handle
			if k == "timestamp" || k == "level" || k == "msg" || k == "caller" || k == "user_id" {
				continue
			}
			if details.Len() > 0 {
				details.WriteString(", ")
			}
			details.WriteString(fmt.Sprintf("%s=%v", k, v))
		}
		entry.Details = details.String()
	}

	// Parse timestamp
	timestamp, err := parseTimestamp(jsonEntry.Timestamp)
	if err != nil {
		return entry, err
	}
	entry.Timestamp = timestamp
	
	// Set other fields
	entry.Level = jsonEntry.Level
	entry.Message = jsonEntry.Msg
	entry.User = jsonEntry.UserID
	
	// Set source from caller if available
	if jsonEntry.Caller != "" {
		entry.Source = jsonEntry.Caller
		entry.Caller = jsonEntry.Caller
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
		"2006-01-02 15:04:05.000 Z", // Format from the example JSON log
		"2006-01-02 15:04:05.000 MST",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestampStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", timestampStr)
}

// trimDuplicateLogInfo removes log entries that contain duplicate information
func trimDuplicateLogInfo(logs []LogEntry) []LogEntry {
	if len(logs) == 0 {
		return logs
	}
	
	// Map to track seen log signatures
	seen := make(map[string]bool)
	var result []LogEntry
	
	for _, entry := range logs {
		// Create a signature of the log based on level, source, and the core message
		// Strip timestamps, specific IDs, and other variable data
		msgCore := entry.Message
		
		// Replace specific patterns that often change but don't affect the message meaning
		// Like IDs, timestamps, specific values
		re := regexp.MustCompile(`\b[0-9a-f]{8,32}\b`) // Match typical IDs
		msgCore = re.ReplaceAllString(msgCore, "ID")
		
		re = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`) // Match dates
		msgCore = re.ReplaceAllString(msgCore, "DATE")
		
		re = regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`) // Match IP addresses
		msgCore = re.ReplaceAllString(msgCore, "IP")
		
		re = regexp.MustCompile(`\d+ms|\d+s|\d+ns`) // Match time durations
		msgCore = re.ReplaceAllString(msgCore, "DURATION")
		
		// Create a signature combining important fields
		signature := fmt.Sprintf("%s:%s:%s", entry.Level, entry.Source, msgCore)
		
		// If we haven't seen this signature, add it to results
		if !seen[signature] {
			seen[signature] = true
			result = append(result, entry)
		}
	}
	
	return result
}

// shouldIncludeEntry checks if a log entry matches all the specified filters
func shouldIncludeEntry(entry LogEntry, searchTerm string, regex *regexp.Regexp, levelFilter, userFilter string, startTime, endTime time.Time) bool {
	// Apply level filter
	if levelFilter != "" && !strings.EqualFold(entry.Level, levelFilter) {
		return false
	}

	// Apply user filter
	if userFilter != "" && !strings.Contains(strings.ToLower(entry.User), strings.ToLower(userFilter)) {
		return false
	}

	// Apply time range filters
	if !startTime.IsZero() && entry.Timestamp.Before(startTime) {
		return false
	}
	if !endTime.IsZero() && entry.Timestamp.After(endTime) {
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

	// Apply regex filter
	if regex != nil {
		// Check if regex matches any field
		if !regex.MatchString(entry.Message) && 
		   !regex.MatchString(entry.Source) && 
		   !regex.MatchString(entry.Details) && 
		   !regex.MatchString(entry.User) {
			return false
		}
	}

	return true
}
