// Package markdown renders post bodies to HTML.
package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// Version identifies the renderer's output, and is stored with cached HTML.
// Bump it whenever the output changes, so cached posts are rendered again.
// Version 1 was the site's previous renderer (SvelteKit).
// Version 2 is the goldmark renderer.
// Version 3 shows HTML tags as text instead of dropping them.
const Version = 3

type Options struct {
	// StorageBaseURL is the public address of uploaded images, used by [img] markup.
	StorageBaseURL string
}

type Renderer struct {
	md goldmark.Markdown
}

// New returns the notes' renderer: CommonMark with strikethrough, HTML shown
// as text, and [img] markup.
func New(opts Options) (*Renderer, error) {
	if opts.StorageBaseURL == "" {
		return nil, errors.New("markdown: StorageBaseURL is required")
	}

	storageBaseURL := strings.TrimSuffix(opts.StorageBaseURL, "/")
	md := goldmark.New(
		goldmark.WithExtensions(extension.Strikethrough),
		// WithParser comes first, because WithParserOptions adds to the current parser
		goldmark.WithParser(textOnlyParser()),
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

// textOnlyParser is goldmark's default parser without the HTML block and
// inline HTML parsers. A tag like <details> then stays text, and the renderer
// escapes it like any other text.
func textOnlyParser() parser.Parser {
	return parser.NewParser(
		parser.WithBlockParsers(without(parser.DefaultBlockParsers(), parser.NewHTMLBlockParser())...),
		parser.WithInlineParsers(without(parser.DefaultInlineParsers(), parser.NewRawHTMLParser())...),
		parser.WithParagraphTransformers(parser.DefaultParagraphTransformers()...),
	)
}

// without returns the parsers whose type differs from the type of drop.
func without(parsers []util.PrioritizedValue, drop any) []util.PrioritizedValue {
	var kept []util.PrioritizedValue
	for _, p := range parsers {
		if reflect.TypeOf(p.Value) != reflect.TypeOf(drop) {
			kept = append(kept, p)
		}
	}
	return kept
}

func (r *Renderer) Render(markdown string) (string, error) {
	var buf bytes.Buffer
	if err := r.md.Convert([]byte(markdown), &buf); err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}
	return buf.String(), nil
}
