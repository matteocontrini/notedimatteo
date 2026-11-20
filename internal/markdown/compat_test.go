package markdown

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Must match storageBaseUrl in testdata/generate.mjs
const storageBaseURL = "https://storage.example.com"

// TestCompatibility compares the output with the SvelteKit app's renderer,
// saved by testdata/generate.mjs. MARKDOWN_COMPAT_DIR can point to another
// directory of .md and .html pairs, for example exported posts with their
// cached HTML, and MARKDOWN_COMPAT_STORAGE_URL sets the storage URL they used.
func TestCompatibility(t *testing.T) {
	dir := os.Getenv("MARKDOWN_COMPAT_DIR")
	if dir == "" {
		dir = filepath.Join("testdata", "compat")
	}
	storageURL := os.Getenv("MARKDOWN_COMPAT_STORAGE_URL")
	if storageURL == "" {
		storageURL = storageBaseURL
	}

	inputs, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) == 0 {
		t.Fatalf("no .md files in %s", dir)
	}

	r, err := New(Options{StorageBaseURL: storageURL})
	if err != nil {
		t.Fatal(err)
	}

	for _, input := range inputs {
		name := strings.TrimSuffix(filepath.Base(input), ".md")
		t.Run(name, func(t *testing.T) {
			markdown, err := os.ReadFile(input)
			if err != nil {
				t.Fatal(err)
			}
			expected, err := os.ReadFile(strings.TrimSuffix(input, ".md") + ".html")
			if err != nil {
				t.Fatal(err)
			}

			got, err := r.Render(string(markdown))
			if err != nil {
				t.Fatal(err)
			}

			want := canonical(t, string(expected))
			if have := canonical(t, got); have != want {
				t.Errorf("output differs from the SvelteKit renderer\n--- want\n%s\n--- got\n%s\n--- raw output\n%s", want, have, got)
			}
		})
	}
}

// canonical parses an HTML fragment and serializes it with decoded text,
// sorted attributes, and CRLF line endings normalized, so that equivalent
// markup compares equal however each renderer escapes it.
func canonical(t *testing.T, fragment string) string {
	t.Helper()

	context := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	nodes, err := html.ParseFragment(strings.NewReader(fragment), context)
	if err != nil {
		t.Fatal(err)
	}

	var b strings.Builder
	var write func(n *html.Node)
	write = func(n *html.Node) {
		switch n.Type {
		case html.TextNode:
			b.WriteString(strings.ReplaceAll(n.Data, "\r\n", "\n"))
		case html.ElementNode:
			attrs := make([]string, 0, len(n.Attr))
			for _, a := range n.Attr {
				attrs = append(attrs, a.Key+"="+a.Val)
			}
			slices.Sort(attrs)
			b.WriteString("<" + n.Data)
			for _, a := range attrs {
				b.WriteString(" " + a)
			}
			b.WriteString(">")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				write(c)
			}
			b.WriteString("</" + n.Data + ">")
		}
	}
	for _, n := range nodes {
		write(n)
	}
	return strings.TrimSpace(b.String())
}
