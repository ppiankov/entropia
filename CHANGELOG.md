# Changelog

All notable changes to Entropia will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.7] - 2026-01-28

### Added

**Wikipedia Conflict Detection:**
- Edit war detection via Wikipedia API revision history analysis
  - Tracks recent edits (last 30 days), revert count, unique editors
  - Calculates edit frequency (edits/day) and conflict severity (low/medium/high)
  - Threshold: High conflict if >10 edits/month AND >3 reverts, OR >5 edits/day
  - Returns transparent diagnostic data (recent_edits, revert_count, unique_editors, edit_frequency)
- Historical entity anachronism detection
  - Scans article text for references to defunct states that no longer exist
  - Database of 8 historical entities: Kyivan Rus (1240), USSR (1991), Yugoslavia (1992), Czechoslovakia (1993), Ottoman Empire (1922), Austria-Hungary (1918), Polish-Lithuanian Commonwealth (1795), Grand Duchy of Lithuania (1795)
  - Only flags entities that ceased to exist >30 years ago (avoids recent political changes)
  - Provides context snippets showing where entities are mentioned
- New signal types: `edit_war`, `historical_entity`
- Automatic detection on Wikipedia URLs (wikipedia.org domain check)
- Example: Borscht article detects 4 historical entities (Kyivan Rus, USSR, Polish-Lithuanian Commonwealth, Grand Duchy of Lithuania)

**TLS/SSL Certificate Validation:**
- Comprehensive certificate information capture during HTTP fetch
  - TLS version (1.0, 1.1, 1.2, 1.3)
  - Certificate subject and issuer
  - Validity dates (NotBefore, NotAfter)
  - Subject Alternative Names (DNS names covered by cert)
  - Expired certificate detection (before NotBefore or after NotAfter)
  - Self-signed certificate detection (issuer == subject)
  - Domain mismatch detection (cert doesn't cover URL hostname)
- Four new security signals added to reports:
  - `no_tls`: Page served over HTTP without encryption [WARNING]
  - `expired_certificate`: Certificate expired or not yet valid [CRITICAL]
  - `self_signed_certificate`: Self-signed cert (not verified by CA) [WARNING]
  - `certificate_mismatch`: Certificate domain doesn't match URL [CRITICAL]
- New `--insecure` flag to skip TLS verification (for self-signed certs in dev/test)
- TLS info added to `fetch_meta.tls` field in JSON reports

**Fixes:**
- Fixed Wikipedia evidence extraction to exclude navigation links (Main Page, Portal, etc.)
  - Previously counted 1177+ internal Wikipedia links as "evidence"
  - Now only external sources count (e.g., Borscht: 1176 evidence links → mostly legitimate external sources)
- Added User-Agent header to Wikipedia API requests (required by Wikipedia)
- Fixed URL encoding for Cyrillic/non-ASCII Wikipedia titles (e.g., "Борщ")
- Increased default scan timeout from 30s to 2 minutes for large Wikipedia pages
- Added 30-second timeout for Wikipedia conflict detection to prevent hangs

**Configuration:**
- New config option: `http.insecure_tls` (boolean, default: false)
- Wired InsecureTLS flag through Fetcher to http.Transport.TLSClientConfig

**Testing:**
- New test program: `cmd/test-wikipedia-conflicts/main.go`
  - Demonstrates edit war detection on Borscht Wikipedia pages
  - Demonstrates historical entity detection on sample text
  - Can be run independently for testing

### Changed

- Wikipedia adapter now passes raw HTML content to DetectWikipediaConflicts() for better text extraction
- FetchResult now includes TLS information in Meta field
- Pipeline generates TLS security signals after fetching, before scoring
- Fetcher now accepts `insecureTLS` parameter to configure TLS verification

### Technical Details

**Edit War Detection Algorithm:**
```
High conflict if:
  (recent_edits > 10 AND revert_count > 3) OR edit_frequency > 5
Medium conflict if:
  (recent_edits > 5 AND revert_count > 1) OR edit_frequency > 2
Low conflict if:
  revert_count > 0
```

**Historical Entity Detection:**
- Searches raw HTML for entity names (primary + aliases)
- Extracts 50-char context before/after each occurrence
- Only signals entities extinct >30 years (2026 - end_year > 30)

**TLS Certificate Validation:**
- Uses `crypto/x509.Certificate.VerifyHostname()` for domain matching
- Captures leaf certificate from `http.Response.TLS.PeerCertificates[0]`
- All certificate issues are WARNING or CRITICAL severity (never INFO)

### Rationale

**Why Wikipedia Conflict Detection?**
As noted by user: "high rate of changes back and forth is a conflict" and "links to non-existent countries is a sign of conflict". Wikipedia edit wars and historical entity anachronisms are strong indicators that content is contested in modern identity/origin disputes.

**Why TLS Certificate Validation?**
As noted by user: "no certificate or no valid certificate is a bad sign". Certificate issues suggest:
- Site not actively maintained (expired certs)
- Misconfiguration (domain mismatch)
- Lower trust (self-signed, no HTTPS)
- Forgot to renew (expired but otherwise valid)

All issues are now transparently reported with full diagnostic data.

### Examples

**Wikipedia Conflict Detection on Borscht:**
```bash
./entropia scan https://en.wikipedia.org/wiki/Borscht --json report.json
```
Signals generated:
- 4 historical entities detected: Kyivan Rus (786 years ago), USSR (35 years ago), Polish-Lithuanian Commonwealth (231 years ago), Grand Duchy of Lithuania (231 years ago)
- No edit war detected (5 recent edits, 1 revert = below threshold)

**TLS Validation on HTTP Site:**
```bash
./entropia scan http://example.com --json report.json
```
Signal generated:
- WARNING: "Page served over HTTP without encryption"

**Skip TLS Verification:**
```bash
./entropia scan https://self-signed.example.com --insecure --json report.json
```

---

## [0.1.0] - 2026-01-27

### Added

**Core Features:**
- Transparent scoring system (0-100 scale) with documented formulas
- Concurrent evidence validation with 20 workers per scan
- Authority classification system (3-tier: Primary/Secondary/Tertiary)
- Staleness detection via HTTP Last-Modified headers (1+ year = stale, 3+ years = very stale)
- Conflict detection for competing origin claims
- Batch processing with configurable worker pools
- Multi-layer caching (memory + disk) with 24h TTL
- Per-domain rate limiting with robots.txt compliance

**Extraction:**
- Keyword-based claim extraction (18 heuristics: "originated", "according to", "is legally", etc.)
- Evidence link extraction with classification (citations, references, external links)
- Domain adapters: Wikipedia, Legal, Generic (fallback)

**LLM Integration (Optional):**
- OpenAI provider (gpt-4o-mini, gpt-4o)
- Anthropic Claude provider (claude-3-5-sonnet, claude-3-5-haiku)
- Ollama provider (local models: llama3.1, mistral, etc.)
- Strict evidence mode (LLM can only cite extracted URLs, prevents hallucinations)
- Separate report output (report.llm.md) with disclaimers

**Scoring Components:**
- Evidence Coverage (0-40 points): `min(evidence_count / claim_count * 40, 40)`
- Authority Distribution (0-30 points): `(primary×3 + secondary×2 + tertiary×1) / (total×3) * 30`
- Freshness (0-20 points): `20 - min(median_age_years * 5, 20)`
- Accessibility (0-10 points): `(accessible_count / total_count) * 10`
- Conflict Penalty (-10 points): Applied for mutually exclusive claims

**CLI Commands:**
- `entropia scan` - Scan single URL
- `entropia batch` - Batch process multiple URLs from file
- `entropia config show` - Display current configuration
- `entropia config init` - Create default config file
- `entropia version` - Show version information

**Output Formats:**
- JSON reports (machine-readable, full data)
- Markdown reports (human-readable, formatted)
- Console summary with support index and signals
- Optional LLM summary (separate file)

**Documentation:**
- Complete CLI guide with examples
- Configuration reference with use cases
- README with architecture overview
- Methodology documentation

**Testing:**
- 380+ unit tests with 92%+ coverage in core packages
- Authority classification tests (80+ cases)
- Concurrent validation tests
- Claim and evidence extraction tests
- LLM summarizer tests with mock providers

**CI/CD:**
- GitHub Actions CI pipeline (build, test, lint)
- Automated multi-platform releases (Linux amd64/arm64, macOS Intel/Apple Silicon, Windows x86_64)
- SHA256 checksums for binary verification

### Known Limitations

**Claim Extraction:**
- Uses keyword-based heuristics (no NLP/ML)
- May miss claims that don't match keyword patterns
- Sentence length filtering (30-500 chars) may exclude valid claims

**Evidence Extraction:**
- Legal document adapter has incomplete evidence extraction (marked as TODO for v0.2)
- Generic adapter may not capture domain-specific citation formats

**Conflict Detection:**
- Limited to origin-based conflicts ("originated in X" vs "originated in Y")
- Does not detect temporal conflicts, quantity conflicts, or negation conflicts

**Validation:**
- Freshness detection requires Last-Modified headers (not all sites provide this)
- Authority classification is domain-based (may not reflect actual source quality)

**LLM Integration:**
- Requires API keys for OpenAI/Anthropic (environment variables or config file)
- Ollama requires local installation and model download
- Strict evidence mode may reject valid summaries if LLM rephrases URLs

### Performance

- Single URL scan: ~2-5 seconds (depending on evidence link count)
- Batch processing: ~100 URLs in ~5-10 minutes with 10 workers
- Concurrent validation: Up to 20 evidence links validated simultaneously per scan

### Dependencies

- Go 1.22 or later
- Key libraries: cobra, viper, go-openai, robotstxt, golang.org/x/net

### Security

- No API keys stored in code
- Configuration via environment variables or YAML files
- robots.txt compliance enforced
- Per-domain rate limiting to prevent abuse

### Philosophy

Entropia is a **non-normative tool** - it evaluates how well claims are supported by evidence, not whether claims are true. It acts as a "mirror, not an oracle."

---

## Future Roadmap

**Planned for v0.2.0:**
- Complete legal document adapter evidence extraction
- Enhanced conflict detection (temporal, quantity, negation)
- Integration tests (end-to-end pipeline with mock HTTP server)
- Structured logging with zerolog
- Golden test cases against manual artifacts
- Additional domain adapters (news sites, academic papers)

**Long-term (v0.3.0+):**
- NLP-based claim extraction (spaCy integration)
- Historical drift tracking (scan over time, detect changes)
- Web UI for report visualization
- Database backend for scan history
- GitHub Action for PR validation
- REST API for integrations

---

[0.1.7]: https://github.com/ppiankov/entropia/releases/tag/v0.1.7
[0.1.0]: https://github.com/ppiankov/entropia/releases/tag/v0.1.0
