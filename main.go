package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// MisuseError represents incorrect command usage (invalid flag values, wrong args, etc.).
// Commands that detect misuse should return this type so main can exit with code 2.
type MisuseError struct{ msg string }

func (e *MisuseError) Error() string { return e.msg }

func newMisuseError(format string, args ...any) error {
	return &MisuseError{msg: fmt.Sprintf(format, args...)}
}

// enteredPreRun is set true once PersistentPreRun fires, meaning cobra accepted the
// command (flags parsed, arg count valid). Errors before this point are misuse (exit 2).
var enteredPreRun bool

// isStdinTTY reports whether os.Stdin is an interactive terminal.
func isStdinTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

var (
	// Global flags
	searchTerm      string
	regexSearch     string
	levelFilter     string
	minLevelFilter  string
	userFilter      string
	startTime       string
	endTime         string
	jsonOutput      bool
	csvOutput       string
	outputFile      string
	analyze         bool
	apiKey          string
	llmProvider     string
	llmModel        string
	trim            bool
	trimJSON        string
	maxEntries      int
	problem         string
	thinkingBudget  int
	ollamaHost      string
	ollamaTimeout   int
	interactive     bool
	autoConfirm     bool
	verbose         bool
	quiet           bool
	verboseAnalysis bool
	rawOutput       bool

	// Global logger
	logger *slog.Logger
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "lamp",
	Short: "lamp is a tool for parsing and analyzing Mattermost log files",
	Long: `lamp (Log Analyser for Mattermost Packet) allows you to parse, filter, and analyze Mattermost log files
and support packets. It provides various filtering options, analysis capabilities,
and AI-powered insights using LLM technology.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		enteredPreRun = true
		initLogger()
	},
}

var fileCmd = &cobra.Command{
	Use:   "file [path...]",
	Short: "Parse and analyze one or more Mattermost log files",
	Long: `Parse and analyze one or more Mattermost log files.

Exit codes:
  0  Success
  1  General error (e.g., file not found, parse failure)
  2  Misuse (e.g., invalid flag value or missing required argument)`,
	Args: cobra.MinimumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveFilterFileExt | cobra.ShellCompDirectiveDefault
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateLevelFlags(levelFilter, minLevelFilter); err != nil {
			return &MisuseError{msg: err.Error()}
		}
		logs, err := loadFileLogs(args)
		if err != nil {
			return err
		}
		return processLogs(logs)
	},
}

var notificationCmd = &cobra.Command{
	Use:   "notification [path]",
	Short: "Parse and analyze a Mattermost notification log file",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveFilterFileExt | cobra.ShellCompDirectiveDefault
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateLevelFlags(levelFilter, minLevelFilter); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}

		filePath := args[0]
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("notification log file '%s' does not exist", filePath)
		}

		logs, err := parseLogFile(filePath, searchTerm, regexSearch, levelFilter, minLevelFilter, userFilter, startTime, endTime)
		if err != nil {
			return fmt.Errorf("error parsing notification log file: %v", err)
		}

		return processLogs(logs)
	},
}

var supportPacketCmd = &cobra.Command{
	Use:   "support-packet [path]",
	Short: "Parse and analyze a Mattermost support packet zip file",
	Long: `Parse and analyze a Mattermost support packet zip file.

Exit codes:
  0  Success
  1  General error (e.g., file not found, parse failure)
  2  Misuse (e.g., invalid flag value or missing required argument)`,
	Args: cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveFilterFileExt | cobra.ShellCompDirectiveDefault
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateLevelFlags(levelFilter, minLevelFilter); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}

		packetPath := args[0]
		if _, err := os.Stat(packetPath); os.IsNotExist(err) {
			return fmt.Errorf("support packet '%s' does not exist", packetPath)
		}

		logs, err := parseSupportPacket(packetPath, searchTerm, regexSearch, levelFilter, minLevelFilter, userFilter, startTime, endTime)
		if err != nil {
			return fmt.Errorf("error parsing support packet: %v", err)
		}

		if verbose {
			fmt.Printf("Debug: processing %d log entries\n", len(logs))
		}

		return processLogs(logs)
	},
}

// aiCmd is the parent for AI-powered log analysis subcommands.
var aiCmd = &cobra.Command{
	Use:   "ai-analyze",
	Short: "Analyze Mattermost logs using AI",
	Long:  `Send parsed log entries to an LLM for analysis. Use subcommands to specify the log source.`,
}

var aiFileCmd = &cobra.Command{
	Use:   "file [path...]",
	Short: "AI analyze one or more Mattermost log files",
	Args:  cobra.MinimumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveFilterFileExt | cobra.ShellCompDirectiveDefault
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logs, err := loadFileLogs(args)
		if err != nil {
			return err
		}
		return runAIAnalysis(logs)
	},
}

var aiNotificationCmd = &cobra.Command{
	Use:   "notification [path]",
	Short: "AI analyze a Mattermost notification log file",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveFilterFileExt | cobra.ShellCompDirectiveDefault
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("notification log file '%s' does not exist", filePath)
		}
		logs, err := parseLogFile(filePath, searchTerm, regexSearch, levelFilter, minLevelFilter, userFilter, startTime, endTime)
		if err != nil {
			return fmt.Errorf("error parsing notification log file: %v", err)
		}
		return runAIAnalysis(logs)
	},
}

var aiSupportPacketCmd = &cobra.Command{
	Use:   "support-packet [path]",
	Short: "AI analyze a Mattermost support packet zip file",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveFilterFileExt | cobra.ShellCompDirectiveDefault
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		packetPath := args[0]
		if _, err := os.Stat(packetPath); os.IsNotExist(err) {
			return fmt.Errorf("support packet '%s' does not exist", packetPath)
		}
		logs, err := parseSupportPacket(packetPath, searchTerm, regexSearch, levelFilter, minLevelFilter, userFilter, startTime, endTime)
		if err != nil {
			return fmt.Errorf("error parsing support packet: %v", err)
		}
		return runAIAnalysis(logs)
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		info, ok := debug.ReadBuildInfo()
		if !ok {
			return fmt.Errorf("could not read build information")
		}
		// Get version (usually from the module path)
		version := info.Main.Version
		if version == "(devel)" {
			version = "dev"
		}
		fmt.Printf("Version:\t%s\n", version)

		// Extract other build information from settings
		var commitDate, gitCommit, gitTreeState string
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.time":
				commitDate = setting.Value
			case "vcs.revision":
				gitCommit = setting.Value
			case "vcs.modified":
				gitTreeState = "clean"
				if setting.Value == "true" {
					gitTreeState = "dirty"
				}
			}
		}

		// Print all version information
		if commitDate != "" {
			fmt.Printf("CommitDate:\t%s\n", commitDate)
		}
		if gitCommit != "" {
			fmt.Printf("GitCommit:\t%s\n", gitCommit)
		}
		fmt.Printf("GitTreeState:\t%s\n", gitTreeState)
		fmt.Printf("GoVersion:\t%s\n", runtime.Version())
		fmt.Printf("Compiler:\t%s\n", runtime.Compiler)
		fmt.Printf("Platform:\t%s/%s\n", runtime.GOARCH, runtime.GOOS)
		return nil
	},
}

// levelRanks maps severity level names to their numeric rank for min-level comparisons.
var levelRanks = map[string]int{"debug": 0, "info": 1, "warn": 2, "error": 3}

// validateLevelFlags returns an error if --level and --min-level are both set, or if
// minLevel is not a recognised value. The caller is responsible for printing to stderr
// and exiting with the appropriate code.
func validateLevelFlags(level, minLevel string) error {
	if level != "" && minLevel != "" {
		return fmt.Errorf("--level and --min-level are mutually exclusive")
	}
	if minLevel != "" {
		if _, ok := levelRanks[strings.ToLower(minLevel)]; !ok {
			return fmt.Errorf("invalid --min-level value %q: valid values are debug, info, warn, error", minLevel)
		}
	}
	return nil
}

// registerFlagCompletion is a helper function that registers flag completion and panics on error
func registerFlagCompletion(cmd *cobra.Command, flag string, completionFunc func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)) {
	if err := cmd.RegisterFlagCompletionFunc(flag, completionFunc); err != nil {
		panic(fmt.Sprintf("failed to register completion for --%s flag: %v", flag, err))
	}
}

// addFilterFlags registers log filtering flags shared by all processing commands.
func addFilterFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&searchTerm, "search", "", "Search term to filter logs")
	cmd.Flags().StringVar(&regexSearch, "regex", "", "Regular expression pattern to filter logs")
	cmd.Flags().StringVar(&levelFilter, "level", "", "Filter logs by level (info, error, debug, etc.)")
	cmd.Flags().StringVar(&minLevelFilter, "min-level", "", "Include only log entries at this severity level or higher (debug, info, warn, error)")
	cmd.Flags().StringVar(&userFilter, "user", "", "Filter logs by username")
	cmd.Flags().StringVar(&startTime, "start", "", "Filter logs after this time (format: 2006-01-02 15:04:05.000)")
	cmd.Flags().StringVar(&endTime, "end", "", "Filter logs before this time (format: 2006-01-02 15:04:05.000)")
	cmd.Flags().BoolVar(&trim, "trim", false, "Remove entries with duplicate information")

	registerFlagCompletion(cmd, "level", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"debug", "info", "warn", "error", "fatal", "panic"}, cobra.ShellCompDirectiveNoFileComp
	})
	registerFlagCompletion(cmd, "min-level", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"debug", "info", "warn", "error"}, cobra.ShellCompDirectiveNoFileComp
	})
	registerFlagCompletion(cmd, "trim", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
	})
	for _, flag := range []string{"start", "end"} {
		registerFlagCompletion(cmd, flag, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return []string{"2006-01-02 15:04:05.000"}, cobra.ShellCompDirectiveNoFileComp
		})
	}
}

// addOutputFlags registers output format flags for standard (non-AI) commands.
func addOutputFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().StringVar(&csvOutput, "csv", "", "Export logs to CSV file at specified path")
	cmd.Flags().StringVar(&outputFile, "output", "", "Save output to file instead of stdout")
	cmd.Flags().BoolVar(&analyze, "analyze", false, "Analyze logs and show statistics")
	cmd.Flags().StringVar(&trimJSON, "trim-json", "", "Write deduplicated logs to a JSON file at specified path")
	cmd.Flags().BoolVar(&interactive, "interactive", false, "Launch interactive TUI mode")
	cmd.Flags().BoolVar(&verboseAnalysis, "verbose-analysis", false, "Show detailed analysis with all sections")
	cmd.Flags().BoolVar(&rawOutput, "raw", false, "Output raw log entries instead of analysis (old default behavior)")

	registerFlagCompletion(cmd, "csv", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveDefault
	})
	registerFlagCompletion(cmd, "output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveDefault
	})
	registerFlagCompletion(cmd, "trim-json", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveDefault
	})
	for _, flag := range []string{"json", "analyze", "interactive", "verbose-analysis", "raw"} {
		registerFlagCompletion(cmd, flag, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
		})
	}
}

// addAIFlags registers LLM-specific flags for ai-analyze subcommands.
func addAIFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key for LLM provider")
	cmd.Flags().StringVar(&llmProvider, "llm-provider", "anthropic", "LLM provider to use (anthropic, openai, gemini, ollama)")
	cmd.Flags().StringVar(&llmModel, "llm-model", "", "LLM model to use (defaults to provider-specific default)")
	cmd.Flags().StringVar(&problem, "problem", "", "Description of the problem you're investigating")
	cmd.Flags().IntVar(&thinkingBudget, "thinking-budget", 0, "Token budget for extended thinking mode (only supported by some models)")
	cmd.Flags().StringVar(&ollamaHost, "ollama-host", "http://localhost:11434", "Ollama server URL (only for ollama provider)")
	cmd.Flags().IntVar(&ollamaTimeout, "ollama-timeout", 120, "Timeout in seconds for Ollama requests (only for ollama provider)")
	cmd.Flags().IntVar(&maxEntries, "max-entries", 100, "Maximum number of log entries to send to LLM")
	cmd.Flags().StringVar(&trimJSON, "trim-json", "", "Write deduplicated logs to a JSON file at specified path")
	cmd.Flags().BoolVar(&autoConfirm, "yes", false, "Skip confirmation prompts and use --max-entries limit")

	registerFlagCompletion(cmd, "llm-provider", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"anthropic", "openai", "gemini", "ollama"}, cobra.ShellCompDirectiveNoFileComp
	})
	registerFlagCompletion(cmd, "llm-model", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		provider := cmd.Flag("llm-provider").Value.String()
		if provider == "" {
			provider = "anthropic"
		}
		var modelNames []string
		models := GetAvailableModels(LLMProvider(provider))
		for _, model := range models {
			modelNames = append(modelNames, model.ID)
		}
		return modelNames, cobra.ShellCompDirectiveNoFileComp
	})
	registerFlagCompletion(cmd, "trim-json", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveDefault
	})
}

func initLogger() {
	// Set log level based on flags
	logLevel := slog.LevelInfo
	switch {
	case quiet:
		logLevel = slog.LevelError
	case verbose:
		logLevel = slog.LevelDebug
	}

	// Create handler with the appropriate level
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})

	// Initialize global logger
	logger = slog.New(handler)
	slog.SetDefault(logger)
}

func init() {
	// Enable command completion
	rootCmd.CompletionOptions.DisableDefaultCmd = false

	// --verbose and --quiet apply to all subcommands
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose output logging")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "Only output errors")
	registerFlagCompletion(rootCmd, "verbose", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
	})
	registerFlagCompletion(rootCmd, "quiet", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
	})

	// Add subcommands to root
	rootCmd.AddCommand(fileCmd)
	rootCmd.AddCommand(notificationCmd)
	rootCmd.AddCommand(supportPacketCmd)
	rootCmd.AddCommand(aiCmd)
	rootCmd.AddCommand(versionCmd)

	// Add AI subcommands
	aiCmd.AddCommand(aiFileCmd)
	aiCmd.AddCommand(aiNotificationCmd)
	aiCmd.AddCommand(aiSupportPacketCmd)

	// Flags for standard log processing commands
	for _, cmd := range []*cobra.Command{fileCmd, notificationCmd, supportPacketCmd} {
		addFilterFlags(cmd)
		addOutputFlags(cmd)
	}

	// Flags for AI analysis commands
	for _, cmd := range []*cobra.Command{aiFileCmd, aiNotificationCmd, aiSupportPacketCmd} {
		addFilterFlags(cmd)
		addAIFlags(cmd)
	}
}

// exitCodeForError returns the appropriate exit code for err.
//
// Exit 2 is used when the error is a MisuseError or when cobra never entered
// PersistentPreRun (meaning it rejected the command before execution — wrong
// arg count, unknown flag, etc.). Exit 1 is used for all other errors.
func exitCodeForError(err error, preRunEntered bool) int {
	if err == nil {
		return 0
	}

	if _, ok := errors.AsType[*MisuseError](err); !preRunEntered || ok {
		return 2
	}
	return 1
}

func main() {
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCodeForError(err, enteredPreRun))
	}
}

// contains checks if a string slice contains a given string
func contains(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

// loadFileLogs loads and merges log entries from one or more files.
func loadFileLogs(paths []string) ([]LogEntry, error) {
	if len(paths) == 1 {
		filePath := paths[0]
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return nil, fmt.Errorf("file '%s' does not exist", filePath)
		}
		logs, err := parseLogFile(filePath, searchTerm, regexSearch, levelFilter, minLevelFilter, userFilter, startTime, endTime)
		if err != nil {
			return nil, fmt.Errorf("error parsing log file: %v", err)
		}
		return logs, nil
	}

	var allLogs []LogEntry

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
	if trim && !autoConfirm && isTerminal() {
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

// isTerminal reports whether stdin is connected to an interactive terminal.
func isTerminal() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
