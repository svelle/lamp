package main

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiFileCommand(t *testing.T) {
	// Initialize the logger for tests
	initLogger()

	// Create temporary directory for test log files
	tempDir, err := os.MkdirTemp("", "lamp-cmd-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test log files with different timestamps
	logFiles := []struct {
		name     string
		contents []string
	}{
		{
			name: "file1.log",
			contents: []string{
				`info [2025-01-01 10:00:00.000 Z] System started caller="system/init.go:42"`,
				`error [2025-01-01 10:05:00.000 Z] Connection failed caller="network/conn.go:123" error="timeout"`,
			},
		},
		{
			name: "file2.log",
			contents: []string{
				`info [2025-01-01 10:02:30.000 Z] User login caller="auth/login.go:55" user_id="user123"`,
				`debug [2025-01-01 10:07:45.000 Z] Cache hit caller="cache/store.go:78" key="session-key"`,
			},
		},
		{
			name: "file3.log",
			contents: []string{
				`info [2025-01-01 10:01:15.000 Z] Config loaded caller="config/loader.go:33"`,
				`warn [2025-01-01 10:06:20.000 Z] High memory usage caller="monitor/usage.go:91" memory_pct=85`,
			},
		},
	}

	// Write test log files
	var filePaths []string
	for _, lf := range logFiles {
		path := filepath.Join(tempDir, lf.name)
		filePaths = append(filePaths, path)
		
		f, err := os.Create(path)
		require.NoError(t, err)
		
		for _, line := range lf.contents {
			_, err = f.WriteString(line + "\n")
			require.NoError(t, err)
		}
		
		f.Close()
	}

	// Test the file command with multiple files
	t.Run("file command with multiple files", func(t *testing.T) {
		// Store original stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Set up command arguments for multiple files
		cmd := &cobra.Command{}
		cmd.Flags().StringVar(&levelFilter, "level", "", "Filter logs by level")
		cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
		cmd.Flags().BoolVar(&trim, "trim", false, "Remove entries with duplicate information")
		
		// Call the RunE function from fileCmd
		err := fileCmd.RunE(cmd, filePaths)
		require.NoError(t, err)
		
		// Restore stdout
		w.Close()
		os.Stdout = oldStdout
		
		var buf bytes.Buffer
		_, err = buf.ReadFrom(r)
		require.NoError(t, err)
		output := buf.String()
		
		// Check output contains expected content
		assert.Contains(t, output, "System started")
		assert.Contains(t, output, "User login")
		assert.Contains(t, output, "Config loaded")
		assert.Contains(t, output, "Connection failed")
		assert.Contains(t, output, "High memory usage")
		assert.Contains(t, output, "Cache hit")
	})

	t.Run("file command with level filter", func(t *testing.T) {
		// Store original stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Set up command with level filter
		cmd := &cobra.Command{}
		cmd.Flags().StringVar(&levelFilter, "level", "", "Filter logs by level")
		cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
		cmd.Flags().BoolVar(&trim, "trim", false, "Remove entries with duplicate information")
		
		// Set the level filter to info
		levelFilter = "info"
		
		// Call the RunE function from fileCmd
		err := fileCmd.RunE(cmd, filePaths)
		require.NoError(t, err)
		
		// Restore stdout and reset levelFilter
		w.Close()
		os.Stdout = oldStdout
		levelFilter = "" // Reset for other tests
		
		var buf bytes.Buffer
		_, err = buf.ReadFrom(r)
		require.NoError(t, err)
		output := buf.String()
		
		// Check output contains only info logs
		assert.Contains(t, output, "System started")
		assert.Contains(t, output, "User login")
		assert.Contains(t, output, "Config loaded")
		
		// These should not be in the output
		assert.NotContains(t, output, "Connection failed") // error level
		assert.NotContains(t, output, "High memory usage") // warn level
		assert.NotContains(t, output, "Cache hit")         // debug level
	})

	t.Run("file command with non-existent file", func(t *testing.T) {
		// For this test, let's test one file at a time since the multiple files implementation
		// handles missing files differently (it skips them)
		nonExistentPath := filepath.Join(tempDir, "nonexistent.log")
		
		// Set up command
		cmd := &cobra.Command{}
		cmd.Flags().StringVar(&levelFilter, "level", "", "Filter logs by level")
		cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
		cmd.Flags().BoolVar(&trim, "trim", false, "Remove entries with duplicate information")
		
		// Call the RunE function with single non-existent file
		// In single file mode, it should return an error
		err := fileCmd.RunE(cmd, []string{nonExistentPath})
		
		// Error should be returned due to missing file
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("file command with mixed valid and invalid files", func(t *testing.T) {
		// Create a list with one valid file and one non-existent file
		mixedPaths := []string{
			filePaths[0],                               // Valid file
			filepath.Join(tempDir, "nonexistent.log"),  // Non-existent file
		}
		
		// Set up command
		cmd := &cobra.Command{}
		cmd.Flags().StringVar(&levelFilter, "level", "", "Filter logs by level")
		cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
		cmd.Flags().BoolVar(&trim, "trim", false, "Remove entries with duplicate information")

		// Create a buffer to capture logs
		var logOutput bytes.Buffer
		
		// Hold the original logger
		origLogger := logger
		
		// Create a new text handler that writes to our buffer
		handler := slog.NewTextHandler(&logOutput, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
		
		// Set up a new logger that writes to our buffer
		logger = slog.New(handler)
		
		// Call the RunE function from fileCmd with mixed files
		// In multiple file mode, it should not return an error for missing files
		err := fileCmd.RunE(cmd, mixedPaths)
		
		// Restore original logger
		logger = origLogger
		
		// Convert the log output to a string
		logString := logOutput.String()
		
		// No error should be returned
		assert.NoError(t, err)
		
		// But a warning should be logged about the missing file
		assert.Contains(t, logString, "does not exist")
		assert.Contains(t, logString, "skipping")
	})

	t.Run("process with multiple files - trim duplicates", func(t *testing.T) {
		// Create a file with duplicate entries
		dupFile := filepath.Join(tempDir, "duplicates.log")
		f, err := os.Create(dupFile)
		require.NoError(t, err)
		
		// Write same log message multiple times with slight variations
		for i := 0; i < 3; i++ {
			_, err = f.WriteString(`info [2025-01-01 11:00:00.000 Z] System check complete caller="system/checks.go:42" status="ok"` + "\n")
			require.NoError(t, err)
		}
		f.Close()
		
		// Store original stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		
		// Set up command with trim flag
		cmd := &cobra.Command{}
		cmd.Flags().StringVar(&levelFilter, "level", "", "Filter logs by level")
		cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
		cmd.Flags().BoolVar(&trim, "trim", false, "Remove entries with duplicate information")
		
		// Enable trimming
		trim = true
		
		// Call the RunE function from fileCmd with just the duplicates file
		err = fileCmd.RunE(cmd, []string{dupFile})
		require.NoError(t, err)
		
		// Restore stdout and reset trim flag
		w.Close()
		os.Stdout = oldStdout
		trim = false // Reset for other tests
		
		var buf bytes.Buffer
		_, err = buf.ReadFrom(r)
		require.NoError(t, err)
		output := buf.String()
		
		// Output should indicate deduplication 
		assert.Contains(t, output, "System check complete")
		
		// Check for indication of repeated entries - could be either "repeated" or "duplicate_count"
		// depending on display format
		assert.True(t, strings.Contains(output, "repeated") || 
		            strings.Contains(output, "duplicate_count") ||
		            strings.Contains(output, "3"), 
		            "Output should indicate duplicated entries")
	})
}