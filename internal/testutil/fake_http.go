// File: internal/testutil/fake_http.go
// Created: 2026-06-20
// Description: Helpers for spinning up an httptest.Server with a fixed handler.

package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// FakeServer starts an httptest.Server that responds to every request
// with the given status and body. The caller receives the server URL and
// is responsible for srv.Close() (or t.Cleanup(srv.Close)).
func FakeServer(t testing.TB, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// FakeHandlerServer starts an httptest.Server with a caller-supplied
// handler. Useful when the test needs to inspect request headers, query
// params, or return different responses per path.
func FakeHandlerServer(t testing.TB, h http.Handler) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}
