package views

import (
	"testing"
	"time"
)

func TestFormatDateAndTime(t *testing.T) {
	tests := []struct {
		utc      string
		wantDate string
		wantTime string
	}{
		// Summer time (UTC+2)
		{"2026-04-16T22:30:00Z", "17 aprile 2026", "00:30"},
		// Standard time (UTC+1)
		{"2026-01-01T08:05:00Z", "1 gennaio 2026", "09:05"},
		{"2025-12-31T23:00:00Z", "1 gennaio 2026", "00:00"},
	}

	for _, tt := range tests {
		ts, err := time.Parse(time.RFC3339, tt.utc)
		if err != nil {
			t.Fatal(err)
		}
		if got := formatDate(ts); got != tt.wantDate {
			t.Errorf("formatDate(%s) = %q, want %q", tt.utc, got, tt.wantDate)
		}
		if got := formatTime(ts); got != tt.wantTime {
			t.Errorf("formatTime(%s) = %q, want %q", tt.utc, got, tt.wantTime)
		}
	}
}

func TestMonthName(t *testing.T) {
	if got := monthName(1); got != "gennaio" {
		t.Errorf("monthName(1) = %q", got)
	}
	if got := monthName(12); got != "dicembre" {
		t.Errorf("monthName(12) = %q", got)
	}
	if got := monthName(13); got != "" {
		t.Errorf("monthName(13) = %q", got)
	}
}

func TestCategoryLabel(t *testing.T) {
	tests := map[string]string{
		"cit":        "Cit.",
		"til":        "TIL",
		"foo-bar":    "Foo Bar",
		"l'arte":     "L'Arte",
		"già-fatto":  "Già Fatto",
		"snake_case": "Snake_case",
		"":           "",
	}

	for slug, want := range tests {
		if got := CategoryLabel(slug); got != want {
			t.Errorf("CategoryLabel(%q) = %q, want %q", slug, got, want)
		}
	}
}
