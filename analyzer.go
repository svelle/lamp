package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// ANSI color constants
const (
	colorHeaderBold = "\033[1;36m" // Bold Cyan
	colorSubHeader  = "\033[1;33m" // Bold Yellow
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
func analyzeAndDisplayStats(logs []LogEntry, writer io.Writer, showDupes bool, verboseAnalysis bool) {
	if len(logs) == 0 {
		_, _ = fmt.Fprintln(writer, "No log entries to analyze.")
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
	displayAnalysis(analysis, writer, isDeduplicated, uniqueEntries, verboseAnalysis)
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

// formatHeaderStats formats the header statistics line
func formatHeaderStats(totalEntries int, isDeduplicated bool, uniqueEntries int, duration time.Duration, errorRate float64) string {
	if isDeduplicated {
		return fmt.Sprintf("%d entries (%d unique) • %s • Error rate: %.1f%%", 
			totalEntries, uniqueEntries, duration, errorRate)
	} else {
		return fmt.Sprintf("%d entries • %s • Error rate: %.1f%%", 
			totalEntries, duration, errorRate)
	}
}

// formatLevelDistribution formats the log level distribution line
func formatLevelDistribution(levelCounts map[string]int, totalEntries int, showPercentages bool) string {
	var parts []string
	for level, count := range levelCounts {
		levelColor := getLevelColor(level)
		if showPercentages {
			percentage := float64(count) / float64(totalEntries) * 100
			parts = append(parts, fmt.Sprintf("%s%s%s:%d(%.0f%%)", levelColor, level, colorReset, count, percentage))
		} else {
			parts = append(parts, fmt.Sprintf("%s%s%s:%d", levelColor, level, colorReset, count))
		}
	}
	return strings.Join(parts, " • ")
}

// formatTopItemsLine formats a line of top N items with counts
func formatTopItemsLine(items []CountedItem, maxItems int, truncateLength int) string {
	var parts []string
	for i, item := range items {
		if i >= maxItems {
			break
		}
		text := item.Item
		if truncateLength > 0 && len(text) > truncateLength {
			text = text[:truncateLength] + "..."
		}
		parts = append(parts, fmt.Sprintf("%s(%d)", text, item.Count))
	}
	return strings.Join(parts, " • ")
}

// findMaxCountAndCreateMap finds the maximum count and creates a map for easier lookup
func findMaxCountAndCreateMap(items []CountedItem) (int, map[string]int) {
	maxCount := 0
	itemMap := make(map[string]int)
	for _, item := range items {
		if item.Count > maxCount {
			maxCount = item.Count
		}
		itemMap[item.Item] = item.Count
	}
	return maxCount, itemMap
}

// createHourMap creates an hour map (int -> int) from BusiestHours and finds max count
func createHourMap(items []CountedItem) (int, map[int]int) {
	maxCount := 0
	hourMap := make(map[int]int)
	for _, item := range items {
		if item.Count > maxCount {
			maxCount = item.Count
		}
		hourNum := 0
		if _, err := fmt.Sscanf(item.Item, "%d", &hourNum); err != nil {
			logger.Debug("Invalid hour format in activity analysis", "hour", item.Item, "error", err)
			continue
		}
		if hourNum < 0 || hourNum >= 24 {
			logger.Debug("Hour outside valid range", "hour", hourNum)
			continue
		}
		hourMap[hourNum] = item.Count
	}
	return maxCount, hourMap
}

// displayAnalysis prints the analysis results
func displayAnalysis(analysis LogAnalysis, writer io.Writer, isDeduplicated bool, uniqueEntries int, verboseAnalysis bool) {

	// Calculate duration once
	duration := analysis.TimeRange.End.Sub(analysis.TimeRange.Start).Round(time.Second)

	headerStats := formatHeaderStats(analysis.TotalEntries, isDeduplicated, uniqueEntries, duration, analysis.ErrorRate)
	
	if verboseAnalysis {
		_, _ = fmt.Fprintf(writer, "\n%s=== MATTERMOST LOG ANALYSIS ===%s\n", colorHeaderBold, colorReset)
		_, _ = fmt.Fprintf(writer, "%s\n", headerStats)
		_, _ = fmt.Fprintf(writer, "%s to %s\n",
			analysis.TimeRange.Start.Format("2006-01-02 15:04:05"),
			analysis.TimeRange.End.Format("2006-01-02 15:04:05"))
	} else {
		_, _ = fmt.Fprintf(writer, "\n%sLOG ANALYSIS%s\n", colorHeaderBold, colorReset)
		_, _ = fmt.Fprintf(writer, "%s\n\n", headerStats)
	}

	// Log level distribution
	levelDistribution := formatLevelDistribution(analysis.LevelCounts, analysis.TotalEntries, verboseAnalysis)
	_, _ = fmt.Fprintf(writer, "%sLevels:%s %s\n", colorSubHeader, colorReset, levelDistribution)

	// Top sources
	if len(analysis.TopSources) > 0 {
		sourcesLine := formatTopItemsLine(analysis.TopSources, 3, 0)
		_, _ = fmt.Fprintf(writer, "%sSources:%s %s\n", colorSubHeader, colorReset, sourcesLine)
	}


	// Top error messages (if any)
	if len(analysis.TopErrorMessages) > 0 {
		truncateLength := 40
		if !verboseAnalysis {
			truncateLength = 30
		}
		errorsLine := formatTopItemsLine(analysis.TopErrorMessages, 3, truncateLength)
		_, _ = fmt.Fprintf(writer, "%sTop Errors:%s %s\n", colorSubHeader, colorReset, errorsLine)
	}

	// Peak hours - only in compact mode
	if !verboseAnalysis {
		// Sort hours by activity and show top 3
		sortedHours := make([]CountedItem, 0, len(analysis.BusiestHours))
		for _, hour := range analysis.BusiestHours {
			if hour.Count > 0 {
				sortedHours = append(sortedHours, hour)
			}
		}
		
		// Sort by count (descending)
		sort.Slice(sortedHours, func(i, j int) bool {
			return sortedHours[i].Count > sortedHours[j].Count
		})
		
		peakHoursLine := formatTopItemsLine(sortedHours, 3, 0)
		// Add 'h' suffix to hours
		peakHoursLine = strings.ReplaceAll(peakHoursLine, "(", "h(")
		_, _ = fmt.Fprintf(writer, "%sPeak Hours:%s %s\n", colorSubHeader, colorReset, peakHoursLine)
	}
	
	
	// Activity by month (if time range spans multiple months) - verbose only
	timeSpan := analysis.TimeRange.End.Sub(analysis.TimeRange.Start)
	if verboseAnalysis && timeSpan.Hours() >= 24*30 && len(analysis.ActivityByMonth) > 0 {
		_, _ = fmt.Fprintf(writer, "%sActivity by Month:%s\n", colorSubHeader, colorReset)
		maxCount, monthMap := findMaxCountAndCreateMap(analysis.ActivityByMonth)
		
		// Display months with bar chart (in calendar order)
		for _, month := range []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"} {
			count := monthMap[month]
			barLength := int(float64(count) / float64(maxCount) * 30)
			bar := strings.Repeat("█", barLength)
			
			// Get dominant log level color for this month
			levelColor := getDominantLevelColor(analysis.MonthLevelCounts[month], count)
			
			_, _ = fmt.Fprintf(writer, "%-9s: %s%s%s (%d)\n", month, levelColor, bar, colorReset, count)
		}
		_, _ = fmt.Fprintln(writer)
	}

	
	// Notification statistics (if present) - only in verbose mode
	if verboseAnalysis && len(analysis.NotificationTypes) > 0 {
		_, _ = fmt.Fprintf(writer, "%sNotification Statistics:%s\n", colorSubHeader, colorReset)
		
		// Notification types
		if len(analysis.NotificationTypes) > 0 {
			_, _ = fmt.Fprintf(writer, "Notification Types:\n")
			for _, nt := range analysis.NotificationTypes {
				_, _ = fmt.Fprintf(writer, "  %s: %d\n", nt.Item, nt.Count)
			}
		}
		
		// Notification statuses
		if len(analysis.NotificationStatuses) > 0 {
			_, _ = fmt.Fprintf(writer, "Notification Statuses:\n")
			for _, ns := range analysis.NotificationStatuses {
				_, _ = fmt.Fprintf(writer, "  %s: %d\n", ns.Item, ns.Count)
			}
		}
		_, _ = fmt.Fprintln(writer)
	}

	// Activity sections at the bottom - verbose only
	if verboseAnalysis {
		// Activity by hour
		_, _ = fmt.Fprintf(writer, "%sActivity by Hour:%s\n", colorSubHeader, colorReset)
		maxCount, hourMap := createHourMap(analysis.BusiestHours)

		// Display hours with bar chart (skip zero activity hours)
		for hour := 0; hour < 24; hour++ {
			count := hourMap[hour]
			if count == 0 {
				continue // Skip hours with no activity
			}
			barLength := int(float64(count) / float64(maxCount) * 15) // Shorter bars
			bar := strings.Repeat("█", barLength)
			
			// Get dominant log level color for this hour
			levelColor := getDominantLevelColor(analysis.HourLevelCounts[hour], count)
			
			_, _ = fmt.Fprintf(writer, "%02d:00: %s%s%s (%d)\n", hour, levelColor, bar, colorReset, count)
		}
		_, _ = fmt.Fprintln(writer)

		// Activity by day of week (if time range spans multiple days)
		timeSpan := analysis.TimeRange.End.Sub(analysis.TimeRange.Start)
		if timeSpan.Hours() >= 24 && len(analysis.ActivityByDayOfWeek) > 0 {
			_, _ = fmt.Fprintf(writer, "%sActivity by Day of Week:%s\n", colorSubHeader, colorReset)
			maxCount, dayMap := findMaxCountAndCreateMap(analysis.ActivityByDayOfWeek)
			
			// Display days with bar chart (in order from Sunday to Saturday, skip zero days)
			dayNames := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
			dayAbbrevs := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
			for i, day := range dayNames {
				count := dayMap[day]
				if count == 0 {
					continue // Skip days with no activity
				}
				barLength := int(float64(count) / float64(maxCount) * 15) // Shorter bars
				bar := strings.Repeat("█", barLength)
				
				// Get dominant log level color for this day
				levelColor := getDominantLevelColor(analysis.DayLevelCounts[day], count)
				
				_, _ = fmt.Fprintf(writer, "%s: %s%s%s (%d)\n", dayAbbrevs[i], levelColor, bar, colorReset, count)
			}
			_, _ = fmt.Fprintln(writer)
		}
	}

	// Footer
	if verboseAnalysis {
		_, _ = fmt.Fprintf(writer, "\n%s=== END OF ANALYSIS ===%s\n\n", colorHeaderBold, colorReset)
	} else {
		_, _ = fmt.Fprintln(writer, "")
	}
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