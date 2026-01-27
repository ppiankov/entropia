package extract

import (
	"strings"

	"github.com/ppiankov/entropia/internal/model"
	"golang.org/x/net/html"
)

// ClaimExtractor extracts claims from HTML
type ClaimExtractor struct {
	keywords []string
}

// NewClaimExtractor creates a new claim extractor
func NewClaimExtractor() *ClaimExtractor {
	return &ClaimExtractor{
		keywords: []string{
			"originated", "origin", "first", "introduced", "invented",
			"according to", "is defined as", "is legally", "under the law",
			"under this act", "shall", "must", "is required", "established",
			"founded", "created", "discovered", "developed",
		},
	}
}

// Extract extracts claims from HTML content
func (e *ClaimExtractor) Extract(htmlContent string) ([]model.Claim, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	// Extract visible text
	text := extractVisibleText(doc)

	// Split into sentences
	sentences := splitSentences(text)

	// Extract claims by keyword matching
	var claims []model.Claim
	for i, sentence := range sentences {
		lower := strings.ToLower(sentence)
		for _, keyword := range e.keywords {
			if strings.Contains(lower, keyword) {
				claims = append(claims, model.Claim{
					Text:      strings.TrimSpace(sentence),
					Heuristic: "keyword:" + keyword,
					Sentence:  i,
				})
				break // Only match once per sentence
			}
		}
	}

	return dedupeClaims(claims), nil
}

// extractVisibleText extracts text nodes from HTML, skipping scripts/styles
func extractVisibleText(n *html.Node) string {
	var buf strings.Builder

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Skip script, style, noscript tags
			switch n.Data {
			case "script", "style", "noscript", "iframe":
				return
			}
		}

		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				buf.WriteString(text)
				buf.WriteString(" ")
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(n)
	return buf.String()
}

// splitSentences splits text into sentences (simple heuristic)
func splitSentences(text string) []string {
	// Replace newlines with spaces
	text = strings.ReplaceAll(text, "\n", " ")

	// Split by sentence terminators
	var sentences []string
	var current strings.Builder

	for i, r := range text {
		current.WriteRune(r)

		// Check for sentence terminators
		if r == '.' || r == '!' || r == '?' {
			// Look ahead to avoid splitting on abbreviations
			if i+1 < len(text) && (text[i+1] == ' ' || text[i+1] == '\t') {
				sentence := strings.TrimSpace(current.String())
				if len(sentence) >= 30 && len(sentence) <= 500 {
					sentences = append(sentences, sentence)
				}
				current.Reset()
			}
		}
	}

	// Add remaining text if it looks like a sentence
	if current.Len() > 0 {
		sentence := strings.TrimSpace(current.String())
		if len(sentence) >= 30 && len(sentence) <= 500 {
			sentences = append(sentences, sentence)
		}
	}

	return sentences
}

// dedupeClaims removes duplicate claims
func dedupeClaims(claims []model.Claim) []model.Claim {
	seen := make(map[string]bool)
	var unique []model.Claim

	for _, claim := range claims {
		key := strings.ToLower(strings.TrimSpace(claim.Text))
		if !seen[key] {
			seen[key] = true
			unique = append(unique, claim)
		}
	}

	return unique
}
