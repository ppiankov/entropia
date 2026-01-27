package adapters

import (
	"net/url"
	"strings"

	"github.com/ppiankov/entropia/internal/model"
	"golang.org/x/net/html"
)

// WikipediaAdapter extracts content specifically from Wikipedia pages
type WikipediaAdapter struct {
	BaseAdapter
	claimKeywords []string
}

// NewWikipediaAdapter creates a new Wikipedia adapter
func NewWikipediaAdapter() *WikipediaAdapter {
	return &WikipediaAdapter{
		claimKeywords: []string{
			"originated", "origin", "first", "introduced", "invented",
			"according to", "is defined as", "established", "founded",
			"created", "discovered", "developed",
		},
	}
}

// Name returns the adapter name
func (a *WikipediaAdapter) Name() string {
	return "wikipedia"
}

// CanHandle checks if this is a Wikipedia URL
func (a *WikipediaAdapter) CanHandle(rawURL string, contentType string) bool {
	return strings.Contains(rawURL, "wikipedia.org")
}

// ExtractClaims extracts claims from Wikipedia with focus on lead section
func (a *WikipediaAdapter) ExtractClaims(doc *html.Node, rawURL string) ([]model.Claim, error) {
	var claims []model.Claim

	// Find the main content area
	content := a.FindFirst(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "div" &&
			(a.HasClass(n, "mw-parser-output") || a.GetAttribute(n, "id") == "mw-content-text")
	})

	if content == nil {
		content = doc
	}

	// Extract from lead section (before first h2)
	leadSection := a.extractLeadSection(content)
	if leadSection != nil {
		claims = append(claims, a.extractClaimsFromSection(leadSection, "lead")...)
	}

	// Extract from specific sections of interest
	sections := a.FindAll(content, func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return false
		}
		// Look for h2/h3 headers with specific keywords
		if n.Data == "h2" || n.Data == "h3" {
			text := strings.ToLower(a.ExtractText(n))
			return strings.Contains(text, "origin") ||
				strings.Contains(text, "history") ||
				strings.Contains(text, "etymology")
		}
		return false
	})

	for _, section := range sections {
		// Get content after this header until next header
		sectionContent := a.getSectionContent(section)
		claims = append(claims, a.extractClaimsFromSection(sectionContent, "section")...)
	}

	return a.dedupeClaims(claims), nil
}

// ExtractEvidence extracts evidence with Wikipedia-specific handling
func (a *WikipediaAdapter) ExtractEvidence(doc *html.Node, rawURL string) ([]model.Evidence, error) {
	baseURL, _ := url.Parse(rawURL)
	var evidence []model.Evidence

	// Extract citation links (class="reference")
	citations := a.FindAll(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "a" && a.HasClass(n, "reference")
	})

	for _, citation := range citations {
		href := a.GetAttribute(citation, "href")
		if href != "" {
			// Find the actual citation target
			if strings.HasPrefix(href, "#") {
				targetID := strings.TrimPrefix(href, "#")
				target := a.FindFirst(doc, func(n *html.Node) bool {
					return a.GetAttribute(n, "id") == targetID
				})

				if target != nil {
					// Extract URLs from the citation
					links := a.FindAll(target, func(n *html.Node) bool {
						return n.Type == html.ElementNode && n.Data == "a" &&
							a.GetAttribute(n, "class") == "external text"
					})

					for _, link := range links {
						linkHref := a.GetAttribute(link, "href")
						if linkHref != "" {
							resolved := resolveURL(baseURL, linkHref)
							if resolved != "" {
								parsed, _ := url.Parse(resolved)
								host := ""
								if parsed != nil {
									host = parsed.Host
								}

								evidence = append(evidence, model.Evidence{
									URL:        resolved,
									Kind:       model.EvidenceKindCitation,
									Host:       host,
									IsSameHost: false,
									Text:       a.ExtractText(link),
								})
							}
						}
					}
				}
			}
		}
	}

	// Extract external links section
	externalLinksSection := a.FindFirst(doc, func(n *html.Node) bool {
		if n.Type == html.ElementNode && (n.Data == "h2" || n.Data == "h3") {
			text := strings.ToLower(a.ExtractText(n))
			return strings.Contains(text, "external link") || strings.Contains(text, "further reading")
		}
		return false
	})

	if externalLinksSection != nil {
		sectionContent := a.getSectionContent(externalLinksSection)
		links := a.FindAll(sectionContent, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "a" && a.GetAttribute(n, "href") != ""
		})

		for _, link := range links {
			href := a.GetAttribute(link, "href")
			resolved := resolveURL(baseURL, href)
			if resolved != "" && !strings.HasPrefix(href, "#") {
				parsed, _ := url.Parse(resolved)
				host := ""
				if parsed != nil {
					host = parsed.Host
				}

				evidence = append(evidence, model.Evidence{
					URL:        resolved,
					Kind:       model.EvidenceKindExternalLink,
					Host:       host,
					IsSameHost: host == baseURL.Host,
					Text:       a.ExtractText(link),
				})
			}
		}
	}

	// Also extract regular external links from body
	bodyLinks := a.FindAll(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "a" &&
			a.HasClass(n, "external")
	})

	for _, link := range bodyLinks {
		href := a.GetAttribute(link, "href")
		if href != "" && !strings.HasPrefix(href, "#") {
			resolved := resolveURL(baseURL, href)
			if resolved != "" {
				parsed, _ := url.Parse(resolved)
				host := ""
				if parsed != nil {
					host = parsed.Host
				}

				evidence = append(evidence, model.Evidence{
					URL:        resolved,
					Kind:       model.EvidenceKindExternalLink,
					Host:       host,
					IsSameHost: false,
					Text:       a.ExtractText(link),
				})
			}
		}
	}

	return a.dedupeEvidence(evidence), nil
}

// extractLeadSection extracts the lead section (before first h2)
func (a *WikipediaAdapter) extractLeadSection(content *html.Node) *html.Node {
	// Create a virtual node to hold lead section content
	lead := &html.Node{Type: html.ElementNode, Data: "div"}

	var inLead = true
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if !inLead {
			return
		}

		// Stop at first h2
		if n.Type == html.ElementNode && n.Data == "h2" {
			inLead = false
			return
		}

		// Skip infoboxes, navigation, and other metadata
		if n.Type == html.ElementNode && n.Data == "table" &&
			(a.HasClass(n, "infobox") || a.HasClass(n, "navbox")) {
			return
		}

		// Add paragraph nodes to lead
		if n.Type == html.ElementNode && n.Data == "p" {
			lead.AppendChild(n)
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(content)
	return lead
}

// getSectionContent gets content after a section header
func (a *WikipediaAdapter) getSectionContent(header *html.Node) *html.Node {
	section := &html.Node{Type: html.ElementNode, Data: "div"}

	// Get siblings until next header of same level
	for sibling := header.NextSibling; sibling != nil; sibling = sibling.NextSibling {
		if sibling.Type == html.ElementNode &&
			(sibling.Data == "h2" || sibling.Data == "h3") {
			break
		}
		section.AppendChild(sibling)
	}

	return section
}

// extractClaimsFromSection extracts claims from a section
func (a *WikipediaAdapter) extractClaimsFromSection(section *html.Node, sectionType string) []model.Claim {
	var claims []model.Claim

	// Extract text from paragraphs
	paragraphs := a.FindAll(section, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "p"
	})

	for i, p := range paragraphs {
		text := a.ExtractText(p)
		sentences := splitSentences(text)

		for _, sentence := range sentences {
			lower := strings.ToLower(sentence)
			for _, keyword := range a.claimKeywords {
				if strings.Contains(lower, keyword) {
					claims = append(claims, model.Claim{
						Text:      strings.TrimSpace(sentence),
						Heuristic: "wikipedia:" + keyword,
						Sentence:  i,
					})
					break
				}
			}
		}
	}

	return claims
}

// Helper functions

func splitSentences(text string) []string {
	text = strings.ReplaceAll(text, "\n", " ")
	var sentences []string
	var current strings.Builder

	for i, r := range text {
		current.WriteRune(r)

		if r == '.' || r == '!' || r == '?' {
			if i+1 < len(text) && (text[i+1] == ' ' || text[i+1] == '\t') {
				sentence := strings.TrimSpace(current.String())
				if len(sentence) >= 30 && len(sentence) <= 500 {
					sentences = append(sentences, sentence)
				}
				current.Reset()
			}
		}
	}

	return sentences
}

func resolveURL(base *url.URL, href string) string {
	if strings.HasPrefix(href, "#") || strings.HasPrefix(href, "javascript:") || strings.HasPrefix(href, "mailto:") {
		return ""
	}

	parsed, err := url.Parse(href)
	if err != nil {
		return ""
	}

	resolved := base.ResolveReference(parsed)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return ""
	}

	return resolved.String()
}

func (a *WikipediaAdapter) dedupeClaims(claims []model.Claim) []model.Claim {
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

func (a *WikipediaAdapter) dedupeEvidence(evidence []model.Evidence) []model.Evidence {
	seen := make(map[string]bool)
	var unique []model.Evidence

	for _, ev := range evidence {
		if !seen[ev.URL] && ev.URL != "" {
			seen[ev.URL] = true
			unique = append(unique, ev)
		}
	}

	return unique
}
