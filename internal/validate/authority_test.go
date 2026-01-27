package validate

import (
	"testing"

	"github.com/ppiankov/entropia/internal/model"
)

func TestAuthorityClassifier_PrimaryDomains(t *testing.T) {
	config := &model.AuthorityConfig{
		PrimaryDomains: []string{
			"legislation.gov.uk",
			"doi.org",
			"scholar.google.com",
		},
		SecondaryDomains: []string{
			"wikipedia.org",
		},
	}

	classifier := NewAuthorityClassifier(config)

	tests := []struct {
		url      string
		expected model.AuthorityTier
		desc     string
	}{
		{
			url:      "https://legislation.gov.uk/ukpga/1998/42",
			expected: model.TierPrimary,
			desc:     "Primary domain exact match",
		},
		{
			url:      "https://www.legislation.gov.uk/statute",
			expected: model.TierPrimary,
			desc:     "Primary domain with subdomain",
		},
		{
			url:      "https://doi.org/10.1234/example",
			expected: model.TierPrimary,
			desc:     "DOI primary source",
		},
		{
			url:      "https://scholar.google.com/citations",
			expected: model.TierPrimary,
			desc:     "Google Scholar primary source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := classifier.Classify(tt.url)
			if result != tt.expected {
				t.Errorf("Expected %v for %s, got %v", tt.expected, tt.url, result)
			}
		})
	}
}

func TestAuthorityClassifier_SecondaryDomains(t *testing.T) {
	config := &model.AuthorityConfig{
		SecondaryDomains: []string{
			"wikipedia.org",
			"britannica.com",
		},
	}

	classifier := NewAuthorityClassifier(config)

	tests := []struct {
		url      string
		expected model.AuthorityTier
		desc     string
	}{
		{
			url:      "https://en.wikipedia.org/wiki/Laksa",
			expected: model.TierSecondary,
			desc:     "Wikipedia secondary source",
		},
		{
			url:      "https://www.britannica.com/topic/democracy",
			expected: model.TierSecondary,
			desc:     "Britannica secondary source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := classifier.Classify(tt.url)
			if result != tt.expected {
				t.Errorf("Expected %v for %s, got %v", tt.expected, tt.url, result)
			}
		})
	}
}

func TestAuthorityClassifier_PathPatterns(t *testing.T) {
	config := &model.AuthorityConfig{
		PathPatterns: []model.PathPattern{
			{Pattern: "/statute/", Tier: "primary"},
			{Pattern: "/legal/", Tier: "primary"},
			{Pattern: "/law/", Tier: "primary"},
		},
	}

	classifier := NewAuthorityClassifier(config)

	tests := []struct {
		url      string
		expected model.AuthorityTier
		desc     string
	}{
		{
			url:      "https://example.com/statute/42",
			expected: model.TierPrimary,
			desc:     "Path pattern /statute/",
		},
		{
			url:      "https://example.org/legal/contracts",
			expected: model.TierPrimary,
			desc:     "Path pattern /legal/",
		},
		{
			url:      "https://example.net/law/cases",
			expected: model.TierPrimary,
			desc:     "Path pattern /law/",
		},
		{
			url:      "https://example.com/blog/post",
			expected: model.TierTertiary,
			desc:     "No matching path pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := classifier.Classify(tt.url)
			if result != tt.expected {
				t.Errorf("Expected %v for %s, got %v", tt.expected, tt.url, result)
			}
		})
	}
}

func TestAuthorityClassifier_TLDHeuristics(t *testing.T) {
	classifier := NewAuthorityClassifier(nil) // Use defaults

	tests := []struct {
		url      string
		expected model.AuthorityTier
		desc     string
	}{
		{
			url:      "https://whitehouse.gov/statements",
			expected: model.TierPrimary,
			desc:     ".gov TLD should be primary",
		},
		{
			url:      "https://mit.edu/research",
			expected: model.TierPrimary,
			desc:     ".edu TLD should be primary",
		},
		{
			url:      "https://oxford.ac.uk/research",
			expected: model.TierPrimary,
			desc:     ".ac.uk TLD should be primary (UK academic)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := classifier.Classify(tt.url)
			if result != tt.expected {
				t.Errorf("Expected %v for %s, got %v", tt.expected, tt.url, result)
			}
		})
	}
}

func TestAuthorityClassifier_DomainMap(t *testing.T) {
	config := &model.AuthorityConfig{
		DomainMap: map[string]string{
			"nytimes.com": "secondary",
			"myblog.com":  "tertiary",
		},
	}

	classifier := NewAuthorityClassifier(config)

	tests := []struct {
		url      string
		expected model.AuthorityTier
		desc     string
	}{
		{
			url:      "https://nytimes.com/article",
			expected: model.TierSecondary,
			desc:     "Explicit domain map to secondary",
		},
		{
			url:      "https://myblog.com/post",
			expected: model.TierTertiary,
			desc:     "Explicit domain map to tertiary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := classifier.Classify(tt.url)
			if result != tt.expected {
				t.Errorf("Expected %v for %s, got %v", tt.expected, tt.url, result)
			}
		})
	}
}

func TestAuthorityClassifier_TertiaryDefault(t *testing.T) {
	classifier := NewAuthorityClassifier(nil)

	tests := []struct {
		url      string
		expected model.AuthorityTier
		desc     string
	}{
		{
			url:      "https://randomsite.com/page",
			expected: model.TierTertiary,
			desc:     "Unknown domain defaults to tertiary",
		},
		{
			url:      "https://blog.example.net/article",
			expected: model.TierTertiary,
			desc:     "Blog domain defaults to tertiary",
		},
		{
			url:      "https://tourism-board.org/visit",
			expected: model.TierTertiary,
			desc:     ".org without other signals defaults to tertiary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := classifier.Classify(tt.url)
			if result != tt.expected {
				t.Errorf("Expected %v for %s, got %v", tt.expected, tt.url, result)
			}
		})
	}
}

func TestAuthorityClassifier_InvalidURLs(t *testing.T) {
	classifier := NewAuthorityClassifier(nil)

	tests := []struct {
		url      string
		expected model.AuthorityTier
		desc     string
	}{
		{
			url:      "not-a-url",
			expected: model.TierTertiary,
			desc:     "Invalid URL defaults to tertiary",
		},
		{
			url:      "://missing-scheme",
			expected: model.TierTertiary,
			desc:     "Malformed URL defaults to tertiary",
		},
		{
			url:      "",
			expected: model.TierTertiary,
			desc:     "Empty URL defaults to tertiary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := classifier.Classify(tt.url)
			if result != tt.expected {
				t.Errorf("Expected %v for %s, got %v", tt.expected, tt.url, result)
			}
		})
	}
}

func TestAuthorityClassifier_PortHandling(t *testing.T) {
	config := &model.AuthorityConfig{
		PrimaryDomains: []string{"example.gov"},
	}

	classifier := NewAuthorityClassifier(config)

	tests := []struct {
		url      string
		expected model.AuthorityTier
		desc     string
	}{
		{
			url:      "https://example.gov:443/page",
			expected: model.TierPrimary,
			desc:     "URL with port should match domain",
		},
		{
			url:      "http://example.gov:8080/page",
			expected: model.TierPrimary,
			desc:     "URL with non-standard port should match domain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := classifier.Classify(tt.url)
			if result != tt.expected {
				t.Errorf("Expected %v for %s, got %v", tt.expected, tt.url, result)
			}
		})
	}
}

func TestParseTierString(t *testing.T) {
	tests := []struct {
		input    string
		expected model.AuthorityTier
		desc     string
	}{
		{input: "primary", expected: model.TierPrimary, desc: "lowercase primary"},
		{input: "Primary", expected: model.TierPrimary, desc: "capitalized primary"},
		{input: "PRIMARY", expected: model.TierPrimary, desc: "uppercase primary"},
		{input: "1", expected: model.TierPrimary, desc: "numeric 1 as primary"},
		{input: "secondary", expected: model.TierSecondary, desc: "lowercase secondary"},
		{input: "2", expected: model.TierSecondary, desc: "numeric 2 as secondary"},
		{input: "tertiary", expected: model.TierTertiary, desc: "lowercase tertiary"},
		{input: "3", expected: model.TierTertiary, desc: "numeric 3 as tertiary"},
		{input: "unknown", expected: model.TierTertiary, desc: "unknown defaults to tertiary"},
		{input: "", expected: model.TierTertiary, desc: "empty defaults to tertiary"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := parseTierString(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v for %s, got %v", tt.expected, tt.input, result)
			}
		})
	}
}

func TestNewAuthorityClassifier_NilConfig(t *testing.T) {
	classifier := NewAuthorityClassifier(nil)

	if classifier == nil {
		t.Fatal("Expected classifier to be created with default config")
	}

	if classifier.config == nil {
		t.Error("Expected config to be initialized with defaults")
	}
}
