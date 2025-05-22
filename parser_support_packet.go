package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// supportPacketConfigContent stores the content of sanitized_config.json for AI analysis
var supportPacketConfigContent string

// clearSupportPacketConfig resets the config content for a new analysis
func clearSupportPacketConfig() {
	supportPacketConfigContent = ""
}

// parseSupportPacket extracts and parses logs from a Mattermost support packet zip file
func parseSupportPacket(zipFilePath, searchTerm, regexPattern, levelFilter, userFilter, startTimeStr, endTimeStr string) ([]LogEntry, error) {
	// Open the zip file
	reader, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open support packet: %v", err)
	}
	defer func() { _ = reader.Close() }()

	var allLogs []LogEntry

	// Create a temporary directory to extract files
	tempDir, err := os.MkdirTemp("", "lamp_support_packet")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }() // Clean up when done

	// Extract sanitized_config.json if needed for AI analysis
	var configPath string
	if aiAnalyze && includeConfig {
		for _, file := range reader.File {
			if strings.HasSuffix(file.Name, "sanitized_config.json") {
				configPath = filepath.Join(tempDir, "sanitized_config.json")
				if err := extractZipFile(file, configPath); err != nil {
					logger.Warn("Failed to extract sanitized_config.json from support packet", "file", file.Name, "error", err)
					configPath = "" // Reset if extraction failed
				} else {
					logger.Debug("Extracted sanitized_config.json for AI analysis", "path", configPath)
				}
				break
			}
		}
	}

	// Look for log files in the zip
	for _, file := range reader.File {
		// Check if it's a log file
		if strings.HasSuffix(file.Name, "mattermost.log") ||
			strings.HasSuffix(file.Name, "notifications.log") ||
			strings.Contains(file.Name, "/logs/") ||
			strings.Contains(file.Name, "\\logs\\") ||
			strings.Contains(file.Name, "notification") {

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

	// Store the config path globally if we extracted it for AI analysis
	// Read the config content now before temp directory cleanup if needed for AI analysis
	if configPath != "" && aiAnalyze && includeConfig {
		if configData, err := os.ReadFile(configPath); err == nil {
			supportPacketConfigContent = string(configData)
			logger.Debug("Loaded sanitized_config.json content for AI analysis", "size", len(configData))
		} else {
			logger.Warn("Failed to read sanitized_config.json content", "path", configPath, "error", err)
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
	defer func() { _ = src.Close() }()

	// Create the destination file
	dest, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer func() { _ = dest.Close() }()

	// Copy the contents
	_, err = io.Copy(dest, src)
	return err
}
