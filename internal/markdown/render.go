package markdown

import (
	"bytes"
	stdhtml "html"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// nodeRenderer drops raw HTML, like remark-rehype without allowDangerousHtml,
// and renders [img] markup: the nodes added by imageTransformer, and markup
// inside code, which the SvelteKit app also turns into images.
type nodeRenderer struct {
	storageBaseURL string
}

func (r *nodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindHTMLBlock, renderNothing)
	reg.Register(ast.KindRawHTML, renderNothing)
	reg.Register(kindImage, renderImage)
	reg.Register(kindLiteral, renderLiteral)
	reg.Register(ast.KindCodeSpan, r.renderCodeSpan)
	reg.Register(ast.KindCodeBlock, r.renderCodeBlock)
	reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
}

func renderNothing(util.BufWriter, []byte, ast.Node, bool) (ast.WalkStatus, error) {
	return ast.WalkSkipChildren, nil
}

func renderImage(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		writeImage(w, node.(*imageNode))
	}
	return ast.WalkContinue, nil
}

func writeImage(w util.BufWriter, n *imageNode) {
	_, _ = w.WriteString(`<a href="` + stdhtml.EscapeString(n.href) + `">`)
	_, _ = w.WriteString(`<img src="` + stdhtml.EscapeString(n.src) + `" alt="` + stdhtml.EscapeString(n.alt) + `" loading="lazy"`)
	if n.width != "" {
		_, _ = w.WriteString(` width="` + n.width + `"`)
	}
	if n.height != "" {
		_, _ = w.WriteString(` height="` + n.height + `"`)
	}
	_, _ = w.WriteString(`></a>`)
}

func renderLiteral(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*literalNode)

	_, _ = w.WriteString(stdhtml.EscapeString(n.text))
	if n.hardBreak {
		_, _ = w.WriteString("<br>\n")
	}
	return ast.WalkContinue, nil
}

// The code renderers below match goldmark's, except for writeCode.

func (r *nodeRenderer) renderCodeSpan(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		_, _ = w.WriteString("</code>")
		return ast.WalkContinue, nil
	}

	var code strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		value := c.(*ast.Text).Segment.Value(source)
		if trimmed, ok := bytes.CutSuffix(value, []byte("\n")); ok {
			code.Write(trimmed)
			code.WriteByte(' ')
		} else {
			code.Write(value)
		}
	}

	_, _ = w.WriteString("<code>")
	r.writeCode(w, code.String())
	return ast.WalkSkipChildren, nil
}

func (r *nodeRenderer) renderCodeBlock(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("<pre><code>")
		r.writeCode(w, lines(source, n))
	} else {
		_, _ = w.WriteString("</code></pre>\n")
	}
	return ast.WalkContinue, nil
}

func (r *nodeRenderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.FencedCodeBlock)
	if entering {
		_, _ = w.WriteString("<pre><code")
		if language := n.Language(source); language != nil {
			_, _ = w.WriteString(` class="language-`)
			html.DefaultWriter.Write(w, language)
			_, _ = w.WriteString(`"`)
		}
		_ = w.WriteByte('>')
		r.writeCode(w, lines(source, n))
	} else {
		_, _ = w.WriteString("</code></pre>\n")
	}
	return ast.WalkContinue, nil
}

func lines(source []byte, n ast.Node) string {
	var b strings.Builder
	for i := range n.Lines().Len() {
		line := n.Lines().At(i)
		b.Write(line.Value(source))
	}
	return b.String()
}

// writeCode writes code escaped, with any [img] markup as images.
//
// TODO: after the migration, show [img] markup in code as written, and remove
// the code renderers that only exist for this.
func (r *nodeRenderer) writeCode(w util.BufWriter, code string) {
	last := 0
	for _, m := range imgPattern.FindAllStringSubmatchIndex(code, -1) {
		_, _ = w.WriteString(stdhtml.EscapeString(code[last:m[0]]))
		writeImage(w, newImage(r.storageBaseURL, code[m[2]:m[3]], code[m[4]:m[5]]))
		last = m[1]
	}
	_, _ = w.WriteString(stdhtml.EscapeString(code[last:]))
}
