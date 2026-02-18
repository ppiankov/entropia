package util

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/temoto/robotstxt"
)

// RobotsChecker checks robots.txt compliance
type RobotsChecker struct {
	cache      map[string]*robotstxt.RobotsData
	mu         sync.RWMutex
	httpClient *http.Client
	userAgent  string
}

// NewRobotsChecker creates a new robots.txt checker
func NewRobotsChecker(userAgent string, timeout time.Duration) *RobotsChecker {
	return &RobotsChecker{
		cache: make(map[string]*robotstxt.RobotsData),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		userAgent: userAgent,
	}
}

// CanFetch checks if the URL can be fetched according to robots.txt
// Returns (allowed, crawlDelay, error)
func (r *RobotsChecker) CanFetch(ctx context.Context, rawURL string) (bool, time.Duration, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false, 0, fmt.Errorf("parse URL: %w", err)
	}

	robotsURL := fmt.Sprintf("%s://%s/robots.txt", parsed.Scheme, parsed.Host)

	// Get or fetch robots.txt data
	data, err := r.getRobotsData(ctx, parsed.Host, robotsURL)
	if err != nil {
		// If we can't fetch robots.txt, allow by default but warn
		return true, 0, nil
	}

	// Check if path is allowed
	allowed := data.TestAgent(parsed.Path, r.userAgent)

	// Get crawl delay
	crawlDelay := time.Duration(0)
	if group := data.FindGroup(r.userAgent); group != nil {
		crawlDelay = group.CrawlDelay
	}

	return allowed, crawlDelay, nil
}

// getRobotsData fetches and caches robots.txt data
func (r *RobotsChecker) getRobotsData(ctx context.Context, host string, robotsURL string) (*robotstxt.RobotsData, error) {
	// Check cache first
	r.mu.RLock()
	data, exists := r.cache[host]
	r.mu.RUnlock()

	if exists {
		return data, nil
	}

	// Fetch robots.txt
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", r.userAgent)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch robots.txt: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// If robots.txt doesn't exist, allow everything
	if resp.StatusCode == 404 {
		data, _ := robotstxt.FromStatusAndBytes(404, nil)
		r.cacheData(host, data)
		return data, nil
	}

	// Parse robots.txt
	data, err2 := robotstxt.FromResponse(resp)
	if err2 != nil {
		return nil, fmt.Errorf("parse robots.txt: %w", err2)
	}

	// Cache the data
	r.cacheData(host, data)

	return data, nil
}

// cacheData caches robots.txt data for a host
func (r *RobotsChecker) cacheData(host string, data *robotstxt.RobotsData) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache[host] = data
}

// Clear clears the robots.txt cache
func (r *RobotsChecker) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache = make(map[string]*robotstxt.RobotsData)
}

// GetCrawlDelay returns the crawl delay for a URL
func (r *RobotsChecker) GetCrawlDelay(ctx context.Context, rawURL string) (time.Duration, error) {
	_, crawlDelay, err := r.CanFetch(ctx, rawURL)
	return crawlDelay, err
}

// IsAllowed is a convenience method that returns only the allowed status
func (r *RobotsChecker) IsAllowed(ctx context.Context, rawURL string) bool {
	allowed, _, _ := r.CanFetch(ctx, rawURL)
	return allowed
}

// NormalizeUserAgent normalizes the user agent string for robots.txt matching
func NormalizeUserAgent(ua string) string {
	// Extract the product name (first token)
	parts := strings.Fields(ua)
	if len(parts) > 0 {
		// Remove version if present
		product := strings.Split(parts[0], "/")[0]
		return product
	}
	return ua
}
