package validate

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/ppiankov/entropia/internal/model"
)

// Validator validates evidence links concurrently
type Validator struct {
	httpClient *http.Client
	maxWorkers int
	authority  *AuthorityClassifier
}

func newProxyFunc(httpProxy, httpsProxy, noProxy string) func(*http.Request) (*url.URL, error) {
	if httpProxy == "" && httpsProxy == "" {
		return http.ProxyFromEnvironment
	}

	return func(req *http.Request) (*url.URL, error) {
		if req.URL.Scheme == "https" && httpsProxy != "" {
			return url.Parse(httpsProxy)
		}
		if httpProxy != "" {
			return url.Parse(httpProxy)
		}
		return http.ProxyFromEnvironment(req)
	}
}

// NewValidator creates a new validator
func NewValidator(timeout time.Duration, maxWorkers int, authConfig *model.AuthorityConfig, httpProxy, httpsProxy, noProxy string) *Validator {
	if maxWorkers <= 0 {
		maxWorkers = 20
	}

	proxyFunc := newProxyFunc(httpProxy, httpsProxy, noProxy)

	return &Validator{
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				Proxy: proxyFunc,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return fmt.Errorf("stopped after 3 redirects")
				}
				return nil
			},
		},
		maxWorkers: maxWorkers,
		authority:  NewAuthorityClassifier(authConfig),
	}
}

// Validate validates all evidence links concurrently
func (v *Validator) Validate(ctx context.Context, evidence []model.Evidence) ([]model.ValidationResult, error) {
	if len(evidence) == 0 {
		return []model.ValidationResult{}, nil
	}

	results := make([]model.ValidationResult, len(evidence))
	var wg sync.WaitGroup

	// Create semaphore to limit concurrent requests
	semaphore := make(chan struct{}, v.maxWorkers)

	for i, ev := range evidence {
		wg.Add(1)
		go func(idx int, e model.Evidence) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case <-ctx.Done():
				results[idx] = model.ValidationResult{
					URL:          e.URL,
					IsAccessible: false,
					Error:        "context cancelled",
				}
				return
			case semaphore <- struct{}{}:
			}

			// Release semaphore when done
			defer func() { <-semaphore }()

			// Validate the evidence
			results[idx] = v.validateSingle(ctx, e)
		}(i, ev)
	}

	// Wait for all validations to complete
	wg.Wait()

	return results, nil
}

// validateSingle validates a single evidence link
func (v *Validator) validateSingle(ctx context.Context, evidence model.Evidence) model.ValidationResult {
	result := model.ValidationResult{
		URL:          evidence.URL,
		IsAccessible: false,
		Authority:    v.authority.Classify(evidence.URL),
	}

	// Create HEAD request
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, evidence.URL, nil)
	if err != nil {
		result.Error = fmt.Sprintf("create request: %v", err)
		result.IsDead = true
		return result
	}

	req.Header.Set("User-Agent", "Entropia/0.1 (+https://github.com/ppiankov/entropia)")

	// Execute request
	resp, err := v.httpClient.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		result.IsDead = true
		return result
	}
	defer func() { _ = resp.Body.Close() }()

	result.StatusCode = resp.StatusCode

	// Check if accessible
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.IsAccessible = true
	} else if resp.StatusCode == 404 || resp.StatusCode == 410 {
		result.IsDead = true
	}

	// Check for redirects
	if resp.Request.URL.String() != evidence.URL {
		result.RedirectURL = resp.Request.URL.String()
	}

	// Parse Last-Modified header
	if lastModified := resp.Header.Get("Last-Modified"); lastModified != "" {
		if t, err := time.Parse(time.RFC1123, lastModified); err == nil {
			result.LastModified = &t

			// Calculate age in days
			ageDays := int(time.Since(t).Hours() / 24)
			result.Age = &ageDays

			// Determine staleness
			if ageDays > 365 {
				result.IsStale = true
			}
			if ageDays > 365*3 {
				result.IsVeryStale = true
			}
		}
	}

	return result
}

// ValidateBatch is a convenience method for validating evidence with default settings
func ValidateBatch(ctx context.Context, evidence []model.Evidence, authConfig *model.AuthorityConfig) ([]model.ValidationResult, error) {
	validator := NewValidator(10*time.Second, 20, authConfig, "", "", "")
	return validator.Validate(ctx, evidence)
}
