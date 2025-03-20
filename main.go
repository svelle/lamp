package main

import (
	"fmt"
	"os"

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
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mlp",
	Short: "Mattermost Log Parser (mlp) is a tool for parsing and analyzing Mattermost log files",
	Long: `Mattermost Log Parser (mlp) allows you to parse, filter, and analyze Mattermost log files
and support packets. It provides various filtering options, analysis capabilities,
and AI-powered insights using Claude AI.`,
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

		return processLogs(logs)
	},
}

func init() {
	// Enable command completion
	rootCmd.CompletionOptions.DisableDefaultCmd = false

	// Add subcommands to root command
	rootCmd.AddCommand(fileCmd)
	rootCmd.AddCommand(supportPacketCmd)

	// Add shared flags to both subcommands
	commands := []*cobra.Command{fileCmd, supportPacketCmd}
	for _, cmd := range commands {
		cmd.Flags().StringVar(&searchTerm, "search", "", "Search term to filter logs")
		cmd.Flags().StringVar(&regexSearch, "regex", "", "Regular expression pattern to filter logs")
		cmd.Flags().StringVar(&levelFilter, "level", "", "Filter logs by level (info, error, debug, etc.)")
		cmd.Flags().StringVar(&userFilter, "user", "", "Filter logs by username")
		cmd.Flags().StringVar(&startTime, "start", "", "Filter logs after this time (format: 2006-01-02T15:04:05)")
		cmd.Flags().StringVar(&endTime, "end", "", "Filter logs before this time (format: 2006-01-02T15:04:05)")
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

		// Add custom completion for flags
		cmd.RegisterFlagCompletionFunc("level", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return []string{"debug", "info", "warn", "error", "fatal", "panic"}, cobra.ShellCompDirectiveNoFileComp
		})

		// Add file completion for flags that expect file paths
		cmd.RegisterFlagCompletionFunc("csv", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveDefault
		})
		cmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveDefault
		})
		cmd.RegisterFlagCompletionFunc("trim-json", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveDefault
		})

		// Add boolean flag completion
		for _, flag := range []string{"json", "analyze", "ai-analyze", "trim", "interactive"} {
			cmd.RegisterFlagCompletionFunc(flag, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
			})
		}

		// Add time format hint completion
		for _, flag := range []string{"start", "end"} {
			cmd.RegisterFlagCompletionFunc(flag, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				return []string{"2006-01-02T15:04:05"}, cobra.ShellCompDirectiveNoFileComp
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
		fmt.Printf("Starting deduplication of %d log entries...\n", len(logs))
		originalCount := len(logs)
		logs = trimDuplicateLogInfo(logs)
		fmt.Printf("Trimmed from %d to %d entries after removing duplicates (removed %d entries)\n",
			originalCount, len(logs), originalCount-len(logs))

		if trimJSON != "" {
			if err := writeLogsToJSON(logs, trimJSON); err != nil {
				return fmt.Errorf("error writing deduplicated logs to JSON: %v", err)
			}
			fmt.Printf("Deduplicated logs written to JSON file: %s\n", trimJSON)
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
		analyzeWithClaude(logs, apiKeyValue, maxEntries, problem, thinkingBudget)
	case analyze:
		analyzeAndDisplayStats(logs, output, !trim)
	case jsonOutput:
		displayLogsJSON(logs, output)
	default:
		displayLogsPretty(logs, output)
	}

	return nil
}
