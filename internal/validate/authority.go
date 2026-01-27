package validate

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/ppiankov/entropia/internal/model"
)

// AuthorityClassifier classifies sources into authority tiers
type AuthorityClassifier struct {
	config          *model.AuthorityConfig
	primaryMap      map[string]bool
	secondaryMap    map[string]bool
	pathPatterns    []*compiledPattern
}

type compiledPattern struct {
	pattern *regexp.Regexp
	tier    model.AuthorityTier
}

// NewAuthorityClassifier creates a new authority classifier
func NewAuthorityClassifier(config *model.AuthorityConfig) *AuthorityClassifier {
	if config == nil {
		config = &model.DefaultConfig().Authority
	}

	classifier := &AuthorityClassifier{
		config:       config,
		primaryMap:   make(map[string]bool),
		secondaryMap: make(map[string]bool),
		pathPatterns: make([]*compiledPattern, 0),
	}

	// Build primary domain map
	for _, domain := range config.PrimaryDomains {
		classifier.primaryMap[domain] = true
	}

	// Build secondary domain map
	for _, domain := range config.SecondaryDomains {
		classifier.secondaryMap[domain] = true
	}

	// Compile path patterns
	for _, pathPattern := range config.PathPatterns {
		if re, err := regexp.Compile(pathPattern.Pattern); err == nil {
			tier := model.TierTertiary
			switch strings.ToLower(pathPattern.Tier) {
			case "primary":
				tier = model.TierPrimary
			case "secondary":
				tier = model.TierSecondary
			}
			classifier.pathPatterns = append(classifier.pathPatterns, &compiledPattern{
				pattern: re,
				tier:    tier,
			})
		}
	}

	return classifier
}

// Classify classifies a URL into an authority tier
func (a *AuthorityClassifier) Classify(rawURL string) model.AuthorityTier {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return model.TierTertiary
	}

	host := parsed.Host
	path := parsed.Path

	// Remove port from host
	if idx := strings.Index(host, ":"); idx > 0 {
		host = host[:idx]
	}

	// Check explicit domain mappings from config
	if a.config.DomainMap != nil {
		if tierStr, ok := a.config.DomainMap[host]; ok {
			return parseTierString(tierStr)
		}
	}

	// Check primary domains
	if a.primaryMap[host] {
		return model.TierPrimary
	}

	// Check if host contains primary domain (e.g., foo.gov.uk contains gov.uk)
	for primaryDomain := range a.primaryMap {
		if strings.HasSuffix(host, "."+primaryDomain) || host == primaryDomain {
			return model.TierPrimary
		}
	}

	// Check secondary domains
	if a.secondaryMap[host] {
		return model.TierSecondary
	}

	// Check if host contains secondary domain
	for secondaryDomain := range a.secondaryMap {
		if strings.HasSuffix(host, "."+secondaryDomain) || host == secondaryDomain {
			return model.TierSecondary
		}
	}

	// Check path patterns
	for _, cp := range a.pathPatterns {
		if cp.pattern.MatchString(path) {
			return cp.tier
		}
	}

	// Check for common TLDs that often indicate authority
	if strings.HasSuffix(host, ".gov") || strings.HasSuffix(host, ".edu") {
		return model.TierPrimary
	}

	// Check for .ac.uk (UK academic institutions)
	if strings.HasSuffix(host, ".ac.uk") {
		return model.TierPrimary
	}

	// Default to tertiary
	return model.TierTertiary
}

// parseTierString converts a tier string to AuthorityTier
func parseTierString(tier string) model.AuthorityTier {
	switch strings.ToLower(tier) {
	case "primary", "1":
		return model.TierPrimary
	case "secondary", "2":
		return model.TierSecondary
	case "tertiary", "3":
		return model.TierTertiary
	default:
		return model.TierTertiary
	}
}
