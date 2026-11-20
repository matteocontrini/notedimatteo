package markdown

import (
	"strings"
	"testing"
)

// These cases differ from the SvelteKit app's renderer on purpose.
func TestImageMarkup(t *testing.T) {
	r, err := New(Options{StorageBaseURL: "https://storage.example.com/"})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		markdown string
		want     string
	}{
		{
			"preview before the path's extension",
			"[img]2025/09/a.b.jpg[/img]",
			`<p><a href="https://storage.example.com/2025/09/a.b.jpg"><img src="https://storage.example.com/2025/09/a.b_preview.jpg" alt="" loading="lazy"></a></p>`,
		},
		{
			"path without an extension",
			"[img]2025/09/abc[/img]",
			`<p><a href="https://storage.example.com/2025/09/abc"><img src="https://storage.example.com/2025/09/abc_preview" alt="" loading="lazy"></a></p>`,
		},
		{
			"empty path",
			"[img][/img] and [img]x.jpg[/img]",
			`<p>[img][/img] and <a href="https://storage.example.com/x.jpg"><img src="https://storage.example.com/x_preview.jpg" alt="" loading="lazy"></a></p>`,
		},
		{
			"only http(s) links",
			`[img link="javascript:alert(1)"]a.jpg[/img] [img link=https://example.com/page]b.jpg[/img]`,
			`<p><a href="https://storage.example.com/a.jpg"><img src="https://storage.example.com/a_preview.jpg" alt="" loading="lazy"></a> <a href="https://example.com/page"><img src="https://storage.example.com/b_preview.jpg" alt="" loading="lazy"></a></p>`,
		},
		{
			"only numeric sizes",
			"[img width=abc height=12px]a.jpg[/img] [img width=640 height=480]b.jpg[/img]",
			`<p><a href="https://storage.example.com/a.jpg"><img src="https://storage.example.com/a_preview.jpg" alt="" loading="lazy"></a> <a href="https://storage.example.com/b.jpg"><img src="https://storage.example.com/b_preview.jpg" alt="" loading="lazy" width="640" height="480"></a></p>`,
		},
		{
			"markup in code stays as written",
			"`[img]a.jpg[/img]`\n\n```\n[img]b.jpg[/img]\n```",
			"<p><code>[img]a.jpg[/img]</code></p>\n<pre><code>[img]b.jpg[/img]\n</code></pre>",
		},
	}

	for _, tt := range tests {
		got, err := r.Render(tt.markdown)
		if err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(got) != tt.want {
			t.Errorf("%s:\n got %s\nwant %s", tt.name, got, tt.want)
		}
	}
}
