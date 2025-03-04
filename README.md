# Mattermost Log Parser (mlp)

A command-line tool for parsing, filtering, and displaying Mattermost log files with enhanced readability.

## Features

- Parse both traditional and JSON-formatted Mattermost log entries
- Filter logs by search term, log level, and username
- Display logs in a human-readable colored format or as JSON
- Support for various Mattermost timestamp formats

## Installation

### Prerequisites

- Go 1.18 or higher

### Building from source

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/mlp.git
   cd mlp
   ```

2. Build the application:
   ```bash
   go build -o mlp
   ```

3. (Optional) Move the binary to a directory in your PATH:
   ```bash
   sudo mv mlp /usr/local/bin/
   ```

## Usage

```
mlp --file <path> [options]
```

or

```
mlp --support-packet <path> [options]
```

### Options

- `--file <path>`: Path to the Mattermost log file
- `--support-packet <path>`: Path to a Mattermost support packet zip file
- `--search <term>`: Search term to filter logs
- `--level <level>`: Filter logs by level (info, error, debug, etc.)
- `--user <username>`: Filter logs by username
- `--json`: Output in JSON format
- `--help`: Show help information

### Examples

Display all logs from a file:
```bash
mlp --file mattermost.log
```

Filter logs containing the word "error":
```bash
mlp --file mattermost.log --search "error"
```

Show only error-level logs:
```bash
mlp --file mattermost.log --level error
```

Filter logs by user and output as JSON:
```bash
mlp --file mattermost.log --user admin --json
```

Combine multiple filters:
```bash
mlp --file mattermost.log --level error --search "database"
```

Parse logs from a Mattermost support packet:
```bash
mlp --support-packet mattermost_support_packet.zip
```

Filter logs from a support packet:
```bash
mlp --support-packet mattermost_support_packet.zip --level error
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

## License

[MIT License](LICENSE)
