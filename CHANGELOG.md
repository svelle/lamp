# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Markdown support for AI analysis results

### Changed
- Significant performance improvements to log trimming functionality:
  - Optimized regex processing with pattern precompilation
  - Enhanced string comparison algorithms with better short-circuiting
  - Added message normalization caching to avoid redundant processing
  - Implemented parallel processing for large log sets (1000+ entries)
  - Added smarter log grouping by log level to reduce comparison space
  - Optimized memory usage with periodic cache clearing and chunked processing

### Fixed
- Ensured filtering happens before trimming to reduce resource usage

## [1.0.0] - 2024-02-27

### Added
- Initial release
- Support for parsing Mattermost log files
- Support for parsing Mattermost support packets
- Filtering options (level, time range, search term, regex)
- Output formats (pretty, JSON, CSV)
- Log analysis with statistics
- AI-powered analysis with Claude
- Interactive TUI mode