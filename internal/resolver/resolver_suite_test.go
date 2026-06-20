// File: internal/resolver/resolver_suite_test.go
// Created: 2026-06-20
// Description: Ginkgo test suite for resolver package.
package resolver

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/orangeboyChen/mcmod-cli/internal/netutil"
)

func TestResolver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Resolver Suite")
}

// Tighten the global retry budget for the suite so the dozens of "tries the
// real API with a bad key" specs do not wedge the test run for minutes when
// the remote returns 403/429. Each retry call costs at most a few ms; the
// suite is single-threaded so the total wall time stays well under a second.
var _ = BeforeSuite(func() {
	netutil.SetDefaultRetry(netutil.RetryConfig{
		MaxAttempts: 1,
		BaseDelay:   time.Millisecond,
	})
})
