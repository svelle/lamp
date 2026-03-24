package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

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
			return &MisuseError{msg: err.Error()}
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
		if err := validateLevelFlags(levelFilter, minLevelFilter); err != nil {
			return &MisuseError{msg: err.Error()}
		}
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
