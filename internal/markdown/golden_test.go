package markdown

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// update rewrites the expected output after an intended change:
// go test ./internal/markdown -update
var update = flag.Bool("update", false, "rewrite testdata/golden/*.html with the current output")

// TestGolden renders each .md file in testdata/golden and compares the output
// with the .html file next to it, byte for byte.
func TestGolden(t *testing.T) {
	inputs, err := filepath.Glob(filepath.Join("testdata", "golden", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) == 0 {
		t.Fatal("no .md files in testdata/golden")
	}

	r, err := New(Options{StorageBaseURL: "https://storage.example.com"})
	if err != nil {
		t.Fatal(err)
	}

	for _, input := range inputs {
		t.Run(strings.TrimSuffix(filepath.Base(input), ".md"), func(t *testing.T) {
			markdown, err := os.ReadFile(input)
			if err != nil {
				t.Fatal(err)
			}
			got, err := r.Render(string(markdown))
			if err != nil {
				t.Fatal(err)
			}

			path := strings.TrimSuffix(input, ".md") + ".html"
			if *update {
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Errorf("output differs from %s (run with -update after an intended change)\n--- want\n%s\n--- got\n%s", path, want, got)
			}
		})
	}
}
