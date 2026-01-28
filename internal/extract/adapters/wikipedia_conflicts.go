package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// HistoricalEntity represents a state/entity that no longer exists
type HistoricalEntity struct {
	Name        string   // Primary name
	Aliases     []string // Alternative names/spellings
	EndYear     int      // Year it ceased to exist
	Description string   // Brief description
}

// Common historical entities that appear in modern origin disputes
var historicalEntities = []HistoricalEntity{
	{
		Name:        "Kyivan Rus",
		Aliases:     []string{"Киевская Русь", "Kievan Rus", "Kievan Rus'", "Kiev Rus"},
		EndYear:     1240,
		Description: "Medieval East Slavic state (9th-13th century)",
	},
	{
		Name:        "USSR",
		Aliases:     []string{"Soviet Union", "СССР", "Советский Союз", "CCCP"},
		EndYear:     1991,
		Description: "Soviet Union (1922-1991)",
	},
	{
		Name:        "Yugoslavia",
		Aliases:     []string{"Jugoslavija", "Југославија", "SFRY", "SFR Yugoslavia"},
		EndYear:     1992,
		Description: "Socialist Federal Republic of Yugoslavia (1945-1992)",
	},
	{
		Name:        "Czechoslovakia",
		Aliases:     []string{"Československo", "ČSSR"},
		EndYear:     1993,
		Description: "Czechoslovakia (1918-1993)",
	},
	{
		Name:        "Ottoman Empire",
		Aliases:     []string{"Osmanlı", "Османская империя"},
		EndYear:     1922,
		Description: "Ottoman Empire (1299-1922)",
	},
	{
		Name:        "Austria-Hungary",
		Aliases:     []string{"Austro-Hungarian Empire", "Österreich-Ungarn"},
		EndYear:     1918,
		Description: "Austria-Hungary (1867-1918)",
	},
	{
		Name:        "Polish-Lithuanian Commonwealth",
		Aliases:     []string{"Commonwealth", "Rzeczpospolita Obojga Narodów"},
		EndYear:     1795,
		Description: "Polish-Lithuanian Commonwealth (1569-1795)",
	},
	{
		Name:        "Grand Duchy of Lithuania",
		Aliases:     []string{"Lietuvos Didžioji Kunigaikštystė"},
		EndYear:     1795,
		Description: "Grand Duchy of Lithuania (1236-1795)",
	},
}

// WikipediaRevision represents a page revision
type WikipediaRevision struct {
	RevID     int    `json:"revid"`
	Timestamp string `json:"timestamp"`
	User      string `json:"user"`
	Comment   string `json:"comment"`
	Size      int    `json:"size"`
}

// WikipediaRevisionsResponse represents the API response
type WikipediaRevisionsResponse struct {
	Query struct {
		Pages map[string]struct {
			Revisions []WikipediaRevision `json:"revisions"`
		} `json:"pages"`
	} `json:"query"`
}

// EditWarIndicators contains metrics for detecting edit wars
type EditWarIndicators struct {
	RecentEdits      int       // Edits in last 30 days
	RevertCount      int       // Number of reverts detected
	UniqueEditors    int       // Number of different editors
	EditFrequency    float64   // Edits per day
	IsHighConflict   bool      // Overall assessment
	LastEditTime     time.Time // Most recent edit
	ConflictSeverity string    // low, medium, high
}

// HistoricalEntityConflict represents detected historical entity usage
type HistoricalEntityConflict struct {
	Entity      HistoricalEntity
	Occurrences int
	Context     []string // Surrounding text snippets
}

// DetectEditWar checks Wikipedia revision history for edit war patterns
func DetectEditWar(ctx context.Context, pageURL string) (*EditWarIndicators, error) {
	// Extract page title from URL
	title, err := extractWikipediaTitle(pageURL)
	if err != nil {
		return nil, err
	}

	// URL-decode the title if it's already encoded, then re-encode it properly
	decodedTitle, err := url.QueryUnescape(title)
	if err != nil {
		decodedTitle = title // Use as-is if decode fails
	}

	// Construct API URL for revision history (last 30 days, max 100 revisions)
	lang := extractWikipediaLang(pageURL)
	apiURL := fmt.Sprintf("https://%s.wikipedia.org/w/api.php?action=query&titles=%s&prop=revisions&rvlimit=100&rvprop=timestamp|user|comment|size&format=json",
		lang, url.QueryEscape(decodedTitle))

	// Make HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	// Set User-Agent (Wikipedia API requires it)
	req.Header.Set("User-Agent", "Entropia/0.1 (+https://github.com/ppiankov/entropia)")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check if response is actually JSON
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Wikipedia API returned status %d", resp.StatusCode)
	}

	var apiResp WikipediaRevisionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode Wikipedia API response: %w", err)
	}

	// Extract revisions from response
	var revisions []WikipediaRevision
	for _, page := range apiResp.Query.Pages {
		revisions = page.Revisions
		break // Only one page expected
	}

	if len(revisions) == 0 {
		return &EditWarIndicators{}, nil
	}

	// Analyze revisions
	indicators := analyzeRevisions(revisions)
	return indicators, nil
}

// analyzeRevisions processes revision data to detect edit war patterns
func analyzeRevisions(revisions []WikipediaRevision) *EditWarIndicators {
	now := time.Now()
	thirtyDaysAgo := now.Add(-30 * 24 * time.Hour)

	indicators := &EditWarIndicators{}
	editorsMap := make(map[string]bool)
	var recentEdits []WikipediaRevision

	// Count recent edits and unique editors
	for _, rev := range revisions {
		t, err := time.Parse(time.RFC3339, rev.Timestamp)
		if err != nil {
			continue
		}

		if t.After(thirtyDaysAgo) {
			recentEdits = append(recentEdits, rev)
			editorsMap[rev.User] = true

			if indicators.LastEditTime.IsZero() || t.After(indicators.LastEditTime) {
				indicators.LastEditTime = t
			}
		}

		// Detect reverts in comment
		comment := strings.ToLower(rev.Comment)
		if strings.Contains(comment, "revert") || strings.Contains(comment, "rv ") ||
			strings.Contains(comment, "undo") || strings.Contains(comment, "undid") {
			indicators.RevertCount++
		}
	}

	indicators.RecentEdits = len(recentEdits)
	indicators.UniqueEditors = len(editorsMap)

	// Calculate edit frequency (edits per day)
	if len(recentEdits) > 0 {
		oldestEdit, _ := time.Parse(time.RFC3339, recentEdits[len(recentEdits)-1].Timestamp)
		daysSinceOldest := now.Sub(oldestEdit).Hours() / 24
		if daysSinceOldest > 0 {
			indicators.EditFrequency = float64(indicators.RecentEdits) / daysSinceOldest
		}
	}

	// Determine conflict severity
	// High conflict: >10 edits/month AND >3 reverts, OR >5 edits/day
	// Medium conflict: >5 edits/month AND >1 revert, OR >2 edits/day
	if (indicators.RecentEdits > 10 && indicators.RevertCount > 3) || indicators.EditFrequency > 5 {
		indicators.IsHighConflict = true
		indicators.ConflictSeverity = "high"
	} else if (indicators.RecentEdits > 5 && indicators.RevertCount > 1) || indicators.EditFrequency > 2 {
		indicators.IsHighConflict = true
		indicators.ConflictSeverity = "medium"
	} else if indicators.RevertCount > 0 {
		indicators.ConflictSeverity = "low"
	}

	return indicators
}

// DetectHistoricalEntities scans text for references to non-existent historical entities
func DetectHistoricalEntities(text string) []HistoricalEntityConflict {
	var conflicts []HistoricalEntityConflict
	textLower := strings.ToLower(text)

	for _, entity := range historicalEntities {
		occurrences := 0
		var contexts []string

		// Check primary name and all aliases
		names := append([]string{entity.Name}, entity.Aliases...)
		for _, name := range names {
			nameLower := strings.ToLower(name)
			if strings.Contains(textLower, nameLower) {
				occurrences++

				// Extract context (50 chars before and after)
				if idx := strings.Index(textLower, nameLower); idx >= 0 {
					start := idx - 50
					if start < 0 {
						start = 0
					}
					end := idx + len(nameLower) + 50
					if end > len(text) {
						end = len(text)
					}
					context := text[start:end]
					contexts = append(contexts, strings.TrimSpace(context))
				}
			}
		}

		if occurrences > 0 {
			conflicts = append(conflicts, HistoricalEntityConflict{
				Entity:      entity,
				Occurrences: occurrences,
				Context:     contexts,
			})
		}
	}

	return conflicts
}

// extractWikipediaTitle extracts the page title from a Wikipedia URL
func extractWikipediaTitle(pageURL string) (string, error) {
	// Handle both encoded and unencoded URLs
	// Example: https://en.wikipedia.org/wiki/Borscht
	// Example: https://en.wikipedia.org/wiki/%D0%91%D0%BE%D1%80%D1%89
	re := regexp.MustCompile(`/wiki/(.+)$`)
	matches := re.FindStringSubmatch(pageURL)
	if len(matches) < 2 {
		return "", fmt.Errorf("invalid Wikipedia URL: %s", pageURL)
	}
	return matches[1], nil
}

// extractWikipediaLang extracts language code from Wikipedia URL
func extractWikipediaLang(pageURL string) string {
	// Example: https://en.wikipedia.org/... -> "en"
	// Example: https://ru.wikipedia.org/... -> "ru"
	re := regexp.MustCompile(`https://([a-z]{2,3})\.wikipedia\.org`)
	matches := re.FindStringSubmatch(pageURL)
	if len(matches) < 2 {
		return "en" // Default to English
	}
	return matches[1]
}
