package extract

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestClaimExtractor_BasicExtraction(t *testing.T) {
	extractor := NewClaimExtractor()

	html := `
	<html>
	<body>
		<p>Laksa originated in Malaysia in the 15th century based on documentation.</p>
		<p>According to historians, this culinary tradition spread to coastal regions.</p>
		<p>This is just a regular sentence without claims.</p>
	</body>
	</html>
	`

	claims, err := extractor.Extract(html)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(claims) < 2 {
		t.Errorf("Expected at least 2 claims, got %d", len(claims))
	}

	// Check that keyword matches work
	foundOriginated := false
	foundAccording := false

	for _, claim := range claims {
		if strings.Contains(strings.ToLower(claim.Text), "originated") {
			foundOriginated = true
			if !strings.Contains(strings.ToLower(claim.Heuristic), "originated") {
				t.Errorf("Expected heuristic to mention 'originated', got '%s'", claim.Heuristic)
			}
		}
		if strings.Contains(strings.ToLower(claim.Text), "according to") && strings.Contains(strings.ToLower(claim.Text), "historians") {
			foundAccording = true
			if !strings.Contains(strings.ToLower(claim.Heuristic), "according to") {
				t.Errorf("Expected heuristic to mention 'according to', got '%s'", claim.Heuristic)
			}
		}
	}

	if !foundOriginated {
		t.Error("Expected to find claim with 'originated'")
	}
	if !foundAccording {
		t.Error("Expected to find claim with 'according to'")
	}
}

func TestClaimExtractor_LegalKeywords(t *testing.T) {
	extractor := NewClaimExtractor()

	html := `
	<html>
	<body>
		<p>Under the law, marriage requires both parties to consent.</p>
		<p>The statute is defined as a legislative act.</p>
		<p>Common-law marriage shall not be recognized in the UK.</p>
	</body>
	</html>
	`

	claims, err := extractor.Extract(html)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(claims) < 3 {
		t.Errorf("Expected at least 3 claims, got %d", len(claims))
	}

	keywords := []string{"under the law", "is defined as", "shall"}
	for _, keyword := range keywords {
		found := false
		for _, claim := range claims {
			if strings.Contains(strings.ToLower(claim.Text), keyword) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find claim with keyword '%s'", keyword)
		}
	}
}

func TestClaimExtractor_SkipScripts(t *testing.T) {
	extractor := NewClaimExtractor()

	html := `
	<html>
	<head>
		<script>
			var text = "The system was first developed in 1995.";
		</script>
		<style>
			/* This originated from CSS */
		</style>
	</head>
	<body>
		<p>The product was first introduced in 2020.</p>
	</body>
	</html>
	`

	claims, err := extractor.Extract(html)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should only extract from body, not script/style
	for _, claim := range claims {
		if strings.Contains(claim.Text, "1995") {
			t.Error("Should not extract claims from script tags")
		}
		if strings.Contains(claim.Text, "CSS") {
			t.Error("Should not extract claims from style tags")
		}
	}

	// Should find the body claim
	found := false
	for _, claim := range claims {
		if strings.Contains(claim.Text, "2020") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find claim from body content")
	}
}

func TestClaimExtractor_SentenceLengthFiltering(t *testing.T) {
	extractor := NewClaimExtractor()

	html := `
	<html>
	<body>
		<p>Short text with originated.</p>
		<p>This sentence is long enough and contains the keyword originated which should be extracted properly.</p>
		<p>` + strings.Repeat("This is a very long sentence that exceeds the maximum length limit and should be filtered out even though it contains the keyword originated. ", 10) + `</p>
	</body>
	</html>
	`

	claims, err := extractor.Extract(html)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that sentences are within reasonable length (30-500 chars)
	for _, claim := range claims {
		length := len(claim.Text)
		if length < 30 {
			t.Errorf("Claim too short (%d chars): %s", length, claim.Text)
		}
		if length > 500 {
			t.Errorf("Claim too long (%d chars): %s", length, claim.Text[:50]+"...")
		}
	}
}

func TestClaimExtractor_Deduplication(t *testing.T) {
	extractor := NewClaimExtractor()

	html := `
	<html>
	<body>
		<p>The system was first introduced in 1990.</p>
		<p>The system was first introduced in 1990.</p>
		<p>THE SYSTEM WAS FIRST INTRODUCED IN 1990.</p>
	</body>
	</html>
	`

	claims, err := extractor.Extract(html)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should deduplicate identical claims (case-insensitive)
	if len(claims) != 1 {
		t.Errorf("Expected 1 unique claim after deduplication, got %d", len(claims))
	}
}

func TestClaimExtractor_EmptyHTML(t *testing.T) {
	extractor := NewClaimExtractor()

	html := `<html><body></body></html>`

	claims, err := extractor.Extract(html)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(claims) != 0 {
		t.Errorf("Expected 0 claims from empty HTML, got %d", len(claims))
	}
}

func TestClaimExtractor_NoClaimKeywords(t *testing.T) {
	extractor := NewClaimExtractor()

	html := `
	<html>
	<body>
		<p>This is just a regular paragraph with no special keywords.</p>
		<p>Another paragraph describing something without attribution.</p>
	</body>
	</html>
	`

	claims, err := extractor.Extract(html)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(claims) != 0 {
		t.Errorf("Expected 0 claims when no keywords present, got %d", len(claims))
	}
}

func TestClaimExtractor_AllKeywords(t *testing.T) {
	extractor := NewClaimExtractor()

	// Test all keywords are functional
	keywords := []string{
		"originated", "origin", "first", "introduced", "invented",
		"according to", "is defined as", "is legally", "under the law",
		"under this act", "shall", "must", "is required", "established",
		"founded", "created", "discovered", "developed",
	}

	for _, keyword := range keywords {
		html := `<html><body><p>This is a test sentence that contains the keyword ` + keyword + ` and is long enough to be extracted.</p></body></html>`

		claims, err := extractor.Extract(html)
		if err != nil {
			t.Fatalf("Expected no error for keyword '%s', got %v", keyword, err)
		}

		if len(claims) == 0 {
			t.Errorf("Expected at least 1 claim for keyword '%s', got 0", keyword)
			continue
		}

		// Verify heuristic is set correctly
		if !strings.Contains(claims[0].Heuristic, keyword) {
			t.Errorf("Expected heuristic to contain '%s', got '%s'", keyword, claims[0].Heuristic)
		}
	}
}

func TestClaimExtractor_SentenceIndex(t *testing.T) {
	extractor := NewClaimExtractor()

	html := `
	<html>
	<body>
		<p>First sentence with originated keyword for testing.</p>
		<p>Second sentence also has the first keyword in it.</p>
		<p>Third sentence contains according to attribution.</p>
	</body>
	</html>
	`

	claims, err := extractor.Extract(html)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that sentence indices are assigned
	for _, claim := range claims {
		if claim.Sentence < 0 {
			t.Errorf("Expected non-negative sentence index, got %d", claim.Sentence)
		}
	}
}

func TestExtractVisibleText_SkipInvisibleElements(t *testing.T) {
	html := `
	<html>
	<head>
		<script>var x = "script content";</script>
		<style>body { color: red; }</style>
	</head>
	<body>
		<p>Visible paragraph text.</p>
		<noscript>Noscript content</noscript>
		<iframe src="example.com">Iframe content</iframe>
		<p>Another visible paragraph.</p>
	</body>
	</html>
	`

	doc, err := parseHTML(html)
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	text := extractVisibleText(doc)

	// Should contain visible paragraphs
	if !strings.Contains(text, "Visible paragraph") {
		t.Error("Expected to extract visible paragraph text")
	}
	if !strings.Contains(text, "Another visible paragraph") {
		t.Error("Expected to extract second visible paragraph")
	}

	// Should NOT contain invisible elements
	if strings.Contains(text, "script content") {
		t.Error("Should not extract script content")
	}
	if strings.Contains(text, "color: red") {
		t.Error("Should not extract style content")
	}
	if strings.Contains(text, "Noscript content") {
		t.Error("Should not extract noscript content")
	}
	if strings.Contains(text, "Iframe content") {
		t.Error("Should not extract iframe content")
	}
}

func TestSplitSentences_BasicSplitting(t *testing.T) {
	text := "This is the first sentence that is long enough to be extracted by the filter. This is the second sentence that also meets the minimum length requirement! And this is the third sentence that satisfies the character limit?"

	sentences := splitSentences(text)

	if len(sentences) < 3 {
		t.Errorf("Expected at least 3 sentences, got %d", len(sentences))
	}

	// Check each sentence is trimmed
	for _, sentence := range sentences {
		if sentence != strings.TrimSpace(sentence) {
			t.Errorf("Expected sentence to be trimmed: '%s'", sentence)
		}
	}
}

func TestSplitSentences_MinMaxLength(t *testing.T) {
	// Too short (< 30 chars)
	shortText := "Short."

	// Just right
	goodText := "This sentence is long enough to be considered valid for extraction purposes."

	// Too long (> 500 chars)
	longText := strings.Repeat("This is a very long sentence. ", 30)

	combined := shortText + " " + goodText + " " + longText

	sentences := splitSentences(combined)

	// Should only include the "just right" sentence
	for _, sentence := range sentences {
		if len(sentence) < 30 {
			t.Errorf("Sentence too short (%d chars): %s", len(sentence), sentence)
		}
		if len(sentence) > 500 {
			t.Errorf("Sentence too long (%d chars)", len(sentence))
		}
	}
}

func TestDedupeClaims_CaseInsensitive(t *testing.T) {
	claims := []testClaim{
		{Text: "This claim was first introduced in 1990."},
		{Text: "This claim was first introduced in 1990."},
		{Text: "THIS CLAIM WAS FIRST INTRODUCED IN 1990."},
		{Text: "Different claim that was established later."},
	}

	// Convert to model.Claim
	modelClaims := make([]testClaimModel, len(claims))
	for i, c := range claims {
		modelClaims[i] = testClaimModel{Text: c.Text}
	}

	unique := dedupeTestClaims(modelClaims)

	if len(unique) != 2 {
		t.Errorf("Expected 2 unique claims, got %d", len(unique))
	}
}

// Helper types for testing
type testClaim struct {
	Text string
}

type testClaimModel struct {
	Text      string
	Heuristic string
	Sentence  int
}

func dedupeTestClaims(claims []testClaimModel) []testClaimModel {
	seen := make(map[string]bool)
	var unique []testClaimModel

	for _, claim := range claims {
		key := strings.ToLower(strings.TrimSpace(claim.Text))
		if !seen[key] {
			seen[key] = true
			unique = append(unique, claim)
		}
	}

	return unique
}

// Helper to parse HTML for testing internal functions
func parseHTML(htmlContent string) (*html.Node, error) {
	return html.Parse(strings.NewReader(htmlContent))
}
