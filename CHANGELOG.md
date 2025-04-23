# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Markdown support for AI analysis results
- Support for multiple LLM providers (Anthropic and OpenAI)
- New `--llm-provider` flag to select the LLM provider
- New `--llm-model` flag to specify LLM model with auto-completion
- Implemented OpenAI API integration for log analysis
- Created central models registry for easier model management

### Changed
- Significant performance improvements to log trimming functionality:
  - Optimized regex processing with pattern precompilation
  - Enhanced string comparison algorithms with better short-circuiting
  - Added message normalization caching to avoid redundant processing
  - Implemented parallel processing for large log sets (1000+ entries)
  - Added smarter log grouping by log level to reduce comparison space
  - Optimized memory usage with periodic cache clearing and chunked processing
- Modified AI analysis to use a provider-agnostic approach
- Environment variable for Anthropic is now `ANTHROPIC_API_KEY`
- Refactored LLM analyzer code into a single, more maintainable module

### Fixed
- Ensured filtering happens before trimming to reduce resource usage

### Breaking Changes
- Removed support for `CLAUDE_API_KEY` environment variable, use `ANTHROPIC_API_KEY` instead

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