// File: internal/cli/commands_config_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for `mcmod config` subcommand.

package cli

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"
	"path/filepath"
)

var _ = Describe("config", func() {
	It("config shows the key", func() {
		chdirTemp(`{"packName":"c","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		_, _, err := runCLI("set", "cf-key", "keyval", "--project")
		Expect(err).NotTo(HaveOccurred())
		stdout, _, err := runCLI("config")
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("keyval"))
	})
	It("config with no key shows (not set)", func() {
		chdirTemp(`{"packName":"c","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		stdout, _, err := runCLI("config")
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("not set"))
	})
	It("config set-cf-key writes key", func() {
		dir := chdirTemp(`{"packName":"c","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		_, _, err := runCLI("config", "set-cf-key", "thekey")
		Expect(err).NotTo(HaveOccurred())
		_, statErr := os.Stat(filepath.Join(dir, ".mcmod", "config.json"))
		Expect(statErr).NotTo(HaveOccurred())
	})
})
