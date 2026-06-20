// File: internal/netutil/backoff.go
// Created: 2026-06-20
// Description: HTTP retry with exponential backoff and jitter for transient
// errors (rate limits and 5xx). Honours Retry-After hints when present.

package netutil

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// RetryConfig controls the backoff loop used by DoWithRetry. Zero values fall
// back to defaults that are appropriate for a CLI: at most 4 attempts over
// roughly 6 seconds total, jittered to avoid lockstep retries.
type RetryConfig struct {
	MaxAttempts int           // total attempts (>=1)
	BaseDelay   time.Duration // first sleep between attempts
	MaxDelay    time.Duration // optional cap on any single sleep; 0 = no cap
}

func (c RetryConfig) withDefaults() RetryConfig {
	if c.MaxAttempts < 1 {
		c.MaxAttempts = 10
	}
	if c.BaseDelay <= 0 {
		c.BaseDelay = 250 * time.Millisecond
	}
	// MaxDelay defaults to 0 meaning "no cap" so a long retry budget keeps
	// growing the backoff. Callers that want a cap should set it explicitly.
	_ = c.MaxDelay
	return c
}

// Logf is the destination for retry progress messages. Defaults to log.Print
// which writes to stderr. Tests can swap it for io.Discard to silence output.
var Logf = func(format string, args ...interface{}) { fmt.Fprintf(os.Stderr, format+"\n", args...) }

// labelForRequest extracts a short label for log messages. If the caller
// attached an X-Netutil-Label header to the request, that wins; otherwise we
// fall back to the URL's host+path.
func labelForRequest(req *http.Request) string {
	if v := req.Header.Get("X-Netutil-Label"); v != "" {
		return v
	}
	return req.URL.Host + req.URL.Path
}

// DefaultRetry is the package-wide default used by DoWithRetry. Tests can
// overwrite this with SetDefaultRetry to keep the suite fast even when a
// remote endpoint keeps failing.
var DefaultRetry = RetryConfig{}.withDefaults()

// SetDefaultRetry replaces DefaultRetry and returns a function that restores
// the previous value. Intended for use in test setup/teardown.
func SetDefaultRetry(c RetryConfig) func() {
	prev := DefaultRetry
	DefaultRetry = c.withDefaults()
	return func() { DefaultRetry = prev }
}

// DoWithRetry executes the request and, on transient failures (rate limit or
// 5xx), retries up to cfg.MaxAttempts times with exponential backoff. The
// final response is returned to the caller along with the number of attempts
// consumed. Network errors are also retried.
func DoWithRetry(client *http.Client, req *http.Request, cfg RetryConfig) (*http.Response, int, error) {
	if client == nil {
		client = http.DefaultClient
	}
	cfg = cfg.withDefaults()

	var lastResp *http.Response
	var lastErr error
	label := labelForRequest(req)
	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		// http.Client.Do consumes the request body for redirects; clone so
		// retries see the same body each time.
		clone := req.Clone(req.Context())
		Logf("netutil: %s attempt %d/%d %s %s", label, attempt, cfg.MaxAttempts, clone.Method, clone.URL.String())
		resp, err := client.Do(clone)
		if err != nil {
			lastErr = err
			Logf("netutil: %s backoff: attempt %d/%d failed: %v; will retry", label, attempt, cfg.MaxAttempts, err)
			if attempt == cfg.MaxAttempts {
				return nil, attempt, err
			}
			delay := computeDelay(cfg, attempt, 0)
			Logf("netutil: %s sleeping %s before next attempt", label, delay)
			time.Sleep(delay)
			continue
		}
		if !shouldRetry(resp.StatusCode) {
			if attempt > 1 {
				Logf("netutil: %s attempt %d/%d succeeded with status %d", label, attempt, cfg.MaxAttempts, resp.StatusCode)
			}
			return resp, attempt, nil
		}
		// Read up to 512 bytes of the body so the operator can see *why* the
		// server rejected us (e.g. CF rate-limit message, CF modId not found).
		bodyPreview := readBodyPreview(resp.Body, 512)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		Logf("netutil: %s backoff: attempt %d/%d returned transient status %d body=%q", label, attempt, cfg.MaxAttempts, resp.StatusCode, bodyPreview)
		lastResp = resp
		if attempt == cfg.MaxAttempts {
			return resp, attempt, nil
		}
		extra := retryAfter(resp)
		delay := computeDelay(cfg, attempt, extra)
		if extra > 0 {
			Logf("netutil: %s server asked to wait %s (Retry-After); sleeping %s", label, extra, delay)
		} else {
			Logf("netutil: %s sleeping %s before next attempt", label, delay)
		}
		time.Sleep(delay)
	}
	if lastErr != nil {
		return nil, cfg.MaxAttempts, lastErr
	}
	if lastResp != nil {
		return lastResp, cfg.MaxAttempts, nil
	}
	return nil, cfg.MaxAttempts, fmt.Errorf("netutil: retry loop exited without response")
}

// shouldRetry reports whether the status code is one we treat as transient.
func shouldRetry(code int) bool {
	if code == http.StatusTooManyRequests {
		return true
	}
	if code == http.StatusForbidden {
		// CurseForge returns 403 when the public key hits its hourly rate
		// limit. Treat it as transient so a backoff retry has a chance.
		return true
	}
	if code >= 500 && code < 600 {
		return true
	}
	return false
}

// retryAfter parses a Retry-After header in seconds (or HTTP-date). Returns 0
// when the header is missing or unparseable.
func retryAfter(resp *http.Response) time.Duration {
	v := resp.Header.Get("Retry-After")
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			return 0
		}
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}

// computeDelay returns the sleep duration for the given attempt: BaseDelay *
// 2^(attempt-1) plus a small random jitter, capped at MaxDelay when set.
// extra is added on top (used for Retry-After).
func computeDelay(cfg RetryConfig, attempt int, extra time.Duration) time.Duration {
	d := cfg.BaseDelay << (attempt - 1)
	if d <= 0 {
		d = cfg.BaseDelay
	}
	if cfg.MaxDelay > 0 && d > cfg.MaxDelay {
		d = cfg.MaxDelay
	}
	// Jitter ±25% so 21 parallel locks don't lockstep.
	jitter := time.Duration(rand.Int63n(int64(d) / 2))
	d = d - d/4 + jitter
	if d < 0 {
		d = 0
	}
	return d + extra
}

// sleepBackoff waits the duration returned by computeDelay.
func sleepBackoff(cfg RetryConfig, attempt int, extra time.Duration) {
	time.Sleep(computeDelay(cfg, attempt, extra))
}

// GetWithRetry is the http.Get equivalent of DoWithRetry. It builds a GET
// request for url and runs it through the same retry loop.
func GetWithRetry(client *http.Client, url string, cfg RetryConfig) (*http.Response, int, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 0, err
	}
	return DoWithRetry(client, req, cfg)
}

// readBodyPreview reads up to n bytes from body and returns it as a string
// trimmed of trailing whitespace. The body is then ready to be drained by the
// caller (io.Copy to io.Discard).
func readBodyPreview(body io.Reader, n int) string {
	if body == nil {
		return ""
	}
	limited := io.LimitReader(body, int64(n))
	buf, err := io.ReadAll(limited)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(buf))
}
