---
name: entropia
description: Non-normative source credibility verifier — evaluates evidence support quality for public claims
user-invocable: false
metadata: {"requires":{"bins":["entropia"]}}
---

# entropia — Source Credibility Verification

You have access to `entropia`, a non-normative diagnostic tool that evaluates how well public claims are supported by available, current, and authoritative sources. It does NOT determine truth — it measures evidence quality.

## Install

```bash
brew install ppiankov/tap/entropia
```

Or from binary:

```bash
curl -L https://github.com/ppiankov/entropia/releases/latest/download/entropia-$(uname -s)-$(uname -m) -o /usr/local/bin/entropia
chmod +x /usr/local/bin/entropia
```

## Commands

### entropia scan

Analyze a single URL for source credibility. Primary command.

**Flags:**
- `--json <path>` — JSON report output path (default: `report.json`)
- `--format json` — alias: use `--json <path>` to write JSON output
- `--md <path>` — markdown report output path
- `--timeout <dur>` — overall scan timeout (default: `2m`)
- `--no-cache` — disable cache
- `--insecure` — skip TLS verification
- `--llm` — enable LLM summary
- `--llm-provider <name>` — openai, anthropic, or ollama (default: `openai`)
- `--llm-model <name>` — model name (default: `gpt-4o-mini`)

**JSON output:**
```json
{
  "subject": "Page Title",
  "source_url": "https://example.com",
  "claims": [
    {"text": "Claim text...", "heuristic": "extraction rule"}
  ],
  "evidence": [
    {"url": "https://source.com", "authority": "primary", "kind": "citation"}
  ],
  "validation": [
    {"url": "https://source.com", "is_accessible": true, "is_stale": false, "is_dead": false}
  ],
  "score": {
    "index": 73,
    "confidence": "high",
    "conflict": false,
    "signals": [
      {"type": "evidence_coverage", "severity": "info", "description": "Good ratio"}
    ]
  }
}
```

**Exit codes:**
- 0: success
- 1: error (details on stderr)

### entropia batch

Process multiple URLs in parallel.

**Flags:**
- `--concurrency <n>` — concurrent workers (default: NumCPU)
- `--output-dir <path>` — report output directory (default: `./entropia-reports`)
- `--timeout <dur>` — total batch timeout (default: `10m`)
- `--scan-timeout <dur>` — per-scan timeout (default: `30s`)

### entropia version

Print version.

**Flags:**
(none)

### entropia init

Not implemented. Reserved for future use to bootstrap config and cache directory.

## Scoring Formula

The Support Index (0-100) is transparent and deterministic:

```
Total = Coverage (40) + Authority (30) + Freshness (20) + Accessibility (10)

Coverage:    min(evidence_count / claim_count * 40, 40)
Authority:   (primary*3 + secondary*2 + tertiary*1) / total * 30
Freshness:   20 - min(median_age_years * 5, 20)
Accessibility: accessible_ratio * 10

Penalty: -10 for competing claims (conflict detected)
```

Confidence: 0-40 = low, 41-70 = medium, 71-100 = high.

## Interpreting Results

| Index | Confidence | Meaning |
|-------|-----------|---------|
| 80-100 | high | Well-supported with current, authoritative sources |
| 60-79 | medium | Decent support, may have stale sources or gaps |
| 40-59 | medium | Significant gaps in evidence coverage or freshness |
| 0-39 | low | Poorly supported — dead links, missing evidence, or conflicts |

When `conflict: true` — entropia found competing claims (e.g., multiple origin attributions). This does NOT mean the content is wrong, only that sources disagree.

## Signal Types

Key diagnostic signals in `.score.signals[]`:

| Signal | Severity | Meaning |
|--------|----------|---------|
| `evidence_coverage` | info | Claims-to-evidence ratio |
| `authority_distribution` | info | Balance of primary/secondary/tertiary sources |
| `freshness` | info/warning | Age of median source |
| `accessibility` | info/warning | Ratio of accessible links |
| `conflict` | warning | Competing claims detected |
| `stale_sources` | warning | Old citations |
| `high_entropy` | warning | High claim density, low support |
| `no_tls` | warning | Page served over HTTP |
| `expired_certificate` | critical | TLS cert expired |
| `edit_war` | warning | Wikipedia: high edit frequency + reverts |

## Integration with noisepan

noisepan calls entropia automatically via `noisepan verify`. No manual wiring needed — just ensure `entropia` is in PATH.

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `OPENAI_API_KEY` | only with `--llm` | OpenAI API key |
| `ANTHROPIC_API_KEY` | only with `--llm --llm-provider anthropic` | Anthropic key |
| `ENTROPIA_CONFIG` | no | Override config file path |

## What this does NOT do

- Does not determine truth — evaluates evidence support only
- Does not use ML for scoring — deterministic formula
- Does not modify content — read-only analysis
- Does not store data remotely — cache is local only
- Does not process PDFs — HTML pages only
- Does not work on auth-required pages (reddit.com, t.me)

## Parsing examples

```bash
# Get support index (0-100)
jq '.score.index' report.json

# Get confidence level
jq '.score.confidence' report.json

# Check for conflicts
jq '.score.conflict' report.json

# List diagnostic signals
jq '.score.signals[] | {type, severity, description}' report.json

# Find dead links
jq '.validation[] | select(.is_dead == true) | .url' report.json

# Find stale sources (>1 year old)
jq '.validation[] | select(.is_stale == true) | .url' report.json

# Count claims vs evidence
jq '{claims: (.claims | length), evidence: (.evidence | length)}' report.json
```

---

This tool follows the [Agent-Native CLI Convention](https://ancc.dev). Validate with: `ancc validate .`
