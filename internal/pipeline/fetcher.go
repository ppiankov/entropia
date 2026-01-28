package pipeline

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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
func NewFetcher(timeout time.Duration, userAgent string, maxBytes int64, insecureTLS bool) *Fetcher {
	// Create custom transport with TLS configuration
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureTLS,
		},
	}

	return &Fetcher{
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
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

	// Capture TLS/certificate information
	meta.TLS = extractTLSInfo(resp, rawURL)

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

// extractTLSInfo extracts TLS/certificate information from the HTTP response
func extractTLSInfo(resp *http.Response, rawURL string) *model.TLSInfo {
	// Check if TLS was used
	if resp.TLS == nil {
		return &model.TLSInfo{
			Enabled: false,
		}
	}

	tlsInfo := &model.TLSInfo{
		Enabled: true,
	}

	// TLS version
	switch resp.TLS.Version {
	case tls.VersionTLS10:
		tlsInfo.Version = "TLS 1.0"
	case tls.VersionTLS11:
		tlsInfo.Version = "TLS 1.1"
	case tls.VersionTLS12:
		tlsInfo.Version = "TLS 1.2"
	case tls.VersionTLS13:
		tlsInfo.Version = "TLS 1.3"
	default:
		tlsInfo.Version = fmt.Sprintf("TLS 0x%04X", resp.TLS.Version)
	}

	// Certificate information (use leaf certificate)
	if len(resp.TLS.PeerCertificates) > 0 {
		cert := resp.TLS.PeerCertificates[0]

		tlsInfo.Subject = cert.Subject.String()
		tlsInfo.Issuer = cert.Issuer.String()
		tlsInfo.NotBefore = cert.NotBefore.Format("2006-01-02")
		tlsInfo.NotAfter = cert.NotAfter.Format("2006-01-02")
		tlsInfo.DNSNames = cert.DNSNames

		// Check if expired
		now := time.Now()
		tlsInfo.Expired = now.Before(cert.NotBefore) || now.After(cert.NotAfter)

		// Check if self-signed (issuer == subject)
		tlsInfo.SelfSigned = cert.Issuer.String() == cert.Subject.String()

		// Check domain mismatch
		parsedURL, err := url.Parse(rawURL)
		if err == nil {
			hostname := parsedURL.Hostname()
			tlsInfo.DomainMismatch = !certMatchesHostname(cert, hostname)
		}
	}

	return tlsInfo
}

// certMatchesHostname checks if the certificate is valid for the given hostname
func certMatchesHostname(cert *x509.Certificate, hostname string) bool {
	// Use the standard library's VerifyHostname method
	err := cert.VerifyHostname(hostname)
	return err == nil
}
