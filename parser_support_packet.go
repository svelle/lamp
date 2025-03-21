package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// parseSupportPacket extracts and parses logs from a Mattermost support packet zip file
func parseSupportPacket(zipFilePath, searchTerm, regexPattern, levelFilter, userFilter, startTimeStr, endTimeStr string) ([]LogEntry, error) {
	// Open the zip file
	reader, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open support packet: %v", err)
	}
	defer reader.Close()

	var allLogs []LogEntry

	// Create a temporary directory to extract files
	tempDir, err := os.MkdirTemp("", "lamp_support_packet")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up when done

	// Look for log files in the zip
	for _, file := range reader.File {
		// Check if it's a log file
		if strings.HasSuffix(file.Name, "mattermost.log") ||
			strings.Contains(file.Name, "/logs/") ||
			strings.Contains(file.Name, "\\logs\\") {

			// Extract the file
			extractedPath := filepath.Join(tempDir, filepath.Base(file.Name))
			if err := extractZipFile(file, extractedPath); err != nil {
				logger.Warn("Failed to extract file from support packet", "file", file.Name, "error", err)
				continue
			}

			// Parse the extracted log file
			logs, err := parseLogFile(extractedPath, searchTerm, regexPattern, levelFilter, userFilter, startTimeStr, endTimeStr)
			if err != nil {
				logger.Warn("Failed to parse log file", "file", file.Name, "error", err)
				continue
			}

			// Add to our collection
			allLogs = append(allLogs, logs...)
		}
	}

	if len(allLogs) == 0 {
		fmt.Println("No log files found in the support packet or no entries matched your criteria.")
	}

	return allLogs, nil
}

// extractZipFile extracts a single file from a zip archive to the specified path
func extractZipFile(file *zip.File, destPath string) error {
	// Open the file inside the zip
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	// Create the destination file
	dest, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dest.Close()

	// Copy the contents
	_, err = io.Copy(dest, src)
	return err
}
