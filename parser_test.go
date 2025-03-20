package main

import (
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
				Timestamp: time.Date(2025, 2, 27, 15, 42, 40, 76000000, time.UTC),
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
				Timestamp: time.Date(2025, 2, 27, 15, 42, 40, 76000000, time.UTC),
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
