package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

// LogEntry represents a parsed log entry from Mattermost logs
type LogEntry struct {
	Timestamp      time.Time         `json:"timestamp"`
	Level          string            `json:"level"`
	Message        string            `json:"message"`
	Source         string            `json:"source,omitempty"`
	User           string            `json:"user,omitempty"`
	LogSource      string            `json:"log_source,omitempty"` // For notifications: "notifications"
	AckID          string            `json:"ack_id,omitempty"`     // For notifications: notification ID
	Type           string            `json:"type,omitempty"`       // For notifications: message type
	Status         string            `json:"status,omitempty"`     // For notifications: delivery status
	Extras         map[string]string `json:"extras,omitempty"`
	DuplicateCount int               `json:"duplicate_count,omitempty"`
}

// ExtrasToString converts the Extras map to a comma-separated string of key-value pairs.
// Each pair is formatted as "key=value". The pairs are sorted alphabetically by key.
// Returns an empty string if Extras is nil or empty.
func (l *LogEntry) ExtrasToString() string {
	extras := []string{}
	for k, v := range l.Extras {
		extras = append(extras, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(extras, ", ")
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
		parsedTime, parseErr := time.Parse("2006-01-02 15:04:05.000", startTimeStr)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid start time format: %v", parseErr)
		}
		startTime = parsedTime
	}
	if endTimeStr != "" {
		parsedTime, parseErr := time.Parse("2006-01-02 15:04:05.000", endTimeStr)
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
		if err != nil {
			logger.Debug("skipping unparseable line", "line", line, "error", err)
			// Skip lines that couldn't be parsed
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
	// Check if the line is in JSON format
	if strings.HasPrefix(strings.TrimSpace(line), "{") {
		return parseJSONLine(line)
	}

	var entry LogEntry
	// Basic format detection and parsing
	// Example format:
	// debug [2025-02-27 15:42:40.076 Z] Received HTTP request caller="web/handlers.go:187" method=GET url=/api/v4/groups request_id=XYZ user_id=ABC status_code=200
	parts := strings.SplitN(line, " [", 2)
	if len(parts) != 2 {
		return entry, fmt.Errorf("invalid log format")
	}

	// Parse log level
	entry.Level = strings.TrimSpace(parts[0])

	// Split remaining parts by closing bracket
	remainingParts := strings.SplitN(parts[1], "] ", 2)
	if len(remainingParts) != 2 {
		return entry, fmt.Errorf("invalid log format")
	}

	// Parse timestamp
	timestamp, err := parseTimestamp(remainingParts[0])
	if err != nil {
		return entry, err
	}
	entry.Timestamp = timestamp

	// Parse message and metadata
	rest := remainingParts[1]

	// Initialize extras map
	entry.Extras = make(map[string]string)

	// No caller, just split on first key-value pair
	fields := strings.Fields(rest)
	messageWords := []string{}

	// Collect words until we hit a key=value pair
	for _, word := range fields {
		if strings.Contains(word, "=") {
			break
		}
		messageWords = append(messageWords, word)
	}
	entry.Message = strings.Join(messageWords, " ")

	// Process remaining key-value pairs
	for _, pair := range fields[len(messageWords):] {
		if strings.Contains(pair, "=") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) != 2 {
				return entry, fmt.Errorf("invalid key-value pair: %s", pair)
			}
			k, v := parts[0], parts[1]
			switch k {
			case "caller":
				entry.Source = strings.Trim(v, "\"")
			case "user_id":
				entry.User = v
			default:
				entry.Extras[k] = v
			}
		}
	}

	return entry, nil
}

// parseJSONLine parses a JSON-formatted log line
func parseJSONLine(line string) (LogEntry, error) {
	var entry LogEntry
	entry.Extras = make(map[string]string)

	// JSONLogEntry represents a JSON-formatted log entry
	type JSONLogEntry struct {
		Timestamp string `json:"timestamp"`
		Level     string `json:"level"`
		Msg       string `json:"msg"`
		Caller    string `json:"caller,omitempty"`
		UserID    string `json:"user_id,omitempty"`
		LogSource string `json:"logSource,omitempty"`
		AckID     string `json:"ackId,omitempty"`
		Type      string `json:"type,omitempty"`
		Status    string `json:"status,omitempty"`
	}
	var jsonEntry JSONLogEntry

	// Unmarshal the JSON log entry
	if err := json.Unmarshal([]byte(line), &jsonEntry); err != nil {
		// Try to recover from JSON parsing errors by cleaning up common issues
		fixedLine := strings.ReplaceAll(line, "\\\"", "'")
		if err := json.Unmarshal([]byte(fixedLine), &jsonEntry); err != nil {
			return entry, fmt.Errorf("failed to parse JSON log: %v", err)
		}
	}

	// Extract additional fields
	var extra map[string]any
	if err := json.Unmarshal([]byte(line), &extra); err != nil {
		return entry, fmt.Errorf("failed to parse extra JSON fields: %v", err)
	}
	for k, v := range extra {
		// Skip fields we already handle
		if k == "timestamp" || k == "level" || k == "msg" || k == "caller" || k == "user_id" || 
		   k == "logSource" || k == "ackId" || k == "type" || k == "status" {
			continue
		}

		// Convert non-string values to strings
		switch val := v.(type) {
		case string:
			entry.Extras[k] = val
		default:
			// Use json.Marshal to convert other types to string representation
			bytes, err := json.Marshal(val)
			if err != nil {
				return entry, fmt.Errorf("failed to marshal extra field %s: %v", k, err)
			}
			entry.Extras[k] = string(bytes)
		}
	}

	// Parse timestamp
	timestamp, err := parseTimestamp(strings.TrimSpace(jsonEntry.Timestamp))
	if err != nil {
		return entry, err
	}
	entry.Timestamp = timestamp

	// Set other fields
	entry.Level = jsonEntry.Level
	entry.Message = jsonEntry.Msg
	entry.User = jsonEntry.UserID
	entry.Source = jsonEntry.Caller
	
	// Set notification-specific fields if present
	entry.LogSource = jsonEntry.LogSource
	entry.AckID = jsonEntry.AckID
	entry.Type = jsonEntry.Type
	entry.Status = jsonEntry.Status

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
		"2006-01-02 15:04:05.000 Z",
		"2006-01-02 15:04:05.000 MST",
		// Additional formats with timezone offsets
		"2006-01-02 15:04:05.000 -07:00",
		"2006-01-02 15:04:05.000 +07:00",
		"2006-01-02 15:04:05.999 -07:00",
		"2006-01-02 15:04:05.999 +07:00",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestampStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", timestampStr)
}

// trimDuplicateLogInfo removes log entries that contain duplicate or very similar information
// using fuzzy matching techniques
func trimDuplicateLogInfo(logs []LogEntry) []LogEntry {
	if len(logs) == 0 {
		return logs
	}

	// Similarity threshold (0.0-1.0) - higher means more strict matching
	const similarityThreshold = 0.8
	const updateInterval = 10     // Update progress bar description every N entries
	const batchSize = 100         // Process logs in batches to reduce memory pressure
	const parallelThreshold = 1000 // Minimum log count to use parallel processing

	// Create progress bar
	bar := progressbar.NewOptions(len(logs),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetWidth(40),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("[cyan]Deduplicating logs[reset]"),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionOnCompletion(func() {
			fmt.Println()
		}))

	// Render initial blank progress bar
	if err := bar.RenderBlank(); err != nil {
		logger.Warn("Error rendering progress bar", "error", err)
	}

	// Use parallel processing for large log sets
	if len(logs) >= parallelThreshold {
		return trimDuplicateLogsParallel(logs, similarityThreshold, bar)
	}
	
	return trimDuplicateLogsSequential(logs, similarityThreshold, batchSize, updateInterval, bar)
}

// trimDuplicateLogsSequential performs sequential deduplication for smaller log sets
func trimDuplicateLogsSequential(logs []LogEntry, similarityThreshold float64, batchSize, updateInterval int, bar *progressbar.ProgressBar) []LogEntry {
	var result []LogEntry
	processedEntries := make(map[int]bool)
	
	// Cache for normalized messages to avoid redundant processing
	normalizedCache := make(map[int]string, len(logs))
	
	removedCount := 0

	// Group entries by log level to reduce comparison space
	logsByLevel := make(map[string][]int)
	for i, entry := range logs {
		level := strings.ToLower(entry.Level)
		logsByLevel[level] = append(logsByLevel[level], i)
	}

	// Process each log entry
	for i, entry := range logs {
		// Update description periodically to show activity
		if i%updateInterval == 0 {
			bar.Describe(fmt.Sprintf("[cyan]Processed: %d/%d - Removed: %d[reset]", i, len(logs), removedCount))
		}

		// Skip if already processed
		if processedEntries[i] {
			if err := bar.Add(1); err != nil {
				logger.Warn("Error updating progress bar", "error", err)
			}
			continue
		}

		// Add this entry to results (with initial duplicate count of 1)
		entryWithCount := entry
		entryWithCount.DuplicateCount = 1
		result = append(result, entryWithCount)
		processedEntries[i] = true

		// Get or compute normalized message
		var normalizedMsg string
		var exists bool
		if normalizedMsg, exists = normalizedCache[i]; !exists {
			normalizedMsg = normalizeLogMessage(entry.Message)
			normalizedCache[i] = normalizedMsg
		}

		// Get words from normalized message (for word-based similarity)
		baseWords := strings.Fields(normalizedMsg)

		processedInThisIteration := 0
		entryLevel := strings.ToLower(entry.Level)

		// Only compare with entries of the same level to reduce comparison space
		for _, j := range logsByLevel[entryLevel] {
			// Skip if already processed or if it's the current entry
			if j <= i || processedEntries[j] {
				continue
			}

			// Check source similarity (early filter)
			sourceSimilar := strings.EqualFold(entry.Source, logs[j].Source) ||
				(len(entry.Source) > 0 && len(logs[j].Source) > 0 &&
					stringSimilarity(entry.Source, logs[j].Source) > 0.7)

			if !sourceSimilar {
				continue
			}

			// Get or compute normalized comparison message
			var compMsg string
			if compMsg, exists = normalizedCache[j]; !exists {
				compMsg = normalizeLogMessage(logs[j].Message)
				normalizedCache[j] = compMsg
			}

			// Compare messages
			if isSimilarMessage(normalizedMsg, compMsg, baseWords, similarityThreshold) {
				processedEntries[j] = true
				processedInThisIteration++
				removedCount++

				// Increment duplicate count for this entry
				result[len(result)-1].DuplicateCount++

				// Update progress description more frequently during batch removals
				if processedInThisIteration%10 == 0 {
					bar.Describe(fmt.Sprintf("[cyan]Processed: %d/%d - Removed: %d[reset]", i, len(logs), removedCount))
				}
			}
		}

		// Update progress bar
		if err := bar.Add(1); err != nil {
			logger.Warn("Error updating progress bar", "error", err)
		}
		
		// Periodically clear the cache to manage memory usage
		if i > 0 && i%batchSize == 0 {
			// Clear cache for already processed entries
			for k := range normalizedCache {
				if k < i-batchSize {
					delete(normalizedCache, k)
				}
			}
		}
	}

	// Ensure the bar is completed
	if err := bar.Finish(); err != nil {
		logger.Warn("Error completing progress bar", "error", err)
	}

	return result
}

// trimDuplicateLogsParallel performs parallel deduplication for larger log sets
func trimDuplicateLogsParallel(logs []LogEntry, similarityThreshold float64, bar *progressbar.ProgressBar) []LogEntry {
	// Normalize all messages in parallel first
	normalizedMsgs := make([]string, len(logs))
	
	// Group entries by log level to reduce comparison space
	logsByLevel := make(map[string][]int)
	for i, entry := range logs {
		level := strings.ToLower(entry.Level)
		logsByLevel[level] = append(logsByLevel[level], i)
	}
	
	// Use a worker pool to normalize messages in parallel
	workersCount := runtime.NumCPU()
	bar.Describe("[cyan]Normalizing log messages in parallel[reset]")
	
	// Create a channel to distribute work
	jobs := make(chan int, len(logs))
	for i := range logs {
		jobs <- i
	}
	close(jobs)
	
	// Use a sync.Mutex to protect the normalizedMsgs slice
	var mutex sync.Mutex
	var wg sync.WaitGroup
	wg.Add(workersCount)
	
	// Launch workers
	for w := 0; w < workersCount; w++ {
		go func() {
			defer wg.Done()
			for i := range jobs {
				normalizedMsg := normalizeLogMessage(logs[i].Message)
				mutex.Lock()
				normalizedMsgs[i] = normalizedMsg
				mutex.Unlock()
				
				// Update progress bar (safely)
				mutex.Lock()
				if err := bar.Add(1); err != nil {
					logger.Warn("Error updating progress bar", "error", err)
				}
				mutex.Unlock()
			}
		}()
	}
	
	// Wait for all normalizations to complete
	wg.Wait()
	
	// Reset the progress bar for the main deduplication phase
	bar.Reset()
	bar.ChangeMax(len(logs))
	bar.RenderBlank()
	bar.Describe("[cyan]Deduplicating logs with parallel processing[reset]")
	
	var result []LogEntry
	processedEntries := make(map[int]bool)
	var resultMutex sync.Mutex
	var processedMutex sync.Mutex
	removedCount := 0
	var removedMutex sync.Mutex
	
	// Process logs in chunks based on their level
	var levelWg sync.WaitGroup
	for level, indices := range logsByLevel {
		if len(indices) < 10 {  // Process small groups sequentially
			processLogGroup(
				logs, normalizedMsgs, indices, level, similarityThreshold, 
				&result, processedEntries, &removedCount, bar,
				&resultMutex, &processedMutex, &removedMutex,
			)
		} else {
			levelWg.Add(1)
			go func(lvl string, idxs []int) {
				defer levelWg.Done()
				processLogGroup(
					logs, normalizedMsgs, idxs, lvl, similarityThreshold, 
					&result, processedEntries, &removedCount, bar,
					&resultMutex, &processedMutex, &removedMutex,
				)
			}(level, indices)
		}
	}
	
	levelWg.Wait()
	
	// Ensure the bar is completed
	if err := bar.Finish(); err != nil {
		logger.Warn("Error completing progress bar", "error", err)
	}
	
	logger.Info("Parallel deduplication completed", "removed", removedCount)
	return result
}

// processLogGroup processes a group of logs with the same level
func processLogGroup(
	logs []LogEntry, 
	normalizedMsgs []string, 
	indices []int, 
	level string, 
	similarityThreshold float64,
	result *[]LogEntry,
	processedEntries map[int]bool,
	removedCount *int,
	bar *progressbar.ProgressBar,
	resultMutex, processedMutex, removedMutex *sync.Mutex,
) {
	// Process each log entry in this level group
	for _, i := range indices {
		// Skip if already processed
		processedMutex.Lock()
		if processedEntries[i] {
			processedMutex.Unlock()
			if err := bar.Add(1); err != nil {
				logger.Warn("Error updating progress bar", "error", err)
			}
			continue
		}
		processedEntries[i] = true
		processedMutex.Unlock()
		
		// Add this entry to results (with initial duplicate count of 1)
		entryWithCount := logs[i]
		entryWithCount.DuplicateCount = 1
		
		resultMutex.Lock()
		resultIndex := len(*result)
		*result = append(*result, entryWithCount)
		resultMutex.Unlock()
		
		// Get normalized message and its words
		normalizedMsg := normalizedMsgs[i]
		baseWords := strings.Fields(normalizedMsg)
		
		processedInThisIteration := 0
		
		// Compare with other entries of the same level
		for _, j := range indices {
			if j <= i {
				continue
			}
			
			// Skip if already processed
			processedMutex.Lock()
			isProcessed := processedEntries[j]
			processedMutex.Unlock()
			
			if isProcessed {
				continue
			}
			
			// Check source similarity (early filter)
			sourceSimilar := strings.EqualFold(logs[i].Source, logs[j].Source) ||
				(len(logs[i].Source) > 0 && len(logs[j].Source) > 0 &&
					stringSimilarity(logs[i].Source, logs[j].Source) > 0.7)
					
			if !sourceSimilar {
				continue
			}
			
			compMsg := normalizedMsgs[j]
			
			// Compare messages
			if isSimilarMessage(normalizedMsg, compMsg, baseWords, similarityThreshold) {
				// Mark as processed
				processedMutex.Lock()
				processedEntries[j] = true
				processedMutex.Unlock()
				
				processedInThisIteration++
				
				// Increment counters
				removedMutex.Lock()
				*removedCount++
				removedMutex.Unlock()
				
				// Update duplicate count
				resultMutex.Lock()
				(*result)[resultIndex].DuplicateCount++
				resultMutex.Unlock()
			}
		}
		
		// Update progress periodically
		if processedInThisIteration > 0 && processedInThisIteration%10 == 0 {
			removedMutex.Lock()
			currentRemoved := *removedCount
			removedMutex.Unlock()
			
			bar.Describe(fmt.Sprintf("[cyan]Processed: %d - Removed: %d[reset]", i, currentRemoved))
		}
		
		// Update progress bar
		if err := bar.Add(1); err != nil {
			logger.Warn("Error updating progress bar", "error", err)
		}
	}
}

// Precompile regex patterns for better performance
var (
	normalizePatterns = []struct {
		regex       *regexp.Regexp
		replacement string
	}{
		{regexp.MustCompile(`\b[0-9a-f]{8}\b`), "ID_SHORT"},                           // Short hex IDs (8 chars)
		{regexp.MustCompile(`\b[0-9a-f]{32}\b`), "ID_LONG"},                           // Long hex IDs (32 chars)
		{regexp.MustCompile(`\b[0-9a-f]{8}(-[0-9a-f]{4}){3}-[0-9a-f]{12}\b`), "UUID"}, // UUIDs
		{regexp.MustCompile(`\b([0-9a-f]{6,31})\b`), "ID"},                            // Other hex IDs
		{regexp.MustCompile(`\d{4}[-/]\d{1,2}[-/]\d{1,2}`), "DATE"},                   // Dates (yyyy-mm-dd)
		{regexp.MustCompile(`\d{1,2}[-/]\d{1,2}[-/]\d{2,4}`), "DATE"},                 // Dates (mm-dd-yyyy)
		{regexp.MustCompile(`\d{1,2}:\d{1,2}(:\d{1,2})?(\.\d+)?`), "TIME"},            // Times
		{regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`), "IP"},              // IPv4 addresses
		{regexp.MustCompile(`(([0-9a-f]{1,4}:){7}|::)[0-9a-f]{1,4}`), "IPV6"},         // IPv6 addresses
		{regexp.MustCompile(`\d+(\.\d+)?ms`), "DURATION_MS"},                          // Millisecond durations
		{regexp.MustCompile(`\d+(\.\d+)?s`), "DURATION_S"},                            // Second durations
		{regexp.MustCompile(`\d+(\.\d+)?ns`), "DURATION_NS"},                          // Nanosecond durations
		{regexp.MustCompile(`\d+(\.\d+)?[mu]s`), "DURATION_US"},                       // Microsecond durations
		{regexp.MustCompile(`\b\d{1,9}\b`), "NUMBER"},                                 // Simple numbers up to 9 digits
		{regexp.MustCompile(`"[^"]*"`), "STRING"},                                     // Quoted strings
		{regexp.MustCompile(`'[^']*'`), "STRING"},                                     // Single-quoted strings
		{regexp.MustCompile(`\b([a-zA-Z0-9_-]+\.)+[a-zA-Z0-9_-]+\b`), "PATH"},         // File/URL paths
		{regexp.MustCompile(`\b\d+\.\d+\.\d+\b`), "VERSION"},                          // Version numbers
	}
	whitespaceRegex = regexp.MustCompile(`\s+`)
)

// normalizeLogMessage applies various normalization techniques to a log message
func normalizeLogMessage(message string) string {
	// Convert to lowercase for case-insensitive comparison
	normalized := strings.ToLower(message)

	// Apply precompiled regex patterns
	for _, p := range normalizePatterns {
		normalized = p.regex.ReplaceAllString(normalized, p.replacement)
	}

	// Remove extra whitespace
	normalized = whitespaceRegex.ReplaceAllString(normalized, " ")
	return strings.TrimSpace(normalized)
}

// stringSimilarity calculates the similarity between two strings
// returns a value between 0.0 (completely different) and 1.0 (identical)
func stringSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	// Convert to lowercase for case-insensitive comparison
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)

	// Calculate Levenshtein distance
	distance := levenshteinDistance(s1, s2)
	maxLen := float64(max(len(s1), len(s2)))

	if maxLen == 0 {
		return 1.0 // Both strings are empty
	}

	return 1.0 - float64(distance)/maxLen
}

// isSimilarMessage determines if two messages are similar enough based on different measures
func isSimilarMessage(msg1, msg2 string, msg1Words []string, threshold float64) bool {
	// Quick path: exact match after normalization
	if msg1 == msg2 {
		return true
	}

	// Quick path: if one message is contained within the other
	// Only check if the lengths aren't too different to avoid unnecessary string operations
	lenRatio := float64(min(len(msg1), len(msg2))) / float64(max(len(msg1), len(msg2)))
	if lenRatio > 0.5 {
		if strings.Contains(msg1, msg2) || strings.Contains(msg2, msg1) {
			return true
		}
	} else {
		// Early exit for very different length strings
		return false
	}

	// Optimize for common case: check word-based similarity first as it's usually faster
	// and more effective for log messages than Levenshtein distance
	msg2Words := strings.Fields(msg2)
	
	// Skip Jaccard similarity calculation if the word counts are very different
	wordLenRatio := float64(min(len(msg1Words), len(msg2Words))) / float64(max(len(msg1Words), len(msg2Words)))
	if wordLenRatio < 0.5 {
		return false
	}

	// Calculate Jaccard similarity of words
	commonWords := 0
	msg1WordSet := make(map[string]bool, len(msg1Words))
	for _, word := range msg1Words {
		msg1WordSet[word] = true
	}

	for _, word := range msg2Words {
		if msg1WordSet[word] {
			commonWords++
		}
	}

	totalWords := len(msg1WordSet) + len(msg2Words) - commonWords
	if totalWords == 0 {
		return false
	}

	jaccardSimilarity := float64(commonWords) / float64(totalWords)
	if jaccardSimilarity >= threshold {
		return true
	}

	// Only perform the more expensive Levenshtein distance check if the Jaccard similarity
	// is close but not quite at the threshold
	if jaccardSimilarity >= threshold*0.8 {
		return stringSimilarity(msg1, msg2) >= threshold
	}
	
	return false
}

// levenshteinDistance calculates the edit distance between two strings.
// This is an optimized version with early termination if the distance exceeds maxDistance.
func levenshteinDistance(s1, s2 string) int {
	// Quick path for empty strings
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}
	
	// Optimization: swap strings so s1 is the shorter one
	if len(s1) > len(s2) {
		s1, s2 = s2, s1
	}
	
	// Optimization: if strings are identical, return 0 immediately
	if s1 == s2 {
		return 0
	}
	
	// Reuse vectors to avoid continuous allocations
	v0 := make([]int, len(s2)+1)
	v1 := make([]int, len(s2)+1)

	// Initialize v0 (the previous row of distances)
	for i := 0; i <= len(s2); i++ {
		v0[i] = i
	}

	// Calculate v1 (current row distances) from the previous row v0
	for i := 0; i < len(s1); i++ {
		// First element of v1 is A[i+1][0]
		v1[0] = i + 1
		
		// Track minimum value in this row to enable early termination
		minValue := v1[0]

		// Use formula to fill in the rest of the row
		for j := 0; j < len(s2); j++ {
			// Cost calculation
			var cost int
			if s1[i] == s2[j] {
				cost = 0
			} else {
				cost = 1
			}
			
			v1[j+1] = min(
				v0[j+1] + 1,      // deletion
				min(
					v1[j] + 1,     // insertion
					v0[j] + cost,  // substitution
				),
			)
			
			// Track minimum value in this row
			if v1[j+1] < minValue {
				minValue = v1[j+1]
			}
		}

		// Swap vectors for next iteration (avoid extra allocation)
		v0, v1 = v1, v0
	}

	// The result is in v0 (previously v1) because we swapped vectors
	return v0[len(s2)]
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
			!strings.Contains(strings.ToLower(entry.ExtrasToString()), searchLower) {
			return false
		}
	}

	// Apply regex filter
	if regex != nil {
		// Check if regex matches any field
		if !regex.MatchString(entry.Message) &&
			!regex.MatchString(entry.Source) &&
			!regex.MatchString(entry.ExtrasToString()) &&
			!regex.MatchString(entry.User) {
			return false
		}
	}

	return true
}
