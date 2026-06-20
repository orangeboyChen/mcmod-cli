// File: internal/cli/commands_release_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for `mcmod lock release` subcommands (set/list/show/delete).

package cli

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"
)

var _ = Describe("release set", func() {
	It("release set writes index", func() {
		chdirTemp(`{"packName":"p","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		_, _, err := runCLI("lock", "release", "set", "1.21.1",
			"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0")
		Expect(err).NotTo(HaveOccurred())
	})
	It("release set with loader and artifacts writes artifact map", func() {
		chdirTemp(`{"packName":"p","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		_, _, err := runCLI("lock", "release", "set", "1.21.1", "neoforge",
			"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0",
			"--artifact-client", "client.jar", "--artifact-server", "server.jar")
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("release list/show/delete", func() {
	It("release list with no index errors", func() {
		chdirTemp(`{"packName":"p","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		_, _, err := runCLI("lock", "release", "list", "1.21.1")
		Expect(err).To(HaveOccurred())
	})
	It("release show without args errors", func() {
		_, _, err := runCLI("lock", "release", "show")
		Expect(err).To(HaveOccurred())
	})
	It("release show with no index errors", func() {
		chdirTemp(`{"packName":"p","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		_, _, err := runCLI("lock", "release", "show", "1.21.1", "0.1.0")
		Expect(err).To(HaveOccurred())
	})
	It("release delete with too few args errors", func() {
		_, _, err := runCLI("lock", "release", "delete")
		Expect(err).To(HaveOccurred())
	})
	It("release delete with no index for loader-specific target succeeds gracefully", func() {
		chdirTemp(`{"packName":"p","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		stdout, _, err := runCLI("lock", "release", "delete", "1.21.1", "0.1.0", "neoforge", "--target", "client")
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("deleted"))
	})
	It("release delete for non-existent version in non-empty index succeeds gracefully", func() {
		chdirTemp(`{"packName":"p","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		_, _, err := runCLI("lock", "release", "set", "1.21.1", "neoforge",
			"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0")
		Expect(err).NotTo(HaveOccurred())
		stdout, _, err := runCLI("lock", "release", "delete", "1.21.1", "99.99.99", "neoforge", "--target", "client")
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("deleted"))
	})
	It("release delete entire version", func() {
		chdirTemp(`{"packName":"p","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		_, _, err := runCLI("lock", "release", "set", "1.21.1", "neoforge",
			"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0")
		Expect(err).NotTo(HaveOccurred())
		_, _, err = runCLI("lock", "release", "delete", "1.21.1", "0.1.0")
		Expect(err).NotTo(HaveOccurred())
	})
	It("release delete non-existing version in non-empty index (whole-record path) errors", func() {
		chdirTemp(`{"packName":"p","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		_, _, err := runCLI("lock", "release", "set", "1.21.1", "neoforge",
			"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0")
		Expect(err).NotTo(HaveOccurred())
		_, _, err = runCLI("lock", "release", "delete", "1.21.1", "9.9.9")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("CLI extra3 - newReleaseDeleteCmd target validation", func() {
	It("invalid --target fails with hint", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "release", "delete", "1.21.1", "0.1.0", "--target", "bogus"})
		err := cmd.Execute()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid"))
	})
})
