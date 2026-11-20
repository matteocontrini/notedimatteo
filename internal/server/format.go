package server

import (
	"fmt"
	"strings"
	"time"
	_ "time/tzdata" // fallback for time zones if the system has no time zone database
)

var rome = mustLoadLocation("Europe/Rome")

var italianMonths = [...]string{
	"gennaio", "febbraio", "marzo", "aprile", "maggio", "giugno",
	"luglio", "agosto", "settembre", "ottobre", "novembre", "dicembre",
}

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// formatDate formats a date like "16 aprile 2026", in Rome's time zone.
func formatDate(t time.Time) string {
	t = t.In(rome)
	return fmt.Sprintf("%d %s %d", t.Day(), italianMonths[t.Month()-1], t.Year())
}

// formatTime formats a time like "16:20", in Rome's time zone.
func formatTime(t time.Time) string {
	return t.In(rome).Format("15:04")
}

// dateKey identifies a day in Rome's time zone.
func dateKey(t time.Time) string {
	return t.In(rome).Format(time.DateOnly)
}

// monthName returns the lowercase Italian name of a month, from 1 to 12.
func monthName(month int) string {
	if month < 1 || month > 12 {
		return ""
	}
	return italianMonths[month-1]
}

var categoryLabels = map[string]string{
	"note":       "Note",
	"til":        "TIL",
	"cit":        "Cit.",
	"segnalibri": "Segnalibri",
	"video":      "Video",
	"progetti":   "Progetti",
}

// categoryLabel returns a category's display name. Unknown categories are
// title-cased like in the SvelteKit app: "foo-bar" becomes "Foo Bar".
func categoryLabel(slug string) string {
	if label, ok := categoryLabels[slug]; ok {
		return label
	}

	var b strings.Builder
	prevWord := false
	for _, r := range strings.ReplaceAll(slug, "-", " ") {
		word := isASCIIWord(r)
		if word && !prevWord {
			r = []rune(strings.ToUpper(string(r)))[0]
		}
		b.WriteRune(r)
		prevWord = word
	}
	return b.String()
}

// isASCIIWord matches JavaScript's \w.
func isASCIIWord(r rune) bool {
	return r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
}
