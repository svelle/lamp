package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// Define command line flags
	filePath := flag.String("file", "", "Path to the Mattermost log file")
	supportPacket := flag.String("support-packet", "", "Path to a Mattermost support packet zip file")
	searchTerm := flag.String("search", "", "Search term to filter logs")
	levelFilter := flag.String("level", "", "Filter logs by level (info, error, debug, etc.)")
	userFilter := flag.String("user", "", "Filter logs by username")
	jsonOutput := flag.Bool("json", false, "Output in JSON format")
	help := flag.Bool("help", false, "Show help information")

	// Parse command line arguments
	flag.Parse()

	// Show help information if requested or no files provided
	if *help || (*filePath == "" && *supportPacket == "") {
		printUsage()
		return
	}

	var logs []LogEntry
	var err error

	if *supportPacket != "" {
		// Process support packet
		if _, err := os.Stat(*supportPacket); os.IsNotExist(err) {
			fmt.Printf("Error: Support packet '%s' does not exist\n", *supportPacket)
			os.Exit(1)
		}
		logs, err = parseSupportPacket(*supportPacket, *searchTerm, *levelFilter, *userFilter)
	} else {
		// Process single log file
		if _, err := os.Stat(*filePath); os.IsNotExist(err) {
			fmt.Printf("Error: File '%s' does not exist\n", *filePath)
			os.Exit(1)
		}
		logs, err = parseLogFile(*filePath, *searchTerm, *levelFilter, *userFilter)
	}
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
	fmt.Println("Usage: mlp --file <path> [options] OR mlp --support-packet <path> [options]")
	fmt.Println("\nOptions:")
	fmt.Println("  --file <path>            Path to the Mattermost log file")
	fmt.Println("  --support-packet <path>  Path to a Mattermost support packet zip file")
	fmt.Println("  --search <term>          Search term to filter logs")
	fmt.Println("  --level <level>          Filter logs by level (info, error, debug, etc.)")
	fmt.Println("  --user <username>        Filter logs by username")
	fmt.Println("  --json                   Output in JSON format")
	fmt.Println("  --help                   Show this help information")
	fmt.Println("\nExamples:")
	fmt.Println("  mlp --file mattermost.log")
	fmt.Println("  mlp --file mattermost.log --search \"error\"")
	fmt.Println("  mlp --file mattermost.log --level error --user admin")
	fmt.Println("  mlp --support-packet mattermost_support_packet.zip")
	fmt.Println("  mlp --support-packet mattermost_support_packet.zip --level error")
}
