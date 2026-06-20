// File: internal/downloader/downloader_suite_test.go
// Created: 2026-06-20
// Description: Ginkgo test suite for downloader package.

package downloader

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/orangeboyChen/mcmod-cli/internal/netutil"
)

func TestDownloader(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Downloader Suite")
}

// Tighten the global retry budget so specs that hit the real network
// (with a bad/missing API key) finish promptly instead of backing off
// for many seconds between attempts.
var _ = BeforeSuite(func() {
	netutil.SetDefaultRetry(netutil.RetryConfig{
		MaxAttempts: 1,
		BaseDelay:   time.Millisecond,
	})
})
