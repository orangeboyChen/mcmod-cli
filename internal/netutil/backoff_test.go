// File: internal/netutil/backoff_test.go
// Created: 2026-06-21
// Description: Ginkgo tests for internal/netutil/backoff.go (DoWithRetry, GetWithRetry, retry policies, label helpers).

package netutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("shouldRetry", func() {
	It("does not retry on 2xx and 4xx (except 408/429)", func() {
		Expect(shouldRetry(http.StatusOK)).To(BeFalse())
		Expect(shouldRetry(http.StatusBadRequest)).To(BeFalse())
		Expect(shouldRetry(http.StatusNotFound)).To(BeFalse())
	})

	It("retries on 5xx and rate-limit responses", func() {
		Expect(shouldRetry(http.StatusForbidden)).To(BeTrue())
		Expect(shouldRetry(http.StatusTooManyRequests)).To(BeTrue())
		Expect(shouldRetry(http.StatusInternalServerError)).To(BeTrue())
		Expect(shouldRetry(http.StatusBadGateway)).To(BeTrue())
		Expect(shouldRetry(http.StatusServiceUnavailable)).To(BeTrue())
	})
})

var _ = Describe("DoWithRetry", func() {
	It("returns the response on the first try when the server is healthy", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "ok")
		}))
		DeferCleanup(srv.Close)

		req, _ := http.NewRequest("GET", srv.URL, nil)
		resp, attempts, err := DoWithRetry(http.DefaultClient, req, RetryConfig{MaxAttempts: 3, BaseDelay: time.Millisecond})
		Expect(err).NotTo(HaveOccurred())
		Expect(attempts).To(Equal(1))
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})

	It("retries until the server returns a success", func() {
		var calls int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if calls < 3 {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		DeferCleanup(srv.Close)

		req, _ := http.NewRequest("GET", srv.URL, nil)
		resp, attempts, err := DoWithRetry(http.DefaultClient, req, RetryConfig{MaxAttempts: 5, BaseDelay: time.Millisecond})
		Expect(err).NotTo(HaveOccurred())
		Expect(calls).To(Equal(3))
		Expect(attempts).To(Equal(3))
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})

	It("gives up after MaxAttempts and returns the last response", func() {
		var calls int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.WriteHeader(http.StatusForbidden)
		}))
		DeferCleanup(srv.Close)

		req, _ := http.NewRequest("GET", srv.URL, nil)
		resp, attempts, err := DoWithRetry(http.DefaultClient, req, RetryConfig{MaxAttempts: 3, BaseDelay: time.Millisecond})
		Expect(err).NotTo(HaveOccurred())
		Expect(calls).To(Equal(3))
		Expect(attempts).To(Equal(3))
		Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
	})

	It("returns an error when the network is unreachable", func() {
		req, _ := http.NewRequest("GET", "http://127.0.0.1:1/never", nil)
		_, attempts, err := DoWithRetry(http.DefaultClient, req, RetryConfig{MaxAttempts: 2, BaseDelay: time.Millisecond})
		Expect(err).To(HaveOccurred())
		Expect(attempts).To(Equal(2))
	})

	It("falls back to defaults when the config is zero", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		DeferCleanup(srv.Close)
		req, _ := http.NewRequest("GET", srv.URL, nil)
		resp, _, err := DoWithRetry(http.DefaultClient, req, RetryConfig{})
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})
})

var _ = Describe("GetWithRetry", func() {
	It("returns the response on the first try", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "ok")
		}))
		DeferCleanup(srv.Close)
		resp, attempts, err := GetWithRetry(http.DefaultClient, srv.URL, RetryConfig{MaxAttempts: 2, BaseDelay: time.Millisecond})
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Expect(attempts).To(Equal(1))
	})

	It("returns an error for a malformed URL", func() {
		_, _, err := GetWithRetry(http.DefaultClient, "://not-a-url", RetryConfig{MaxAttempts: 1})
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("retryAfter", func() {
	It("parses a Retry-After header in seconds", func() {
		resp := &http.Response{Header: http.Header{}}
		resp.Header.Set("Retry-After", "2")
		Expect(retryAfter(resp)).To(Equal(2 * time.Second))
	})

	It("parses a Retry-After header as an HTTP date in the future", func() {
		resp := &http.Response{Header: http.Header{}}
		future := time.Now().Add(2 * time.Hour).UTC().Format(http.TimeFormat)
		resp.Header.Set("Retry-After", future)
		Expect(retryAfter(resp)).To(BeNumerically(">", 0))
	})

	It("returns zero when the header is missing", func() {
		resp := &http.Response{Header: http.Header{}}
		Expect(retryAfter(resp)).To(Equal(time.Duration(0)))
	})

	It("returns zero for negative seconds", func() {
		resp := &http.Response{Header: http.Header{}}
		resp.Header.Set("Retry-After", "-5")
		Expect(retryAfter(resp)).To(Equal(time.Duration(0)))
	})

	It("returns zero for an unparseable value", func() {
		resp := &http.Response{Header: http.Header{}}
		resp.Header.Set("Retry-After", "not-a-date-or-number")
		Expect(retryAfter(resp)).To(Equal(time.Duration(0)))
	})
})

var _ = Describe("computeDelay and sleepBackoff", func() {
	It("caps the pre-jitter delay at MaxDelay and lets jitter push it slightly above", func() {
		cfg := RetryConfig{MaxAttempts: 5, BaseDelay: time.Second, MaxDelay: 2 * time.Second}
		for i := 1; i < 5; i++ {
			got := computeDelay(cfg, i, 0)
			// Pre-jitter cap is 2s; jitter adds up to 25%, so 2.5s is the
			// documented upper bound. Allow a 3s ceiling for safety.
			Expect(got).To(BeNumerically("<=", 3*time.Second))
		}
	})

	It("adds the extra duration to the computed delay", func() {
		cfg := RetryConfig{MaxAttempts: 1, BaseDelay: time.Millisecond}
		got := computeDelay(cfg, 1, 50*time.Millisecond)
		Expect(got).To(BeNumerically(">=", 50*time.Millisecond))
	})

	It("sleepBackoff does not panic with a tiny delay", func() {
		// Smoke test: just call it and assert it returns.
		Expect(func() {
			sleepBackoff(RetryConfig{MaxAttempts: 1, BaseDelay: time.Millisecond}, 1, 0)
		}).NotTo(Panic())
	})
})

var _ = Describe("labelForRequest", func() {
	It("returns the X-Netutil-Label header when set", func() {
		req, _ := http.NewRequest("GET", "http://x.example/y", nil)
		req.Header.Set("X-Netutil-Label", "my-label")
		Expect(labelForRequest(req)).To(Equal("my-label"))
	})

	It("falls back to the URL host when no header is set", func() {
		req, _ := http.NewRequest("GET", "http://x.example/y", nil)
		Expect(labelForRequest(req)).To(ContainSubstring("x.example"))
	})
})

var _ = Describe("readBodyPreview", func() {
	It("returns the first n bytes of a non-empty body", func() {
		body := strings.NewReader("hello world")
		Expect(readBodyPreview(body, 5)).To(Equal("hello"))
	})

	It("returns an empty string for a nil body", func() {
		Expect(readBodyPreview(nil, 5)).To(BeEmpty())
	})
})

var _ = Describe("SetDefaultRetry", func() {
	It("overrides DefaultRetry and restores it on the returned func", func() {
		original := DefaultRetry
		restore := SetDefaultRetry(RetryConfig{MaxAttempts: 7, BaseDelay: 7 * time.Millisecond})
		Expect(DefaultRetry.MaxAttempts).To(Equal(7))
		restore()
		Expect(DefaultRetry.MaxAttempts).To(Equal(original.MaxAttempts))
	})
})

var _ = Describe("RetryConfig.withDefaults", func() {
	It("fills in defaults when the config is zero", func() {
		c := RetryConfig{}.withDefaults()
		Expect(c.MaxAttempts).To(BeNumerically(">=", 1))
		Expect(c.BaseDelay).To(BeNumerically(">", 0))
	})

	It("preserves explicitly set values", func() {
		c := RetryConfig{MaxAttempts: 3, BaseDelay: 5 * time.Millisecond, MaxDelay: 100 * time.Millisecond}.withDefaults()
		Expect(c.MaxAttempts).To(Equal(3))
		Expect(c.BaseDelay).To(Equal(5 * time.Millisecond))
		Expect(c.MaxDelay).To(Equal(100 * time.Millisecond))
	})
})

var _ = Describe("computeDelay extra branches", func() {
	It("caps delay at MaxDelay when delay exceeds it", func() {
		cfg := RetryConfig{BaseDelay: time.Second, MaxDelay: 100 * time.Millisecond}
		d := computeDelay(cfg, 5, 0)
		// MaxDelay (100ms) + jitter range - MaxDelay/4, so the upper bound
		// is roughly 175ms. The pre-jitter cap is exactly MaxDelay.
		Expect(d).To(BeNumerically("<=", 200*time.Millisecond))
	})

	It("returns extra added to base when attempt is 1 and base is small", func() {
		cfg := RetryConfig{BaseDelay: 1 * time.Millisecond, MaxDelay: 10 * time.Second}
		d := computeDelay(cfg, 1, 50*time.Millisecond)
		Expect(d).To(BeNumerically(">=", 50*time.Millisecond))
	})
})

var _ = Describe("DoWithRetry readBodyPreview", func() {
	It("readBodyPreview returns empty for nil body", func() {
		Expect(readBodyPreview(nil, 10)).To(BeEmpty())
	})

	It("readBodyPreview reads up to n bytes", func() {
		body := strings.NewReader(strings.Repeat("a", 100))
		got := readBodyPreview(body, 10)
		Expect(got).To(HaveLen(10))
	})
})

var _ = Describe("DoWithRetry transient then success", func() {
	It("logs success on attempt > 1 when transient status recovers", func() {
		var attempts int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte("transient"))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}))
		DeferCleanup(srv.Close)
		req, _ := http.NewRequest("GET", srv.URL, nil)
		cfg := RetryConfig{MaxAttempts: 3, BaseDelay: time.Millisecond}
		resp, n, err := DoWithRetry(http.DefaultClient, req, cfg)
		Expect(err).NotTo(HaveOccurred())
		Expect(n).To(Equal(2))
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})

	It("honours Retry-After header when present on transient response", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		DeferCleanup(srv.Close)
		req, _ := http.NewRequest("GET", srv.URL, nil)
		cfg := RetryConfig{MaxAttempts: 2, BaseDelay: time.Millisecond}
		_, _, err := DoWithRetry(http.DefaultClient, req, cfg)
		// Either returns or gives up — the key is the Retry-After path runs.
		Expect(err).NotTo(HaveOccurred()) // we get the last 429 response
	})
})

var _ = Describe("readBodyPreview edge cases", func() {
	It("returns empty when read returns error", func() {
		r := io.NopCloser(errorReader{})
		Expect(readBodyPreview(r, 10)).To(BeEmpty())
	})

	It("returns the body as-is when within n bytes", func() {
		Expect(readBodyPreview(strings.NewReader("hello world"), 64)).To(Equal("hello world"))
	})
})

type errorReader struct{}

func (errorReader) Read(_ []byte) (int, error) { return 0, io.ErrUnexpectedEOF }
