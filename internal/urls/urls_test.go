package urls

import "testing"

func TestURLs(t *testing.T) {
	tests := []struct {
		got  string
		want string
	}{
		{Post(42, ""), "/42"},
		{Post(42, "a-slug"), "/42/a-slug"},
		{Edit(42), "/42/edit"},
		{Category("segnalibri"), "/segnalibri"},
		{Tag("c#"), "/archivio/c%23"},
		{Tag("a b"), "/archivio/a%20b"},
		{Month(2026, 4), "/archivio/2026/04"},
		{Page("/", 1), "/"},
		{Page("/segnalibri", 2), "/segnalibri?page=2"},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("got %q, want %q", tt.got, tt.want)
		}
	}
}
