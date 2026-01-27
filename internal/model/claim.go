package model

// Claim represents a factual assertion extracted from the source
type Claim struct {
	Text      string `json:"text"`                // The claim text itself
	Heuristic string `json:"heuristic,omitempty"` // Which extraction rule matched (e.g., "keyword:originated")
	Sentence  int    `json:"sentence,omitempty"`  // Sentence index in source (0-based)
}

// ClaimType categorizes the nature of the claim
type ClaimType string

const (
	ClaimTypeOrigin      ClaimType = "origin"       // Claims about origin/first occurrence
	ClaimTypeAttribution ClaimType = "attribution"  // Claims about who did/created something
	ClaimTypeAuthority   ClaimType = "authority"    // Claims about legal/official status
	ClaimTypeExistence   ClaimType = "existence"    // Claims about something existing
	ClaimTypeDefinition  ClaimType = "definition"   // Definitional claims
)
