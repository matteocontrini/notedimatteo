// Package markdown renders post bodies to HTML.
package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// Version identifies the renderer's output, and is stored with cached HTML.
// Bump it whenever the output changes, so cached posts are rendered again.
// Version 1 was the site's previous renderer.
const Version = 2

type Options struct {
	// StorageBaseURL is the public address of uploaded images, used by [img] markup.
	StorageBaseURL string
}

type Renderer struct {
	md goldmark.Markdown
}

// New returns the notes' renderer: CommonMark with strikethrough, raw HTML
// dropped, and [img] markup.
func New(opts Options) (*Renderer, error) {
	if opts.StorageBaseURL == "" {
		return nil, errors.New("markdown: StorageBaseURL is required")
	}

	storageBaseURL := strings.TrimSuffix(opts.StorageBaseURL, "/")
	md := goldmark.New(
		goldmark.WithExtensions(extension.Strikethrough),
		goldmark.WithParserOptions(
			parser.WithASTTransformers(util.Prioritized(&imageTransformer{storageBaseURL: storageBaseURL}, 100)),
		),
		goldmark.WithRendererOptions(
			// A lower number takes precedence over goldmark's HTML renderer (1000)
			renderer.WithNodeRenderers(util.Prioritized(&nodeRenderer{}, 100)),
		),
	)

	return &Renderer{md: md}, nil
}

func (r *Renderer) Render(markdown string) (string, error) {
	var buf bytes.Buffer
	if err := r.md.Convert([]byte(markdown), &buf); err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}
	return buf.String(), nil
}
