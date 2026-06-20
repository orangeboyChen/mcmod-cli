// File: internal/netutil/backoff_suite_test.go
// Created: 2026-06-21
// Description: Ginkgo test suite for the netutil package.

package netutil

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestNetutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Netutil Suite")
}
