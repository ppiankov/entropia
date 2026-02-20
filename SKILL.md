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

| Command | What it does |
|---------|-------------|
| `entropia scan <url>` | Analyze a single URL |
| `entropia batch <file>` | Process multiple URLs in parallel |
| `entropia version` | Print version |

## Key Flags

### scan

| Flag | Default | Description |
|------|---------|-------------|
| `--json <path>` | `report.json` | JSON report output path |
| `--md <path>` | — | Markdown report output path |
| `--timeout <dur>` | `2m` | Overall scan timeout |
| `--no-cache` | false | Disable cache |
| `--insecure` | false | Skip TLS verification |
| `--llm` | false | Enable LLM summary |
| `--llm-provider` | `openai` | openai, anthropic, or ollama |
| `--llm-model` | `gpt-4o-mini` | Model name |

### batch

| Flag | Default | Description |
|------|---------|-------------|
| `--concurrency <n>` | NumCPU | Concurrent workers |
| `--output-dir <path>` | `./entropia-reports` | Report output directory |
| `--timeout <dur>` | `10m` | Total batch timeout |
| `--scan-timeout <dur>` | `30s` | Per-scan timeout |

## Agent Usage Pattern

For programmatic use, always request JSON output:

```bash
entropia scan https://example.com --json /tmp/report.json
```

### JSON Output Structure (key fields)

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

### Parsing Examples

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

## Typical Workflows

### Single URL verification

```bash
entropia scan https://example.com/article --json report.json
score=$(jq '.score.index' report.json)
if [ "$score" -lt 60 ]; then
  echo "Low support: $score/100"
fi
```

### Batch documentation audit

```bash
echo "https://docs.example.com/page1" > urls.txt
echo "https://docs.example.com/page2" >> urls.txt
entropia batch urls.txt --concurrency 5 --output-dir ./reports
```

### Integration with noisepan

noisepan calls entropia automatically via `noisepan verify`. No manual wiring needed — just ensure `entropia` is in PATH.

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

## What entropia Does NOT Do

- Does not determine truth — evaluates evidence support only
- Does not use ML for scoring — deterministic formula
- Does not modify content — read-only analysis
- Does not store data remotely — cache is local only
- Does not process PDFs — HTML pages only
- Does not work on auth-required pages (reddit.com, t.me)

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `OPENAI_API_KEY` | only with `--llm` | OpenAI API key |
| `ANTHROPIC_API_KEY` | only with `--llm --llm-provider anthropic` | Anthropic key |
| `ENTROPIA_CONFIG` | no | Override config file path |

## Exit Codes

- `0` — success
- `1` — error (details on stderr)
