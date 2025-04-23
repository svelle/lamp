# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands
- Build: `go build`
- Run: `go run main.go`
- Test all: `go test ./...`
- Test single: `go test -v -run TestFunctionName`
- Lint: `golint ./...`

## Code Style Guidelines
- Formatting: Use Go standard formatting (`gofmt` or `go fmt`)
- Imports: Group standard library, third-party, and local imports
- Error handling: Always check and handle errors appropriately
- Testing: Use testify package for assertions and test helpers
- Naming: Use camelCase for variables and PascalCase for exported functions
- Comments: Document exported functions following Go convention
- Log levels: Use debug/info/warn/error consistently based on severity

## Project Specific Conventions
- Parse functions return both result and error
- Support both JSON and plain text log formats
- Always sort log entries by timestamp
- UI components use tview and tcell packages
- Maintain backward compatibility with existing log formats