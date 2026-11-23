package db

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// timeFormat is how the database stores times: in UTC, with milliseconds, like
// SQLite's date functions. Every stored time has the same width, so comparing
// and sorting the text compares and sorts the times.
const timeFormat = "2006-01-02 15:04:05.000"

// Time is a time stored in the database. The queries use it for time columns;
// the other files convert it to time.Time.
type Time struct {
	time.Time
}

// currentTime returns the current time truncated to milliseconds, the precision
// of every stored time.
func currentTime() Time {
	return Time{time.Now().Truncate(time.Millisecond)}
}

// Value writes the time in timeFormat. It drops whatever is smaller than a
// millisecond.
func (t Time) Value() (driver.Value, error) {
	return t.UTC().Format(timeFormat), nil
}

// Scan reads a time in timeFormat.
func (t *Time) Scan(value any) error {
	var text string
	switch v := value.(type) {
	case string:
		text = v
	case []byte:
		text = string(v)
	default:
		return fmt.Errorf("scan time: %T isn't text", value)
	}
	parsed, err := time.Parse(timeFormat, text)
	if err != nil {
		return fmt.Errorf("scan time: %w", err)
	}
	t.Time = parsed
	return nil
}

// timePointer converts a nullable Time to a nullable time.Time.
func timePointer(t *Time) *time.Time {
	if t == nil {
		return nil
	}
	return &t.Time
}
