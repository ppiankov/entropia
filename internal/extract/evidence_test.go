package extract

import (
	"net/url"
	"strings"
	"testing"

	"github.com/ppiankov/entropia/internal/model"
	"golang.org/x/net/html"
)

func TestEvidenceExtractor_BasicExtraction(t *testing.T) {
	extractor := NewEvidenceExtractor()

	html := `
	<html>
	<body>
		<p>Some text with a <a href="https://example.com/page1">link</a>.</p>
		<p>Another <a href="https://example.org/page2">external link</a>.</p>
	</body>
	</html>
	`

	sourceURL := "https://mysite.com/article"
	evidence, err := extractor.Extract(html, sourceURL)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(evidence) != 2 {
		t.Errorf("Expected 2 evidence links, got %d", len(evidence))
	}

	// Check URLs are extracted
	foundExample := false
	foundExampleOrg := false

	for _, ev := range evidence {
		if strings.Contains(ev.URL, "example.com") {
			foundExample = true
		}
		if strings.Contains(ev.URL, "example.org") {
			foundExampleOrg = true
		}
	}

	if !foundExample {
		t.Error("Expected to find example.com link")
	}
	if !foundExampleOrg {
		t.Error("Expected to find example.org link")
	}
}

func TestEvidenceExtractor_RelativeURLs(t *testing.T) {
	extractor := NewEvidenceExtractor()

	html := `
	<html>
	<body>
		<a href="/relative/path">Relative link</a>
		<a href="../parent/path">Parent relative link</a>
		<a href="same-level.html">Same level link</a>
	</body>
	</html>
	`

	sourceURL := "https://example.com/articles/page1"
	evidence, err := extractor.Extract(html, sourceURL)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// All relative URLs should be resolved to absolute
	for _, ev := range evidence {
		if !strings.HasPrefix(ev.URL, "https://") && !strings.HasPrefix(ev.URL, "http://") {
			t.Errorf("Expected absolute URL, got %s", ev.URL)
		}
	}
}

func TestEvidenceExtractor_SkipAnchors(t *testing.T) {
	extractor := NewEvidenceExtractor()

	html := `
	<html>
	<body>
		<a href="#section1">Section 1</a>
		<a href="#section2">Section 2</a>
		<a href="https://example.com/page">Real link</a>
	</body>
	</html>
	`

	sourceURL := "https://mysite.com/article"
	evidence, err := extractor.Extract(html, sourceURL)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should only extract the real link, not anchors
	if len(evidence) != 1 {
		t.Errorf("Expected 1 evidence link (anchors skipped), got %d", len(evidence))
	}

	if evidence[0].URL != "https://example.com/page" {
		t.Errorf("Expected real link, got %s", evidence[0].URL)
	}
}

func TestEvidenceExtractor_SkipJavaScriptAndMailto(t *testing.T) {
	extractor := NewEvidenceExtractor()

	html := `
	<html>
	<body>
		<a href="javascript:void(0)">JavaScript link</a>
		<a href="mailto:user@example.com">Email link</a>
		<a href="https://example.com/valid">Valid link</a>
	</body>
	</html>
	`

	sourceURL := "https://mysite.com/article"
	evidence, err := extractor.Extract(html, sourceURL)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should only extract valid http/https links
	if len(evidence) != 1 {
		t.Errorf("Expected 1 evidence link, got %d", len(evidence))
	}

	if evidence[0].URL != "https://example.com/valid" {
		t.Errorf("Expected valid link, got %s", evidence[0].URL)
	}
}

func TestEvidenceExtractor_CitationClassification(t *testing.T) {
	extractor := NewEvidenceExtractor()

	html := `
	<html>
	<body>
		<a href="https://example.com/cite/123" class="reference">Citation</a>
		<a href="https://example.com/#ref1">Reference anchor</a>
		<a href="https://example.com/footnote/5">Footnote</a>
		<a href="https://example.com/article">Regular link</a>
	</body>
	</html>
	`

	sourceURL := "https://mysite.com/article"
	evidence, err := extractor.Extract(html, sourceURL)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check classification
	kindCounts := make(map[model.EvidenceKind]int)
	for _, ev := range evidence {
		kindCounts[ev.Kind]++
	}

	if kindCounts[model.EvidenceKindCitation] == 0 {
		t.Error("Expected at least one citation")
	}

	if kindCounts[model.EvidenceKindReference] == 0 {
		t.Error("Expected at least one reference")
	}

	if kindCounts[model.EvidenceKindExternalLink] == 0 {
		t.Error("Expected at least one external link")
	}
}

func TestEvidenceExtractor_SameHostDetection(t *testing.T) {
	extractor := NewEvidenceExtractor()

	html := `
	<html>
	<body>
		<a href="https://mysite.com/page1">Same host</a>
		<a href="https://example.com/page2">Different host</a>
	</body>
	</html>
	`

	sourceURL := "https://mysite.com/article"
	evidence, err := extractor.Extract(html, sourceURL)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	sameHostCount := 0
	diffHostCount := 0

	for _, ev := range evidence {
		if ev.IsSameHost {
			sameHostCount++
		} else {
			diffHostCount++
		}
	}

	if sameHostCount != 1 {
		t.Errorf("Expected 1 same-host link, got %d", sameHostCount)
	}

	if diffHostCount != 1 {
		t.Errorf("Expected 1 different-host link, got %d", diffHostCount)
	}
}

func TestEvidenceExtractor_LinkText(t *testing.T) {
	extractor := NewEvidenceExtractor()

	html := `
	<html>
	<body>
		<a href="https://example.com/page1">Link Text</a>
		<a href="https://example.com/page2"><span>Nested span text</span></a>
	</body>
	</html>
	`

	sourceURL := "https://mysite.com/article"
	evidence, err := extractor.Extract(html, sourceURL)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that link text is extracted
	foundLinkText := false
	for _, ev := range evidence {
		if ev.Text == "Link Text" {
			foundLinkText = true
		}
	}

	if !foundLinkText {
		t.Error("Expected to extract link text 'Link Text'")
	}
}

func TestEvidenceExtractor_Deduplication(t *testing.T) {
	extractor := NewEvidenceExtractor()

	html := `
	<html>
	<body>
		<a href="https://example.com/page">First mention</a>
		<a href="https://example.com/page">Second mention</a>
		<a href="https://example.com/page">Third mention</a>
	</body>
	</html>
	`

	sourceURL := "https://mysite.com/article"
	evidence, err := extractor.Extract(html, sourceURL)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should deduplicate by URL
	if len(evidence) != 1 {
		t.Errorf("Expected 1 unique evidence link after deduplication, got %d", len(evidence))
	}
}

func TestEvidenceExtractor_EmptyHTML(t *testing.T) {
	extractor := NewEvidenceExtractor()

	html := `<html><body></body></html>`

	sourceURL := "https://mysite.com/article"
	evidence, err := extractor.Extract(html, sourceURL)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(evidence) != 0 {
		t.Errorf("Expected 0 evidence from empty HTML, got %d", len(evidence))
	}
}

func TestEvidenceExtractor_NoLinks(t *testing.T) {
	extractor := NewEvidenceExtractor()

	html := `
	<html>
	<body>
		<p>This is just text without any links.</p>
		<div>More text content.</div>
	</body>
	</html>
	`

	sourceURL := "https://mysite.com/article"
	evidence, err := extractor.Extract(html, sourceURL)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(evidence) != 0 {
		t.Errorf("Expected 0 evidence when no links present, got %d", len(evidence))
	}
}

func TestEvidenceExtractor_HostExtraction(t *testing.T) {
	extractor := NewEvidenceExtractor()

	html := `
	<html>
	<body>
		<a href="https://example.com:8080/page">Link with port</a>
		<a href="https://subdomain.example.org/page">Subdomain link</a>
	</body>
	</html>
	`

	sourceURL := "https://mysite.com/article"
	evidence, err := extractor.Extract(html, sourceURL)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that hosts are extracted correctly
	for _, ev := range evidence {
		if ev.Host == "" {
			t.Errorf("Expected host to be extracted for URL %s", ev.URL)
		}

		if strings.Contains(ev.URL, "example.com") && !strings.Contains(ev.Host, "example.com") {
			t.Errorf("Expected host to contain example.com, got %s", ev.Host)
		}

		if strings.Contains(ev.URL, "example.org") && !strings.Contains(ev.Host, "example.org") {
			t.Errorf("Expected host to contain example.org, got %s", ev.Host)
		}
	}
}

func TestResolveURL_AbsoluteURLs(t *testing.T) {
	tests := []struct {
		base     string
		href     string
		expected string
		desc     string
	}{
		{
			base:     "https://example.com/page",
			href:     "https://external.com/link",
			expected: "https://external.com/link",
			desc:     "Absolute URL unchanged",
		},
		{
			base:     "https://example.com/page",
			href:     "http://external.com/link",
			expected: "http://external.com/link",
			desc:     "HTTP absolute URL unchanged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			base, _ := parseURL(tt.base)
			result := resolveURL(base, tt.href)

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestResolveURL_RelativeURLs(t *testing.T) {
	tests := []struct {
		base     string
		href     string
		expected string
		desc     string
	}{
		{
			base:     "https://example.com/path/page.html",
			href:     "/absolute/path",
			expected: "https://example.com/absolute/path",
			desc:     "Absolute path",
		},
		{
			base:     "https://example.com/path/page.html",
			href:     "relative.html",
			expected: "https://example.com/path/relative.html",
			desc:     "Relative path",
		},
		{
			base:     "https://example.com/path/page.html",
			href:     "../parent.html",
			expected: "https://example.com/parent.html",
			desc:     "Parent directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			base, _ := parseURL(tt.base)
			result := resolveURL(base, tt.href)

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestResolveURL_SkipInvalid(t *testing.T) {
	tests := []struct {
		base string
		href string
		desc string
	}{
		{
			base: "https://example.com/page",
			href: "#anchor",
			desc: "Skip anchor",
		},
		{
			base: "https://example.com/page",
			href: "javascript:void(0)",
			desc: "Skip javascript:",
		},
		{
			base: "https://example.com/page",
			href: "mailto:user@example.com",
			desc: "Skip mailto:",
		},
		{
			base: "https://example.com/page",
			href: "ftp://example.com/file",
			desc: "Skip non-http/https schemes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			base, _ := parseURL(tt.base)
			result := resolveURL(base, tt.href)

			if result != "" {
				t.Errorf("Expected empty result for %s, got %s", tt.href, result)
			}
		})
	}
}

func TestClassifyEvidenceKind_Citation(t *testing.T) {
	tests := []struct {
		href     string
		class    string
		expected model.EvidenceKind
		desc     string
	}{
		{
			href:     "https://example.com/cite/123",
			class:    "",
			expected: model.EvidenceKindCitation,
			desc:     "URL with 'cite'",
		},
		{
			href:     "https://example.com/article#ref1",
			class:    "",
			expected: model.EvidenceKindCitation,
			desc:     "URL with '#ref'",
		},
		{
			href:     "https://example.com/page",
			class:    "reference",
			expected: model.EvidenceKindCitation,
			desc:     "Class 'reference'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// Create mock node with class attribute
			node := createMockNode(tt.href, tt.class)
			result := classifyEvidenceKind(tt.href, node)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestClassifyEvidenceKind_Reference(t *testing.T) {
	tests := []struct {
		href     string
		expected model.EvidenceKind
		desc     string
	}{
		{
			href:     "https://example.com/reference/123",
			expected: model.EvidenceKindReference,
			desc:     "URL with 'reference'",
		},
		{
			href:     "https://example.com/footnote/5",
			expected: model.EvidenceKindReference,
			desc:     "URL with 'footnote'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			node := createMockNode(tt.href, "")
			result := classifyEvidenceKind(tt.href, node)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestClassifyEvidenceKind_ExternalLink(t *testing.T) {
	href := "https://example.com/article"
	node := createMockNode(href, "")

	result := classifyEvidenceKind(href, node)

	if result != model.EvidenceKindExternalLink {
		t.Errorf("Expected EvidenceKindExternalLink, got %v", result)
	}
}

// Helper functions for testing
func parseURL(rawURL string) (*url.URL, error) {
	return url.Parse(rawURL)
}

func createMockNode(href, class string) *html.Node {
	node := &html.Node{
		Type: html.ElementNode,
		Data: "a",
		Attr: []html.Attribute{},
	}

	if class != "" {
		node.Attr = append(node.Attr, html.Attribute{
			Key: "class",
			Val: class,
		})
	}

	return node
}
