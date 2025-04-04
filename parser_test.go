package main

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePlainTextLogLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    LogEntry
		wantErr bool
	}{
		{
			name: "basic plain text log",
			line: `debug [2025-02-27 15:42:40.076 Z] Received HTTP request caller="web/handlers.go:187" method=GET url=/api/v4/groups request_id=1yuo8z88cp8nzxza6w9ij6khnr user_id=gyd6suh8a3fcukcaqkn3zo3o9y status_code=200`,
			want: LogEntry{
				Timestamp: mustParseTime(t, "2025-02-27 15:42:40.076 Z"),
				Level:     "debug",
				Message:   "Received HTTP request",
				Source:    "web/handlers.go:187",
				User:      "gyd6suh8a3fcukcaqkn3zo3o9y",
				Extras: map[string]string{
					"method":      "GET",
					"url":         "/api/v4/groups",
					"request_id":  "1yuo8z88cp8nzxza6w9ij6khnr",
					"status_code": "200",
				},
			},
			wantErr: false,
		},
		{
			name: "plain text log without caller",
			line: `info [2025-02-27 15:42:40.076 Z] User logged in user_id=abc123 ip_address=192.168.1.1`,
			want: LogEntry{
				Timestamp: mustParseTime(t, "2025-02-27 15:42:40.076 Z"),
				Level:     "info",
				Message:   "User logged in",
				User:      "abc123",
				Extras: map[string]string{
					"ip_address": "192.168.1.1",
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid plain text format",
			line:    `not a valid log line`,
			wantErr: true,
		},
		{
			name:    "empty line",
			line:    "",
			wantErr: true,
		},
		// TODO: Fix implementation to handle this case
		// {
		// 	name: "plain text log with license info",
		// 	line: `info  [2025-03-20 11:02:02.785 +01:00] Set license caller="platform/license.go:392" id=K9fGlbHegqb5F4KjP3zaoNqZ4L issued_at="2024-10-15 13:39:48.515 +02:00" starts_at="2024-10-15 13:39:48.515 +02:00" expires_at="2026-10-15 06:00:00.000 +02:00" sku_name=Enterprise sku_short_name=enterprise is_trial=false is_gov_sku=false customer_id=p9un369a67ksmj4yd6i6ib39wh features.users=200000 features=mfa=true,message_export=true,guest_accounts_permissions=true,elastic_search=true,id_loaded=true,office365=true,compliance=true,email_notification_contents=true,cloud=false,shared_channels=true,saml=true,enterprise_plugins=true,future=true,metrics=true,mhpns=true,data_retention=true,guest_accounts=true,outgoing_oauth_connections=true,lock_teammate_name_display=true,advanced_logging=true,google=true,openid=true,custom_permissions_schemes=true,ldap=true,ldap_groups=true,cluster=true,remote_cluster_service=true`,
		// 	want: LogEntry{
		// 		Timestamp: mustParseTime(t, "2025-03-20 10:02:02.785 Z"),
		// 		Level:     "info",
		// 		Message:   "Set license",
		// 		Source:    "platform/license.go:392",
		// 		Extras: map[string]string{
		// 			"id":             "K9fGlbHegqb5F4KjP3zaoNqZ4L",
		// 			"issued_at":      "2024-10-15 13:39:48.515 +02:00",
		// 			"starts_at":      "2024-10-15 13:39:48.515 +02:00",
		// 			"expires_at":     "2026-10-15 06:00:00.000 +02:00",
		// 			"sku_name":       "Enterprise",
		// 			"sku_short_name": "enterprise",
		// 			"is_trial":       "false",
		// 			"is_gov_sku":     "false",
		// 			"customer_id":    "p9un369a67ksmj4yd6i6ib39wh",
		// 			"features.users": "200000",
		// 			"features":       "mfa=true,message_export=true,guest_accounts_permissions=true,elastic_search=true,id_loaded=true,office365=true,compliance=true,email_notification_contents=true,cloud=false,shared_channels=true,saml=true,enterprise_plugins=true,future=true,metrics=true,mhpns=true,data_retention=true,guest_accounts=true,outgoing_oauth_connections=true,lock_teammate_name_display=true,advanced_logging=true,google=true,openid=true,custom_permissions_schemes=true,ldap=true,ldap_groups=true,cluster=true,remote_cluster_service=true",
		// 		},
		// 	},
		// 	wantErr: false,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLine(tt.line)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.True(t, got.Timestamp.Equal(tt.want.Timestamp))
			assert.Equal(t, tt.want.Level, got.Level)
			assert.Equal(t, tt.want.Message, got.Message)
			assert.Equal(t, tt.want.Source, got.Source)
			assert.Equal(t, tt.want.User, got.User)
			assert.Equal(t, tt.want.Extras, got.Extras)
		})
	}
}

func TestParseJSONLine(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    LogEntry
		wantErr bool
	}{
		{
			name: "valid JSON log format",
			input: `{
				"timestamp": "2025-02-27T15:42:40.076Z",
				"level": "debug",
				"msg": "Received HTTP request",
				"caller": "web/handlers.go:187",
				"user_id": "ABC123",
				"method": "GET",
				"url": "/api/v4/groups",
				"request_id": "XYZ789",
				"err": "some error",
				"status_code": "200"
			}`,
			want: LogEntry{
				Timestamp: mustParseTime(t, "2025-02-27 15:42:40.076 Z"),
				Level:     "debug",
				Message:   "Received HTTP request",
				Source:    "web/handlers.go:187",
				User:      "ABC123",
				Extras: map[string]string{
					"method":      "GET",
					"url":         "/api/v4/groups",
					"request_id":  "XYZ789",
					"status_code": "200",
					"err":         "some error",
				},
			},
			wantErr: false,
		},
		{
			name:  "complex JSON with license info",
			input: `{"timestamp":"2025-02-19 13:00:19.541 +01:00","level":"info","msg":"Set license","caller":"platform/license.go:392","id":"ntisr7wfwbghpyakh87fazbqma","issued_at":"2023-03-06 18:51:19.000 +01:00","starts_at":"2023-03-06 18:51:19.000 +01:00","expires_at":"2025-03-06 18:51:19.000 +01:00","sku_name":"Enterprise Dev","sku_short_name":"enterprise","is_trial":false,"is_gov_sku":false,"customer_id":"p9un369a67gimj4yd6i6ib39wh","features.users":200000,"features":{"advanced_logging":true,"cloud":false,"cluster":true,"compliance":true,"custom_permissions_schemes":true,"data_retention":true,"elastic_search":true,"email_notification_contents":true,"enterprise_plugins":true,"future":true,"google":true,"guest_accounts":true,"guest_accounts_permissions":true,"id_loaded":true,"ldap":true,"ldap_groups":true,"lock_teammate_name_display":true,"message_export":true,"metrics":true,"mfa":true,"mhpns":true,"office365":true,"openid":true,"outgoing_oauth_connections":true,"remote_cluster_service":true,"saml":true,"shared_channels":true}}`,
			want: LogEntry{
				Timestamp: mustParseTime(t, "2025-02-19 12:00:19.541 Z"), // Adjusted for UTC
				Level:     "info",
				Message:   "Set license",
				Source:    "platform/license.go:392",
				Extras: map[string]string{
					"id":             "ntisr7wfwbghpyakh87fazbqma",
					"issued_at":      "2023-03-06 18:51:19.000 +01:00",
					"starts_at":      "2023-03-06 18:51:19.000 +01:00",
					"expires_at":     "2025-03-06 18:51:19.000 +01:00",
					"sku_name":       "Enterprise Dev",
					"sku_short_name": "enterprise",
					"is_trial":       "false",
					"is_gov_sku":     "false",
					"customer_id":    "p9un369a67gimj4yd6i6ib39wh",
					"features.users": "200000",
					"features":       `{"advanced_logging":true,"cloud":false,"cluster":true,"compliance":true,"custom_permissions_schemes":true,"data_retention":true,"elastic_search":true,"email_notification_contents":true,"enterprise_plugins":true,"future":true,"google":true,"guest_accounts":true,"guest_accounts_permissions":true,"id_loaded":true,"ldap":true,"ldap_groups":true,"lock_teammate_name_display":true,"message_export":true,"metrics":true,"mfa":true,"mhpns":true,"office365":true,"openid":true,"outgoing_oauth_connections":true,"remote_cluster_service":true,"saml":true,"shared_channels":true}`,
				},
			},
			wantErr: false,
		},
		{
			name: "JSON with escaped quotes",
			input: `{
				"timestamp": "2025-02-27T15:42:40.076Z",
				"level": "error",
				"msg": "Error processing request with \"special\" characters",
				"caller": "api/handler.go:42"
			}`,
			want: LogEntry{
				Timestamp: mustParseTime(t, "2025-02-27 15:42:40.076 Z"),
				Level:     "error",
				Message:   "Error processing request with \"special\" characters",
				Source:    "api/handler.go:42",
				Extras:    map[string]string{},
			},
			wantErr: false,
		},
		{
			name:    "invalid JSON format",
			input:   `{"timestamp": "2025-02-27T15:42:40.076Z", "level": "debug", "msg": "incomplete json...`,
			wantErr: true,
		},
		{
			name:    "empty JSON",
			input:   "{}",
			want:    LogEntry{Extras: map[string]string{}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseJSONLine(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			assert.True(t, got.Timestamp.Equal(tt.want.Timestamp))
			assert.Equal(t, tt.want.Level, got.Level)
			assert.Equal(t, tt.want.Message, got.Message)
			assert.Equal(t, tt.want.Source, got.Source)
			assert.Equal(t, tt.want.User, got.User)
			assert.Equal(t, tt.want.Extras, got.Extras)
		})
	}
}

// Helper function to parse time without error handling for test data
func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	tme, err := time.Parse("2006-01-02 15:04:05.000 Z", s)
	require.NoError(t, err)
	return tme
}

func TestMultiFileLogProcessing(t *testing.T) {
	// Initialize the logger for tests
	initLogger()

	// Create temporary directory for test log files
	tempDir, err := os.MkdirTemp("", "lamp-test-")
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

	// Test parsing multiple log files
	t.Run("parse multiple log files", func(t *testing.T) {
		var allLogs []LogEntry
		
		// Process each file
		for _, filePath := range filePaths {
			logs, err := parseLogFile(filePath, "", "", "", "", "", "")
			require.NoError(t, err)
			allLogs = append(allLogs, logs...)
		}
		
		// Verify we got all 6 log entries
		assert.Equal(t, 6, len(allLogs))
		
		// Sort logs by timestamp (as would happen in the main.go file)
		sort.Slice(allLogs, func(i, j int) bool {
			return allLogs[i].Timestamp.Before(allLogs[j].Timestamp)
		})
		
		// Verify correct ordering after sorting
		expectedOrder := []string{
			"System started",
			"Config loaded",
			"User login",
			"Connection failed",
			"High memory usage",
			"Cache hit",
		}
		
		for i, msg := range expectedOrder {
			assert.Contains(t, allLogs[i].Message, msg, "Log entry %d should be %s", i, msg)
		}
	})

	t.Run("parse multiple log files with filters", func(t *testing.T) {
		var allLogs []LogEntry
		
		// Process each file with level filter
		for _, filePath := range filePaths {
			logs, err := parseLogFile(filePath, "", "", "info", "", "", "")
			require.NoError(t, err)
			allLogs = append(allLogs, logs...)
		}
		
		// We should only have the 3 info logs
		assert.Equal(t, 3, len(allLogs))
		
		// Verify all entries have info level
		for _, entry := range allLogs {
			assert.Equal(t, "info", entry.Level)
		}
		
		// Sort logs by timestamp
		sort.Slice(allLogs, func(i, j int) bool {
			return allLogs[i].Timestamp.Before(allLogs[j].Timestamp)
		})
		
		// Verify correct ordering of info logs
		expectedOrder := []string{
			"System started",
			"Config loaded",
			"User login",
		}
		
		for i, msg := range expectedOrder {
			assert.Contains(t, allLogs[i].Message, msg, "Log entry %d should be %s", i, msg)
		}
	})

	t.Run("parse with time range filter", func(t *testing.T) {
		var allLogs []LogEntry
		
		// Process each file with time range filter
		// Get logs between 10:01:00 and 10:06:00
		startTime := "2025-01-01 10:01:00.000"
		endTime := "2025-01-01 10:06:00.000"
		
		for _, filePath := range filePaths {
			logs, err := parseLogFile(filePath, "", "", "", "", startTime, endTime)
			require.NoError(t, err)
			allLogs = append(allLogs, logs...)
		}
		
		// We should have 3 logs in this time range
		assert.Equal(t, 3, len(allLogs))
		
		// Sort logs by timestamp
		sort.Slice(allLogs, func(i, j int) bool {
			return allLogs[i].Timestamp.Before(allLogs[j].Timestamp)
		})
		
		// Verify timestamps are within range
		startTimeParsed, _ := time.Parse("2006-01-02 15:04:05.000", startTime)
		endTimeParsed, _ := time.Parse("2006-01-02 15:04:05.000", endTime)
		
		for _, entry := range allLogs {
			assert.True(t, !entry.Timestamp.Before(startTimeParsed))
			assert.True(t, !entry.Timestamp.After(endTimeParsed))
		}
	})

	t.Run("handle missing file gracefully", func(t *testing.T) {
		var allLogs []LogEntry
		
		// Create a list with one valid file and one non-existent file
		mixedPaths := []string{
			filePaths[0], // Valid file
			filepath.Join(tempDir, "nonexistent.log"), // Non-existent file
		}
		
		// Process each file, skipping errors
		for _, filePath := range mixedPaths {
			logs, err := parseLogFile(filePath, "", "", "", "", "", "")
			if err == nil {
				allLogs = append(allLogs, logs...)
			}
		}
		
		// We should still have logs from the valid file
		assert.Equal(t, 2, len(allLogs))
	})
}
