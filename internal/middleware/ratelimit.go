package middleware

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitedTransport wraps an http.RoundTripper with:
//   - Minimum interval between requests (XURRENT_HTTP_MIN_INTERVAL, e.g. "0.35" seconds)
//   - 429 Too Many Requests handling with Retry-After header
//   - Exponential backoff with jitter for transient errors
type RateLimitedTransport struct {
	base       http.RoundTripper
	minInterval time.Duration
	mu         sync.Mutex
	limiter    *rate.Limiter
}

// NewRateLimitedTransport creates a rate-limited transport.
// minInterval is the minimum time between consecutive requests.
// If minInterval is 0 (or negative), no rate limiting is applied.
func NewRateLimitedTransport(base http.RoundTripper, minInterval time.Duration) *RateLimitedTransport {
	rt := &RateLimitedTransport{
		base:       base,
		minInterval: minInterval,
	}
	if minInterval > 0 {
		rt.limiter = rate.NewLimiter(rate.Every(minInterval), 1)
	}
	return rt
}

// RoundTrip implements http.RoundTripper.
func (t *RateLimitedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	maxRetries := 5
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Wait for rate limiter slot.
		if t.limiter != nil {
			if err := t.limiter.Wait(req.Context()); err != nil {
				return nil, fmt.Errorf("rate limit context: %w", err)
			}
		}

		resp, err := t.base.RoundTrip(req)
		if err != nil {
			// Network error — backoff and retry.
			if isRetryable(err) && attempt < maxRetries-1 {
				lastErr = err
				backoff := backoffDuration(attempt)
				select {
				case <-time.After(backoff):
					continue
				case <-req.Context().Done():
					return nil, req.Context().Err()
				}
			}
			return resp, err
		}

		// Handle 429 Too Many Requests.
		if resp.StatusCode == http.StatusTooManyRequests {
			_ = resp.Body.Close()
			retryAfter := parseRetryAfter(resp)
			if retryAfter == 0 {
				retryAfter = backoffDuration(attempt)
			}
			lastErr = fmt.Errorf("rate limited (429), retry after %v", retryAfter)
			select {
			case <-time.After(retryAfter):
				continue
			case <-req.Context().Done():
				return nil, req.Context().Err()
			}
		}

		// Handle 5xx transient errors with Retry-After.
		if resp.StatusCode >= 500 && resp.StatusCode < 600 && attempt < maxRetries-1 {
			_ = resp.Body.Close()
			retryAfter := parseRetryAfter(resp)
			if retryAfter == 0 {
				retryAfter = backoffDuration(attempt)
			}
			lastErr = fmt.Errorf("server error (HTTP %d), retrying in %v", resp.StatusCode, retryAfter)
			select {
			case <-time.After(retryAfter):
				continue
			case <-req.Context().Done():
				return nil, req.Context().Err()
			}
		}

		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	// Context cancellation and deadline are not retryable.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

// backoffDuration returns an exponential backoff with jitter.
func backoffDuration(attempt int) time.Duration {
	// Compute base with float to detect overflow before converting.
	p := math.Pow(2, float64(attempt))
	if p > float64(30*time.Second)/(100*float64(time.Millisecond)) || p < 0 {
		p = float64(30*time.Second) / (100 * float64(time.Millisecond))
	}
	base := time.Duration(p) * 100 * time.Millisecond
	if base <= 0 {
		base = 30 * time.Second
	}
	// Add jitter: ±50%.
	jitter := time.Duration(rand.Int64N(max(1, int64(base)/2)))
	if rand.IntN(2) == 0 {
		return base + jitter
	}
	return base - jitter
}

// parseRetryAfter parses the Retry-After header (seconds or HTTP-date).
func parseRetryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}
	ra := resp.Header.Get("Retry-After")
	if ra == "" {
		return 0
	}
	if sec, err := strconv.Atoi(ra); err == nil && sec > 0 {
		return time.Duration(sec) * time.Second
	}
	if t, err := time.Parse(time.RFC1123, ra); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

// WithRateLimit adds rate-limited transport to an http.Client.
// Reads XURRENT_HTTP_MIN_INTERVAL from environment.
func WithRateLimit(client *http.Client) *http.Client {
	d := httpMinIntervalFromEnv()
	if d <= 0 {
		return client
	}
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	client.Transport = NewRateLimitedTransport(base, d)
	return client
}

func httpMinIntervalFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv("XURRENT_HTTP_MIN_INTERVAL"))
	if raw == "" {
		return 0
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil || f <= 0 {
		return 0
	}
	return time.Duration(f * float64(time.Second))
}
