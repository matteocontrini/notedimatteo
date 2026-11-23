package db

import (
	"slices"
	"testing"
	"time"
)

func TestCountByMonth(t *testing.T) {
	rome, err := time.LoadLocation("Europe/Rome")
	if err != nil {
		t.Fatal(err)
	}
	utc := func(value string) time.Time {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			t.Fatal(err)
		}
		return parsed
	}

	times := []time.Time{
		utc("2026-12-31T23:30:00Z"), // 1 January 2027, 00:30 in Rome (UTC+1)
		utc("2026-12-31T22:30:00Z"), // 31 December 2026, 23:30 in Rome
		utc("2026-09-30T22:30:00Z"), // 1 October, 00:30 in Rome (UTC+2)
		utc("2026-09-30T21:30:00Z"), // 30 September, 23:30 in Rome
		utc("2026-09-10T10:00:00Z"),
	}
	want := []MonthCount{
		{Year: 2027, Month: 1, Count: 1},
		{Year: 2026, Month: 12, Count: 1},
		{Year: 2026, Month: 10, Count: 1},
		{Year: 2026, Month: 9, Count: 2},
	}
	if got := countByMonth(times, rome); !slices.Equal(got, want) {
		t.Errorf("countByMonth = %v, want %v", got, want)
	}
	if got := countByMonth(nil, rome); len(got) != 0 {
		t.Errorf("countByMonth(nil) = %v", got)
	}
}
