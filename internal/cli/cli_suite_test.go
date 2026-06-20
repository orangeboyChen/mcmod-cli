// File: internal/cli/cli_suite_test.go
// Created: 2026-06-20
// Description: Ginkgo test suite for CLI package.
package cli

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/orangeboyChen/mcmod-cli/internal/netutil"
)

func TestCLI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CLI Suite")
}

// Keep CLI command tests fast even when underlying resolver/downloader calls
// hit a remote API that returns 403/429. Default retry backoff is fine for
// production but would make the suite hang for minutes.
var _ = BeforeSuite(func() {
	netutil.SetDefaultRetry(netutil.RetryConfig{
		MaxAttempts: 1,
		BaseDelay:   time.Millisecond,
	})
})
