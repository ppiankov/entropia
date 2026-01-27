# Entropia Configuration Guide

Complete guide to configuring Entropia for your use case.

## Table of Contents

- [Configuration Hierarchy](#configuration-hierarchy)
- [Configuration File](#configuration-file)
- [Configuration Options](#configuration-options)
- [Environment Variables](#environment-variables)
- [Use Cases](#use-cases)

---

## Configuration Hierarchy

Entropia loads configuration from multiple sources with the following priority (highest to lowest):

1. **CLI flags** - Explicit command-line arguments
2. **Environment variables** - `ENTROPIA_*` variables
3. **Configuration file** - `~/.entropia/config.yaml`
4. **Built-in defaults** - Sensible defaults for most use cases

**Example:**
```bash
# Config file sets timeout to 30s
# CLI flag overrides to 60s
entropia scan https://example.com --timeout 60s
```

---

## Configuration File

### Location

Default: `~/.entropia/config.yaml`

Override with:
```bash
entropia scan https://example.com --config /path/to/custom.yaml
```

Or environment variable:
```bash
export ENTROPIA_CONFIG=/path/to/custom.yaml
entropia scan https://example.com
```

### Creating Configuration File

**Option 1: Use init command**
```bash
entropia config init
```

This creates `~/.entropia/config.yaml` with all options documented.

**Option 2: Manual creation**

Create `~/.entropia/config.yaml`:
```yaml
http:
  timeout: 30s
  user_agent: "Entropia/0.1 (+https://github.com/ppiankov/entropia)"
  follow_redirects: true
  max_redirects: 3
  max_body_bytes: 2000000

concurrency:
  workers: 4
  validation_workers: 20

rate_limiting:
  requests_per_second: 2.0
  respect_robots_txt: true
  burst_size: 5

cache:
  enabled: true
  ttl: 24h
  dir: ~/.entropia/cache

llm:
  provider: ""  # openai, anthropic, ollama, or "" (disabled)
  model: gpt-4o-mini
  strict_evidence: true
  timeout: 20
  max_tokens: 500

output:
  format: both  # json, markdown, or both
  dir: ./entropia-reports
  verbose: false

authority:
  primary_domains:
    - legislation.gov.uk
    - doi.org
    - scholar.google.com
    - jstor.org
    - arxiv.org
  secondary_domains:
    - wikipedia.org
    - britannica.com
    - nytimes.com
    - bbc.com
  path_patterns:
    - pattern: /statute/
      tier: primary
    - pattern: /legal/
      tier: primary
```

---

## Configuration Options

### HTTP Settings

Controls how Entropia fetches web pages.

```yaml
http:
  timeout: 30s                # HTTP request timeout
  user_agent: "Entropia/0.1"  # User-Agent header
  follow_redirects: true      # Follow HTTP redirects
  max_redirects: 3            # Maximum redirect hops
  max_body_bytes: 2000000     # Max response size (2MB)
```

**Use Cases:**
- **Slow sites**: Increase `timeout` to `60s` or more
- **Large pages**: Increase `max_body_bytes` to `10000000` (10MB)
- **Custom identification**: Set `user_agent` to include your contact info

### Concurrency Settings

Controls parallel processing.

```yaml
concurrency:
  workers: 4                  # Batch mode workers (URL scanning)
  validation_workers: 20      # Evidence validation goroutines
```

**Recommendations:**
- `workers`: Set to number of CPU cores (default: `runtime.NumCPU()`)
- `validation_workers`: Keep at 20 for I/O-bound tasks (HTTP HEAD requests)

**Examples:**
```yaml
# Conservative (low resource usage)
concurrency:
  workers: 2
  validation_workers: 10

# Aggressive (fast processing)
concurrency:
  workers: 16
  validation_workers: 50
```

### Rate Limiting

Prevents overwhelming target servers.

```yaml
rate_limiting:
  requests_per_second: 2.0   # Rate limit per domain
  respect_robots_txt: true   # Honor robots.txt directives
  burst_size: 5              # Burst allowance
```

**Important:**
- `requests_per_second`: Average rate (2 requests/second = polite default)
- `burst_size`: Allows brief bursts above average rate
- `respect_robots_txt`: **Always keep true** for ethical scraping

**robots.txt compliance:**
- Entropia parses `/robots.txt` for each domain
- Respects `Crawl-delay` directive
- Skips disallowed paths
- Cannot be disabled (ethical requirement)

### Caching

Speeds up repeated scans.

```yaml
cache:
  enabled: true              # Enable/disable caching
  ttl: 24h                   # Time-to-live for cached responses
  dir: ~/.entropia/cache     # Cache directory
```

**Cache Behavior:**
- **Memory cache**: LRU cache for recent fetches (500MB limit)
- **Disk cache**: Persistent storage with TTL expiration
- **Cache key**: URL + timestamp (rounded to TTL boundary)

**Use Cases:**
```yaml
# Development (aggressive caching)
cache:
  enabled: true
  ttl: 168h  # 1 week

# Production (conservative caching)
cache:
  enabled: true
  ttl: 1h    # 1 hour

# Always fresh (disable caching)
cache:
  enabled: false
```

**Clearing cache:**
```bash
rm -rf ~/.entropia/cache
```

### LLM Configuration

Controls optional AI-generated summaries.

```yaml
llm:
  provider: ""               # openai, anthropic, ollama, or "" (disabled)
  model: gpt-4o-mini         # Model name
  api_key: ""                # API key (use env vars instead!)
  base_url: ""               # Custom endpoint (for Ollama)
  strict_evidence: true      # Enforce URL allowlist (ALWAYS keep true)
  timeout: 20                # LLM request timeout (seconds)
  max_tokens: 500            # Max output tokens
```

**Security Note:** Store API keys in environment variables, NOT config files:
```bash
export OPENAI_API_KEY=sk-...
export ANTHROPIC_API_KEY=sk-ant-...
```

**Provider-Specific Settings:**

#### OpenAI
```yaml
llm:
  provider: openai
  model: gpt-4o-mini  # or gpt-4o, gpt-4o-turbo
```

Environment: `OPENAI_API_KEY=sk-...`

Cost: ~$0.15 per 1M input tokens (gpt-4o-mini)

#### Anthropic Claude
```yaml
llm:
  provider: anthropic
  model: claude-3-5-haiku-20241022  # or claude-3-5-sonnet-20241022
```

Environment: `ANTHROPIC_API_KEY=sk-ant-...`

Cost: Varies by model

#### Ollama (Local)
```yaml
llm:
  provider: ollama
  model: llama3.1:8b  # or mistral, phi, etc.
  base_url: http://localhost:11434
```

Environment: `OLLAMA_BASE_URL=http://localhost:11434` (optional)

Cost: Free (runs locally)

**Strict Evidence Mode:**
- `strict_evidence: true` (default, **never change**)
- Prevents LLM from citing URLs not in the evidence list
- Rejects responses that leak citations
- Critical for maintaining Entropia's credibility

### Output Settings

Controls report generation.

```yaml
output:
  format: both               # json, markdown, or both
  dir: ./entropia-reports    # Output directory for batch mode
  verbose: false             # Verbose console output
```

**Format Options:**
- `json`: Machine-readable, full data
- `markdown`: Human-readable, formatted
- `both`: Generate both formats (recommended)

### Authority Classification

Defines source quality tiers.

```yaml
authority:
  primary_domains:           # Tier 1: Authoritative sources
    - legislation.gov.uk
    - doi.org
    - scholar.google.com
    - jstor.org
    - arxiv.org

  secondary_domains:         # Tier 2: Reputable sources
    - wikipedia.org
    - britannica.com
    - nytimes.com
    - bbc.com

  path_patterns:             # URL pattern-based classification
    - pattern: /statute/
      tier: primary
    - pattern: /legal/
      tier: primary
```

**Authority Tiers:**
1. **Primary** (Tier 1): Laws, academic papers, official documents
2. **Secondary** (Tier 2): Encyclopedias, major publishers
3. **Tertiary** (Tier 3): Blogs, personal sites (default)

**Customization Example:**
```yaml
authority:
  primary_domains:
    - arxiv.org
    - pubmed.ncbi.nlm.nih.gov
    - your-organization.com  # Add your own authoritative sources

  secondary_domains:
    - medium.com
    - dev.to

  path_patterns:
    - pattern: /research/
      tier: primary
    - pattern: /blog/
      tier: tertiary
```

---

## Environment Variables

### LLM Provider Keys

| Variable | Description | Example |
|----------|-------------|---------|
| `OPENAI_API_KEY` | OpenAI API key | `sk-proj-...` |
| `ANTHROPIC_API_KEY` | Anthropic API key | `sk-ant-...` |
| `OLLAMA_BASE_URL` | Ollama server URL | `http://localhost:11434` |

**Usage:**
```bash
export OPENAI_API_KEY=sk-your-key
entropia scan https://example.com --llm --llm-provider openai
```

### Configuration Path

| Variable | Description | Example |
|----------|-------------|---------|
| `ENTROPIA_CONFIG` | Config file path | `/etc/entropia/config.yaml` |

**Usage:**
```bash
export ENTROPIA_CONFIG=/etc/entropia/custom.yaml
entropia scan https://example.com
```

---

## Use Cases

### Use Case 1: Wikipedia Editor - Citation Quality Audits

**Goal:** Audit citation quality across 100+ articles weekly

**Configuration:**
```yaml
concurrency:
  workers: 10              # Process 10 articles in parallel
  validation_workers: 30   # Check many citations per article

rate_limiting:
  requests_per_second: 3.0 # Be more aggressive (Wikipedia is robust)

cache:
  enabled: true
  ttl: 1h                  # Short TTL (citations change frequently)

authority:
  secondary_domains:
    - wikipedia.org        # Expect secondary sources
    - britannica.com
```

**Workflow:**
```bash
# Create article list
cat > wikipedia-articles.txt <<EOF
https://en.wikipedia.org/wiki/Laksa
https://en.wikipedia.org/wiki/Common-law_marriage
https://en.wikipedia.org/wiki/List_of_common_misconceptions
EOF

# Run weekly audit
entropia batch wikipedia-articles.txt --concurrency 10
```

### Use Case 2: Legal Compliance - Policy Document Verification

**Goal:** Ensure company policies cite current laws

**Configuration:**
```yaml
concurrency:
  workers: 2               # Conservative (legal sites are sensitive)
  validation_workers: 10

rate_limiting:
  requests_per_second: 1.0 # Be polite to government sites

authority:
  primary_domains:
    - legislation.gov.uk
    - law.cornell.edu
    - your-company.com     # Internal policy site

cache:
  enabled: false           # Always fetch latest (laws change)
```

**Workflow:**
```bash
# Scan company policy page
entropia scan https://your-company.com/policies/data-protection --no-cache

# Alert if support index drops below 70
SCORE=$(entropia scan https://your-company.com/policies/data-protection --json - | jq '.score.index')
if [ "$SCORE" -lt 70 ]; then
  echo "WARNING: Policy support quality dropped to $SCORE"
fi
```

### Use Case 3: Academic Researcher - Literature Review Quality

**Goal:** Evaluate citation quality in research papers

**Configuration:**
```yaml
authority:
  primary_domains:
    - doi.org
    - arxiv.org
    - pubmed.ncbi.nlm.nih.gov
    - scholar.google.com

  secondary_domains:
    - researchgate.net

llm:
  provider: ollama         # Use free local model
  model: llama3.1:8b
  strict_evidence: true

output:
  format: both             # JSON for analysis, Markdown for reading
```

**Workflow:**
```bash
# Scan paper landing page
entropia scan https://arxiv.org/abs/2301.12345 --llm --md paper-review.md

# Compare multiple papers
cat > papers.txt <<EOF
https://arxiv.org/abs/2301.12345
https://arxiv.org/abs/2302.67890
EOF

entropia batch papers.txt --concurrency 3 --llm
```

### Use Case 4: Content Manager - Documentation Maintenance

**Goal:** Monitor 1000+ documentation pages for link decay

**Configuration:**
```yaml
concurrency:
  workers: 20              # High parallelism
  validation_workers: 50

cache:
  enabled: true
  ttl: 24h                 # Daily checks

output:
  format: json             # Machine-readable for automation
  dir: /var/log/entropia

authority:
  secondary_domains:
    - docs.example.com     # Your docs site
```

**Workflow:**
```bash
# Daily cron job
0 2 * * * entropia batch /etc/entropia/docs-urls.txt --output-dir /var/log/entropia/$(date +\%Y\%m\%d)

# Alert on high dead link ratio
for report in /var/log/entropia/$(date +\%Y\%m\%d)/*.json; do
  DEAD=$(jq '[.validation[] | select(.is_accessible == false)] | length' $report)
  TOTAL=$(jq '.validation | length' $report)
  RATIO=$(echo "scale=2; $DEAD / $TOTAL" | bc)
  if (( $(echo "$RATIO > 0.1" | bc -l) )); then
    echo "ALERT: $report has $DEAD/$TOTAL dead links (${RATIO}%)"
  fi
done
```

### Use Case 5: CI/CD Pipeline - PR Validation

**Goal:** Block PRs that introduce low-quality citations

**Configuration:**
```yaml
concurrency:
  workers: 5
  validation_workers: 20

cache:
  enabled: false           # Always fresh for PR checks

output:
  format: json
```

**GitHub Action Workflow:**
```yaml
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
            entropia batch urls.txt --no-cache --output-dir ./reports
          fi

      - name: Check quality threshold
        run: |
          for report in ./reports/*.json; do
            SCORE=$(jq '.score.index' $report)
            if [ "$SCORE" -lt 60 ]; then
              echo "ERROR: $(basename $report) has support index $SCORE (minimum: 60)"
              exit 1
            fi
          done
```

---

## Advanced Configuration

### Custom Scoring Rules

(Feature planned for v0.2.0)

```yaml
scoring:
  rules_file: ~/.entropia/scoring_rules.json
```

### Domain-Specific Timeouts

```yaml
# Not yet supported - planned feature
http:
  domain_timeouts:
    slow-site.com: 60s
    fast-site.com: 10s
```

### Proxy Support

```yaml
# Not yet supported - planned feature
http:
  proxy: http://proxy.example.com:8080
```

---

## Troubleshooting

### "No configuration file found"

This is normal if you haven't created one. Entropia uses built-in defaults.

**Solution:** Create config file:
```bash
entropia config init
```

### "Invalid configuration"

**Symptoms:** YAML parsing errors

**Solutions:**
1. Validate YAML syntax: http://www.yamllint.com/
2. Check for tabs (use spaces only)
3. Regenerate default config:
```bash
rm ~/.entropia/config.yaml
entropia config init
```

### "Config file ignored"

**Symptoms:** CLI flags not overriding config

**Solution:** Check configuration hierarchy:
```bash
entropia config show
```

---

## Next Steps

- Read the [CLI Guide](CLI_GUIDE.md) for command reference
- See [README.md](../README.md) for project overview
- Check [METHODOLOGY.md](METHODOLOGY.md) for scoring logic

---

**Questions or Issues?**

- GitHub Issues: https://github.com/ppiankov/entropia/issues
- Documentation: https://github.com/ppiankov/entropia
