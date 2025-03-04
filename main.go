package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// Define command line flags
	filePath := flag.String("file", "", "Path to the Mattermost log file")
	searchTerm := flag.String("search", "", "Search term to filter logs")
	levelFilter := flag.String("level", "", "Filter logs by level (info, error, debug, etc.)")
	userFilter := flag.String("user", "", "Filter logs by username")
	jsonOutput := flag.Bool("json", false, "Output in JSON format")
	help := flag.Bool("help", false, "Show help information")

	// Parse command line arguments
	flag.Parse()

	// Show help information if requested or no file provided
	if *help || *filePath == "" {
		printUsage()
		return
	}

	// Check if the file exists
	if _, err := os.Stat(*filePath); os.IsNotExist(err) {
		fmt.Printf("Error: File '%s' does not exist\n", *filePath)
		os.Exit(1)
	}

	// Parse and filter logs
	logs, err := parseLogFile(*filePath, *searchTerm, *levelFilter, *userFilter)
	if err != nil {
		fmt.Printf("Error parsing log file: %v\n", err)
		os.Exit(1)
	}

	// Display logs in the requested format
	if *jsonOutput {
		displayLogsJSON(logs)
	} else {
		displayLogsPretty(logs)
	}
}

func printUsage() {
	fmt.Println("Mattermost Log Parser (mlp)")
	fmt.Println("Usage: mlp --file <path> [options]")
	fmt.Println("\nOptions:")
	fmt.Println("  --file <path>     Path to the Mattermost log file (required)")
	fmt.Println("  --search <term>   Search term to filter logs")
	fmt.Println("  --level <level>   Filter logs by level (info, error, debug, etc.)")
	fmt.Println("  --user <username> Filter logs by username")
	fmt.Println("  --json            Output in JSON format")
	fmt.Println("  --help            Show this help information")
	fmt.Println("\nExamples:")
	fmt.Println("  mlp --file mattermost.log")
	fmt.Println("  mlp --file mattermost.log --search \"error\"")
	fmt.Println("  mlp --file mattermost.log --level error --user admin")
}
