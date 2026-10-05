package middleware

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRateLimitTransport_NilLimiter(t *testing.T) {
	t.Parallel()
	rt := NewRateLimitedTransport(http.DefaultTransport, 0)
	require.NotNil(t, rt)

	// Verify it doesn't panic on RoundTrip with no limiter.
	// We don't actually make a request — just check construction.
}

func TestRateLimitTransport_WithLimiter(t *testing.T) {
	t.Parallel()
	rt := NewRateLimitedTransport(http.DefaultTransport, 100*time.Millisecond)
	require.NotNil(t, rt)
}

func TestBackoffDuration(t *testing.T) {
	t.Parallel()
	d0 := backoffDuration(0)
	d1 := backoffDuration(1)
	d2 := backoffDuration(2)
	dMax := backoffDuration(100)

	// With ±50% jitter, values may not be strictly monotonic.
	// Verify they're all positive and within expected range.
	require.Greater(t, d0, time.Duration(0))
	require.Greater(t, d1, time.Duration(0))
	require.Greater(t, d2, time.Duration(0))
	require.LessOrEqual(t, dMax, 45*time.Second)
}

func TestParseRetryAfter_Seconds(t *testing.T) {
	t.Parallel()
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", "30")
	d := parseRetryAfter(resp)
	require.Equal(t, 30*time.Second, d)
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	t.Parallel()
	future := time.Now().Add(5 * time.Second).UTC().Format(time.RFC1123)
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", future)
	d := parseRetryAfter(resp)
	require.Greater(t, d, time.Duration(0))
	require.LessOrEqual(t, d, 6*time.Second)
}

func TestParseRetryAfter_Empty(t *testing.T) {
	t.Parallel()
	resp := &http.Response{Header: http.Header{}}
	require.Equal(t, time.Duration(0), parseRetryAfter(resp))
}

func TestParseRetryAfter_Nil(t *testing.T) {
	t.Parallel()
	require.Equal(t, time.Duration(0), parseRetryAfter(nil))
}

func TestWithRateLimit_NoEnv(t *testing.T) {
	t.Parallel()
	client := &http.Client{}
	result := WithRateLimit(client)
	require.NotNil(t, result)
	// No min interval set — transport should be unchanged.
}

func TestWithRateLimit_CustomMinInterval(t *testing.T) {
	t.Setenv("XURRENT_HTTP_MIN_INTERVAL", "0.5")
	client := &http.Client{}
	result := WithRateLimit(client)
	require.NotNil(t, result)
	_, ok := result.Transport.(*RateLimitedTransport)
	require.True(t, ok, "transport should be RateLimitedTransport")

	// Reset env so other tests aren't affected.
	t.Setenv("XURRENT_HTTP_MIN_INTERVAL", "")
}

func TestCache_SetGet(t *testing.T) {
	t.Parallel()
	c := NewCache(10 * time.Second)
	defer c.Stop()

	c.Set("key1", []byte("value1"), 5*time.Second)
	v, ok := c.Get("key1")
	require.True(t, ok)
	require.Equal(t, "value1", string(v))

	_, ok = c.Get("nonexistent")
	require.False(t, ok)
}

func TestCache_TTLExpiry(t *testing.T) {
	t.Parallel()
	c := NewCache(10 * time.Second)
	defer c.Stop()

	c.Set("key1", []byte("value1"), 1*time.Millisecond)
	time.Sleep(10 * time.Millisecond)
	_, ok := c.Get("key1")
	require.False(t, ok)
}

func TestCache_SetNil(t *testing.T) {
	t.Parallel()
	c := NewCache(10 * time.Second)
	defer c.Stop()

	c.Set("k", nil, 1*time.Millisecond)
	time.Sleep(10 * time.Millisecond)
	_, ok := c.Get("k")
	require.False(t, ok)
}

func TestCache_Delete(t *testing.T) {
	t.Parallel()
	c := NewCache(10 * time.Second)
	defer c.Stop()

	c.Set("key1", []byte("value1"), time.Minute)
	require.Equal(t, 1, c.Size())
	c.Delete("key1")
	require.Equal(t, 0, c.Size())
}

func TestCache_DeletePrefix(t *testing.T) {
	t.Parallel()
	c := NewCache(10 * time.Second)
	defer c.Stop()

	c.Set("acct1:query:teams:1", []byte("a"), time.Minute)
	c.Set("acct1:query:sites:1", []byte("b"), time.Minute)
	c.Set("acct2:query:teams:1", []byte("c"), time.Minute)

	c.DeletePrefix("acct1:")
	require.Equal(t, 1, c.Size())
	_, ok := c.Get("acct2:query:teams:1")
	require.True(t, ok)
}

func TestCache_InvalidateAll(t *testing.T) {
	t.Parallel()
	c := NewCache(10 * time.Second)
	defer c.Stop()

	c.Set("a", []byte("1"), time.Minute)
	c.Set("b", []byte("2"), time.Minute)
	c.InvalidateAll()
	require.Equal(t, 0, c.Size())
}

func TestCache_SetJSON(t *testing.T) {
	t.Parallel()
	c := NewCache(10 * time.Second)
	defer c.Stop()

	err := c.SetJSON("key", map[string]any{"name": "test"}, time.Minute)
	require.NoError(t, err)

	v, ok := c.Get("key")
	require.True(t, ok)
	require.Contains(t, string(v), `"name"`)
}

func TestCache_Stop(t *testing.T) {
	t.Parallel()
	c := NewCache(10 * time.Second)
	c.Stop()
	// Double-stop should not panic.
	c.Stop()
}

func TestCache_GetReturnsCopy(t *testing.T) {
	t.Parallel()
	c := NewCache(10 * time.Second)
	defer c.Stop()

	original := []byte("original")
	c.Set("key", original, time.Minute)

	got, ok := c.Get("key")
	require.True(t, ok)
	require.Equal(t, original, got)

	// Mutate the returned slice, verify cache is not mutated.
	got[0] = 'X'
	got2, _ := c.Get("key")
	require.Equal(t, original, got2)
}

func TestResponseShaper_JSONCompact(t *testing.T) {
	t.Parallel()
	rs := NewResponseShaper()
	rs.Compact = true
	s, truncated, err := rs.ShapeJSON(map[string]any{"name": "test", "id": 42})
	require.NoError(t, err)
	require.False(t, truncated)
	require.Contains(t, s, `"name"`)
	require.Contains(t, s, `42`)
}

func TestResponseShaper_JSONTruncation(t *testing.T) {
	t.Parallel()
	rs := NewResponseShaper()
	rs.MaxChars = 20

	// Create a >20 char JSON string.
	data := map[string]any{"very_long_key_name": "a very long string value"}
	s, truncated, err := rs.ShapeJSON(data)
	require.NoError(t, err)
	require.True(t, truncated)
	require.Contains(t, s, "truncated")
	require.LessOrEqual(t, len(s), 20+50) // room for truncation message
}

func TestResponseShaper_TextTruncation(t *testing.T) {
	t.Parallel()
	rs := NewResponseShaper()
	rs.MaxChars = 5

	s, truncated := rs.TruncateText("hello world")
	require.True(t, truncated)
	require.Contains(t, s, "hello")
	require.Contains(t, s, "truncated")
}

func TestResponseShaper_TextNoTruncation(t *testing.T) {
	t.Parallel()
	rs := NewResponseShaper()
	rs.MaxChars = 100

	s, truncated := rs.TruncateText("short")
	require.False(t, truncated)
	require.Equal(t, "short", s)
}

func TestResponseShaper_CSV(t *testing.T) {
	t.Parallel()
	rs := NewResponseShaper()

	rows := []map[string]any{
		{"id": float64(1), "name": "Alice"},
		{"id": float64(2), "name": "Bob"},
	}
	cols := []string{"id", "name"}

	s, truncated, err := rs.ShapeCSV(rows, cols)
	require.NoError(t, err)
	require.False(t, truncated)
	require.Contains(t, s, "Alice")
	require.Contains(t, s, "Bob")
	require.Contains(t, s, "id,name")
}

func TestResponseShaper_CSVEmpty(t *testing.T) {
	t.Parallel()
	rs := NewResponseShaper()
	s, truncated, err := rs.ShapeCSV(nil, nil)
	require.NoError(t, err)
	require.False(t, truncated)
	require.Empty(t, s)
}

func TestResponseShaper_CSVTruncation(t *testing.T) {
	t.Parallel()
	rs := NewResponseShaper()
	rs.MaxChars = 5

	rows := []map[string]any{
		{"id": float64(1), "name": "Alice"},
	}
	s, truncated, err := rs.ShapeCSV(rows, []string{"id", "name"})
	require.NoError(t, err)
	require.True(t, truncated)
	require.Contains(t, s, "truncated")
}

func TestResponseShaper_FromEnv(t *testing.T) {
	t.Setenv("XURRENT_TOOL_MAX_RESPONSE_CHARS", "50000")
	t.Setenv("XURRENT_TOOL_JSON_COMPACT", "0")
	rs := NewResponseShaper()
	require.Equal(t, 50000, rs.MaxChars)
	require.False(t, rs.Compact)
}

func TestLogger_Default(t *testing.T) {
	logger := Logger()
	require.NotNil(t, logger)
}

func TestLogger_Debug(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	logger := Logger()
	require.NotNil(t, logger)
}

func TestLogger_Text(t *testing.T) {
	t.Setenv("LOG_FORMAT", "text")
	logger := Logger()
	require.NotNil(t, logger)
}
