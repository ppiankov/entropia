# Work Orders — entropia

Evidence support and decay diagnostics tool. Improvement and hardening work orders.

Status key: `[ ]` planned, `[~]` in progress, `[x]` done

---

## Phase 1: Code Quality

### WO-E01: Fix lint errors (unchecked error returns)

**Status:** `[x]` done
**Priority:** high — 19 violations, blocks clean CI

### Summary

golangci-lint reports 19 errors, all unchecked error returns. Fix every one. Most are `resp.Body.Close()`, `os.Remove()`, `fmt.Fprintf()`, and `file.Close()` calls that silently drop errors.

### Scope

| File | Change |
|------|--------|
| `internal/cache/disk.go` | Handle `os.Remove()` error |
| `internal/pipeline/fetcher.go` | Handle `resp.Body.Close()` errors |
| `internal/pipeline/renderer.go` | Handle `fmt.Fprintf()` errors |
| `internal/validate/validator.go` | Handle `resp.Body.Close()` errors |
| `internal/llm/openai.go` | Handle `resp.Body.Close()` error |
| `internal/llm/anthropic.go` | Handle `resp.Body.Close()` error |
| `internal/llm/ollama.go` | Handle `resp.Body.Close()` error |
| `internal/worker/batch.go` | Handle `file.Close()` error |
| `internal/extract/adapters/wikipedia_conflicts.go` | Fix error string capitalization |

### Acceptance criteria

- [ ] `golangci-lint run` reports zero errors
- [ ] All `resp.Body.Close()` errors handled (use `defer func() { _ = resp.Body.Close() }()` pattern or check+log)
- [ ] All `os.Remove()` errors checked
- [ ] Renderer errors propagated to caller
- [ ] `make test && make lint` pass

---

### WO-E02: Use LDFLAGS for version injection

**Status:** `[x]` done
**Priority:** high — version is hardcoded, violates project standards

### Summary

Version is hardcoded as `"entropia v0.1.14"` in `internal/cli/root.go`. Should be injected at build time via LDFLAGS, consistent with all other ppiankov projects.

### Scope

| File | Change |
|------|--------|
| `internal/cli/root.go` | Add `var Version = "dev"` and `var Commit = "none"`, use in versionCmd |
| `Makefile` | Add LDFLAGS to `go build`: `-X github.com/ppiankov/entropia/internal/cli.Version=$(VERSION_NUM) -X github.com/ppiankov/entropia/internal/cli.Commit=$(COMMIT)` |

### Acceptance criteria

- [ ] `var Version` and `var Commit` declared in root.go
- [ ] `versionCmd` prints `entropia vX.Y.Z (commit)` using injected values
- [ ] Makefile passes LDFLAGS on build
- [ ] Default values (`dev`, `none`) work when building without flags
- [ ] `make build && bin/entropia version` shows injected version
- [ ] `make test && make lint` pass

---

### WO-E03: Add project CLAUDE.md

**Status:** `[x]` done
**Priority:** medium — guides future agentic work on this repo

### Summary

Create a project-level `CLAUDE.md` with entropia-specific conventions: architecture, test commands, scoring methodology, key design decisions. Follows the pattern from noisepan and other ppiankov repos.

### Scope

| File | Change |
|------|--------|
| `CLAUDE.md` | New file: project-specific instructions for AI agents |

### Acceptance criteria

- [ ] Documents build/test/lint commands
- [ ] Documents architecture (`cmd/entropia/main.go` → `internal/` packages)
- [ ] Documents key design decisions (non-normative, deterministic scoring, adapter pattern)
- [ ] Documents conventions (Go 1.25+, LDFLAGS, test patterns)
- [ ] References `docs/work-orders.md` for pending work
- [ ] Lean — no duplication of global CLAUDE.md rules

---

### WO-E04: Align Go version to 1.25

**Status:** `[x]` done
**Priority:** medium — go.mod says 1.24, CI uses 1.25, README says 1.22+

### Summary

Go version is inconsistent across the project. Align everything to 1.25 (current ppiankov standard).

### Scope

| File | Change |
|------|--------|
| `go.mod` | Change `go 1.24.0` to `go 1.25` |
| `README.md` | Update Go version reference |

### Acceptance criteria

- [ ] `go.mod` specifies `go 1.25`
- [ ] `go mod tidy` produces no diff
- [ ] README version reference updated
- [ ] `make test && make lint` pass

---

## Phase 2: Test Coverage

### WO-E05: Worker package test coverage

**Status:** `[x]` done
**Priority:** medium — 0% coverage on concurrency primitives

### Summary

`internal/worker/` contains the worker pool and batch processing logic used by the `batch` command. Currently has 0% test coverage. Add unit tests for pool creation, task distribution, and batch file parsing.

### Scope

| File | Change |
|------|--------|
| `internal/worker/pool_test.go` | New file: tests for worker pool — creation, task execution, error propagation |
| `internal/worker/batch_test.go` | New file: tests for batch file reading — valid input, empty file, malformed URLs |

### Acceptance criteria

- [ ] Worker pool tests: concurrent task execution, error handling, worker count respected
- [ ] Batch tests: file parsing, blank lines skipped, URL validation
- [ ] Coverage for worker package ≥ 80%
- [ ] No flaky tests — deterministic assertions with sync primitives
- [ ] `make test && make lint` pass

---

### WO-E06: LLM provider test coverage

**Status:** `[x]` done
**Priority:** low — 25.7% coverage, providers need mock HTTP tests

### Summary

LLM providers (OpenAI, Anthropic, Ollama) have low test coverage. Add httptest-based mock tests for each provider's API interaction.

### Scope

| File | Change |
|------|--------|
| `internal/llm/openai_test.go` | Add/extend: mock HTTP server tests for chat completion flow |
| `internal/llm/anthropic_test.go` | Add/extend: mock HTTP server tests for messages API |
| `internal/llm/ollama_test.go` | Add/extend: mock HTTP server tests for generate API |

### Acceptance criteria

- [ ] Each provider tested with httptest mock server
- [ ] Tests cover: successful response, API error, timeout, malformed response
- [ ] Coverage for llm package ≥ 60%
- [ ] `make test && make lint` pass

---

## Phase 3: Features

### WO-E07: Proxy support

**Status:** `[x]` done
**Priority:** high — required for corporate/intraweb audits
**Depends on:** WO-E01

### Summary

Add HTTP proxy support for all network operations. Entropia must route all outbound requests through configured proxies when set. This enables corporate intraweb audits where direct internet access is blocked.

### Scope

| File | Change |
|------|--------|
| `internal/pipeline/fetcher.go` | Respect HTTP_PROXY/HTTPS_PROXY/NO_PROXY env vars via custom http.Transport |
| `internal/validate/validator.go` | Propagate proxy settings to validation HTTP client |
| `internal/llm/openai.go` | Propagate proxy settings to OpenAI HTTP client |
| `internal/llm/anthropic.go` | Propagate proxy settings to Anthropic HTTP client |
| `internal/llm/ollama.go` | Propagate proxy settings to Ollama HTTP client |
| `internal/cli/scan.go` | Add `--http-proxy` and `--https-proxy` flags (optional, override env vars) |
| `docs/CONFIGURATION.md` | Document proxy configuration |

### Acceptance criteria

- [ ] Standard env vars respected: `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`
- [ ] CLI flags `--http-proxy` and `--https-proxy` override env vars when set
- [ ] All HTTP clients (fetcher, validator, LLM providers) route through proxy
- [ ] `NO_PROXY` patterns correctly bypass proxy for matching hosts
- [ ] Tests verify proxy propagation (httptest with proxy simulation)
- [ ] `make test && make lint` pass

### Notes

- Go's `http.ProxyFromEnvironment` already handles standard env vars — ensure custom transports use it
- LLM providers create their own HTTP clients — each must be updated
- Per TODO.md: "sensitive leakage can occur via reports and exports even without LLM usage"

---

### WO-E08: Homebrew packaging and release workflow update

**Status:** `[x]` done
**Priority:** high — enables `brew install ppiankov/tap/entropia`

### Summary

Update release workflow to match ppiankov standard pattern: tar.gz archives with LDFLAGS, CHANGELOG extraction, and automatic homebrew-tap formula update. Create Formula template with VERSION/SHA256 placeholders.

### Scope

| File | Change |
|------|--------|
| `.github/workflows/release.yml` | Rewrite: tar.gz packaging, LDFLAGS, homebrew-tap update via GitHub API |
| `Formula/entropia.rb` | New file: formula template with placeholders |

### Acceptance criteria

- [x] Release builds tar.gz archives (not flat binaries)
- [x] LDFLAGS inject version and commit
- [x] CHANGELOG.md parsed for release notes
- [x] Homebrew formula auto-updated in ppiankov/homebrew-tap on release
- [x] `HOMEBREW_TAP_TOKEN` secret required on the repo
- [x] 4 platforms: linux/darwin x amd64/arm64 (no Windows)

---

## Execution Order

```
WO-E01 (lint fixes) ──→ WO-E07 (proxy support)
WO-E02 (LDFLAGS) ─────→ standalone
WO-E03 (CLAUDE.md) ───→ standalone
WO-E04 (Go version) ──→ standalone
WO-E05 (worker tests) → standalone
WO-E06 (LLM tests) ───→ standalone
WO-E08 (Homebrew) ────→ done
```

E01-E04 are independent and can be parallelized. E07 depends on E01 (clean lint baseline before adding new code).
