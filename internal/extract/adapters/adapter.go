package adapters

import (
	"strings"

	"github.com/ppiankov/entropia/internal/model"
	"golang.org/x/net/html"
)

// Adapter defines the interface for domain-specific extractors
type Adapter interface {
	// Name returns the adapter name
	Name() string

	// CanHandle checks if this adapter can handle the given URL/content
	CanHandle(url string, contentType string) bool

	// ExtractClaims extracts claims from the HTML document
	ExtractClaims(doc *html.Node, url string) ([]model.Claim, error)

	// ExtractEvidence extracts evidence links from the HTML document
	ExtractEvidence(doc *html.Node, url string) ([]model.Evidence, error)
}

// Registry manages domain adapters
type Registry struct {
	adapters []Adapter
	generic  Adapter
}

// NewRegistry creates a new adapter registry
func NewRegistry() *Registry {
	registry := &Registry{
		adapters: make([]Adapter, 0),
	}

	// Register built-in adapters
	registry.Register(NewWikipediaAdapter())
	registry.Register(NewLegalAdapter())

	// Set generic adapter as fallback
	registry.generic = NewGenericAdapter()

	return registry
}

// Register registers a new adapter
func (r *Registry) Register(adapter Adapter) {
	r.adapters = append(r.adapters, adapter)
}

// FindAdapter finds the best adapter for the given URL and content type
func (r *Registry) FindAdapter(url string, contentType string) Adapter {
	// Try specific adapters first
	for _, adapter := range r.adapters {
		if adapter.CanHandle(url, contentType) {
			return adapter
		}
	}

	// Fall back to generic adapter
	return r.generic
}

// BaseAdapter provides common functionality for adapters
type BaseAdapter struct{}

// ParseHTML parses HTML string into a node tree
func (b *BaseAdapter) ParseHTML(htmlContent string) (*html.Node, error) {
	return html.Parse(strings.NewReader(htmlContent))
}

// ExtractText extracts text content from a node
func (b *BaseAdapter) ExtractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return strings.TrimSpace(n.Data)
	}

	var buf strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		buf.WriteString(b.ExtractText(c))
		buf.WriteString(" ")
	}
	return strings.TrimSpace(buf.String())
}

// HasClass checks if a node has a specific CSS class
func (b *BaseAdapter) HasClass(n *html.Node, className string) bool {
	if n.Type != html.ElementNode {
		return false
	}

	for _, attr := range n.Attr {
		if attr.Key == "class" {
			classes := strings.Fields(attr.Val)
			for _, class := range classes {
				if class == className {
					return true
				}
			}
		}
	}
	return false
}

// GetAttribute gets an attribute value from a node
func (b *BaseAdapter) GetAttribute(n *html.Node, attrKey string) string {
	for _, attr := range n.Attr {
		if attr.Key == attrKey {
			return attr.Val
		}
	}
	return ""
}

// FindAll finds all nodes matching a predicate
func (b *BaseAdapter) FindAll(n *html.Node, predicate func(*html.Node) bool) []*html.Node {
	var results []*html.Node

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if predicate(node) {
			results = append(results, node)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(n)
	return results
}

// FindFirst finds the first node matching a predicate
func (b *BaseAdapter) FindFirst(n *html.Node, predicate func(*html.Node) bool) *html.Node {
	var result *html.Node

	var walk func(*html.Node) bool
	walk = func(node *html.Node) bool {
		if predicate(node) {
			result = node
			return true
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if walk(c) {
				return true
			}
		}
		return false
	}

	walk(n)
	return result
}
