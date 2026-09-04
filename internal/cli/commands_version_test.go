// File: internal/cli/commands_version_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for `mcmod version` subcommand.

package cli

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("version", func() {
	It("version prints the mcmod release version", func() {
		stdout, _, err := runCLI("version")
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("mcmod version " + domain.Version))
	})

	It("version output is non-empty and single-line", func() {
		stdout, _, err := runCLI("version")
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).NotTo(BeEmpty())
		Expect(stdout).NotTo(ContainSubstring("\n\n"))
	})
})
