package pipeline

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ppiankov/entropia/internal/model"
)

// Fetcher fetches HTML content from URLs
type Fetcher struct {
	httpClient *http.Client
	userAgent  string
	maxBytes   int64
}

// NewFetcher creates a new Fetcher with the given configuration
func NewFetcher(timeout time.Duration, userAgent string, maxBytes int64) *Fetcher {
	return &Fetcher{
		httpClient: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return fmt.Errorf("stopped after 3 redirects")
				}
				return nil
			},
		},
		userAgent: userAgent,
		maxBytes:  maxBytes,
	}
}

// FetchResult contains the fetched HTML and metadata
type FetchResult struct {
	HTML     string
	Meta     model.FetchMeta
	Subject  string
	FinalURL string
}

// Fetch retrieves HTML content from the given URL
func (f *Fetcher) Fetch(ctx context.Context, rawURL string) (*FetchResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	meta := model.FetchMeta{
		StatusCode:   resp.StatusCode,
		ContentType:  resp.Header.Get("Content-Type"),
		LastModified: resp.Header.Get("Last-Modified"),
		ETag:         resp.Header.Get("ETag"),
		Headers:      make(map[string]string),
	}

	// Store selected headers
	for _, key := range []string{"Content-Length", "Server", "Cache-Control"} {
		if val := resp.Header.Get(key); val != "" {
			meta.Headers[key] = val
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status: %d %s", resp.StatusCode, resp.Status)
	}

	// Read body with size limit
	limitedReader := io.LimitReader(resp.Body, f.maxBytes)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	finalURL := resp.Request.URL.String()
	subject := extractSubject(finalURL)

	return &FetchResult{
		HTML:     string(body),
		Meta:     meta,
		Subject:  subject,
		FinalURL: finalURL,
	}, nil
}

// extractSubject extracts a human-readable subject from the URL
func extractSubject(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	path := strings.Trim(parsed.Path, "/")
	if path == "" {
		return parsed.Host
	}

	// Extract last path segment
	segments := strings.Split(path, "/")
	last := segments[len(segments)-1]

	// De-slugify: replace underscores and hyphens with spaces
	last = strings.ReplaceAll(last, "_", " ")
	last = strings.ReplaceAll(last, "-", " ")

	// Remove file extensions
	if idx := strings.LastIndex(last, "."); idx > 0 {
		last = last[:idx]
	}

	return last
}
