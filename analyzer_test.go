package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Uses the mustParseTime function from parser_test.go

func TestAnalyzeLogs(t *testing.T) {
	// Create test log entries with different timestamps spanning multiple days and months
	logs := []LogEntry{
		// Day 1 - January 1st
		{
			Timestamp: mustParseTime(t, "2025-01-01 10:00:00.000 Z"),
			Level:     "INFO",
			Message:   "System started",
			Source:    "system/init.go:42",
		},
		{
			Timestamp: mustParseTime(t, "2025-01-01 10:30:00.000 Z"),
			Level:     "INFO",
			Message:   "User login",
			Source:    "auth/login.go:55",
			User:      "user123",
		},
		{
			Timestamp: mustParseTime(t, "2025-01-01 11:05:00.000 Z"),
			Level:     "ERROR",
			Message:   "Database connection failed",
			Source:    "db/conn.go:77",
		},
		{
			Timestamp: mustParseTime(t, "2025-01-01 12:15:00.000 Z"),
			Level:     "WARN",
			Message:   "High CPU usage",
			Source:    "monitor/cpu.go:30",
		},
		
		// Day 2 - January 2nd
		{
			Timestamp: mustParseTime(t, "2025-01-02 09:20:00.000 Z"),
			Level:     "INFO",
			Message:   "Backup started",
			Source:    "backup/scheduler.go:42",
		},
		{
			Timestamp: mustParseTime(t, "2025-01-02 10:45:00.000 Z"),
			Level:     "DEBUG",
			Message:   "Cache invalidated",
			Source:    "cache/manager.go:55",
		},
		
		// Day 3 - January 3rd
		{
			Timestamp: mustParseTime(t, "2025-01-03 11:30:00.000 Z"),
			Level:     "ERROR",
			Message:   "Failed to send email",
			Source:    "email/sender.go:87",
		},
		
		// February 1st (different month)
		{
			Timestamp: mustParseTime(t, "2025-02-01 14:10:00.000 Z"),
			Level:     "INFO",
			Message:   "Monthly maintenance started",
			Source:    "maintenance/scheduler.go:22",
		},
		
		// March 1st (different month)
		{
			Timestamp: mustParseTime(t, "2025-03-01 15:45:00.000 Z"),
			Level:     "INFO",
			Message:   "System updated",
			Source:    "system/updater.go:64",
		},
	}

	t.Run("analyze basic statistics", func(t *testing.T) {
		analysis := analyzeLogs(logs, false)
		
		// Check total entries
		assert.Equal(t, 9, analysis.TotalEntries)
		
		// Check time range
		assert.Equal(t, mustParseTime(t, "2025-01-01 10:00:00.000 Z"), analysis.TimeRange.Start)
		assert.Equal(t, mustParseTime(t, "2025-03-01 15:45:00.000 Z"), analysis.TimeRange.End)
		
		// Check level counts
		assert.Equal(t, 5, analysis.LevelCounts["INFO"])
		assert.Equal(t, 2, analysis.LevelCounts["ERROR"])
		assert.Equal(t, 1, analysis.LevelCounts["WARN"])
		assert.Equal(t, 1, analysis.LevelCounts["DEBUG"])
		
		// Check error rate (~22.22%)
		assert.InDelta(t, 22.22, analysis.ErrorRate, 0.1)
	})

	t.Run("analyze hour distribution", func(t *testing.T) {
		analysis := analyzeLogs(logs, false)
		hourMap := make(map[string]int)
		
		for _, hour := range analysis.BusiestHours {
			hourMap[hour.Item] = hour.Count
		}
		
		assert.Equal(t, 1, hourMap["9"])   // 09:00 hour
		assert.Equal(t, 3, hourMap["10"])  // 10:00 hour (busiest, 3 logs)
		assert.Equal(t, 2, hourMap["11"])  // 11:00 hour
		assert.Equal(t, 1, hourMap["12"])  // 12:00 hour
		assert.Equal(t, 1, hourMap["14"])  // 14:00 hour
		assert.Equal(t, 1, hourMap["15"])  // 15:00 hour
	})

	t.Run("analyze day of week distribution", func(t *testing.T) {
		analysis := analyzeLogs(logs, false)
		dayMap := make(map[string]int)
		
		for _, day := range analysis.ActivityByDayOfWeek {
			dayMap[day.Item] = day.Count
		}
		
		// In 2025, Jan 1 is a Wednesday, Jan 2 is Thursday, Jan 3 is Friday,
		// Feb 1 is Saturday, Mar 1 is Saturday
		assert.Equal(t, 4, dayMap["Wednesday"]) // Most entries on Wednesday
		assert.Equal(t, 2, dayMap["Thursday"])
		assert.Equal(t, 1, dayMap["Friday"])
		assert.Equal(t, 2, dayMap["Saturday"])
	})

	t.Run("analyze month distribution", func(t *testing.T) {
		analysis := analyzeLogs(logs, false)
		monthMap := make(map[string]int)
		
		for _, month := range analysis.ActivityByMonth {
			monthMap[month.Item] = month.Count
		}
		
		assert.Equal(t, 7, monthMap["January"])  // Most entries in January
		assert.Equal(t, 1, monthMap["February"])
		assert.Equal(t, 1, monthMap["March"])
	})

	t.Run("analyze level distribution by hour", func(t *testing.T) {
		analysis := analyzeLogs(logs, false)
		
		// Check hour 10 level distribution
		hourLevels := analysis.HourLevelCounts[10]
		assert.Equal(t, 2, hourLevels["INFO"])
		assert.Equal(t, 0, hourLevels["ERROR"])  // We don't have ERROR logs at 10 hour
		
		// Check hour 11 level distribution
		hourLevels = analysis.HourLevelCounts[11]
		assert.Equal(t, 0, hourLevels["INFO"])  // The actual values in the test data
		assert.Equal(t, 2, hourLevels["ERROR"])
	})

	t.Run("analyze level distribution by day", func(t *testing.T) {
		analysis := analyzeLogs(logs, false)
		
		// Check Wednesday level distribution
		wedLevels := analysis.DayLevelCounts["Wednesday"]
		assert.Equal(t, 2, wedLevels["INFO"])
		assert.Equal(t, 1, wedLevels["ERROR"])
		assert.Equal(t, 1, wedLevels["WARN"])
		
		// Check Thursday level distribution
		thuLevels := analysis.DayLevelCounts["Thursday"]
		assert.Equal(t, 1, thuLevels["INFO"])
		assert.Equal(t, 1, thuLevels["DEBUG"])
	})

	t.Run("analyze level distribution by month", func(t *testing.T) {
		analysis := analyzeLogs(logs, false)
		
		// Check January level distribution
		janLevels := analysis.MonthLevelCounts["January"]
		assert.Equal(t, 3, janLevels["INFO"])
		assert.Equal(t, 2, janLevels["ERROR"])
		assert.Equal(t, 1, janLevels["WARN"])
		assert.Equal(t, 1, janLevels["DEBUG"])
	})
}

func TestGetDominantLevelColor(t *testing.T) {
	tests := []struct {
		name        string
		levelCounts map[string]int
		totalCount  int
		wantColor   string
	}{
		{
			name: "dominant error level",
			levelCounts: map[string]int{
				"ERROR": 6,
				"INFO":  3,
				"WARN":  1,
			},
			totalCount: 10,
			wantColor:  "\033[31m", // Red for ERROR
		},
		{
			name: "dominant info level",
			levelCounts: map[string]int{
				"INFO":  7,
				"DEBUG": 2,
				"ERROR": 1,
			},
			totalCount: 10,
			wantColor:  "\033[32m", // Green for INFO
		},
		{
			name: "no dominant level (mixed)",
			levelCounts: map[string]int{
				"INFO":  4,
				"ERROR": 3,
				"WARN":  3,
			},
			totalCount: 10,
			wantColor:  "\033[0m", // Reset color (no dominant level)
		},
		{
			name:        "empty level counts",
			levelCounts: map[string]int{},
			totalCount:  0,
			wantColor:   "\033[0m", // Reset color for empty counts
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getDominantLevelColor(tt.levelCounts, tt.totalCount)
			assert.Equal(t, tt.wantColor, got)
		})
	}
}

func TestDisplayAnalysis(t *testing.T) {
	// Create a sample analysis with all required components
	analysis := LogAnalysis{
		TotalEntries: 10,
		TimeRange: TimeRange{
			Start: mustParseTime(t, "2025-01-01 10:00:00.000 Z"),
			End:   mustParseTime(t, "2025-03-01 15:45:00.000 Z"),
		},
		LevelCounts: map[string]int{
			"INFO":  5,
			"ERROR": 2,
			"WARN":  2,
			"DEBUG": 1,
		},
		ErrorRate: 20.0,
		BusiestHours: []CountedItem{
			{Item: "10", Count: 3},
			{Item: "11", Count: 2},
			{Item: "9", Count: 2},
			{Item: "14", Count: 1},
			{Item: "15", Count: 1},
			{Item: "16", Count: 1},
		},
		ActivityByDayOfWeek: []CountedItem{
			{Item: "Wednesday", Count: 4},
			{Item: "Thursday", Count: 3},
			{Item: "Friday", Count: 2},
			{Item: "Saturday", Count: 1},
		},
		ActivityByMonth: []CountedItem{
			{Item: "January", Count: 6},
			{Item: "February", Count: 2},
			{Item: "March", Count: 2},
		},
		HourLevelCounts: map[int]map[string]int{
			10: {"INFO": 2, "ERROR": 1},
			11: {"INFO": 1, "WARN": 1},
		},
		DayLevelCounts: map[string]map[string]int{
			"Wednesday": {"INFO": 3, "ERROR": 1},
			"Thursday":  {"INFO": 1, "WARN": 2},
		},
		MonthLevelCounts: map[string]map[string]int{
			"January":  {"INFO": 3, "ERROR": 2, "WARN": 1},
			"February": {"INFO": 1, "DEBUG": 1},
			"March":    {"INFO": 1},
		},
		TopSources: []CountedItem{
			{Item: "system/init.go:42", Count: 2},
			{Item: "auth/login.go:55", Count: 1},
		},
	}

	t.Run("display analysis output formatting", func(t *testing.T) {
		var buf bytes.Buffer
		displayAnalysis(analysis, &buf, false, 10)
		output := buf.String()
		
		// Check that all expected sections are present
		assert.Contains(t, output, "=== MATTERMOST LOG ANALYSIS ===")
		assert.Contains(t, output, "Basic Statistics:")
		assert.Contains(t, output, "Log Level Distribution:")
		assert.Contains(t, output, "Activity by Hour:")
		assert.Contains(t, output, "Activity by Day of Week:")
		assert.Contains(t, output, "Activity by Month:")
		
		// Check time formatting
		assert.Contains(t, output, "2025-01-01 10:00:00")
		assert.Contains(t, output, "2025-03-01 15:45:00")
		
		// Check level distribution
		assert.Contains(t, output, "INFO")
		assert.Contains(t, output, "ERROR")
		
		// Check error rate
		assert.Contains(t, output, "Error Rate: 20.00%")
	})

	t.Run("display analysis with deduplication info", func(t *testing.T) {
		var buf bytes.Buffer
		displayAnalysis(analysis, &buf, true, 8) // 8 unique entries out of 10 total
		output := buf.String()
		
		// Check deduplication info
		assert.Contains(t, output, "Unique Log Entries: 8")
		assert.Contains(t, output, "Total Log Entries: 10")
		assert.Contains(t, output, "Deduplication Ratio")
	})

	t.Run("display time range condition for day and month charts", func(t *testing.T) {
		// Create an analysis with short time range (less than 24 hours)
		shortAnalysis := analysis
		shortAnalysis.TimeRange = TimeRange{
			Start: mustParseTime(t, "2025-01-01 10:00:00.000 Z"),
			End:   mustParseTime(t, "2025-01-01 15:45:00.000 Z"),
		}
		
		var buf bytes.Buffer
		displayAnalysis(shortAnalysis, &buf, false, 10)
		output := buf.String()
		
		// Day of week chart should NOT be present for short time ranges
		assert.NotContains(t, output, "Activity by Day of Week:")
		
		// Month chart should NOT be present for short time ranges
		assert.NotContains(t, output, "Activity by Month:")
	})
}

func TestAnalyzeAndDisplayStats(t *testing.T) {
	logs := []LogEntry{
		{
			Timestamp: mustParseTime(t, "2025-01-01 10:00:00.000 Z"),
			Level:     "INFO",
			Message:   "System started",
			Source:    "system/init.go:42",
		},
		{
			Timestamp: mustParseTime(t, "2025-01-02 10:30:00.000 Z"),
			Level:     "ERROR",
			Message:   "Database connection failed",
			Source:    "db/conn.go:77",
		},
		{
			Timestamp: mustParseTime(t, "2025-01-03 11:05:00.000 Z"),
			Level:     "WARN",
			Message:   "High CPU usage",
			Source:    "monitor/cpu.go:30",
		},
	}

	t.Run("display stats without duplicates", func(t *testing.T) {
		var buf bytes.Buffer
		analyzeAndDisplayStats(logs, &buf, false)
		output := buf.String()
		
		assert.Contains(t, output, "Total Log Entries: 3")
		assert.NotContains(t, output, "Deduplication Ratio")
	})

	t.Run("handle empty logs", func(t *testing.T) {
		var buf bytes.Buffer
		analyzeAndDisplayStats([]LogEntry{}, &buf, false)
		output := buf.String()
		
		assert.Contains(t, output, "No log entries to analyze.")
	})

	t.Run("display stats with duplicates", func(t *testing.T) {
		// Create logs with duplicate counts
		duplicateLogs := []LogEntry{
			{
				Timestamp:      mustParseTime(t, "2025-01-01 10:00:00.000 Z"),
				Level:          "INFO",
				Message:        "System started",
				Source:         "system/init.go:42",
				DuplicateCount: 3,
			},
			{
				Timestamp:      mustParseTime(t, "2025-01-02 10:30:00.000 Z"),
				Level:          "ERROR",
				Message:        "Database connection failed",
				Source:         "db/conn.go:77",
				DuplicateCount: 2,
			},
		}
		
		var buf bytes.Buffer
		analyzeAndDisplayStats(duplicateLogs, &buf, true)
		output := buf.String()
		
		assert.Contains(t, output, "Unique Log Entries: 2")
		assert.Contains(t, output, "Total Log Entries: 5")
		assert.Contains(t, output, "Deduplication Ratio")
	})
}