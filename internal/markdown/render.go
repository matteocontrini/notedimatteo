package markdown

import (
	stdhtml "html"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// nodeRenderer drops raw HTML tags, keeping the text between them, and renders
// the nodes added by imageTransformer.
type nodeRenderer struct{}

func (r *nodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindHTMLBlock, renderNothing)
	reg.Register(ast.KindRawHTML, renderNothing)
	reg.Register(kindImage, renderImage)
	reg.Register(kindLiteral, renderLiteral)
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
