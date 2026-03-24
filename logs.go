package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/schollz/progressbar/v3"
)

// contains checks if a string slice contains a given string.
func contains(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

// loadFileLogs loads and merges log entries from one or more files.
// A path of "-" reads from stdin.
func loadFileLogs(paths []string) ([]LogEntry, error) {
	if len(paths) == 1 {
		filePath := paths[0]
		if filePath == "-" {
			if isStdinTTY() {
				return nil, &MisuseError{msg: "stdin is a TTY; pipe log data into lamp or provide a file path"}
			}
			if interactive {
				return nil, &MisuseError{msg: "--interactive (TUI mode) cannot be used with stdin input; provide a file path instead"}
			}
			logs, err := parseLogReader(os.Stdin, searchTerm, regexSearch, levelFilter, minLevelFilter, userFilter, startTime, endTime)
			if err != nil {
				return nil, fmt.Errorf("error parsing stdin: %v", err)
			}
			logger.Debug("Processed file", "file", "<stdin>", "entries", len(logs))
			return logs, nil
		}
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return nil, fmt.Errorf("file '%s' does not exist", filePath)
		}
		logs, err := parseLogFile(filePath, searchTerm, regexSearch, levelFilter, minLevelFilter, userFilter, startTime, endTime)
		if err != nil {
			return nil, fmt.Errorf("error parsing log file: %v", err)
		}
		return logs, nil
	}

	// Multiple files mode — validate stdin args before opening the progress bar
	for _, filePath := range paths {
		if filePath == "-" {
			if isStdinTTY() {
				return nil, &MisuseError{msg: "stdin is a TTY; pipe log data into lamp or provide a file path"}
			}
			if interactive {
				return nil, &MisuseError{msg: "--interactive (TUI mode) cannot be used with stdin input; provide a file path instead"}
			}
		}
	}

	var allLogs []LogEntry
	stdinConsumed := false

	bar := progressbar.NewOptions(len(paths),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetWidth(40),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("[cyan]Processing log files[reset]"),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	for _, filePath := range paths {
		if err := bar.Add(1); err != nil {
			logger.Warn("Error updating progress bar", "error", err)
		}

		if filePath == "-" {
			if stdinConsumed {
				logger.Warn("Duplicate \"-\" argument, stdin already consumed, skipping")
				continue
			}
			logs, err := parseLogReader(os.Stdin, searchTerm, regexSearch, levelFilter, minLevelFilter, userFilter, startTime, endTime)
			stdinConsumed = true
			if err != nil {
				logger.Warn("Error parsing stdin, skipping", "error", err)
				continue
			}
			allLogs = append(allLogs, logs...)
			logger.Debug("Processed file", "file", "<stdin>", "entries", len(logs))
			continue
		}

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			logger.Warn("File does not exist, skipping", "file", filePath)
			continue
		}

		logs, err := parseLogFile(filePath, searchTerm, regexSearch, levelFilter, minLevelFilter, userFilter, startTime, endTime)
		if err != nil {
			logger.Warn("Error parsing log file, skipping", "file", filePath, "error", err)
			continue
		}

		allLogs = append(allLogs, logs...)
		logger.Debug("Processed file", "file", filePath, "entries", len(logs))
	}

	if len(allLogs) == 0 {
		return nil, fmt.Errorf("no valid log entries found in any of the provided files")
	}

	sort.Slice(allLogs, func(i, j int) bool {
		return allLogs[i].Timestamp.Before(allLogs[j].Timestamp)
	})

	logger.Info("Finished processing files", "total_files", len(paths), "total_entries", len(allLogs))
	return allLogs, nil
}

// applyTrim deduplicates logs when --trim is set, optionally writing results to --trim-json.
func applyTrim(logs []LogEntry) ([]LogEntry, error) {
	if !trim {
		return logs, nil
	}
	logger.Info("Starting deduplication", "count", len(logs))
	originalCount := len(logs)
	logs = trimDuplicateLogInfo(logs)
	logger.Info("finished deduplication",
		"original", originalCount,
		"final", len(logs),
		"removed", originalCount-len(logs))
	if trimJSON != "" {
		if err := writeLogsToJSON(logs, trimJSON); err != nil {
			return nil, fmt.Errorf("error writing deduplicated logs to JSON: %v", err)
		}
		logger.Info("wrote deduplicated logs", "file", trimJSON)
	}
	return logs, nil
}

// processLogs handles output for the standard (non-AI) commands.
func processLogs(logs []LogEntry) error {
	var err error
	if logs, err = applyTrim(logs); err != nil {
		return err
	}

	// Set output destination
	output := os.Stdout
	if outputFile != "" {
		file, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("error creating output file: %v", err)
		}
		defer func() { _ = file.Close() }()
		output = file
		fmt.Printf("Writing output to %s\n", outputFile)
	}

	// Handle interactive mode
	if interactive {
		return launchInteractiveMode(logs)
	}

	// Export to CSV if requested
	if csvOutput != "" {
		if err := exportToCSV(logs, csvOutput); err != nil {
			return fmt.Errorf("error exporting to CSV: %v", err)
		}
		fmt.Printf("Logs exported to CSV file: %s\n", csvOutput)
		return nil
	}

	// Display logs in the requested format
	switch {
	case analyze:
		analyzeAndDisplayStats(logs, output, !trim, verboseAnalysis)
	case jsonOutput:
		displayLogsJSON(logs, output)
	case rawOutput:
		displayLogsPretty(logs, output)
	default:
		// Default to compact analysis instead of dumping all logs
		analyzeAndDisplayStats(logs, output, !trim, verboseAnalysis)
	}

	return nil
}

// runAIAnalysis sends logs to the configured LLM for analysis.
func runAIAnalysis(logs []LogEntry) error {
	provider := LLMProvider(llmProvider)
	if provider == "" {
		provider = ProviderAnthropic
	}

	// Validate provider
	supportedProviders := []string{"anthropic", "openai", "gemini", "ollama"}
	if !contains(supportedProviders, string(provider)) {
		return newMisuseError("invalid LLM provider: %s. Supported providers are: %s", provider, strings.Join(supportedProviders, ", "))
	}

	// Validate API key (not required for Ollama)
	apiKeyValue := apiKey
	if provider != ProviderOllama {
		if apiKeyValue == "" {
			envVar := getAPIKeyEnvVar(provider)
			apiKeyValue = os.Getenv(envVar)
			if apiKeyValue == "" {
				return fmt.Errorf("%s API key is required for AI analysis. Set with --api-key or %s environment variable",
					provider, envVar)
			}
		}
	}

	var applyTrimErr error
	if logs, applyTrimErr = applyTrim(logs); applyTrimErr != nil {
		return applyTrimErr
	}

	// After trimming, ask if user wants to send all remaining entries
	entriesForAnalysis := maxEntries
	if trim && !autoConfirm && isStdinTTY() {
		fmt.Printf("After trimming, there are %d log entries. Would you like to analyze all of them? (y/n): ", len(logs))
		var response string
		_, err := fmt.Scanln(&response)
		if err != nil {
			response = "n"
		}
		if strings.ToLower(response) == "y" || strings.ToLower(response) == "yes" {
			entriesForAnalysis = len(logs)
		}
	}

	model := llmModel
	if model == "" {
		model = GetDefaultModel(provider)
	}
	config := LLMConfig{
		Provider:       provider,
		Model:          model,
		APIKey:         apiKeyValue,
		MaxEntries:     entriesForAnalysis,
		Problem:        problem,
		ThinkingBudget: thinkingBudget,
		OllamaHost:     ollamaHost,
		OllamaTimeout:  ollamaTimeout,
	}

	if err := analyzeWithLLM(logs, config); err != nil {
		return fmt.Errorf("error during LLM analysis: %v", err)
	}
	return nil
}
