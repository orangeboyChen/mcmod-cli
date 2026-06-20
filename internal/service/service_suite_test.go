// File: internal/service/service_suite_test.go
// Created: 2026-06-20
// Description: Ginkgo test suite for service package.
package service

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/orangeboyChen/mcmod-cli/internal/netutil"
)

func TestService(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Suite")
}

// Tighten the global retry budget. Service tests build zips that can touch
// the downloader; with the default 10-attempt exponential backoff, a single
// 403 from a remote API multiplies into tens of seconds of sleep per test.
var _ = BeforeSuite(func() {
	netutil.SetDefaultRetry(netutil.RetryConfig{
		MaxAttempts: 1,
		BaseDelay:   time.Millisecond,
	})
})
