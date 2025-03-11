package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	// Define command line flags
	filePath := flag.String("file", "", "Path to the Mattermost log file")
	supportPacket := flag.String("support-packet", "", "Path to a Mattermost support packet zip file")
	searchTerm := flag.String("search", "", "Search term to filter logs")
	regexSearch := flag.String("regex", "", "Regular expression pattern to filter logs")
	levelFilter := flag.String("level", "", "Filter logs by level (info, error, debug, etc.)")
	userFilter := flag.String("user", "", "Filter logs by username")
	startTime := flag.String("start", "", "Filter logs after this time (format: 2006-01-02T15:04:05)")
	endTime := flag.String("end", "", "Filter logs before this time (format: 2006-01-02T15:04:05)")
	jsonOutput := flag.Bool("json", false, "Output in JSON format")
	csvOutput := flag.String("csv", "", "Output logs to CSV file at specified path")
	outputFile := flag.String("output", "", "Save output to file instead of stdout")
	analyze := flag.Bool("analyze", false, "Analyze logs and show statistics")
	aiAnalyze := flag.Bool("ai-analyze", false, "Analyze logs using Claude AI")
	apiKey := flag.String("api-key", "", "Claude API key for AI analysis")
	trim := flag.Bool("trim", false, "Remove entries with duplicate information")
	maxEntries := flag.Int("max-entries", 100, "Maximum number of log entries to send to Claude AI")
	problem := flag.String("problem", "", "Description of the problem you're investigating (helps guide AI analysis)")
	interactive := flag.Bool("interactive", false, "Launch interactive TUI mode")
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
		logs, err = parseSupportPacket(*supportPacket, *searchTerm, *regexSearch, *levelFilter, *userFilter, *startTime, *endTime)
	} else {
		// Process single log file
		if _, err := os.Stat(*filePath); os.IsNotExist(err) {
			fmt.Printf("Error: File '%s' does not exist\n", *filePath)
			os.Exit(1)
		}
		logs, err = parseLogFile(*filePath, *searchTerm, *regexSearch, *levelFilter, *userFilter, *startTime, *endTime)
	}
	if err != nil {
		fmt.Printf("Error parsing log file: %v\n", err)
		os.Exit(1)
	}
	
	// Apply trim if requested
	if *trim {
		logs = trimDuplicateLogInfo(logs)
		fmt.Printf("Trimmed to %d entries after removing duplicates\n", len(logs))
	}

	// Redirect output if requested
	var outputWriter io.Writer = os.Stdout
	if *outputFile != "" {
		file, err := os.Create(*outputFile)
		if err != nil {
			fmt.Printf("Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		outputWriter = file
		fmt.Printf("Writing output to %s\n", *outputFile)
	}

	// Handle interactive mode
	if *interactive {
		if err := launchInteractiveMode(logs); err != nil {
			fmt.Printf("Error in interactive mode: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Export to CSV if requested
	if *csvOutput != "" {
		if err := exportToCSV(logs, *csvOutput); err != nil {
			fmt.Printf("Error exporting to CSV: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Logs exported to CSV file: %s\n", *csvOutput)
		return
	}

	// Display logs in the requested format
	if *aiAnalyze {
		// Get API key from flag or environment variable
		apiKeyValue := *apiKey
		if apiKeyValue == "" {
			apiKeyValue = os.Getenv("CLAUDE_API_KEY")
			if apiKeyValue == "" {
				fmt.Println("Error: Claude API key is required for AI analysis.")
				fmt.Println("Provide it using --api-key flag or set the CLAUDE_API_KEY environment variable.")
				os.Exit(1)
			}
		}
		analyzeWithClaude(logs, apiKeyValue, *maxEntries, *problem)
	} else if *analyze {
		analyzeAndDisplayStats(logs, outputWriter)
	} else if *jsonOutput {
		displayLogsJSON(logs, outputWriter)
	} else {
		displayLogsPretty(logs, outputWriter)
	}
}

func printUsage() {
	fmt.Println("Mattermost Log Parser (mlp)")
	fmt.Println("Usage: mlp --file <path> [options] OR mlp --support-packet <path> [options]")
	fmt.Println("\nOptions:")
	fmt.Println("  --file <path>            Path to the Mattermost log file")
	fmt.Println("  --support-packet <path>  Path to a Mattermost support packet zip file")
	fmt.Println("  --search <term>          Search term to filter logs")
	fmt.Println("  --regex <pattern>        Regular expression pattern to filter logs")
	fmt.Println("  --level <level>          Filter logs by level (info, error, debug, etc.)")
	fmt.Println("  --user <username>        Filter logs by username")
	fmt.Println("  --start <time>           Filter logs after this time (format: 2006-01-02T15:04:05)")
	fmt.Println("  --end <time>             Filter logs before this time (format: 2006-01-02T15:04:05)")
	fmt.Println("  --json                   Output in JSON format")
	fmt.Println("  --csv <path>             Export logs to CSV file at specified path")
	fmt.Println("  --output <path>          Save output to file instead of stdout")
	fmt.Println("  --analyze                Analyze logs and show statistics")
	fmt.Println("  --ai-analyze             Analyze logs using Claude AI")
	fmt.Println("  --api-key <key>          Claude API key for AI analysis (or set CLAUDE_API_KEY env var)")
	fmt.Println("  --trim                   Remove entries with duplicate information")
	fmt.Println("  --max-entries <num>      Maximum number of log entries to send to Claude AI (default: 100)")
	fmt.Println("  --problem \"<description>\" Description of the problem you're investigating (helps guide AI analysis)")
	fmt.Println("  --interactive            Launch interactive TUI mode for exploring logs")
	fmt.Println("  --help                   Show this help information")
	fmt.Println("\nExamples:")
	fmt.Println("  mlp --file mattermost.log")
	fmt.Println("  mlp --file mattermost.log --search \"error\"")
	fmt.Println("  mlp --file mattermost.log --level error --user admin")
	fmt.Println("  mlp --support-packet mattermost_support_packet.zip")
	fmt.Println("  mlp --support-packet mattermost_support_packet.zip --level error")
	fmt.Println("  mlp --file mattermost.log --analyze")
	fmt.Println("  mlp --support-packet mattermost_support_packet.zip --analyze")
	fmt.Println("  mlp --file mattermost.log --ai-analyze --api-key YOUR_API_KEY")
	fmt.Println("  mlp --file mattermost.log --trim --level error")
}
