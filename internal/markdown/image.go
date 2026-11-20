package markdown

import (
	"bufio"
	"bytes"
	stdhtml "html"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

// The [img] markup and its output match the SvelteKit app's rehype plugin:
//
//	[img width=1200 height=800 alt="…" link=https://…]2025/09/abc.jpg[/img]
//
// becomes a lazy-loaded image of the _preview variant, linked to the original
// or to the link attribute.
var (
	// The path can't be empty or contain brackets or line breaks.
	imgPattern       = regexp.MustCompile(`\[img([^\]]*)\]([^\[\]\n]+?)\[/img\]`)
	attributePattern = regexp.MustCompile(`(\w+)=(?:"([^"]*)"|(\S+))`)
)

var kindImage = ast.NewNodeKind("StorageImage")

type imageNode struct {
	ast.BaseInline
	href   string
	src    string
	alt    string
	width  string
	height string
}

func (n *imageNode) Kind() ast.NodeKind { return kindImage }

func (n *imageNode) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }

var kindLiteral = ast.NewNodeKind("LiteralText")

// literalNode is text taken out of goldmark's Text nodes, already decoded.
type literalNode struct {
	ast.BaseInline
	text      string
	hardBreak bool
}

func (n *literalNode) Kind() ast.NodeKind { return kindLiteral }

func (n *literalNode) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }

type imageTransformer struct {
	storageBaseURL string
}

func (t *imageTransformer) Transform(doc *ast.Document, reader text.Reader, _ parser.Context) {
	source := reader.Source()

	// Collect parents first, because transforming changes the tree
	var parents []ast.Node
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		// [img] markup in code stays as written; goldmark's code span renderer
		// also only accepts Text children
		if n.Kind() == ast.KindCodeSpan {
			return ast.WalkSkipChildren, nil
		}
		if n.HasChildren() {
			parents = append(parents, n)
		}
		return ast.WalkContinue, nil
	})

	for _, parent := range parents {
		t.transformChildren(parent, source)
	}
}

// transformChildren looks for [img] markup in each run of adjacent text nodes.
// remark merges adjacent text into one node where goldmark splits it, for
// example around a bracket that doesn't start a link, so runs are matched as a
// whole. A hard line break ends a run, like remark's break element.
func (t *imageTransformer) transformChildren(parent ast.Node, source []byte) {
	var run []*ast.Text
	flush := func() {
		if len(run) > 0 {
			t.replaceRun(parent, run, source)
		}
		run = nil
	}

	for child := parent.FirstChild(); child != nil; {
		next := child.NextSibling()
		textNode, ok := child.(*ast.Text)
		if !ok || textNode.IsRaw() {
			flush()
		} else {
			run = append(run, textNode)
			if textNode.HardLineBreak() {
				flush()
			}
		}
		child = next
	}
	flush()
}

func (t *imageTransformer) replaceRun(parent ast.Node, run []*ast.Text, source []byte) {
	var value strings.Builder
	for _, node := range run {
		value.WriteString(decodeText(node.Segment.Value(source)))
		if node.SoftLineBreak() {
			value.WriteByte('\n')
		}
	}

	combined := value.String()
	matches := imgPattern.FindAllStringSubmatchIndex(combined, -1)
	if matches == nil {
		return
	}

	var replacement []ast.Node
	last := 0
	for _, m := range matches {
		if m[0] > last {
			replacement = append(replacement, &literalNode{text: combined[last:m[0]]})
		}
		replacement = append(replacement, newImage(t.storageBaseURL, combined[m[2]:m[3]], combined[m[4]:m[5]]))
		last = m[1]
	}
	if last < len(combined) {
		replacement = append(replacement, &literalNode{text: combined[last:]})
	}
	if run[len(run)-1].HardLineBreak() {
		replacement = append(replacement, &literalNode{hardBreak: true})
	}

	for _, node := range replacement {
		parent.InsertBefore(parent, run[0], node)
	}
	for _, node := range run {
		parent.RemoveChild(parent, node)
	}
}

func newImage(storageBaseURL, attributeString, imagePath string) *imageNode {
	attributes := map[string]string{}
	for _, m := range attributePattern.FindAllStringSubmatch(strings.TrimSpace(attributeString), -1) {
		value := m[2]
		if value == "" {
			value = m[3]
		}
		attributes[m[1]] = value
	}

	ext := path.Ext(imagePath)
	image := &imageNode{
		href: storageBaseURL + "/" + imagePath,
		src:  storageBaseURL + "/" + strings.TrimSuffix(imagePath, ext) + "_preview" + ext,
		alt:  attributes["alt"],
	}
	if link, err := url.Parse(attributes["link"]); err == nil && (link.Scheme == "http" || link.Scheme == "https") {
		image.href = attributes["link"]
	}
	if width, err := strconv.Atoi(attributes["width"]); err == nil {
		image.width = strconv.Itoa(width)
	}
	if height, err := strconv.Atoi(attributes["height"]); err == nil {
		image.height = strconv.Itoa(height)
	}
	return image
}

// decodeText returns a text segment's plain value, with backslash escapes and
// character references resolved, as goldmark's renderer would output it.
func decodeText(segment []byte) string {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	html.DefaultWriter.Write(w, segment)
	_ = w.Flush()
	return stdhtml.UnescapeString(buf.String())
}
