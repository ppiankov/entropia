# Entropia CLI Guide

Complete reference for using Entropia from the command line.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Commands](#commands)
  - [scan](#scan)
  - [batch](#batch)
  - [config](#config)
- [Global Flags](#global-flags)
- [Examples](#examples)
- [Environment Variables](#environment-variables)

---

## Installation

### From Binary

Download the latest release from GitHub:

```bash
# macOS/Linux
curl -L https://github.com/ppiankov/entropia/releases/latest/download/entropia-$(uname -s)-$(uname -m) -o entropia
chmod +x entropia
sudo mv entropia /usr/local/bin/
```

### From Source

```bash
git clone https://github.com/ppiankov/entropia.git
cd entropia
make build
sudo cp bin/entropia /usr/local/bin/
```

### Verify Installation

```bash
entropia --version
```

---

## Quick Start

### Scan a Single URL

```bash
entropia scan https://en.wikipedia.org/wiki/Laksa
```

This generates:
- `report.json` - Machine-readable report with full data
- Console output - Human-readable summary

### Scan Multiple URLs

Create a file `urls.txt`:
```
https://en.wikipedia.org/wiki/Laksa
https://en.wikipedia.org/wiki/Common-law_marriage
https://en.wikipedia.org/wiki/List_of_common_misconceptions
```

Run batch scan:
```bash
entropia batch urls.txt --concurrency 5
```

### Generate LLM Summary

```bash
export OPENAI_API_KEY=sk-...
entropia scan https://example.com --llm --llm-provider openai --llm-model gpt-4o-mini
```

---

## Commands

### `scan`

Scan a single URL and generate an evidence quality report.

**Usage:**
```bash
entropia scan <url> [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--json` | string | `report.json` | Output JSON path |
| `--md` | string | `""` | Output Markdown path (optional) |
| `--timeout` | duration | `30s` | HTTP fetch timeout |
| `--ua` | string | `"Entropia/0.1 ..."` | HTTP User-Agent |
| `--max-bytes` | int | `2000000` | Max response size (2MB) |
| `--no-cache` | bool | `false` | Disable cache (force fresh fetch) |
| `--llm` | bool | `false` | Enable LLM summary generation |
| `--llm-provider` | string | `"openai"` | LLM provider (openai, anthropic, ollama) |
| `--llm-model` | string | `"gpt-4o-mini"` | LLM model name |

**Examples:**

```bash
# Basic scan
entropia scan https://example.com

# Save as JSON and Markdown
entropia scan https://example.com --json output.json --md output.md

# With LLM summary (OpenAI)
export OPENAI_API_KEY=sk-...
entropia scan https://example.com --llm --llm-provider openai

# With LLM summary (Anthropic Claude)
export ANTHROPIC_API_KEY=sk-ant-...
entropia scan https://example.com --llm --llm-provider anthropic --llm-model claude-3-5-haiku-20241022

# With LLM summary (Ollama local)
entropia scan https://example.com --llm --llm-provider ollama --llm-model llama3.1:8b

# Force fresh fetch (bypass cache)
entropia scan https://example.com --no-cache

# Verbose output
entropia scan https://example.com -v
```

---

### `batch`

Scan multiple URLs from a file in parallel.

**Usage:**
```bash
entropia batch <file> [flags]
```

**Input File Format:**
- One URL per line
- Empty lines and lines starting with `#` are ignored

Example `urls.txt`:
```
# Wikipedia articles
https://en.wikipedia.org/wiki/Laksa
https://en.wikipedia.org/wiki/Common-law_marriage

# Legal documents
https://www.legislation.gov.uk/ukpga/1998/42/contents
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--concurrency` | int | `NumCPU()` | Number of concurrent workers |
| `--output-dir` | string | `./entropia-reports` | Output directory for reports |
| `--timeout` | duration | `10m` | Total batch timeout |
| `--scan-timeout` | duration | `30s` | Timeout for individual scans |
| `--ua` | string | `"Entropia/0.1 ..."` | HTTP User-Agent |
| `--no-cache` | bool | `false` | Disable cache |
| `--llm` | bool | `false` | Enable LLM summaries |
| `--llm-provider` | string | `"openai"` | LLM provider |
| `--llm-model` | string | `"gpt-4o-mini"` | LLM model |

**Examples:**

```bash
# Basic batch scan (uses all CPU cores)
entropia batch urls.txt

# Limit to 5 concurrent workers
entropia batch urls.txt --concurrency 5

# Custom output directory
entropia batch urls.txt --output-dir ./my-reports

# With LLM summaries
export OPENAI_API_KEY=sk-...
entropia batch urls.txt --llm --concurrency 3

# Verbose mode
entropia batch urls.txt -v
```

**Output:**
- Each URL generates: `<slug>.json` and `<slug>.md` in the output directory
- Console shows progress and summary statistics

---

### `config`

Manage Entropia configuration.

#### `config show`

Display current configuration including all sources (defaults, config file, environment variables, flags).

**Usage:**
```bash
entropia config show
```

**Output:**
- Full YAML configuration
- Config file path (if loaded)
- Configuration hierarchy explanation

#### `config init`

Create a default configuration file at `~/.entropia/config.yaml`.

**Usage:**
```bash
entropia config init
```

**Output:**
- Creates `~/.entropia/config.yaml` with all available options documented
- Fails if config already exists (delete first to recreate)

---

## Global Flags

These flags work with all commands:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-v, --verbose` | bool | `false` | Verbose output with debug information |
| `--config` | string | `~/.entropia/config.yaml` | Path to config file |

**Examples:**

```bash
# Verbose output
entropia scan https://example.com -v

# Use custom config file
entropia scan https://example.com --config ./my-config.yaml
```

---

## Examples

### Example 1: Audit Wikipedia Article Quality

```bash
# Scan article
entropia scan https://en.wikipedia.org/wiki/Laksa --json laksa.json --md laksa.md

# View report
cat laksa.md

# Check support index
jq '.score.index' laksa.json
```

### Example 2: Batch Scan Legal Documents

Create `legal-urls.txt`:
```
https://www.legislation.gov.uk/ukpga/1998/42/contents
https://www.law.cornell.edu/uscode/text/17/107
```

Run batch:
```bash
entropia batch legal-urls.txt --concurrency 2 --output-dir ./legal-reports
```

### Example 3: Compare Evidence Quality Over Time

```bash
# Initial scan
entropia scan https://example.com --json baseline.json

# Wait some time...

# Rescan (bypass cache)
entropia scan https://example.com --json current.json --no-cache

# Compare support indexes
echo "Baseline: $(jq '.score.index' baseline.json)"
echo "Current: $(jq '.score.index' current.json)"
```

### Example 4: Use LLM Summaries for Reports

```bash
# Configure API key
export OPENAI_API_KEY=sk-your-key-here

# Scan with GPT-4o-mini
entropia scan https://example.com --llm --llm-model gpt-4o-mini --md report.md

# View LLM summary (separate file)
cat report.llm.md
```

### Example 5: High-Concurrency Batch Processing

```bash
# Process 100 URLs with 20 workers
entropia batch large-batch.txt --concurrency 20 --timeout 30m
```

---

## Environment Variables

### LLM Provider API Keys

| Variable | Description |
|----------|-------------|
| `OPENAI_API_KEY` | OpenAI API key for GPT models |
| `ANTHROPIC_API_KEY` | Anthropic API key for Claude models |
| `OLLAMA_BASE_URL` | Ollama API base URL (default: `http://localhost:11434`) |

**Usage:**

```bash
# OpenAI
export OPENAI_API_KEY=sk-your-openai-key
entropia scan https://example.com --llm --llm-provider openai

# Anthropic Claude
export ANTHROPIC_API_KEY=sk-ant-your-anthropic-key
entropia scan https://example.com --llm --llm-provider anthropic

# Ollama (local)
export OLLAMA_BASE_URL=http://localhost:11434
entropia scan https://example.com --llm --llm-provider ollama --llm-model llama3.1:8b
```

### Configuration Override

| Variable | Description |
|----------|-------------|
| `ENTROPIA_CONFIG` | Path to configuration file (overrides default) |

---

## Tips & Best Practices

### 1. Use Batch Mode for Multiple URLs

Batch mode is much faster than running individual scans:

```bash
# Slow (sequential)
for url in $(cat urls.txt); do entropia scan $url; done

# Fast (parallel)
entropia batch urls.txt --concurrency 10
```

### 2. Cache Management

Cache speeds up repeated scans but can hide changes. Use `--no-cache` to force fresh data:

```bash
# Use cache (fast, may be stale)
entropia scan https://example.com

# Bypass cache (slow, always fresh)
entropia scan https://example.com --no-cache
```

Cache location: `~/.entropia/cache/` (24-hour TTL by default)

### 3. LLM Cost Management

LLM summaries cost money (API usage). Recommendations:

- Use `gpt-4o-mini` (cheaper) instead of `gpt-4o` for most cases
- Use Ollama (free, local) for privacy-sensitive scans
- Only enable `--llm` when you need human-readable summaries

### 4. Monitoring Batch Progress

Use verbose mode to see detailed progress:

```bash
entropia batch urls.txt -v --concurrency 5
```

### 5. Handling Large Batches

For very large batches (1000+ URLs):

```bash
# Split into smaller batches
split -l 100 large-batch.txt batch-part-

# Process each part
for part in batch-part-*; do
  entropia batch $part --concurrency 10 --output-dir ./reports
done
```

---

## Troubleshooting

### "OPENAI_API_KEY environment variable not set"

**Solution**: Export your API key before running:
```bash
export OPENAI_API_KEY=sk-your-key-here
```

### "config file already exists"

**Solution**: Delete existing config or use a different path:
```bash
rm ~/.entropia/config.yaml
entropia config init
```

### "context deadline exceeded"

**Solution**: Increase timeout:
```bash
entropia scan https://slow-site.com --timeout 60s
```

### "too many open files"

**Solution**: Reduce concurrency or increase system limits:
```bash
# Reduce workers
entropia batch urls.txt --concurrency 5

# Or increase OS limit (macOS/Linux)
ulimit -n 4096
```

### "unexpected status: 403 403 Forbidden"

**Cause**: Some sites block bot user agents.

**Solution**: Use a browser user agent:
```bash
entropia scan https://example.com --ua "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
```

### "robots.txt disallows fetching"

**Solution**: Respect robots.txt (tool won't bypass it). Check if you have permission to scan the site.

---

## Next Steps

- Read the [Configuration Guide](CONFIGURATION.md) for advanced settings
- See [README.md](../README.md) for project overview and principles
- Check [METHODOLOGY.md](METHODOLOGY.md) for scoring details

---

**Questions or Issues?**

- GitHub Issues: https://github.com/ppiankov/entropia/issues
- Documentation: https://github.com/ppiankov/entropia
