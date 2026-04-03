package handlers

import (
	"testing"
	"time"
)

func TestParseFlexibleTime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, got time.Time)
	}{
		{
			name:  "RFC3339",
			input: "2025-06-15T10:30:00Z",
			check: func(t *testing.T, got time.Time) {
				if got.Year() != 2025 || got.Month() != 6 || got.Day() != 15 {
					t.Errorf("unexpected date: %v", got)
				}
			},
		},
		{
			name:  "RFC3339 with offset",
			input: "2025-06-15T10:30:00+03:00",
			check: func(t *testing.T, got time.Time) {
				if got.Year() != 2025 || got.Month() != 6 || got.Day() != 15 {
					t.Errorf("unexpected date: %v", got)
				}
			},
		},
		{
			name:  "Go time.Time.String() UTC",
			input: "2025-06-15 10:30:00 +0000 UTC",
			check: func(t *testing.T, got time.Time) {
				if got.Year() != 2025 || got.Month() != 6 || got.Day() != 15 {
					t.Errorf("unexpected date: %v", got)
				}
			},
		},
		{
			name:  "Go time.Time.String() with timezone",
			input: "2025-06-15 10:30:00 -0700 MST",
			check: func(t *testing.T, got time.Time) {
				if got.Year() != 2025 || got.Month() != 6 || got.Day() != 15 {
					t.Errorf("unexpected date: %v", got)
				}
			},
		},
		{
			name:  "SQLite default format",
			input: "2025-06-15 10:30:00",
			check: func(t *testing.T, got time.Time) {
				if got.Year() != 2025 || got.Month() != 6 || got.Day() != 15 {
					t.Errorf("unexpected date: %v", got)
				}
				if got.Hour() != 10 || got.Minute() != 30 {
					t.Errorf("unexpected time: %v", got)
				}
			},
		},
		{
			name:  "SQLite with timezone",
			input: "2025-06-15 10:30:00-07:00",
			check: func(t *testing.T, got time.Time) {
				if got.Year() != 2025 || got.Month() != 6 || got.Day() != 15 {
					t.Errorf("unexpected date: %v", got)
				}
			},
		},
		{
			name:  "ISO 8601 T Z",
			input: "2025-06-15T10:30:00Z",
			check: func(t *testing.T, got time.Time) {
				if got.Year() != 2025 || got.Month() != 6 || got.Day() != 15 {
					t.Errorf("unexpected date: %v", got)
				}
			},
		},
		{
			name:  "Go format with monotonic clock suffix",
			input: "2025-06-15 10:30:00 +0000 UTC m=+3608.716621793",
			check: func(t *testing.T, got time.Time) {
				if got.Year() != 2025 || got.Month() != 6 || got.Day() != 15 {
					t.Errorf("unexpected date: %v", got)
				}
			},
		},
		{
			name:    "completely invalid",
			input:   "not-a-date",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFlexibleTime(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}
