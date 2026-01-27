package adapters

import (
	"github.com/ppiankov/entropia/internal/extract"
	"github.com/ppiankov/entropia/internal/model"
	"golang.org/x/net/html"
)

// GenericAdapter is the fallback adapter for unknown domains
type GenericAdapter struct {
	BaseAdapter
	claimExtractor    *extract.ClaimExtractor
	evidenceExtractor *extract.EvidenceExtractor
}

// NewGenericAdapter creates a new generic adapter
func NewGenericAdapter() *GenericAdapter {
	return &GenericAdapter{
		claimExtractor:    extract.NewClaimExtractor(),
		evidenceExtractor: extract.NewEvidenceExtractor(),
	}
}

// Name returns the adapter name
func (a *GenericAdapter) Name() string {
	return "generic"
}

// CanHandle always returns true (fallback adapter)
func (a *GenericAdapter) CanHandle(url string, contentType string) bool {
	return true
}

// ExtractClaims delegates to the generic claim extractor
func (a *GenericAdapter) ExtractClaims(doc *html.Node, url string) ([]model.Claim, error) {
	// Convert HTML node back to string for the generic extractor
	// This is not efficient but maintains compatibility
	htmlContent := renderHTML(doc)
	return a.claimExtractor.Extract(htmlContent)
}

// ExtractEvidence delegates to the generic evidence extractor
func (a *GenericAdapter) ExtractEvidence(doc *html.Node, url string) ([]model.Evidence, error) {
	htmlContent := renderHTML(doc)
	return a.evidenceExtractor.Extract(htmlContent, url)
}

// renderHTML renders an HTML node back to string
func renderHTML(n *html.Node) string {
	// Simple serialization - just get the text content
	return extractAllText(n)
}

func extractAllText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	var result string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result += extractAllText(c)
	}
	return result
}
