package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

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
			return &MisuseError{msg: err.Error()}
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
		if err := validateLevelFlags(levelFilter, minLevelFilter); err != nil {
			return &MisuseError{msg: err.Error()}
		}
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
