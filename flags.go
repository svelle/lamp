package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

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

// registerFlagCompletion is a helper function that registers flag completion and panics on error.
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
