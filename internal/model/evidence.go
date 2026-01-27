package model

import "time"

// Evidence represents a cited source or outbound reference
type Evidence struct {
	URL        string        `json:"url"`                  // Full URL
	Kind       EvidenceKind  `json:"kind"`                 // citation, external_link, reference
	Host       string        `json:"host,omitempty"`       // Domain name
	IsSameHost bool          `json:"is_same_host"`         // Whether it's same domain as source
	Authority  AuthorityTier `json:"authority,omitempty"`  // Source authority classification
	Text       string        `json:"text,omitempty"`       // Link anchor text
}

// EvidenceKind classifies the type of evidence
type EvidenceKind string

const (
	EvidenceKindCitation     EvidenceKind = "citation"      // Formal citation (e.g., Wikipedia references)
	EvidenceKindExternalLink EvidenceKind = "external_link" // Outbound link
	EvidenceKindReference    EvidenceKind = "reference"     // Named reference
)

// AuthorityTier represents the classification of source authority
type AuthorityTier int

const (
	TierUnknown   AuthorityTier = 0 // Not yet classified
	TierPrimary   AuthorityTier = 1 // Laws, statutes, academic papers, official documents
	TierSecondary AuthorityTier = 2 // Encyclopedias, major publishers, reputable media
	TierTertiary  AuthorityTier = 3 // Blogs, personal websites, tourism sites
)

func (t AuthorityTier) String() string {
	switch t {
	case TierPrimary:
		return "primary"
	case TierSecondary:
		return "secondary"
	case TierTertiary:
		return "tertiary"
	default:
		return "unknown"
	}
}

// ValidationResult contains the result of evidence validation
type ValidationResult struct {
	URL          string        `json:"url"`
	IsAccessible bool          `json:"is_accessible"`
	StatusCode   int           `json:"status_code,omitempty"`
	LastModified *time.Time    `json:"last_modified,omitempty"`
	Age          *int          `json:"age_days,omitempty"`    // Days since last modified
	IsStale      bool          `json:"is_stale"`              // > 1 year old
	IsVeryStale  bool          `json:"is_very_stale"`         // > 3 years old
	IsDead       bool          `json:"is_dead"`               // 404, 410, or timeout
	RedirectURL  string        `json:"redirect_url,omitempty"` // If redirected
	Authority    AuthorityTier `json:"authority"`
	Error        string        `json:"error,omitempty"`
}
