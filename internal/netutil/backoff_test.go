// File: internal/netutil/backoff_test.go
// Created: 2026-06-20
// Description: Unit tests for the HTTP retry helper.

package netutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestShouldRetry(t *testing.T) {
	cases := []struct {
		code int
		want bool
	}{
		{http.StatusOK, false},
		{http.StatusBadRequest, false},
		{http.StatusNotFound, false},
		{http.StatusForbidden, true},
		{http.StatusTooManyRequests, true},
		{http.StatusInternalServerError, true},
		{http.StatusBadGateway, true},
		{http.StatusServiceUnavailable, true},
	}
	for _, c := range cases {
		if got := shouldRetry(c.code); got != c.want {
			t.Errorf("shouldRetry(%d)=%v want %v", c.code, got, c.want)
		}
	}
}

func TestDoWithRetrySuccessOnFirstTry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, attempts, err := DoWithRetry(http.DefaultClient, req, RetryConfig{MaxAttempts: 3, BaseDelay: 1 * time.Millisecond})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if attempts != 1 {
		t.Errorf("attempts=%d want 1", attempts)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status=%d want 200", resp.StatusCode)
	}
}

func TestDoWithRetryRetriesThenSucceeds(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, attempts, err := DoWithRetry(http.DefaultClient, req, RetryConfig{MaxAttempts: 5, BaseDelay: 1 * time.Millisecond})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if calls != 3 {
		t.Errorf("calls=%d want 3", calls)
	}
	if attempts != 3 {
		t.Errorf("attempts=%d want 3", attempts)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status=%d want 200", resp.StatusCode)
	}
}

func TestDoWithRetryGivesUpAfterMax(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, attempts, err := DoWithRetry(http.DefaultClient, req, RetryConfig{MaxAttempts: 3, BaseDelay: 1 * time.Millisecond})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if calls != 3 {
		t.Errorf("calls=%d want 3", calls)
	}
	if attempts != 3 {
		t.Errorf("attempts=%d want 3", attempts)
	}
	if resp.StatusCode != 403 {
		t.Errorf("status=%d want 403", resp.StatusCode)
	}
}

func TestRetryAfterParseSeconds(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", "2")
	if got := retryAfter(resp); got != 2*time.Second {
		t.Errorf("retryAfter=%v want 2s", got)
	}
}

func TestRetryAfterMissing(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	if got := retryAfter(resp); got != 0 {
		t.Errorf("retryAfter=%v want 0", got)
	}
}
