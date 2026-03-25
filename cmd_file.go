package main

import "github.com/spf13/cobra"

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

var aiFileCmd = &cobra.Command{
	Use:   "file [path...]",
	Short: "AI analyze one or more Mattermost log files",
	Args:  cobra.MinimumNArgs(1),
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
		return runAIAnalysis(logs)
	},
}
