// File: internal/resolver/coverage_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for resolver package.
package resolver

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("Resolver", func() {
	It("ResolveSource empty type errors", func() {
		_, err := ResolveSource(domain.ModSource{}, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("ResolveSource unknown type errors", func() {
		_, err := ResolveSource(domain.ModSource{Type: "unknown"}, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("LoaderToCF maps correctly", func() {
		Expect(LoaderToCF("fabric")).To(Equal(4))
		Expect(LoaderToCF("neoforge")).To(Equal(6))
		Expect(LoaderToCF("forge")).To(Equal(0))
	})
	It("CurseForgeByQuery without key errors", func() {
		_, err := ResolveCurseForgeByQuery("test", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("CurseForgeByID without key errors", func() {
		_, err := ResolveCurseForgeByID(123, 456)
		Expect(err).To(HaveOccurred())
	})
	It("GitHubRelease replaces placeholders", func() {
		src, err := ResolveGitHubRelease("owner/repo", "v{mcVersion}", "asset-{tag}-{loader}.jar", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(src.Tag).To(Equal("v1.21.1"))
		Expect(src.AssetName).To(Equal("asset-v1.21.1-neoforge.jar"))
	})
	It("GitHubRelease direct pattern works", func() {
		src, err := ResolveGitHubRelease("o/r", "v1", "asset.jar", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(src.AssetName).To(Equal("asset.jar"))
	})
	It("ResolveLocalSource fails for missing file", func() {
		_, err := ResolveLocalSource("./missing.jar", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("ResolveLocalSource works for existing file", func() {
		dir := GinkgoT().TempDir()
		p := filepath.Join(dir, "mod.jar")
		Expect(os.WriteFile(p, []byte("data"), 0644)).To(Succeed())
		src, err := ResolveLocalSource(p, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(src.FileName).To(Equal("mod.jar"))
	})
	It("GitPackage fails for nonexistent repo", func() {
		_, err := ResolveGitPackage("owner/nonexistent-repo-xyz", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
})
