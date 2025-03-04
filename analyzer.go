package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// LogAnalysis contains statistics and insights from log entries
type LogAnalysis struct {
	TotalEntries      int
	TimeRange         TimeRange
	LevelCounts       map[string]int
	TopSources        []CountedItem
	TopUsers          []CountedItem
	TopErrorMessages  []CountedItem
	ErrorRate         float64
	BusiestHours      []CountedItem
	CommonPatterns    []CountedItem
}

// TimeRange represents the time span of analyzed logs
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// CountedItem represents an item with its count
type CountedItem struct {
	Item  string
	Count int
}

// analyzeAndDisplayStats analyzes log entries and displays statistics
func analyzeAndDisplayStats(logs []LogEntry) {
	if len(logs) == 0 {
		fmt.Println("No log entries to analyze.")
		return
	}

	analysis := analyzeLogs(logs)
	displayAnalysis(analysis)
}

// analyzeLogs performs analysis on log entries
func analyzeLogs(logs []LogEntry) LogAnalysis {
	analysis := LogAnalysis{
		TotalEntries: len(logs),
		LevelCounts:  make(map[string]int),
	}

	// Initialize maps for counting
	sourceCounts := make(map[string]int)
	userCounts := make(map[string]int)
	errorMsgCounts := make(map[string]int)
	hourCounts := make(map[int]int)
	patternCounts := make(map[string]int)

	// Set initial time range
	if len(logs) > 0 {
		analysis.TimeRange.Start = logs[0].Timestamp
		analysis.TimeRange.End = logs[0].Timestamp
	}

	// Process each log entry
	for _, log := range logs {
		// Update time range
		if log.Timestamp.Before(analysis.TimeRange.Start) {
			analysis.TimeRange.Start = log.Timestamp
		}
		if log.Timestamp.After(analysis.TimeRange.End) {
			analysis.TimeRange.End = log.Timestamp
		}

		// Count log levels
		analysis.LevelCounts[strings.ToUpper(log.Level)]++

		// Count sources
		if log.Source != "" {
			sourceCounts[log.Source]++
		}

		// Count users
		if log.User != "" {
			userCounts[log.User]++
		}

		// Count error messages
		if strings.EqualFold(log.Level, "error") || strings.EqualFold(log.Level, "fatal") {
			// Get first 50 chars of message or full message if shorter
			shortMsg := log.Message
			if len(shortMsg) > 50 {
				shortMsg = shortMsg[:50] + "..."
			}
			errorMsgCounts[shortMsg]++
		}

		// Count activity by hour
		hour := log.Timestamp.Hour()
		hourCounts[hour]++

		// Identify common patterns in messages
		words := strings.Fields(log.Message)
		if len(words) > 0 {
			pattern := words[0]
			if len(words) > 1 {
				pattern += " " + words[1]
			}
			patternCounts[pattern]++
		}
	}

	// Calculate error rate
	errorCount := analysis.LevelCounts["ERROR"] + analysis.LevelCounts["FATAL"]
	analysis.ErrorRate = float64(errorCount) / float64(analysis.TotalEntries) * 100

	// Convert maps to sorted slices
	analysis.TopSources = mapToSortedSlice(sourceCounts, 10)
	analysis.TopUsers = mapToSortedSlice(userCounts, 10)
	analysis.TopErrorMessages = mapToSortedSlice(errorMsgCounts, 10)
	
	// Convert hourCounts (map[int]int) to string keys for mapToSortedSlice
	hourCountsStr := make(map[string]int)
	for hour, count := range hourCounts {
		hourCountsStr[fmt.Sprintf("%d", hour)] = count
	}
	analysis.BusiestHours = mapToSortedSlice(hourCountsStr, 24)
	
	analysis.CommonPatterns = mapToSortedSlice(patternCounts, 10)

	return analysis
}

// mapToSortedSlice converts a map to a sorted slice of CountedItems
func mapToSortedSlice(m map[string]int, limit int) []CountedItem {
	var items []CountedItem
	for k, v := range m {
		items = append(items, CountedItem{Item: k, Count: v})
	}

	// Sort by count (descending)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Count > items[j].Count
	})

	// Limit the number of items
	if len(items) > limit {
		items = items[:limit]
	}

	return items
}

// displayAnalysis prints the analysis results
func displayAnalysis(analysis LogAnalysis) {
	// ANSI color codes
	headerColor := "\033[1;36m" // Bold Cyan
	subHeaderColor := "\033[1;33m" // Bold Yellow
	resetColor := "\033[0m"
	
	fmt.Printf("\n%s=== MATTERMOST LOG ANALYSIS ===%s\n\n", headerColor, resetColor)
	
	// Basic statistics
	fmt.Printf("%sBasic Statistics:%s\n", subHeaderColor, resetColor)
	fmt.Printf("Total Log Entries: %d\n", analysis.TotalEntries)
	fmt.Printf("Time Range: %s to %s\n", 
		analysis.TimeRange.Start.Format("2006-01-02 15:04:05"),
		analysis.TimeRange.End.Format("2006-01-02 15:04:05"))
	fmt.Printf("Duration: %s\n\n", analysis.TimeRange.End.Sub(analysis.TimeRange.Start).Round(time.Second))
	
	// Log level distribution
	fmt.Printf("%sLog Level Distribution:%s\n", subHeaderColor, resetColor)
	for level, count := range analysis.LevelCounts {
		percentage := float64(count) / float64(analysis.TotalEntries) * 100
		levelColor := getLevelColor(level)
		fmt.Printf("%s%s%s: %d (%.1f%%)\n", levelColor, level, resetColor, count, percentage)
	}
	fmt.Printf("Error Rate: %.2f%%\n\n", analysis.ErrorRate)
	
	// Top sources
	fmt.Printf("%sTop Log Sources:%s\n", subHeaderColor, resetColor)
	for i, source := range analysis.TopSources {
		if i < 5 {
			fmt.Printf("%d. %s (%d entries)\n", i+1, source.Item, source.Count)
		}
	}
	fmt.Println()
	
	// Top users (if any)
	if len(analysis.TopUsers) > 0 {
		fmt.Printf("%sTop Active Users:%s\n", subHeaderColor, resetColor)
		for i, user := range analysis.TopUsers {
			if i < 5 {
				fmt.Printf("%d. %s (%d entries)\n", i+1, user.Item, user.Count)
			}
		}
		fmt.Println()
	}
	
	// Top error messages (if any)
	if len(analysis.TopErrorMessages) > 0 {
		fmt.Printf("%sTop Error Messages:%s\n", subHeaderColor, resetColor)
		for i, err := range analysis.TopErrorMessages {
			if i < 5 {
				fmt.Printf("%d. %s (%d occurrences)\n", i+1, err.Item, err.Count)
			}
		}
		fmt.Println()
	}
	
	// Busiest hours
	fmt.Printf("%sActivity by Hour:%s\n", subHeaderColor, resetColor)
	// Find the max count for scaling
	maxCount := 0
	for _, hour := range analysis.BusiestHours {
		if hour.Count > maxCount {
			maxCount = hour.Count
		}
	}
	
	// Create a map for easier hour lookup
	hourMap := make(map[int]int)
	for _, hour := range analysis.BusiestHours {
		hourNum := 0
		fmt.Sscanf(hour.Item, "%d", &hourNum)
		hourMap[hourNum] = hour.Count
	}
	
	// Display hours with bar chart
	for hour := 0; hour < 24; hour++ {
		count := hourMap[hour]
		barLength := int(float64(count) / float64(maxCount) * 30)
		bar := strings.Repeat("█", barLength)
		fmt.Printf("%02d:00: %s (%d)\n", hour, bar, count)
	}
	fmt.Println()
	
	// Common message patterns
	fmt.Printf("%sCommon Message Patterns:%s\n", subHeaderColor, resetColor)
	for i, pattern := range analysis.CommonPatterns {
		if i < 5 {
			fmt.Printf("%d. \"%s\" (%d occurrences)\n", i+1, pattern.Item, pattern.Count)
		}
	}
	
	fmt.Printf("\n%s=== END OF ANALYSIS ===%s\n\n", headerColor, resetColor)
}

// getLevelColor returns the ANSI color code for a log level
func getLevelColor(level string) string {
	switch strings.ToUpper(level) {
	case "ERROR", "FATAL", "CRITICAL":
		return "\033[31m" // Red
	case "WARN", "WARNING":
		return "\033[33m" // Yellow
	case "INFO":
		return "\033[32m" // Green
	case "DEBUG":
		return "\033[34m" // Blue
	default:
		return "\033[0m" // Reset
	}
}
