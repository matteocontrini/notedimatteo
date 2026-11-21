package views

import (
	"fmt"
	"strings"
	"time"
	_ "time/tzdata" // fallback for time zones if the system has no time zone database

	"github.com/matteocontrini/notedimatteo/internal/db"
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

// formatDateTime formats a time like "21/11/2026, 14:05", in Rome's time zone.
func formatDateTime(t time.Time) string {
	return t.In(rome).Format("02/01/2006, 15:04")
}

// timeUnits lists the units relativeTime uses. next is how many of a unit make
// the next one, like 60 seconds in a minute.
var timeUnits = []struct {
	name string
	next float64
}{
	{"second", 60},
	{"minute", 60},
	{"hour", 24},
	{"day", 7},
	{"week", 365.0 / 7 / 12},
	{"month", 12},
	{"year", 0},
}

// relativeTime says how long before now t was, in English, with the wording of
// the timeago.js package: "just now", "10 seconds ago", "1 week ago".
func relativeTime(t, now time.Time) string {
	amount := max(now.Sub(t).Seconds(), 0)
	unit := 0
	for unit < len(timeUnits)-1 && amount >= timeUnits[unit].next {
		amount /= timeUnits[unit].next
		unit++
	}

	n := int(amount)
	if unit == 0 && n < 10 {
		return "just now"
	}
	name := timeUnits[unit].name
	if n > 1 {
		name += "s"
	}
	return fmt.Sprintf("%d %s ago", n, name)
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

// MonthName returns the lowercase Italian name of a month, from 1 to 12.
func MonthName(month int) string {
	return monthName(month)
}

// CapitalizedMonthName returns the Italian name of a month with a capital letter.
func CapitalizedMonthName(month int) string {
	name := monthName(month)
	if name == "" {
		return ""
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

// PostTitle is a post's title, or "Nota #670 del 19/11/2026" for untitled posts.
func PostTitle(post db.Post) string {
	if post.Title != "" {
		return post.Title
	}
	return fmt.Sprintf("Nota #%d del %s", post.Number, post.Timestamp().In(rome).Format("02/01/2006"))
}

// Categories lists the categories in the order the editor offers them.
var Categories = []string{"note", "til", "cit", "segnalibri", "video", "progetti"}

var categoryLabels = map[string]string{
	"note":       "Note",
	"til":        "TIL",
	"cit":        "Cit.",
	"segnalibri": "Segnalibri",
	"video":      "Video",
	"progetti":   "Progetti",
}

// IsCategory reports whether a category has its own page.
func IsCategory(slug string) bool {
	_, ok := categoryLabels[slug]
	return ok
}

// CategoryLabel returns a category's display name. Unknown categories are
// title-cased like in the SvelteKit app: "foo-bar" becomes "Foo Bar".
func CategoryLabel(slug string) string {
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
