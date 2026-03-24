package main

import "github.com/spf13/cobra"

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
)

// rootCmd represents the base command when called without any subcommands.
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

// aiCmd is the parent for AI-powered log analysis subcommands.
var aiCmd = &cobra.Command{
	Use:   "ai-analyze",
	Short: "Analyze Mattermost logs using AI",
	Long:  `Send parsed log entries to an LLM for analysis. Use subcommands to specify the log source.`,
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
