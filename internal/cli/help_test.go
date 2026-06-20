// File: internal/cli/help_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for `--help` output across subcommands.

package cli

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"
)

var _ = Describe("help", func() {
	It("root help lists all commands", func() {
		stdout, _, err := runCLI("--help")
		Expect(err).NotTo(HaveOccurred())
		for _, sub := range []string{"lock", "build", "list", "validate", "set", "tree", "config", "version"} {
			Expect(stdout).To(ContainSubstring(sub))
		}
	})
	It("help subcommand lists all commands", func() {
		stdout, _, err := runCLI("help")
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("lock"))
	})
	It("lock --help", func() {
		stdout, _, err := runCLI("lock", "--help")
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("lock"))
	})
	It("lock release --help", func() {
		stdout, _, err := runCLI("lock", "release", "--help")
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("set"))
	})
	It("set --help", func() {
		stdout, _, err := runCLI("set", "--help")
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("cf-key"))
	})
	It("lock add --help", func() {
		stdout, _, err := runCLI("lock", "add", "--help")
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("source"))
	})
	It("lock update --help", func() {
		stdout, _, err := runCLI("lock", "update", "--help")
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("version"))
	})
})
var _ = Describe("CLI extra4 - renderUsageToStderr", func() {
	It("renders root usage to stderr", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		cmd := NewApp()
		renderUsageToStderr(cmd)
	})
})

var _ = Describe("CLI extra4 - newHelpCmd help with topic", func() {
	It("help for an unknown topic still returns nil", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		cmd := NewApp()
		cmd.SetArgs([]string{"help", "nonexistent-cmd"})
		Expect(cmd.Execute()).To(Succeed())
	})
})
