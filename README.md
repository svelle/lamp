# lamp 

`lamp` (Log Analyser for Mattermost Packet) is a command-line tool for parsing, filtering, and displaying Mattermost log files with enhanced readability.

## Features

- Parse both traditional and JSON-formatted Mattermost log entries
- Filter logs by search term, log level, and username
- Display logs in a human-readable colored format or as JSON
- Support for various Mattermost timestamp formats

## Installation

### Installing from releases

1. Visit the [releases page](https://github.com/svelle/lamp/releases/latest) on GitHub
2. Download the release for your operating system and architecture
3. Extract the downloaded archive
4. Move the `lamp` binary to a directory in your PATH:

   ```bash
   # Linux/macOS
   sudo mv lamp /usr/local/bin/

   # Windows
   # Move lamp.exe to a directory in your PATH
   ```

5. Verify the installation:
   ```bash
   lamp --version
   ```


### Building from source

**Prerequisites**:

- Go 1.23 or higher

1. Install directly using go install:
   ```bash
   go install github.com/svelle/lamp@latest
   ```

The binary will be installed to your `$GOPATH/bin` directory, which should be in your PATH. If it's not, add the following to your shell's configuration file (e.g., `~/.bashrc`, `~/.zshrc`):
   ```bash
   export PATH=$PATH:$(go env GOPATH)/bin
   ```

## Usage

Basic command structure:
```bash
lamp <command> [flags]
```

### Commands

- `file <path>`: Parse a single Mattermost log file
- `support-packet <path>`: Parse a Mattermost support packet zip file
- `version`: Print version and build information
- `completion`: Generate shell completion scripts
- `help`: Help about any command

### Options

- `--search <term>`: Search term to filter logs
- `--regex <pattern>`: Regular expression pattern to filter logs
- `--level <level>`: Filter logs by level (info, error, debug, etc.)
- `--user <username>`: Filter logs by username
- `--start <time>`: Filter logs after this time (format: 2006-01-02 15:04:05.000)
- `--end <time>`: Filter logs before this time (format: 2006-01-02 15:04:05.000)
- `--json`: Output in JSON format
- `--csv <path>`: Export logs to CSV file at specified path
- `--output <path>`: Save output to file instead of stdout
- `--analyze`: Analyze logs and show statistics
- `--ai-analyze`: Analyze logs using Claude AI
- `--api-key <key>`: Claude API key for AI analysis (or set CLAUDE_API_KEY environment variable)
- `--max-entries <num>`: Maximum number of log entries to send to Claude AI (default: 100)
- `--problem "<description>"`: Description of the problem you're investigating (helps guide AI analysis)
- `--interactive`: Launch interactive TUI mode for exploring logs
- `--verbose`: Enable debug level logging output
- `--quiet`: Only output errors (suppresses info, warn, and debug messages)
- `--help`: Show help information for any command

### Shell Completion

The tool supports shell completion for commands, flags, and arguments. To enable completion:

```bash
# For bash
lamp completion bash > /etc/bash_completion.d/lamp

# For zsh
lamp completion zsh > "${fpath[1]}/_lamp"

# For fish
lamp completion fish > ~/.config/fish/completions/lamp.fish
```

After enabling completion, you can use:
- Tab completion for commands: `lamp [tab]`
- Tab completion for log files: `lamp file [tab]`
- Tab completion for flag values: `lamp file log.txt --level [tab]`
- File path completion for relevant flags: `lamp file log.txt --output [tab]`

### Examples

Parse a single log file:
```bash
lamp file mattermost.log
```

Parse a support packet:
```bash
lamp support-packet mattermost_support_packet.zip
```

Filter logs containing the word "error":
```bash
lamp file mattermost.log --search "error"
```

Show only error-level logs:
```bash
lamp file mattermost.log --level error
```

Filter logs by user and output as JSON:
```bash
lamp file mattermost.log --user admin --json
```

Combine multiple filters:
```bash
lamp file mattermost.log --level error --search "database"
```

Filter logs by time range:
```bash
lamp file mattermost.log --start 2023-01-01T00:00:00 --end 2023-01-02T00:00:00
```

Use regular expressions for advanced filtering:
```bash
lamp file mattermost.log --regex "error.*database"
```

Export logs to CSV for spreadsheet analysis:
```bash
lamp file mattermost.log --csv logs_export.csv
```

Save output to a file:
```bash
lamp file mattermost.log --analyze --output analysis_report.txt
```

Show version information:
```bash
lamp version
```

Launch interactive TUI mode for exploring logs:
```bash
lamp file mattermost.log --interactive
```

Analyze logs using Claude AI:
```bash
# Using command line flag
lamp file mattermost.log --ai-analyze --api-key YOUR_API_KEY

# Using environment variable
export CLAUDE_API_KEY=YOUR_API_KEY
lamp file mattermost.log --ai-analyze

# Specify maximum number of log entries to analyze
lamp file mattermost.log --ai-analyze --max-entries 200

# Provide a problem statement to guide the analysis
lamp file mattermost.log --ai-analyze --problem "Users are reporting authentication failures"
```

Analyze support packet logs using Claude AI:
```bash
# Using command line flag
lamp support-packet mattermost_support_packet.zip --ai-analyze --api-key YOUR_API_KEY

# Using environment variable
export CLAUDE_API_KEY=YOUR_API_KEY
lamp support-packet mattermost_support_packet.zip --ai-analyze
```

## Output Format

### Pretty Print (Default)

The default output format includes:
- Colored timestamps and log levels for better readability
- Highlighted source/caller information
- Structured display of user information and additional details
- Summary count of displayed log entries

### JSON Output

When using the `--json` flag, the output will be formatted as a JSON array of log entries, useful for further processing or integration with other tools.

## Supported Log Formats

The parser supports both traditional Mattermost log formats and the newer JSON-formatted logs:

### Traditional Format
```
2023-04-15T14:22:34.123Z [INFO] api.user.login.success user_id=abc123 ip_address=192.168.1.1
```

### JSON Format
```
{"timestamp":"2025-02-14 17:11:10.308 Z","level":"debug","msg":"Email batching job ran.","caller":"email/email_batching.go:138","number_of_users":0}
```

## Support Packet Processing

The tool can extract and parse log files from Mattermost support packets. Support packets are ZIP files that contain server logs, configuration information, and diagnostic data. When using the `--support-packet` option, the tool will:

1. Extract log files from the ZIP archive
2. Parse each log file
3. Apply any specified filters (search term, level, user)
4. Display the combined results

This is particularly useful for analyzing logs from multi-node Mattermost deployments where each node's logs are included in the support packet.

## Log Analysis

The `--analyze` option provides a high-level overview of the log data, including:

- Basic statistics (total entries, time range, duration)
- Log level distribution and error rate
- Top log sources and active users
- Most frequent error messages
- Activity patterns by hour
- Common message patterns

This analysis helps quickly identify trends, issues, and patterns in large log files without having to manually review thousands of entries.

## Advanced Filtering

The tool provides several ways to filter logs:

- **Text Search**: Use `--search` to find logs containing specific text
- **Regular Expressions**: Use `--regex` for pattern matching
- **Level Filtering**: Use `--level` to focus on specific log levels
- **User Filtering**: Use `--user` to find logs related to specific users
- **Time Range**: Use `--start` and `--end` to filter logs within a specific time period

## Output Options

You can control how the results are displayed or saved:

- **Pretty Print**: Default colored output for human readability
- **JSON Format**: Use `--json` for machine-readable output
- **CSV Export**: Use `--csv` to export logs to a CSV file for spreadsheet analysis
- **File Output**: Use `--output` to save results to a file instead of displaying on screen

## Interactive Mode

The `--interactive` option launches a terminal-based UI that allows you to:

- Browse through logs with keyboard navigation
- Filter logs interactively
- View detailed information about each log entry
- Search within the loaded logs

This mode is particularly useful for exploring large log files or investigating complex issues.

## AI-Powered Log Analysis

The `--ai-analyze` option uses Claude Sonnet API to provide an intelligent analysis of your logs. This feature:

- Sends a sample of your logs to Claude for analysis
- Provides a comprehensive report of issues and patterns
- Identifies potential root causes for errors
- Offers recommendations for resolution
- Gives context and insights that might not be obvious from statistical analysis

To use this feature, you need a Claude API key from Anthropic. You can obtain one by signing up at [https://console.anthropic.com/](https://console.anthropic.com/).

You can provide the API key in two ways:
1. Using the `--api-key` command line flag
2. Setting the `CLAUDE_API_KEY` environment variable (more secure)

Note: When using AI analysis, a limited number of log entries are sent to the Claude API to stay within token limits. By default, the tool sends up to 100 entries, but you can adjust this with the `--max-entries` flag.

You can also provide a problem statement with the `--problem` flag to help guide the AI analysis toward specific issues you're investigating.

## Logging

`lamp` uses structured logging for its output. By default, it logs at the INFO level. You can modify the logging level using these flags:

- `--verbose`: Show detailed debug information
- `--quiet`: Only show error messages

These flags are mutually exclusive - if both are provided, `--quiet` takes precedence.

Examples:
```bash
# Default behavior - show info and error messages
lamp file logfile.txt

# Show detailed debug information
lamp file logfile.txt --verbose

# Only show errors
lamp file logfile.txt --quiet
```
## License

[Apache License 2.0](LICENSE)
