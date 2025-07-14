package classifier

import (
	"strings"

	"golang.org/x/net/html"
)

// --- Config & Constants ---

// Block-level HTML tags considered for text extraction
var blockTags = map[string]bool{
	"div": true, "p": true, "blockquote": true, "section": true, "article": true, "main": true, "li": true, "ul": true, "ol": true,
}

// Main content selectors for root node detection
var mainContentSelectors = []string{"article", "main", "#content", ".post-content", "#main", ".entry-content"}

// Define textBlock struct used for block extraction
type textBlock struct {
	Text      string
	Selector  string
	Path      string
	Depth     int
	Colorized bool
}

// PageFeatureExtractor is responsible for HTML traversal and feature extraction
// It is decoupled from classification logic for SOLID separation

type PageFeatureExtractor struct{}

func NewPageFeatureExtractor() *PageFeatureExtractor {
	return &PageFeatureExtractor{}
}

// ExtractFeatures parses HTML and returns text blocks and pattern stats
func (e *PageFeatureExtractor) ExtractFeatures(htmlStr string) ([]textBlock, PatternStats, error) {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return nil, PatternStats{}, err
	}
	root := findMainContentNode(doc)
	if root == nil {
		root = doc // fallback
	}
	blocks := extractTextBlocks(root)
	stats := buildPatternStats(blocks)
	return blocks, stats, nil
}

// findMainContentNode tries common selectors, else picks node with most text
func findMainContentNode(n *html.Node) *html.Node {
	candidates := findNodesBySelectors(n, mainContentSelectors)
	if len(candidates) > 0 {
		return candidates[0]
	}
	return findNodeWithMostText(n)
}

func findNodesBySelectors(n *html.Node, selectors []string) []*html.Node {
	var nodes []*html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, sel := range selectors {
				if nodeMatchesSelector(n, sel) {
					nodes = append(nodes, n)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return nodes
}

// extractTextBlocks traverses the DOM and extracts candidate text blocks for quote mining
func extractTextBlocks(root *html.Node) []textBlock {
	var blocks []textBlock
	var visit func(n *html.Node, depth int)
	visit = func(n *html.Node, depth int) {
		if n.Type == html.ElementNode {
			selector := buildSelector(n)
			path := buildPath(n)
			colorized := hasColorStyle(n)
			if isBlockElement(n.Data) {
				text := extractNodeText(n)
				if len(strings.TrimSpace(text)) > 0 {
					blocks = append(blocks, textBlock{
						Text:      strings.TrimSpace(text),
						Selector:  selector,
						Path:      path,
						Depth:     depth,
						Colorized: colorized,
					})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c, depth+1)
		}
	}
	visit(root, 0)
	return blocks
}

func buildSelector(n *html.Node) string {
	if n.Type != html.ElementNode {
		return ""
	}
	id := ""
	class := ""
	for _, attr := range n.Attr {
		if attr.Key == "id" && attr.Val != "" {
			id = "#" + attr.Val
		}
		if attr.Key == "class" && attr.Val != "" {
			class = "." + strings.ReplaceAll(attr.Val, " ", ".")
		}
	}
	if id != "" {
		return n.Data + id
	}
	if class != "" {
		return n.Data + class
	}
	return n.Data
}

func buildPath(n *html.Node) string {
	var parts []string
	for cur := n; cur != nil && cur.Type == html.ElementNode; cur = cur.Parent {
		parts = append([]string{cur.Data}, parts...)
	}
	return strings.Join(parts, "/")
}

func hasColorStyle(n *html.Node) bool {
	for _, attr := range n.Attr {
		if attr.Key == "style" && strings.Contains(attr.Val, "color") {
			return true
		}
	}
	return false
}

func isBlockElement(tag string) bool {
	return blockTags[strings.ToLower(tag)]
}

func extractNodeText(n *html.Node) string {
	var sb strings.Builder
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return sb.String()
}

func findNodeWithMostText(n *html.Node) *html.Node {
	var maxNode *html.Node
	maxLen := 0
	var f func(*html.Node)
	f = func(node *html.Node) {
		if node.Type == html.ElementNode {
			text := extractNodeText(node)
			l := len(strings.TrimSpace(text))
			if l > maxLen {
				maxLen = l
				maxNode = node
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	if maxNode != nil {
		return maxNode
	}
	return n
}

func nodeMatchesSelector(n *html.Node, sel string) bool {
	sel = strings.TrimSpace(sel)
	if sel == "" || n.Type != html.ElementNode {
		return false
	}
	if strings.HasPrefix(sel, "#") {
		id := ""
		for _, attr := range n.Attr {
			if attr.Key == "id" {
				id = attr.Val
				break
			}
		}
		return id == sel[1:]
	}
	if strings.HasPrefix(sel, ".") {
		class := ""
		for _, attr := range n.Attr {
			if attr.Key == "class" {
				class = attr.Val
				break
			}
		}
		for _, c := range strings.Fields(class) {
			if c == sel[1:] {
				return true
			}
		}
		return false
	}
	return strings.EqualFold(n.Data, sel)
}
