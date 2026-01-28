package extract

import (
	"net/url"
	"strings"

	"github.com/ppiankov/entropia/internal/model"
	"golang.org/x/net/html"
)

// EvidenceExtractor extracts evidence links from HTML
type EvidenceExtractor struct{}

// NewEvidenceExtractor creates a new evidence extractor
func NewEvidenceExtractor() *EvidenceExtractor {
	return &EvidenceExtractor{}
}

// Extract extracts evidence links from HTML content
func (e *EvidenceExtractor) Extract(htmlContent string, sourceURL string) ([]model.Evidence, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	baseURL, err := url.Parse(sourceURL)
	if err != nil {
		return nil, err
	}

	var evidence []model.Evidence
	var walk func(*html.Node)

	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			href := ""
			text := ""

			for _, attr := range n.Attr {
				if attr.Key == "href" {
					href = strings.TrimSpace(attr.Val)
				}
			}

			// Extract link text
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				text = strings.TrimSpace(n.FirstChild.Data)
			}

			if href != "" {
				resolvedURL := resolveURL(baseURL, href)
				if resolvedURL != "" && !isWikipediaNavigationLink(resolvedURL, baseURL.String()) {
					parsed, _ := url.Parse(resolvedURL)
					host := ""
					if parsed != nil {
						host = parsed.Host
					}

					evidence = append(evidence, model.Evidence{
						URL:        resolvedURL,
						Kind:       classifyEvidenceKind(href, n),
						Host:       host,
						IsSameHost: host == baseURL.Host,
						Text:       text,
					})
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(doc)

	return dedupeEvidence(evidence), nil
}

// resolveURL resolves a relative URL against a base URL
func resolveURL(base *url.URL, href string) string {
	// Skip anchors
	if strings.HasPrefix(href, "#") {
		return ""
	}

	// Skip javascript: and mailto: links
	if strings.HasPrefix(href, "javascript:") || strings.HasPrefix(href, "mailto:") {
		return ""
	}

	parsed, err := url.Parse(href)
	if err != nil {
		return ""
	}

	resolved := base.ResolveReference(parsed)

	// Only keep http/https URLs
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return ""
	}

	return resolved.String()
}

// classifyEvidenceKind determines the kind of evidence link
func classifyEvidenceKind(href string, n *html.Node) model.EvidenceKind {
	lower := strings.ToLower(href)

	// Check for citation markers
	if strings.Contains(lower, "cite") || strings.Contains(lower, "#ref") {
		return model.EvidenceKindCitation
	}

	// Check for reference class
	for _, attr := range n.Attr {
		if attr.Key == "class" && strings.Contains(attr.Val, "reference") {
			return model.EvidenceKindCitation
		}
	}

	// Check for common reference patterns
	if strings.Contains(lower, "reference") || strings.Contains(lower, "footnote") {
		return model.EvidenceKindReference
	}

	return model.EvidenceKindExternalLink
}

// isWikipediaNavigationLink checks if a URL is a Wikipedia UI/navigation link
func isWikipediaNavigationLink(resolvedURL, sourceURL string) bool {
	// Only apply to Wikipedia pages
	if !strings.Contains(sourceURL, "wikipedia.org") {
		return false
	}

	parsedURL, err := url.Parse(resolvedURL)
	if err != nil {
		return false
	}

	// Skip if not same Wikipedia domain
	parsedSource, err := url.Parse(sourceURL)
	if err != nil {
		return false
	}
	if parsedURL.Host != parsedSource.Host {
		return false // External link, keep it
	}

	path := parsedURL.Path

	// Filter Wikipedia navigation, UI, and meta pages
	navigationPrefixes := []string{
		"/wiki/Main_Page",
		"/wiki/Wikipedia:",
		"/wiki/Portal:",
		"/wiki/Special:",
		"/wiki/Help:",
		"/wiki/Talk:",
		"/wiki/File:",
		"/wiki/Template:",
		"/wiki/Template_talk:",
		"/wiki/Category:",
		"/w/index.php", // Edit/history links
	}

	for _, prefix := range navigationPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	// Also filter if it's the same page as the source (self-reference)
	if parsedURL.Path == parsedSource.Path {
		return true
	}

	return false
}

// dedupeEvidence removes duplicate evidence links
func dedupeEvidence(evidence []model.Evidence) []model.Evidence {
	seen := make(map[string]bool)
	var unique []model.Evidence

	for _, ev := range evidence {
		if !seen[ev.URL] {
			seen[ev.URL] = true
			unique = append(unique, ev)
		}
	}

	return unique
}
