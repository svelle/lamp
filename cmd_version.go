package main

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"
)

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
