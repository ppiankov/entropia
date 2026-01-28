# Changelog

All notable changes to Entropia will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.11] - 2026-01-28

### Added
- robots.txt file
- Authority classification comments in config.example.yaml
- Language support section in README.md

### Changed
- Simplified authority configuration in config.example.yaml

---

## [0.1.10] - 2026-01-28

### Added
- BorshWars case study in FIELD_NOTES.md

---

## [0.1.9] - 2026-01-28

### Added
- FIELD_NOTES.md

---

## [0.1.8] - 2026-01-28

### Added
- Wikipedia edit war detection via API (tracks edits, reverts, edit frequency)
- Historical entity detection for defunct states (8 entities: Kyivan Rus, USSR, Yugoslavia, etc.)
- TLS certificate validation (version, expiration, self-signed, domain mismatch)
- Freshness anomaly signal (flags topics with suspiciously recent sources)
- New signals: `edit_war`, `historical_entity`, `no_tls`, `expired_certificate`, `self_signed_certificate`, `certificate_mismatch`, `freshness_anomaly`
- `--insecure` flag to skip TLS verification
- Test program: `cmd/test-wikipedia-conflicts/main.go`

### Fixed
- Wikipedia evidence extraction now excludes navigation links (Main Page, Portal, Special:, Help:, Talk:)
- URL encoding for Cyrillic Wikipedia titles
- User-Agent header for Wikipedia API requests
- go vet warnings

### Changed
- Default scan timeout increased from 30s to 2m
- Fetcher accepts `insecureTLS` parameter
- FetchResult includes TLS information in Meta field

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

[0.1.10]: https://github.com/ppiankov/entropia/releases/tag/v0.1.10
[0.1.0]: https://github.com/ppiankov/entropia/releases/tag/v0.1.0
