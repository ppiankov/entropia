# Contributing to Entropia

Thank you for your interest in contributing to Entropia! This document provides guidelines for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Testing Guidelines](#testing-guidelines)
- [Code Style](#code-style)
- [Pull Request Process](#pull-request-process)
- [Areas Where Help is Needed](#areas-where-help-is-needed)

---

## Code of Conduct

This project follows the principle of **non-normative evaluation** - we evaluate support quality, not truth. Please keep this philosophy in mind when contributing:

- Be respectful and inclusive
- Focus on technical merit and evidence-based discussions
- Avoid making normative claims or truth assertions in code/docs
- Keep scoring formulas transparent and explainable

---

## Getting Started

### Prerequisites

- **Go 1.22 or later** ([download here](https://go.dev/dl/))
- **Git** for version control
- **Make** (optional, but recommended)

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/entropia.git
   cd entropia
   ```
3. Add upstream remote:
   ```bash
   git remote add upstream https://github.com/ppiankov/entropia.git
   ```

---

## Development Setup

### Install Dependencies

```bash
go mod download
```

### Build the Binary

```bash
make build
# or
go build -o bin/entropia ./cmd/entropia
```

### Verify Installation

```bash
./bin/entropia version
./bin/entropia scan https://en.wikipedia.org/wiki/Laksa --json test.json
```

### Run Tests

```bash
make test
# or
go test -v -race -cover ./...
```

### Lint Code

```bash
make lint
# or
golangci-lint run
```

---

## Making Changes

### Branch Naming

- **Feature**: `feature/description` (e.g., `feature/add-news-adapter`)
- **Bug fix**: `fix/description` (e.g., `fix/validation-timeout`)
- **Documentation**: `docs/description` (e.g., `docs/update-readme`)
- **Refactoring**: `refactor/description` (e.g., `refactor/scoring-engine`)

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring (no functional changes)
- `perf`: Performance improvements
- `chore`: Maintenance tasks (dependencies, build config)

**Examples:**
```bash
feat(extract): add news site adapter for NYTimes
fix(validate): handle timeout errors gracefully
docs(readme): update installation instructions
test(score): add edge cases for conflict detection
```

---

## Testing Guidelines

### Writing Tests

1. **All new features must include tests**
2. **Test file naming**: `*_test.go` in the same package
3. **Test function naming**: `Test<FunctionName>_<Scenario>`

**Example:**
```go
// internal/validate/authority_test.go
func TestAuthorityClassifier_PrimaryDomains(t *testing.T) {
    // Test primary domain classification
}

func TestAuthorityClassifier_InvalidURLs(t *testing.T) {
    // Test error handling for invalid URLs
}
```

### Test Coverage Goals

- **Core packages** (score, validate, extract): 80%+ coverage
- **CLI and utilities**: 60%+ coverage
- **Overall project**: 70%+ coverage

### Running Specific Tests

```bash
# Run tests for a specific package
go test -v ./internal/validate/...

# Run a specific test
go test -v -run TestAuthorityClassifier_PrimaryDomains ./internal/validate

# Run tests with coverage report
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration Tests

Integration tests should:
- Use mock HTTP servers (httptest.NewServer)
- Test full pipeline: fetch → extract → validate → score
- Verify JSON output matches expected schema
- Test concurrency and error handling

---

## Code Style

### Go Conventions

- **Follow standard Go style** (use `gofmt`, `golangci-lint`)
- **Descriptive names**: Prefer clarity over brevity
- **Error handling**: Always check and handle errors explicitly
- **Comments**: Use godoc format for public APIs

### Naming Conventions

**Variables:**
```go
// Good
evidenceCount := len(evidence)
primaryDomains := []string{"doi.org", "scholar.google.com"}

// Bad
ec := len(evidence)  // Too cryptic
primary_domains := []string{...}  // Use camelCase, not snake_case
```

**Functions:**
```go
// Public functions: Capitalized
func NewScorer() *Scorer { ... }
func Calculate(claims []Claim) Score { ... }

// Private functions: lowercase
func calculateCoverage(claims, evidence int) int { ... }
func validateURL(url string) error { ... }
```

**Interfaces:**
```go
// Use -er suffix when possible
type Validator interface { ... }
type Scorer interface { ... }
type Provider interface { ... }
```

### Transparency Principle

**All scoring formulas must be documented and explainable:**

```go
// Good
// calculateCoverageScore computes evidence coverage (0-40 points)
// Formula: min(evidence_count / claim_count * 40, 40)
// Rationale: Rewards 1:1 coverage, caps at 150% to avoid over-weighting
func calculateCoverageScore(claims, evidence int) int {
    if claims == 0 {
        return 0
    }
    ratio := float64(evidence) / float64(claims)
    score := int(ratio * 40)
    if score > 40 {
        score = 40
    }
    return score
}

// Bad (opaque magic numbers)
func calculateCoverageScore(claims, evidence int) int {
    return min(evidence * 40 / claims, 40)  // What is 40? Why cap here?
}
```

### Error Messages

```go
// Good: Actionable error messages
return fmt.Errorf("failed to validate evidence URL %s: %w", url, err)
return fmt.Errorf("OPENAI_API_KEY not set. Set with: export OPENAI_API_KEY=sk-...")

// Bad: Vague errors
return fmt.Errorf("error")
return err  // Without context
```

---

## Pull Request Process

### Before Submitting

1. **Update your branch** with latest upstream changes:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Run tests and linting**:
   ```bash
   make test
   make lint
   ```

3. **Update documentation** if adding features:
   - Update `README.md` if changing user-facing behavior
   - Update `docs/CLI_GUIDE.md` if adding CLI flags
   - Update `CHANGELOG.md` under "Unreleased" section

4. **Add tests** for new features or bug fixes

### PR Title and Description

**Title format:**
```
<type>(<scope>): <brief description>
```

**Description should include:**
- **What**: What does this PR change?
- **Why**: Why is this change needed?
- **How**: Brief explanation of the approach
- **Testing**: How was this tested?
- **Screenshots** (if applicable)

**Example:**
```markdown
## What
Adds NYTimes domain adapter for improved claim extraction from news articles.

## Why
Generic adapter misses citations in news articles due to different HTML structure.

## How
- Created `NewsAdapter` implementing `Adapter` interface
- Added pattern matching for `<cite>` tags
- Integrated with adapter registry

## Testing
- Added 15 unit tests for NewsAdapter
- Tested against 5 real NYTimes articles
- Coverage: 85%

## Checklist
- [x] Tests added and passing
- [x] Documentation updated
- [x] golangci-lint passes
- [x] CHANGELOG.md updated
```

### Review Process

1. **Automated checks**: CI must pass (build, test, lint)
2. **Code review**: At least one maintainer approval required
3. **Testing**: Reviewer may request additional tests
4. **Documentation**: Verify docs are clear and complete

### Merging

- **Squash commits** for small PRs (1-3 commits)
- **Rebase and merge** for larger feature branches
- **Delete branch** after merge

---

## Areas Where Help is Needed

We welcome contributions in these areas:

### 1. Domain Adapters (High Priority)
- **News sites**: NYTimes, BBC, Reuters, Guardian
- **Academic**: arXiv, JSTOR, PubMed, IEEE Xplore
- **Blogs**: Medium, Substack, personal blogs
- **Legal**: law.cornell.edu, more UK/EU legal databases

**What to do:**
- Implement `Adapter` interface in `internal/extract/adapters/`
- Add URL pattern matching
- Extract domain-specific citations and claims
- Add comprehensive tests

### 2. Enhanced Claim Extraction (Medium Priority)
- NLP integration (spaCy, Stanford CoreNLP)
- Grammatical pattern matching
- Configurable keyword sets per domain
- False positive reduction

### 3. Testing (High Priority)
- Integration tests (end-to-end pipeline)
- Golden test cases (compare against manual artifacts)
- Performance benchmarks
- Edge case coverage

### 4. Documentation (Medium Priority)
- Tutorials and guides
- Video walkthroughs
- API documentation (godoc)
- More usage examples

### 5. LLM Providers (Low Priority)
- Google Gemini integration
- Grok API support
- z.ai integration (when API stabilizes)
- Local model optimizations for Ollama

### 6. Features (Future)
- Historical drift tracking
- Web UI for report visualization
- Database backend (SQLite/PostgreSQL)
- REST API
- GitHub Action for PR validation

### 7. Performance (Low Priority)
- Memory optimization for large batch scans
- Parallel batch processing improvements
- Cache hit rate optimization

---

## Questions?

- **GitHub Issues**: Open an issue for questions or bug reports
- **Discussions**: Use GitHub Discussions for general questions
- **Documentation**: Check docs/ directory for guides

---

## License

By contributing to Entropia, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to Entropia! Together we can build a transparent, non-normative tool for evidence quality evaluation.
