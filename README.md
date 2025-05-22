# lamp 

`lamp` (Log Analyser for Mattermost Packet) is a command-line tool for parsing, filtering, and analyzing Mattermost log files with enhanced readability and intelligent insights.

## Features

- **Smart analysis by default**: Automatically provides compact insights instead of overwhelming log dumps
- Parse both traditional and JSON-formatted Mattermost log entries
- Filter logs by search term, log level, username, and time ranges
- **Intelligent log analysis** with statistics, patterns, and trends
- **AI-powered analysis** using Claude, GPT, Gemini, or local models via Ollama
- Display logs in human-readable colored format, JSON, or export to CSV
- **Interactive terminal UI** for exploring large log files
- Support for various Mattermost timestamp formats and support packets

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
   lamp version
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

- `file <path...>`: Parse and analyze one or more Mattermost log files
- `notification <path>`: Parse and analyze a Mattermost notification log file  
- `support-packet <path>`: Parse and analyze a Mattermost support packet zip file
- `version`: Print version and build information
- `completion`: Generate shell completion scripts
- `help`: Help about any command

### Options

#### Analysis Options
- `--ai-analyze`: Analyze logs using AI (Claude, GPT, Gemini, or Ollama)
- `--analyze`: Show compact statistical analysis (same as default)
- `--verbose-analysis`: Show detailed analysis with full sections
- `--raw`: Output raw log entries instead of analysis

#### AI Configuration  
- `--api-key <key>`: API key for LLM provider
- `--llm-provider <provider>`: LLM provider (anthropic, openai, gemini, ollama) (default: anthropic)
- `--llm-model <model>`: LLM model to use (autocompletes based on provider)
- `--max-entries <num>`: Maximum log entries to send to AI (default: 100)
- `--problem "<description>"`: Problem description to guide AI analysis
- `--thinking-budget <tokens>`: Token budget for Claude's extended thinking mode
- `--ollama-host <url>`: Ollama server URL (default: http://localhost:11434)
- `--ollama-timeout <seconds>`: Ollama request timeout (default: 120)
- `--include-config`: Include Mattermost configuration in AI analysis (support-packet only)

#### Filtering Options
- `--search <term>`: Search term to filter logs
- `--regex <pattern>`: Regular expression pattern to filter logs  
- `--level <level>`: Filter by log level (info, error, debug, etc.) - supports autocomplete
- `--user <username>`: Filter logs by username
- `--start <time>`: Filter logs after this time (format: 2006-01-02 15:04:05.000)
- `--end <time>`: Filter logs before this time (format: 2006-01-02 15:04:05.000)
- `--trim`: Remove entries with duplicate information
- `--trim-json <path>`: Write deduplicated logs to JSON file

#### Output Options
- `--json`: Output in JSON format
- `--csv <path>`: Export logs to CSV file - supports file path autocomplete
- `--output <path>`: Save output to file - supports file path autocomplete
- `--interactive`: Launch interactive TUI mode for exploring logs

#### Logging Options
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

#### Basic Usage (New Default Behavior)

Analyze a single log file (shows compact analysis by default):
```bash
lamp file mattermost.log
```

Analyze multiple log files:
```bash
lamp file mattermost.log mattermost2.log mattermost3.log
```

Analyze a support packet:
```bash
lamp support-packet mattermost_support_packet.zip
```

Get detailed analysis with full sections:
```bash
lamp file mattermost.log --verbose-analysis
```

Get raw log output (old default behavior):
```bash
lamp file mattermost.log --raw
```

#### Filtering Examples

Filter logs containing "error" (still shows analysis by default):
```bash
lamp file mattermost.log --search "error"
```

Show analysis of only error-level logs:
```bash
lamp file mattermost.log --level error
```

Filter logs by user and export as JSON:
```bash
lamp file mattermost.log --user admin --json
```

Combine multiple filters:
```bash
lamp file mattermost.log --level error --search "database"
```

Get raw filtered logs instead of analysis:
```bash
lamp file mattermost.log --level error --raw
```

#### Advanced Filtering

Filter logs by time range:
```bash
lamp file mattermost.log --start 2023-01-01T00:00:00 --end 2023-01-02T00:00:00
```

Use regular expressions for advanced filtering:
```bash
lamp file mattermost.log --regex "error.*database"
```

#### Export and Output Options

Export analysis to CSV for spreadsheet analysis:
```bash
lamp file mattermost.log --csv logs_export.csv
```

Save analysis to a file:
```bash
lamp file mattermost.log --output analysis_report.txt
```

Export raw logs to CSV:
```bash
lamp file mattermost.log --raw --csv raw_logs.csv
```

#### Interactive and AI Analysis

Launch interactive TUI mode for exploring logs:
```bash
lamp file mattermost.log --interactive
```

Show version information:
```bash
lamp version
```

#### AI-Powered Analysis
```bash
# Using Anthropic Claude (default provider)
lamp file mattermost.log --ai-analyze --api-key YOUR_API_KEY

# Using environment variables
export ANTHROPIC_API_KEY=YOUR_API_KEY
lamp file mattermost.log --ai-analyze

# Using OpenAI GPT models
export OPENAI_API_KEY=YOUR_API_KEY
lamp file mattermost.log --ai-analyze --llm-provider openai

# Using Google Gemini models
export GEMINI_API_KEY=YOUR_API_KEY
lamp file mattermost.log --ai-analyze --llm-provider gemini

# Using local Ollama models (no API key required)
lamp file mattermost.log --ai-analyze --llm-provider ollama --llm-model llama3

# Using a specific provider and model with autocomplete
lamp file mattermost.log --ai-analyze --llm-provider anthropic --llm-model claude-opus-4-20250514

# Specify maximum number of log entries to analyze
lamp file mattermost.log --ai-analyze --max-entries 200

# Provide a problem statement to guide the analysis
lamp file mattermost.log --ai-analyze --problem "Users are reporting authentication failures"

# Use extended thinking mode with Claude (more detailed analysis)
lamp file mattermost.log --ai-analyze --thinking-budget 10000
```

Support packet AI analysis:
```bash
# Analyze entire support packet with AI
export ANTHROPIC_API_KEY=YOUR_API_KEY
lamp support-packet mattermost_support_packet.zip --ai-analyze

# Include Mattermost configuration for comprehensive analysis
lamp support-packet mattermost_support_packet.zip --ai-analyze --include-config

# Use Ollama for local analysis with configuration
lamp support-packet mattermost_support_packet.zip --ai-analyze --include-config --llm-provider ollama
```

## Output Formats

### Analysis (Default)

The default output provides a **compact analysis** instead of raw log dumps:
- **Header**: Entry count, duration, and error rate
- **Log levels**: Distribution with colored counts  
- **Top sources**: Most active log sources
- **Top errors**: Most frequent error messages (truncated for readability)
- **Peak hours**: Busiest time periods

Use `--verbose-analysis` for detailed analysis with full activity charts and patterns.

### Raw Log Output

When using the `--raw` flag, output includes:
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

The tool can extract and parse log files from Mattermost support packets. Support packets are ZIP files that contain server logs, configuration information, and diagnostic data. When using the `support-packet` command, the tool will:

1. Extract log files from the ZIP archive
2. Parse each log file
3. Apply any specified filters (search term, level, user)
4. Display the combined results

For AI analysis, you can also include the Mattermost configuration:

- Use `--include-config` to extract and include `sanitized_config.json` in the AI analysis
- This provides additional context about server configuration, helping AI identify misconfigurations
- Configuration data is only included when explicitly requested with the flag
- The AI can then correlate log issues with configuration settings for more comprehensive insights

This is particularly useful for analyzing logs from multi-node Mattermost deployments where each node's logs are included in the support packet, and when you need to understand how configuration affects the observed issues.

## Log Analysis

**Compact analysis** (now the default) provides a quick overview:
- Basic statistics (total entries, time range, duration, error rate)
- Log level distribution with colored counts
- Top 3 log sources and error messages  
- Top 3 peak activity hours

**Detailed analysis** (`--verbose-analysis`) includes additional insights:
- Full 24-hour activity charts with colored bars (skips zero-activity hours)
- Day-of-week activity patterns (when spanning multiple days)
- Monthly activity patterns (when spanning multiple months)
- Only shows sections with relevant data

**Explicit analysis** (`--analyze`) is the same as the default compact analysis.

This smart analysis helps quickly identify trends, issues, and patterns in large log files without having to manually review thousands of entries or deal with overwhelming terminal output.

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

The `--ai-analyze` option uses AI to provide an intelligent analysis of your logs. This feature:

- Sends a sample of your logs to an LLM for analysis
- Provides a comprehensive report of issues and patterns
- Identifies potential root causes for errors
- Offers recommendations for resolution
- Gives context and insights that might not be obvious from statistical analysis
- For support packets: optionally includes Mattermost configuration data (`--include-config`) to identify misconfigurations and provide configuration-specific recommendations

**Supported LLM providers:**
- **Anthropic Claude** (default) - Get API key from [console.anthropic.com](https://console.anthropic.com/)
- **OpenAI GPT** - Get API key from [platform.openai.com](https://platform.openai.com/)  
- **Google Gemini** - Get API key from [console.cloud.google.com](https://console.cloud.google.com/)
- **Ollama** - Local models, no API key required ([ollama.ai](https://ollama.ai/))

**API key configuration:**
1. Command line flag: `--api-key YOUR_KEY`
2. Environment variables:
   - `ANTHROPIC_API_KEY` for Claude
   - `OPENAI_API_KEY` for GPT models
   - `GEMINI_API_KEY` for Gemini models
   - No key needed for Ollama

**Provider and model selection:**
- `--llm-provider`: Choose provider (anthropic, openai, gemini, ollama)
- `--llm-model`: Specify model with **tab autocomplete** based on selected provider
- Models automatically complete based on your chosen provider

Note: When using AI analysis, a limited number of log entries are sent to the LLM provider to stay within token limits. By default, the tool sends up to 100 entries, but you can adjust this with the `--max-entries` flag.

You can also provide a problem statement with the `--problem` flag to help guide the AI analysis toward specific issues you're investigating.

## Logging

`lamp` uses structured logging for its output. By default, it logs at the INFO level. You can modify the logging level using these flags:

- `--verbose`: Show detailed debug information
- `--quiet`: Only show error messages

These flags are mutually exclusive - if both are provided, `--quiet` takes precedence.

Examples:
```bash
# Default behavior - show compact analysis with info and error messages
lamp file logfile.txt

# Show detailed debug information  
lamp file logfile.txt --verbose

# Only show errors
lamp file logfile.txt --quiet

# Get raw logs with detailed debug information
lamp file logfile.txt --raw --verbose
```
## License

[Apache License 2.0](LICENSE)
