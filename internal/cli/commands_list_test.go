// File: internal/cli/commands_list_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for `mcmod list` subcommand.

package cli

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("list", func() {
	It("list without spec errors", func() {
		chdirTemp("")
		_, _, err := runCLI("list")
		Expect(err).To(HaveOccurred())
	})
	It("list with empty mods prints (empty) for each scope", func() {
		chdirTemp(`{"packName":"empty","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		stdout, _, err := runCLI("list")
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("(empty)"))
		Expect(stdout).To(ContainSubstring("[Server]"))
		Expect(stdout).To(ContainSubstring("[Client]"))
		Expect(stdout).To(ContainSubstring("[Shared]"))
	})
})
