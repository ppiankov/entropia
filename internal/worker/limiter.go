package worker

import (
	"context"
	"net/url"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Limiter implements per-domain rate limiting
type Limiter struct {
	limiters     map[string]*rate.Limiter
	mu           sync.RWMutex
	defaultRate  rate.Limit
	defaultBurst int
}

// NewLimiter creates a new rate limiter
func NewLimiter(requestsPerSecond float64, burst int) *Limiter {
	if burst <= 0 {
		burst = 5
	}

	return &Limiter{
		limiters:     make(map[string]*rate.Limiter),
		defaultRate:  rate.Limit(requestsPerSecond),
		defaultBurst: burst,
	}
}

// Wait waits for rate limit clearance for the given URL
func (l *Limiter) Wait(ctx context.Context, rawURL string) error {
	domain, err := extractDomain(rawURL)
	if err != nil {
		return err
	}

	limiter := l.getLimiter(domain)
	return limiter.Wait(ctx)
}

// Allow checks if a request is allowed without waiting
func (l *Limiter) Allow(rawURL string) bool {
	domain, err := extractDomain(rawURL)
	if err != nil {
		return false
	}

	limiter := l.getLimiter(domain)
	return limiter.Allow()
}

// getLimiter returns the rate limiter for a domain
func (l *Limiter) getLimiter(domain string) *rate.Limiter {
	l.mu.RLock()
	limiter, exists := l.limiters[domain]
	l.mu.RUnlock()

	if exists {
		return limiter
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists := l.limiters[domain]; exists {
		return limiter
	}

	// Create new limiter for this domain
	limiter = rate.NewLimiter(l.defaultRate, l.defaultBurst)
	l.limiters[domain] = limiter

	return limiter
}

// SetDomainRate sets a custom rate limit for a specific domain
func (l *Limiter) SetDomainRate(domain string, requestsPerSecond float64, burst int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if burst <= 0 {
		burst = l.defaultBurst
	}

	l.limiters[domain] = rate.NewLimiter(rate.Limit(requestsPerSecond), burst)
}

// extractDomain extracts the domain from a URL
func extractDomain(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return parsed.Host, nil
}

// WaitWithDelay waits for rate limit and adds an additional delay
func (l *Limiter) WaitWithDelay(ctx context.Context, rawURL string, additionalDelay time.Duration) error {
	if err := l.Wait(ctx, rawURL); err != nil {
		return err
	}

	if additionalDelay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(additionalDelay):
		}
	}

	return nil
}
