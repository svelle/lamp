package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// LogAnalysis contains statistics and insights from log entries
type LogAnalysis struct {
	TotalEntries        int
	TimeRange           TimeRange
	LevelCounts         map[string]int
	TopSources          []CountedItem
	TopUsers            []CountedItem
	TopErrorMessages    []CountedItem
	ErrorRate           float64
	BusiestHours        []CountedItem
	ActivityByDayOfWeek []CountedItem
	ActivityByMonth     []CountedItem
	HourLevelCounts     map[int]map[string]int    // Hour -> Level -> Count
	DayLevelCounts      map[string]map[string]int // Day -> Level -> Count
	MonthLevelCounts    map[string]map[string]int // Month -> Level -> Count
	CommonPatterns      []CountedItem
	NotificationTypes   []CountedItem   // For notification logs: message, clear, etc.
	NotificationStatuses []CountedItem  // For notification logs: Sent, Received, etc.
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
func analyzeAndDisplayStats(logs []LogEntry, writer io.Writer, showDupes bool) {
	if len(logs) == 0 {
		fmt.Fprintln(writer, "No log entries to analyze.")
		return
	}

	// Check if any logs have duplicate counts
	hasDuplicateCounts := false
	uniqueEntries := len(logs)
	totalEntries := 0

	for _, log := range logs {
		count := log.DuplicateCount
		if count > 1 {
			hasDuplicateCounts = true
		}

		if count == 0 {
			count = 1
		}
		totalEntries += count
	}

	// Only consider logs deduplicated if they actually have duplicate counts AND showDupes is true
	isDeduplicated := hasDuplicateCounts && totalEntries > uniqueEntries && showDupes

	analysis := analyzeLogs(logs, showDupes)
	displayAnalysis(analysis, writer, isDeduplicated, uniqueEntries)
}

// analyzeLogs performs analysis on log entries
func analyzeLogs(logs []LogEntry, showDupes bool) LogAnalysis {
	analysis := LogAnalysis{
		TotalEntries:     len(logs),
		LevelCounts:      make(map[string]int),
		HourLevelCounts:  make(map[int]map[string]int),
		DayLevelCounts:   make(map[string]map[string]int),
		MonthLevelCounts: make(map[string]map[string]int),
	}

	// Initialize maps for counting
	sourceCounts := make(map[string]int)
	userCounts := make(map[string]int)
	errorMsgCounts := make(map[string]int)
	hourCounts := make(map[int]int)
	dayOfWeekCounts := make(map[string]int)
	monthCounts := make(map[string]int)
	patternCounts := make(map[string]int)
	notificationTypeCounts := make(map[string]int)
	notificationStatusCounts := make(map[string]int)

	// Set initial time range
	if len(logs) > 0 {
		analysis.TimeRange.Start = logs[0].Timestamp
		analysis.TimeRange.End = logs[0].Timestamp
	}

	// Track total entries including duplicates
	totalWithDuplicates := 0

	// Process each log entry
	for _, log := range logs {
		// Get the count (either the duplicate count or 1 if not set)
		count := 1
		if showDupes && log.DuplicateCount > 1 {
			count = log.DuplicateCount
		}
		totalWithDuplicates += count
		// Update time range
		if log.Timestamp.Before(analysis.TimeRange.Start) {
			analysis.TimeRange.Start = log.Timestamp
		}
		if log.Timestamp.After(analysis.TimeRange.End) {
			analysis.TimeRange.End = log.Timestamp
		}

		// Count log levels
		level := strings.ToUpper(log.Level)
		analysis.LevelCounts[level] += count

		// Count sources
		if log.Source != "" {
			sourceCounts[log.Source] += count
		}

		// Count users
		if log.User != "" {
			userCounts[log.User] += count
		}

		// Count error messages
		if strings.EqualFold(log.Level, "error") || strings.EqualFold(log.Level, "fatal") {
			// Get first 50 chars of message or full message if shorter
			shortMsg := log.Message
			if len(shortMsg) > 50 {
				shortMsg = shortMsg[:50] + "..."
			}
			errorMsgCounts[shortMsg] += count
		}

		// Count activity by hour
		hour := log.Timestamp.Hour()
		hourCounts[hour] += count
		
		// Track level distribution by hour
		if _, exists := analysis.HourLevelCounts[hour]; !exists {
			analysis.HourLevelCounts[hour] = make(map[string]int)
		}
		analysis.HourLevelCounts[hour][level] += count
		
		// Count activity by day of week
		dayOfWeek := log.Timestamp.Weekday().String()
		dayOfWeekCounts[dayOfWeek] += count
		
		// Track level distribution by day of week
		if _, exists := analysis.DayLevelCounts[dayOfWeek]; !exists {
			analysis.DayLevelCounts[dayOfWeek] = make(map[string]int)
		}
		analysis.DayLevelCounts[dayOfWeek][level] += count
		
		// Count activity by month
		month := log.Timestamp.Month().String()
		monthCounts[month] += count
		
		// Track level distribution by month
		if _, exists := analysis.MonthLevelCounts[month]; !exists {
			analysis.MonthLevelCounts[month] = make(map[string]int)
		}
		analysis.MonthLevelCounts[month][level] += count

		// Identify common patterns in messages
		words := strings.Fields(log.Message)
		if len(words) > 0 {
			pattern := words[0]
			if len(words) > 1 {
				pattern += " " + words[1]
			}
			patternCounts[pattern] += count
		}
		
		// Count notification types and statuses if present
		if log.LogSource == "notifications" {
			if log.Type != "" {
				notificationTypeCounts[log.Type] += count
			}
			if log.Status != "" {
				notificationStatusCounts[log.Status] += count
			}
		}
	}

	// Calculate error rate
	errorCount := analysis.LevelCounts["ERROR"] + analysis.LevelCounts["FATAL"]
	analysis.ErrorRate = float64(errorCount) / float64(totalWithDuplicates) * 100

	// Update total entries to include duplicates
	analysis.TotalEntries = totalWithDuplicates

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
	
	// Add day of week and month activity
	analysis.ActivityByDayOfWeek = mapToSortedSlice(dayOfWeekCounts, 7)
	analysis.ActivityByMonth = mapToSortedSlice(monthCounts, 12)

	analysis.CommonPatterns = mapToSortedSlice(patternCounts, 10)
	
	// Add notification-specific information if present
	analysis.NotificationTypes = mapToSortedSlice(notificationTypeCounts, 10) 
	analysis.NotificationStatuses = mapToSortedSlice(notificationStatusCounts, 10)

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

// getDominantLevelColor returns the color of the most common log level for a period
func getDominantLevelColor(levelCounts map[string]int, totalCount int) string {
	if totalCount == 0 {
		return "\033[0m" // Reset color if no entries
	}
	
	// Define log level colors
	levelColors := map[string]string{
		"ERROR":    "\033[31m", // Red
		"FATAL":    "\033[31m", // Red
		"CRITICAL": "\033[31m", // Red
		"WARN":     "\033[33m", // Yellow
		"WARNING":  "\033[33m", // Yellow
		"INFO":     "\033[32m", // Green
		"DEBUG":    "\033[34m", // Blue
	}
	
	// Find the dominant level (highest percentage)
	var dominantLevel string
	highestCount := 0
	
	for level, count := range levelCounts {
		if count > highestCount {
			highestCount = count
			dominantLevel = level
		}
	}
	
	// Calculate percentage of dominant level
	percentage := float64(highestCount) / float64(totalCount) * 100
	
	// Only color if the dominant level represents at least 50% of entries
	if percentage >= 50 {
		if color, exists := levelColors[dominantLevel]; exists {
			return color
		}
	}
	
	return "\033[0m" // Default to reset color
}

// displayAnalysis prints the analysis results
func displayAnalysis(analysis LogAnalysis, writer io.Writer, isDeduplicated bool, uniqueEntries int) {
	// ANSI color codes
	headerColor := "\033[1;36m"    // Bold Cyan
	subHeaderColor := "\033[1;33m" // Bold Yellow
	resetColor := "\033[0m"

	fmt.Fprintf(writer, "\n%s=== MATTERMOST LOG ANALYSIS ===%s\n\n", headerColor, resetColor)

	// Basic statistics
	fmt.Fprintf(writer, "%sBasic Statistics:%s\n", subHeaderColor, resetColor)
	if isDeduplicated {
		fmt.Fprintf(writer, "Unique Log Entries: %d\n", uniqueEntries)
		fmt.Fprintf(writer, "Total Log Entries: %d (including %d duplicates)\n",
			analysis.TotalEntries, analysis.TotalEntries-uniqueEntries)
		fmt.Fprintf(writer, "Deduplication Ratio: %.2f:1\n", float64(analysis.TotalEntries)/float64(uniqueEntries))
	} else {
		fmt.Fprintf(writer, "Total Log Entries: %d\n", analysis.TotalEntries)
	}
	fmt.Fprintf(writer, "Time Range: %s to %s\n",
		analysis.TimeRange.Start.Format("2006-01-02 15:04:05"),
		analysis.TimeRange.End.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(writer, "Duration: %s\n\n", analysis.TimeRange.End.Sub(analysis.TimeRange.Start).Round(time.Second))

	// Log level distribution
	fmt.Fprintf(writer, "%sLog Level Distribution:%s\n", subHeaderColor, resetColor)
	for level, count := range analysis.LevelCounts {
		percentage := float64(count) / float64(analysis.TotalEntries) * 100
		levelColor := getLevelColor(level)
		fmt.Fprintf(writer, "%s%s%s: %d (%.1f%%)\n", levelColor, level, resetColor, count, percentage)
	}
	fmt.Fprintf(writer, "Error Rate: %.2f%%\n\n", analysis.ErrorRate)

	// Top sources
	fmt.Fprintf(writer, "%sTop Log Sources:%s\n", subHeaderColor, resetColor)
	for i, source := range analysis.TopSources {
		if i < 5 {
			fmt.Fprintf(writer, "%d. %s (%d entries)\n", i+1, source.Item, source.Count)
		}
	}
	fmt.Fprintln(writer)

	// Top users (if any)
	if len(analysis.TopUsers) > 0 {
		fmt.Fprintf(writer, "%sTop Active Users:%s\n", subHeaderColor, resetColor)
		for i, user := range analysis.TopUsers {
			if i < 5 {
				fmt.Fprintf(writer, "%d. %s (%d entries)\n", i+1, user.Item, user.Count)
			}
		}
		fmt.Fprintln(writer)
	}

	// Top error messages (if any)
	if len(analysis.TopErrorMessages) > 0 {
		fmt.Fprintf(writer, "%sTop Error Messages:%s\n", subHeaderColor, resetColor)
		for i, err := range analysis.TopErrorMessages {
			if i < 5 {
				fmt.Fprintf(writer, "%d. %s (%d occurrences)\n", i+1, err.Item, err.Count)
			}
		}
		fmt.Fprintln(writer)
	}

	// Busiest hours
	fmt.Fprintf(writer, "%sActivity by Hour:%s\n", subHeaderColor, resetColor)
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
		if _, err := fmt.Sscanf(hour.Item, "%d", &hourNum); err != nil {
			logger.Debug("Invalid hour format in activity analysis", "hour", hour.Item, "error", err)
			// Skip invalid hour entries
			continue
		}
		if hourNum < 0 || hourNum >= 24 {
			logger.Debug("Hour outside valid range", "hour", hourNum)
			// Skip hours outside valid range
			continue
		}
		hourMap[hourNum] = hour.Count
	}

	// Display hours with bar chart
	for hour := 0; hour < 24; hour++ {
		count := hourMap[hour]
		barLength := int(float64(count) / float64(maxCount) * 30)
		bar := strings.Repeat("█", barLength)
		
		// Get dominant log level color for this hour
		levelColor := getDominantLevelColor(analysis.HourLevelCounts[hour], count)
		
		fmt.Fprintf(writer, "%02d:00: %s%s%s (%d)\n", hour, levelColor, bar, resetColor, count)
	}
	fmt.Fprintln(writer)
	
	// Activity by day of week (if time range spans multiple days)
	timeSpan := analysis.TimeRange.End.Sub(analysis.TimeRange.Start)
	if timeSpan.Hours() >= 24 && len(analysis.ActivityByDayOfWeek) > 0 {
		fmt.Fprintf(writer, "%sActivity by Day of Week:%s\n", subHeaderColor, resetColor)
		// Find the max count for scaling
		maxCount = 0
		for _, day := range analysis.ActivityByDayOfWeek {
			if day.Count > maxCount {
				maxCount = day.Count
			}
		}
		
		// Create a map for easier day lookup
		dayMap := make(map[string]int)
		for _, day := range analysis.ActivityByDayOfWeek {
			dayMap[day.Item] = day.Count
		}
		
		// Display days with bar chart (in order from Sunday to Saturday)
		for _, day := range []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"} {
			count := dayMap[day]
			barLength := int(float64(count) / float64(maxCount) * 30)
			bar := strings.Repeat("█", barLength)
			
			// Get dominant log level color for this day
			levelColor := getDominantLevelColor(analysis.DayLevelCounts[day], count)
			
			fmt.Fprintf(writer, "%-9s: %s%s%s (%d)\n", day, levelColor, bar, resetColor, count)
		}
		fmt.Fprintln(writer)
	}
	
	// Activity by month (if time range spans multiple months)
	if timeSpan.Hours() >= 24*30 && len(analysis.ActivityByMonth) > 0 {
		fmt.Fprintf(writer, "%sActivity by Month:%s\n", subHeaderColor, resetColor)
		// Find the max count for scaling
		maxCount = 0
		for _, month := range analysis.ActivityByMonth {
			if month.Count > maxCount {
				maxCount = month.Count
			}
		}
		
		// Create a map for easier month lookup
		monthMap := make(map[string]int)
		for _, month := range analysis.ActivityByMonth {
			monthMap[month.Item] = month.Count
		}
		
		// Display months with bar chart (in calendar order)
		for _, month := range []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"} {
			count := monthMap[month]
			barLength := int(float64(count) / float64(maxCount) * 30)
			bar := strings.Repeat("█", barLength)
			
			// Get dominant log level color for this month
			levelColor := getDominantLevelColor(analysis.MonthLevelCounts[month], count)
			
			fmt.Fprintf(writer, "%-9s: %s%s%s (%d)\n", month, levelColor, bar, resetColor, count)
		}
		fmt.Fprintln(writer)
	}

	// Common message patterns
	fmt.Fprintf(writer, "%sCommon Message Patterns:%s\n", subHeaderColor, resetColor)
	for i, pattern := range analysis.CommonPatterns {
		if i < 5 {
			fmt.Fprintf(writer, "%d. \"%s\" (%d occurrences)\n", i+1, pattern.Item, pattern.Count)
		}
	}
	fmt.Fprintln(writer)
	
	// Notification statistics (if present)
	if len(analysis.NotificationTypes) > 0 {
		fmt.Fprintf(writer, "%sNotification Statistics:%s\n", subHeaderColor, resetColor)
		
		// Notification types
		if len(analysis.NotificationTypes) > 0 {
			fmt.Fprintf(writer, "Notification Types:\n")
			for _, nt := range analysis.NotificationTypes {
				fmt.Fprintf(writer, "  %s: %d\n", nt.Item, nt.Count)
			}
		}
		
		// Notification statuses
		if len(analysis.NotificationStatuses) > 0 {
			fmt.Fprintf(writer, "Notification Statuses:\n")
			for _, ns := range analysis.NotificationStatuses {
				fmt.Fprintf(writer, "  %s: %d\n", ns.Item, ns.Count)
			}
		}
		fmt.Fprintln(writer)
	}

	fmt.Fprintf(writer, "\n%s=== END OF ANALYSIS ===%s\n\n", headerColor, resetColor)
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