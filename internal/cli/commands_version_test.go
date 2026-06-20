// File: internal/cli/commands_version_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for `mcmod version` subcommand.

package cli

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("version", func() {
	It("version prints version info", func() {
		stdout, _, err := runCLI("version")
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("mcmod version"))
	})
})
