package adapters

import (
	"strings"

	"github.com/ppiankov/entropia/internal/model"
	"golang.org/x/net/html"
)

// LegalAdapter extracts content from legal documents
type LegalAdapter struct {
	BaseAdapter
	legalKeywords []string
	legalDomains  map[string]bool
}

// NewLegalAdapter creates a new legal document adapter
func NewLegalAdapter() *LegalAdapter {
	return &LegalAdapter{
		legalKeywords: []string{
			"shall", "must", "is required", "is defined as",
			"under this act", "under the law", "according to",
			"statute", "regulation", "provision",
		},
		legalDomains: map[string]bool{
			"legislation.gov.uk": true,
			"law.cornell.edu":    true,
			"gov.uk":             true,
			"justice.gov":        true,
		},
	}
}

// Name returns the adapter name
func (a *LegalAdapter) Name() string {
	return "legal"
}

// CanHandle checks if this is a legal document URL
func (a *LegalAdapter) CanHandle(rawURL string, contentType string) bool {
	lowerURL := strings.ToLower(rawURL)

	// Check for legal domains
	for domain := range a.legalDomains {
		if strings.Contains(lowerURL, domain) {
			return true
		}
	}

	// Check for legal path patterns
	if strings.Contains(lowerURL, "/statute") ||
		strings.Contains(lowerURL, "/legal") ||
		strings.Contains(lowerURL, "/law") ||
		strings.Contains(lowerURL, "/regulation") {
		return true
	}

	return false
}

// ExtractClaims extracts claims from legal documents
func (a *LegalAdapter) ExtractClaims(doc *html.Node, rawURL string) ([]model.Claim, error) {
	var claims []model.Claim

	// Focus on main content areas
	mainContent := a.FindFirst(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "main"
	})

	if mainContent == nil {
		// Fallback to article or body
		mainContent = a.FindFirst(doc, func(n *html.Node) bool {
			return n.Type == html.ElementNode &&
				(n.Data == "article" || a.GetAttribute(n, "role") == "main")
		})
	}

	if mainContent == nil {
		mainContent = doc
	}

	// Extract text from sections and paragraphs
	textNodes := a.FindAll(mainContent, func(n *html.Node) bool {
		return n.Type == html.ElementNode && (n.Data == "p" || n.Data == "section" || n.Data == "div")
	})

	for i, node := range textNodes {
		text := a.ExtractText(node)
		sentences := splitSentences(text)

		for _, sentence := range sentences {
			lower := strings.ToLower(sentence)
			for _, keyword := range a.legalKeywords {
				if strings.Contains(lower, keyword) {
					claims = append(claims, model.Claim{
						Text:      strings.TrimSpace(sentence),
						Heuristic: "legal:" + keyword,
						Sentence:  i,
					})
					break
				}
			}
		}
	}

	return a.dedupeClaims(claims), nil
}

// ExtractEvidence extracts evidence from legal documents
func (a *LegalAdapter) ExtractEvidence(doc *html.Node, rawURL string) ([]model.Evidence, error) {
	// Legal documents typically reference other laws and statutes
	// For now, use generic evidence extraction
	// This can be enhanced later with specific legal citation parsing

	var evidence []model.Evidence

	// Find all links
	links := a.FindAll(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "a" && a.GetAttribute(n, "href") != ""
	})

	for _, link := range links {
		_ = a.GetAttribute(link, "href")
		// NOTE: Evidence extraction for legal documents is planned for v0.2.0
		// Legal citations require specialized parsing (e.g., Bluebook format, statutory references)
		// For now, this adapter focuses on claim extraction from legal text
	}

	return evidence, nil
}

func (a *LegalAdapter) dedupeClaims(claims []model.Claim) []model.Claim {
	seen := make(map[string]bool)
	var unique []model.Claim

	for _, claim := range claims {
		key := strings.ToLower(strings.TrimSpace(claim.Text))
		if !seen[key] && key != "" {
			seen[key] = true
			unique = append(unique, claim)
		}
	}

	return unique
}
