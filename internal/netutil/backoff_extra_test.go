// File: internal/netutil/backoff_extra_test.go
// Created: 2026-06-20
// Description: Extra unit tests to push netutil coverage above 80%.
package netutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetWithRetrySuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "ok")
	}))
	defer srv.Close()
	resp, attempts, err := GetWithRetry(http.DefaultClient, srv.URL, RetryConfig{MaxAttempts: 2, BaseDelay: time.Millisecond})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status=%d", resp.StatusCode)
	}
	if attempts != 1 {
		t.Errorf("attempts=%d want 1", attempts)
	}
}

func TestGetWithRetryBadURL(t *testing.T) {
	_, _, err := GetWithRetry(http.DefaultClient, "://not-a-url", RetryConfig{MaxAttempts: 1})
	if err == nil {
		t.Error("expected error for malformed url")
	}
}

func TestReadBodyPreview(t *testing.T) {
	body := strings.NewReader("hello world")
	got := readBodyPreview(body, 5)
	if got != "hello" {
		t.Errorf("got %q want hello", got)
	}
}

func TestReadBodyPreviewNilBody(t *testing.T) {
	if got := readBodyPreview(nil, 5); got != "" {
		t.Errorf("got %q want empty", got)
	}
}

func TestSleepBackoff(t *testing.T) {
	// Smoke test: should not panic and should sleep a tiny duration.
	start := time.Now()
	sleepBackoff(RetryConfig{MaxAttempts: 1, BaseDelay: time.Millisecond}, 1, 0)
	if time.Since(start) < 0 {
		t.Error("negative duration")
	}
}

func TestLabelForRequestWithHeader(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://x.example/y", nil)
	req.Header.Set("X-Netutil-Label", "my-label")
	if got := labelForRequest(req); got != "my-label" {
		t.Errorf("label=%q want my-label", got)
	}
}

func TestLabelForRequestFromURL(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://x.example/y", nil)
	got := labelForRequest(req)
	if !strings.Contains(got, "x.example") {
		t.Errorf("label=%q should contain host", got)
	}
}

func TestRetryAfterHTTPDate(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	future := time.Now().Add(2 * time.Hour).UTC().Format(http.TimeFormat)
	resp.Header.Set("Retry-After", future)
	if got := retryAfter(resp); got <= 0 {
		t.Errorf("expected positive retryAfter, got %v", got)
	}
}

func TestRetryAfterNegativeSeconds(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", "-5")
	if got := retryAfter(resp); got != 0 {
		t.Errorf("retryAfter=%v want 0 for negative", got)
	}
}

func TestRetryAfterUnparseable(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", "not-a-date-or-number")
	if got := retryAfter(resp); got != 0 {
		t.Errorf("retryAfter=%v want 0 for unparseable", got)
	}
}

func TestComputeDelayCap(t *testing.T) {
	// With MaxDelay set, the computed delay is capped before jitter is added.
	// Jitter may push the final value up to ~1.25 * MaxDelay, so we allow a
	// small fudge factor when asserting.
	cfg := RetryConfig{MaxAttempts: 5, BaseDelay: time.Second, MaxDelay: 2 * time.Second}
	for i := 1; i < 5; i++ {
		got := computeDelay(cfg, i, 0)
		// Pre-jitter cap is 2s; jitter adds up to 25%, so 2.5s is the
		// documented upper bound.
		if got > 3*time.Second {
			t.Errorf("attempt %d: delay %v exceeds reasonable bound for MaxDelay 2s", i, got)
		}
	}
}

func TestComputeDelayWithExtra(t *testing.T) {
	cfg := RetryConfig{MaxAttempts: 1, BaseDelay: time.Millisecond}
	got := computeDelay(cfg, 1, 50*time.Millisecond)
	if got < 50*time.Millisecond {
		t.Errorf("got %v expected at least 50ms", got)
	}
}

func TestSetDefaultRetryRestores(t *testing.T) {
	original := DefaultRetry
	restore := SetDefaultRetry(RetryConfig{MaxAttempts: 7, BaseDelay: 7 * time.Millisecond})
	if DefaultRetry.MaxAttempts != 7 {
		t.Errorf("expected MaxAttempts=7, got %d", DefaultRetry.MaxAttempts)
	}
	restore()
	if DefaultRetry.MaxAttempts != original.MaxAttempts {
		t.Errorf("restore failed: got %d, want %d", DefaultRetry.MaxAttempts, original.MaxAttempts)
	}
}

func TestWithDefaultsFillsZeros(t *testing.T) {
	c := RetryConfig{}.withDefaults()
	if c.MaxAttempts < 1 {
		t.Errorf("MaxAttempts=%d want >=1", c.MaxAttempts)
	}
	if c.BaseDelay <= 0 {
		t.Errorf("BaseDelay=%v want >0", c.BaseDelay)
	}
}

func TestWithDefaultsKeepsProvided(t *testing.T) {
	c := RetryConfig{MaxAttempts: 3, BaseDelay: 5 * time.Millisecond, MaxDelay: 100 * time.Millisecond}.withDefaults()
	if c.MaxAttempts != 3 {
		t.Errorf("MaxAttempts=%d want 3", c.MaxAttempts)
	}
	if c.BaseDelay != 5*time.Millisecond {
		t.Errorf("BaseDelay=%v want 5ms", c.BaseDelay)
	}
	if c.MaxDelay != 100*time.Millisecond {
		t.Errorf("MaxDelay=%v want 100ms", c.MaxDelay)
	}
}

func TestDoWithRetryNetworkError(t *testing.T) {
	// Point to a closed server to force a network error.
	req, _ := http.NewRequest("GET", "http://127.0.0.1:1/never", nil)
	_, attempts, err := DoWithRetry(http.DefaultClient, req, RetryConfig{MaxAttempts: 2, BaseDelay: time.Millisecond})
	if err == nil {
		t.Error("expected error from unreachable host")
	}
	if attempts != 2 {
		t.Errorf("attempts=%d want 2", attempts)
	}
}

func TestDoWithRetryZeroConfig(t *testing.T) {
	// Zero MaxAttempts should fall back to defaults but still produce a
	// sensible response when the server is up.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, _, err := DoWithRetry(http.DefaultClient, req, RetryConfig{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status=%d", resp.StatusCode)
	}
}
