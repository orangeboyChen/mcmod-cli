// File: internal/cache/cache_suite_test.go
// Created: 2026-06-20
// Description: Ginkgo test suite for cache package.
package cache

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/orangeboyChen/mcmod-cli/internal/netutil"
)

func TestCache(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cache Suite")
}

// Keep retry budget tight so any tests in this suite that touch netutil don't
// wedge the run on a remote 403/429.
var _ = BeforeSuite(func() {
	netutil.SetDefaultRetry(netutil.RetryConfig{
		MaxAttempts: 1,
		BaseDelay:   time.Millisecond,
	})
})
