package worker

import (
	"context"
	"testing"
	"time"
)

func TestLimiter_New(t *testing.T) {
	limiter := NewLimiter(10, 5)
	if limiter.defaultBurst != 5 {
		t.Errorf("expected burst 5, got %d", limiter.defaultBurst)
	}
	
	l2 := NewLimiter(10, -1)
	if l2.defaultBurst != 5 {
		t.Errorf("expected default burst 5 for negative input, got %d", l2.defaultBurst)
	}
}

func TestLimiter_Wait(t *testing.T) {
	limiter := NewLimiter(100, 1) // 100 rps, burst 1
	ctx := context.Background()

	url := "http://example.com/foo"
	if err := limiter.Wait(ctx, url); err != nil {
		t.Errorf("wait failed: %v", err)
	}
	
	// Different domain should also work
	if err := limiter.Wait(ctx, "http://google.com"); err != nil {
		t.Errorf("wait failed: %v", err)
	}
}

func TestLimiter_WaitWithDelay(t *testing.T) {
	limiter := NewLimiter(100, 1)
	ctx := context.Background()
	
	start := time.Now()
	err := limiter.WaitWithDelay(ctx, "http://example.com", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("WaitWithDelay failed: %v", err)
	}
	
	duration := time.Since(start)
	if duration < 50*time.Millisecond {
		t.Errorf("expected delay >= 50ms, got %v", duration)
	}
}

func TestLimiter_RateLimit(t *testing.T) {
	// 1 rps, burst 1
	limiter := NewLimiter(1, 1)
	ctx := context.Background()
	url := "http://example.com"
	
	// First request ok
	if err := limiter.Wait(ctx, url); err != nil {
		t.Errorf("first wait failed: %v", err)
	}
	
	// Second request should block or be allowed?
	// Wait() blocks. Allow() returns false immediately.
	// Since we used burst 1, token is consumed.
	
	if limiter.Allow(url) {
		t.Errorf("expected allow to fail (exhausted tokens)")
	}
	
	// Different domain should be allowed
	if !limiter.Allow("http://other.com") {
		t.Errorf("expected allow for other domain")
	}
}

func TestLimiter_SetDomainRate(t *testing.T) {
	limiter := NewLimiter(10, 10) // fast default
	domain := "slow.com"
	
	// Set strict limit for specific domain
	limiter.SetDomainRate(domain, 0.1, 1) // very slow
	
	// First request passes (burst 1)
	if !limiter.Allow("http://" + domain) {
		t.Errorf("first request should pass")
	}
	
	// Second request fails
	if limiter.Allow("http://" + domain) {
		t.Errorf("second request should fail")
	}
	
	// Other domain still fast
	if !limiter.Allow("http://fast.com") {
		t.Errorf("other domain should pass")
	}
}

func TestExtractDomain(t *testing.T) {
	domain, err := extractDomain("http://example.com/foo")
	if err != nil {
		t.Fatalf("extractDomain failed: %v", err)
	}
	if domain != "example.com" {
		t.Errorf("expected example.com, got %s", domain)
	}
	
	_, err = extractDomain("::invalid")
	if err == nil {
		t.Errorf("expected error for invalid URL")
	}
}
