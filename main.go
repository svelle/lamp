package main

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	searchTerm     string
	regexSearch    string
	levelFilter    string
	userFilter     string
	startTime      string
	endTime        string
	jsonOutput     bool
	csvOutput      string
	outputFile     string
	analyze        bool
	aiAnalyze      bool
	apiKey         string
	trim           bool
	trimJSON       string
	maxEntries     int
	problem        string
	thinkingBudget int
	interactive    bool
	verbose        bool
	quiet          bool

	// Global logger
	logger *slog.Logger
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "lamp",
	Short: "lamp is a tool for parsing and analyzing Mattermost log files",
	Long: `lamp (Log Analyser for Mattermost Packet) allows you to parse, filter, and analyze Mattermost log files
and support packets. It provides various filtering options, analysis capabilities,
and AI-powered insights using Claude AI.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initLogger()
	},
}

var fileCmd = &cobra.Command{
	Use:   "file [path]",
	Short: "Parse a single Mattermost log file",
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
			return fmt.Errorf("file '%s' does not exist", filePath)
		}

		logs, err := parseLogFile(filePath, searchTerm, regexSearch, levelFilter, userFilter, startTime, endTime)
		if err != nil {
			return fmt.Errorf("error parsing log file: %v", err)
		}

		return processLogs(logs)
	},
}

var notificationCmd = &cobra.Command{
	Use:   "notification [path]",
	Short: "Parse a Mattermost notification log file",
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

		logs, err := parseLogFile(filePath, searchTerm, regexSearch, levelFilter, userFilter, startTime, endTime)
		if err != nil {
			return fmt.Errorf("error parsing notification log file: %v", err)
		}

		return processLogs(logs)
	},
}

var supportPacketCmd = &cobra.Command{
	Use:   "support-packet [path]",
	Short: "Parse a Mattermost support packet zip file",
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

		logs, err := parseSupportPacket(packetPath, searchTerm, regexSearch, levelFilter, userFilter, startTime, endTime)
		if err != nil {
			return fmt.Errorf("error parsing support packet: %v", err)
		}

		if verbose {
			fmt.Printf("Debug: processing %d log entries\n", len(logs))
		}

		return processLogs(logs)
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

// registerFlagCompletion is a helper function that registers flag completion and panics on error
func registerFlagCompletion(cmd *cobra.Command, flag string, completionFunc func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)) {
	if err := cmd.RegisterFlagCompletionFunc(flag, completionFunc); err != nil {
		panic(fmt.Sprintf("failed to register completion for --%s flag: %v", flag, err))
	}
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

	// Add subcommands to root command
	rootCmd.AddCommand(fileCmd)
	rootCmd.AddCommand(notificationCmd)
	rootCmd.AddCommand(supportPacketCmd)
	rootCmd.AddCommand(versionCmd)

	// Add shared flags to both subcommands
	commands := []*cobra.Command{fileCmd, notificationCmd, supportPacketCmd}
	for _, cmd := range commands {
		cmd.Flags().StringVar(&searchTerm, "search", "", "Search term to filter logs")
		cmd.Flags().StringVar(&regexSearch, "regex", "", "Regular expression pattern to filter logs")
		cmd.Flags().StringVar(&levelFilter, "level", "", "Filter logs by level (info, error, debug, etc.)")
		cmd.Flags().StringVar(&userFilter, "user", "", "Filter logs by username")
		cmd.Flags().StringVar(&startTime, "start", "", "Filter logs after this time (format: 2006-01-02 15:04:05.000)")
		cmd.Flags().StringVar(&endTime, "end", "", "Filter logs before this time (format: 2006-01-02 15:04:05.000)")
		cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
		cmd.Flags().StringVar(&csvOutput, "csv", "", "Export logs to CSV file at specified path")
		cmd.Flags().StringVar(&outputFile, "output", "", "Save output to file instead of stdout")
		cmd.Flags().BoolVar(&analyze, "analyze", false, "Analyze logs and show statistics")
		cmd.Flags().BoolVar(&aiAnalyze, "ai-analyze", false, "Analyze logs using Claude AI")
		cmd.Flags().StringVar(&apiKey, "api-key", "", "Claude API key for AI analysis")
		cmd.Flags().BoolVar(&trim, "trim", false, "Remove entries with duplicate information")
		cmd.Flags().StringVar(&trimJSON, "trim-json", "", "Write deduplicated logs to a JSON file at specified path")
		cmd.Flags().IntVar(&maxEntries, "max-entries", 100, "Maximum number of log entries to send to Claude AI")
		cmd.Flags().StringVar(&problem, "problem", "", "Description of the problem you're investigating")
		cmd.Flags().IntVar(&thinkingBudget, "thinking-budget", 0, "Token budget for Claude's extended thinking mode")
		cmd.Flags().BoolVar(&interactive, "interactive", false, "Launch interactive TUI mode")
		cmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose output logging")
		cmd.Flags().BoolVar(&quiet, "quiet", false, "Only output errors")

		// Add custom completion for flags
		registerFlagCompletion(cmd, "level", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return []string{"debug", "info", "warn", "error", "fatal", "panic"}, cobra.ShellCompDirectiveNoFileComp
		})

		// Add file completion for flags that expect file paths
		registerFlagCompletion(cmd, "csv", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveDefault
		})

		registerFlagCompletion(cmd, "output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveDefault
		})

		registerFlagCompletion(cmd, "trim-json", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveDefault
		})

		// Add boolean flag completion
		for _, flag := range []string{"json", "analyze", "ai-analyze", "trim", "interactive", "verbose", "quiet"} {
			registerFlagCompletion(cmd, flag, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
			})
		}

		// Add time format hint completion
		for _, flag := range []string{"start", "end"} {
			registerFlagCompletion(cmd, flag, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				return []string{"2006-01-02 15:04:05.000"}, cobra.ShellCompDirectiveNoFileComp
			})
		}
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// processLogs handles the common log processing logic
func processLogs(logs []LogEntry) error {
	// Apply trim if requested
	if trim {
		logger.Info("Starting deduplication", "count", len(logs))
		originalCount := len(logs)
		logs = trimDuplicateLogInfo(logs)
		logger.Info("finished deduplication",
			"original", originalCount,
			"final", len(logs),
			"removed", originalCount-len(logs))

		if trimJSON != "" {
			if err := writeLogsToJSON(logs, trimJSON); err != nil {
				return fmt.Errorf("error writing deduplicated logs to JSON: %v", err)
			}
			logger.Info("wrote deduplicated logs", "file", trimJSON)
		}
	}

	// Set output destination
	output := os.Stdout
	if outputFile != "" {
		file, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("error creating output file: %v", err)
		}
		defer file.Close()
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
	case aiAnalyze:
		apiKeyValue := apiKey
		if apiKeyValue == "" {
			apiKeyValue = os.Getenv("CLAUDE_API_KEY")
			if apiKeyValue == "" {
				return fmt.Errorf("Claude API key is required for AI analysis")
			}
		}
		
		// If trim was used, ask if user wants to send all remaining lines
		entriesForAnalysis := maxEntries
		if trim {
			fmt.Printf("After trimming, there are %d log entries. Would you like to analyze all of them? (y/n): ", len(logs))
			var response string
			_, err := fmt.Scanln(&response)
			if err != nil {
				// Default to 'no' if there's an error with input
				response = "n"
			}
			
			if strings.ToLower(response) == "y" || strings.ToLower(response) == "yes" {
				entriesForAnalysis = len(logs)
			}
		}
		
		analyzeWithClaude(logs, apiKeyValue, entriesForAnalysis, problem, thinkingBudget)
	case analyze:
		analyzeAndDisplayStats(logs, output, !trim)
	case jsonOutput:
		displayLogsJSON(logs, output)
	default:
		displayLogsPretty(logs, output)
	}

	return nil
}
