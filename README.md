# Entropia

**A non-normative tool for detecting entropy, decay, and support gaps in public claims.**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)

Entropia evaluates how well claims are supported by available, current, and authoritative sources. It highlights conflicts, gaps, and drift between claims and evidence.

**Entropia is a mirror, not an oracle** - it does NOT determine truth, but shows the quality of evidence support.

---

## 🚀 Quick Start

### Installation

**From Binary (Recommended):**
```bash
# macOS/Linux
curl -L https://github.com/ppiankov/entropia/releases/latest/download/entropia-$(uname -s)-$(uname -m) -o entropia
chmod +x entropia
sudo mv entropia /usr/local/bin/
```

**From Source:**
```bash
git clone https://github.com/ppiankov/entropia.git
cd entropia
make build
sudo cp bin/entropia /usr/local/bin/
```

**Verify:**
```bash
entropia --version
```

### Basic Usage

**Scan a single URL:**
```bash
entropia scan https://en.wikipedia.org/wiki/Laksa
```

**Scan multiple URLs in parallel:**
```bash
# Create a file with URLs (one per line)
cat > urls.txt <<EOF
https://en.wikipedia.org/wiki/Laksa
https://en.wikipedia.org/wiki/Common-law_marriage
https://en.wikipedia.org/wiki/List_of_common_misconceptions
EOF

# Scan with 5 concurrent workers
entropia batch urls.txt --concurrency 5
```

**Generate AI summary:**
```bash
export OPENAI_API_KEY=sk-...
entropia scan https://example.com --llm --llm-provider openai
```

**Skip TLS verification (for self-signed certificates):**
```bash
entropia scan https://self-signed.example.com --insecure
```

---

## 🆕 New in v0.1.7

### Wikipedia Conflict Detection

Automatically detects contested content on Wikipedia pages:

**Edit War Detection:**
- Analyzes Wikipedia revision history via API
- Tracks edit frequency, revert count, unique editors
- Flags high-conflict articles (>10 edits/month + >3 reverts, OR >5 edits/day)
- Example: Flags articles with competing national identity claims

**Historical Entity Anachronisms:**
- Detects references to 8 defunct states: Kyivan Rus (1240), USSR (1991), Yugoslavia (1992), Czechoslovakia (1993), Ottoman Empire (1922), Austria-Hungary (1918), Polish-Lithuanian Commonwealth (1795), Grand Duchy of Lithuania (1795)
- Only flags entities extinct >30 years (avoids recent political changes)
- Provides context snippets showing where entities are mentioned
- Example: Borscht article references 4 historical entities (Kyivan Rus, USSR, Polish-Lithuanian Commonwealth, Grand Duchy of Lithuania)

### TLS/SSL Security Validation

Captures and validates certificate information for all scanned URLs:

**Certificate Information:**
- TLS version (1.0, 1.1, 1.2, 1.3)
- Subject, issuer, validity dates
- Subject Alternative Names (DNS names)
- Expiration status, self-signed detection, domain mismatch

**Security Signals:**
- 🔴 **No TLS**: Page served over HTTP (no encryption) - WARNING
- 🔴 **Expired Certificate**: Certificate expired or not yet valid - CRITICAL
- ⚠️ **Self-Signed**: Certificate not verified by trusted CA - WARNING
- 🔴 **Domain Mismatch**: Certificate issued for different domain - CRITICAL

**Use `--insecure` flag** to bypass certificate verification for development/testing.

### Freshness Anomaly Detection

Detects when topics have suspiciously recent sources:

**When It Triggers:**
- More than 50 sources with age data
- Median age less than 1 year
- Topic likely has historical significance

**What It Means:**
For historical topics (like a 700-year-old soup), having ALL sources be very recent (<1 year) suggests:
- Ongoing content disputes
- Edit wars or frequent revisions
- Unstable information rather than established facts

**Example (Borscht):**
- 860 sources with age data
- Median age: 0.003 years (about 1 day!)
- Signal: ⚠️ "Suspiciously recent sources: all evidence very new despite topic likely being historical"

### Improved Evidence Quality

**Wikipedia Navigation Link Filtering:**
Previously, Wikipedia UI/navigation links were incorrectly counted as evidence:
- Main Page, Portal, Special: pages, Help: pages
- Talk pages, edit/history links
- Self-references

Now properly filtered:
- Example: Borscht 1177 → 978 evidence links (removed ~200 navigation links)
- Only external sources and legitimate cross-references count
- More accurate evidence-to-claim ratios

---

## 📚 What Entropia Does

✅ **Extracts claims** from public web pages
✅ **Maps claims to sources** (citations, references, evidence links)
✅ **Validates evidence quality** (accessibility, freshness, authority)
✅ **Detects conflicts** (competing claims, contradictions)
✅ **Scores support quality** (transparent, formula-based: 0-100 scale)
✅ **Generates reports** (JSON for automation, Markdown for humans)
✅ **Wikipedia conflict detection** (edit wars, historical entity anachronisms)
✅ **TLS/SSL security validation** (expired certs, self-signed, domain mismatch)
✅ **Freshness anomaly detection** (suspiciously recent sources for historical topics)

### Non-Normative Philosophy

Entropia **evaluates support**, not truth. It answers:
- "How well is this claim supported by cited sources?"
- "Are the sources accessible, current, and authoritative?"
- "Do sources conflict with each other?"

Entropia **does NOT answer**:
- "Is this claim true?"
- "Which source is correct?"
- "What should I believe?"

**Use cases:**
- Wikipedia editors auditing citation quality
- Compliance officers verifying policy document sources
- Researchers evaluating literature review quality
- Content managers monitoring documentation for link decay
- Journalists assessing source quality in investigative reporting

---

## 🔍 How It Works

### 1. Claim Extraction

Identifies factual and attributional claims using keyword-based heuristics:
- "originated in...", "first introduced...", "according to..."
- "is defined as...", "under the law...", "statute requires..."

### 2. Evidence Extraction

Extracts and classifies evidence links:
- **Citations**: Numbered references (Wikipedia's `class="reference"`)
- **External links**: "External links" and "Further reading" sections
- **References**: Inline and bibliographic citations

### 3. Evidence Validation (Concurrent)

For each evidence URL, checks:
- **Accessibility**: HTTP HEAD request to detect 404s, timeouts
- **Freshness**: `Last-Modified` header to calculate age
- **Authority**: Domain-based classification (primary/secondary/tertiary tiers)
- **TLS Security**: Certificate validity, expiration, domain matching

For Wikipedia pages, also detects:
- **Edit Wars**: High edit frequency and revert patterns via Wikipedia API
- **Historical Entities**: References to defunct states (Kyivan Rus, USSR, etc.)

**Concurrency:** Validates up to 20 evidence URLs in parallel per scan.

### 4. Transparent Scoring

Calculates a **Support Index** (0-100) using documented formulas:

```
Total Score = Coverage (40) + Authority (30) + Freshness (20) + Accessibility (10)
```

- **Coverage (0-40 pts)**: Evidence-to-claim ratio
  `score = min(evidence_count / claim_count * 40, 40)`

- **Authority (0-30 pts)**: Source quality tier balance
  `score = (primary * 3 + secondary * 2 + tertiary * 1) / total * 30`

- **Freshness (0-20 pts)**: Median age of sources
  `score = 20 - min(median_age_years * 5, 20)`

- **Accessibility (0-10 pts)**: Ratio of accessible links
  `score = accessible_ratio * 10`

**Conflict Penalty:** -10 points for competing claims (e.g., "originated in Malaysia" AND "originated in Indonesia")

All scores include transparent diagnostic signals with formulas, inputs, and rationale.

### 5. Report Generation

Outputs:
- **JSON**: Machine-readable report with full data
- **Markdown**: Human-readable formatted report
- **LLM Summary (optional)**: AI-generated summary with **strict evidence mode** (only cites URLs from extracted evidence)

---

## 📖 Documentation

- **[CLI Guide](docs/CLI_GUIDE.md)** - Complete command reference
- **[Configuration Guide](docs/CONFIGURATION.md)** - Advanced configuration options
- **[Principles](PRINCIPLES.md)** - Core design principles
- **[Methodology](docs/METHODOLOGY.md)** - Scoring and extraction methodology

---

## 🏗️ Architecture

**Built with Go** for performance and concurrency:

- **Concurrent validation**: 20 goroutines per scan for evidence checking
- **Batch processing**: Worker pools for scanning 100+ URLs in parallel
- **Multi-layer caching**: Memory + disk with TTL-based expiration
- **Rate limiting**: Per-domain rate limiting with robots.txt compliance
- **Domain adapters**: Pluggable extractors for Wikipedia, legal documents, generic HTML
- **LLM providers**: OpenAI, Anthropic Claude, Ollama (local) with citation leak prevention

**Key Components:**
```
entropia/
├── cmd/entropia/           # CLI entry point
├── internal/
│   ├── pipeline/           # Orchestration (fetch → extract → validate → score)
│   ├── extract/            # Claim & evidence extraction with domain adapters
│   ├── validate/           # Concurrent evidence validation
│   ├── score/              # Transparent scoring engine
│   ├── llm/                # Multi-provider LLM integration
│   ├── worker/             # Concurrency primitives (pools, limiters)
│   ├── cache/              # Multi-layer caching
│   └── model/              # Data structures
└── docs/                   # Documentation
```

---

## 🎯 Examples

### Example 1: Wikipedia Citation Quality Audit

```bash
entropia scan https://en.wikipedia.org/wiki/Laksa --json laksa.json --md laksa.md
cat laksa.md
```

**Output snippet:**
```
═══════════════════════════════════════════════════════════
  Entropia Report: Laksa
═══════════════════════════════════════════════════════════

  Support Index:  73 / 100  (high confidence)
  Claims:         75
  Evidence:       112

  Signals:
  ⚠️  conflict_detected: Competing origin claims
  ℹ️  evidence_coverage: Good evidence-to-claim ratio
  ℹ️  freshness: Median source age 2.3 years
```

### Example 2: Batch Processing for Documentation Maintenance

```bash
# Create URL list
cat > docs-urls.txt <<EOF
https://docs.example.com/getting-started
https://docs.example.com/api-reference
https://docs.example.com/deployment
EOF

# Scan all pages concurrently
entropia batch docs-urls.txt --concurrency 3 --output-dir ./reports

# Check for pages with low support
for report in ./reports/*.json; do
  SCORE=$(jq '.score.index' $report)
  if [ "$SCORE" -lt 60 ]; then
    echo "WARNING: $(basename $report) has low support index: $SCORE"
  fi
done
```

### Example 3: CI/CD Integration - Block Low-Quality PRs

```yaml
# .github/workflows/citation-check.yml
name: Citation Quality Check
on: [pull_request]

jobs:
  entropia:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Install Entropia
        run: |
          curl -L https://github.com/ppiankov/entropia/releases/latest/download/entropia-Linux-x86_64 -o /usr/local/bin/entropia
          chmod +x /usr/local/bin/entropia
      - name: Extract URLs from changed files
        run: |
          git diff origin/main --name-only | xargs grep -oP 'https?://[^\s]+' > urls.txt || true
      - name: Scan URLs
        run: |
          if [ -s urls.txt ]; then
            entropia batch urls.txt --no-cache
          fi
      - name: Check quality threshold
        run: |
          for report in ./entropia-reports/*.json; do
            SCORE=$(jq '.score.index' $report)
            if [ "$SCORE" -lt 60 ]; then
              echo "ERROR: Low support index: $SCORE"
              exit 1
            fi
          done
```

### Example 4: LLM-Powered Reports

```bash
# Configure API key
export OPENAI_API_KEY=sk-...

# Scan with AI summary (gpt-4o-mini)
entropia scan https://example.com --llm --llm-model gpt-4o-mini --md report.md

# View main report
cat report.md

# View AI summary (separate file)
cat report.llm.md
```

**Note:** LLM summaries use **strict evidence mode** - the AI can ONLY cite URLs from the extracted evidence list, preventing hallucinations.
Entropia never treats LLM output as evidence; LLMs are used only for summarization of already-validated data.

---

## 🌟 Why Entropia Is Unique

Entropia targets an adjacent problem space to fact-checking and link checking: **evidence support drift**.:

| Feature | Entropia | Fact-Checkers | Link Checkers | Citation Managers |
|---------|----------|---------------|---------------|-------------------|
| **Non-normative** (evaluates support, not truth) | ✅ | ❌ | ⚠️ | ⚠️ |
| **Staleness detection** (via Last-Modified) | ✅ | ❌ | ❌ | ❌ |
| **Authority classification** (3-tier system) | ✅ | ⚠️ | ❌ | ❌ |
| **Conflict detection** (competing claims) | ✅ | ✅ | ❌ | ❌ |
| **Batch processing at scale** (100+ URLs) | ✅ | ❌ | ✅ | ⚠️ |
| **Transparent scoring** (documented formulas) | ✅ | ❌ | ⚠️ | ❌ |

**Entropia combines 5 capabilities that currently require 3-4 separate tools + manual review.**

---

## 🛠️ Development

### Build from Source

**Requirements:**
- Go 1.22 or later
- Make (optional, for convenience)

**Build:**
```bash
git clone https://github.com/ppiankov/entropia.git
cd entropia
make build      # or: go build -o bin/entropia ./cmd/entropia
```

**Run Tests:**
```bash
make test       # or: go test ./...
```

### Project Status

**Current Version:** v0.1.7

**Implemented:**
- ✅ Core pipeline (fetch → extract → validate → score → render)
- ✅ CLI with scan and batch commands
- ✅ Concurrent evidence validation (20 workers)
- ✅ 3 LLM providers (OpenAI, Anthropic, Ollama) with strict evidence mode
- ✅ Domain adapters (Wikipedia, Legal, Generic)
- ✅ Transparent scoring engine
- ✅ Multi-layer caching and rate limiting
- ✅ robots.txt compliance
- ✅ **Wikipedia conflict detection** (edit wars + historical entities)
- ✅ **TLS/SSL certificate validation** with security signals
- ✅ 380+ unit tests with 92%+ coverage

**Roadmap (Post-v0.1):**
- 🔮 Additional domain adapters (news sites, academic papers, blogs)
- 🔮 Historical drift tracking (scan over time, detect changes)
- 🔮 Web UI for report visualization
- 🔮 Database backend for report history
- 🔮 GitHub Action for PR validation
- 🔮 REST API for integrations

---

## 📝 Manual Artifacts (v0.1 Methodology Validation)

This repository includes **3 manually-created artifacts** to validate the methodology:

| Artifact | Focus | Support Index | Key Finding |
|---------|-------|---------------|-------------|
| [Laksa Origin](artifacts/laksa-origin/) | Cultural attribution dispute | 63/100 | **Conflict detected**: Competing origin claims (Malaysia vs Indonesia vs Singapore) |
| [UK Common-law Marriage](artifacts/uk-common-law-marriage/) | Legal misconception | 82/100 | High authority sources, but Wikipedia article contradicts popular belief |
| [Wikipedia Misconceptions](artifacts/wikipedia-misconceptions/) | High-noise claim environment | ~60/100 | Large claim count (500+), many unsupported assertions |

Each artifact contains:
- Source HTML snapshot
- Extracted claims
- Evidence mapping
- Validation results
- Support index calculation
- Diagnostic signals

**Automated implementation (v0.1.0) reproduces these results programmatically.**

---

## 🤝 Contributing

Contributions are welcome! Areas where help is needed:

1. **Domain Adapters**: Add extractors for news sites, academic papers, blogs
2. **Testing**: Unit tests, integration tests, golden test cases
3. **Documentation**: Tutorials, use case guides, API docs
4. **LLM Providers**: Add support for Gemini, Grok, z.ai
5. **Performance**: Optimize caching, reduce memory usage
6. **Features**: Historical tracking, web UI, database backend

**Guidelines:**
- Follow Go conventions and formatting (gofmt, golint)
- Add tests for new features
- Update documentation
- Maintain non-normative philosophy (evaluate support, not truth)

---

## 📄 License

MIT License - see [LICENSE](LICENSE)

---

## 🙏 Acknowledgments

Entropia was built to address a gap - systematic, non-normative evaluation of evidence support.

**Philosophy:** "We do not decide what is correct. We just do the checks on what there is and provide the reality check."

---

## 🔗 Links

- **Documentation**: [docs/](docs/)
- **Issue Tracker**: [GitHub Issues](https://github.com/ppiankov/entropia/issues)
- **Releases**: [GitHub Releases](https://github.com/ppiankov/entropia/releases)

---
