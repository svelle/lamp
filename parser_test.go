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
