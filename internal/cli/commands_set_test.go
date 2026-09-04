// File: internal/cli/commands_set_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for `mcmod set` subcommand.

package cli

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("set", func() {
	It("set cf-key with only one arg errors", func() {
		_, _, err := runCLI("set", "cf-key")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("hint"))
	})
	It("set with wrong first arg errors", func() {
		_, _, err := runCLI("set", "wrong-arg", "value")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("hint"))
	})
	It("set cf-key --global writes user config", func() {
		dir := chdirTemp(`{"packName":"t","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		_, _, err := runCLI("set", "cf-key", "globalkey", "--global")
		Expect(err).NotTo(HaveOccurred())
		// No project file expected.
		_, statErr := os.Stat(filepath.Join(dir, ".mcmod", "config.json"))
		Expect(os.IsNotExist(statErr)).To(BeTrue())
	})
})
