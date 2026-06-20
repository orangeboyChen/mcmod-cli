// File: internal/resolver/push80_test.go
// Created: 2026-06-20
// Description: Push resolver coverage.
package resolver

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("Resolver push80", func() {
	It("curseforge query with key calls API", func() {
		os.Setenv("CURSEFORGE_API_KEY", "test-key")
		defer os.Unsetenv("CURSEFORGE_API_KEY")
		_, err := ResolveCurseForgeByQuery("test-mod", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("curseforge by id without key", func() {
		_, err := ResolveCurseForgeByID(1, 2)
		Expect(err).To(HaveOccurred())
	})
	It("github release mcVersion placeholder", func() {
		s, err := ResolveGitHubRelease("o/r", "v{mcVersion}", "a-{tag}.jar", "1.21.1", "fabric")
		Expect(err).NotTo(HaveOccurred())
		Expect(s.Tag).To(Equal("v1.21.1"))
		Expect(s.AssetName).To(Equal("a-v1.21.1.jar"))
	})
	It("github release loader placeholder", func() {
		s, err := ResolveGitHubRelease("o/r", "v1", "m-{loader}.jar", "1.21.1", "fabric")
		Expect(err).NotTo(HaveOccurred())
		Expect(s.AssetName).To(Equal("m-fabric.jar"))
	})
	It("github release wildcard fails for bad repo", func() {
		_, err := ResolveGitHubRelease("o/nonexistent9999", "*", "a.jar", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("local source existing jar", func() {
		dir := GinkgoT().TempDir()
		p := filepath.Join(dir, "test-mod.jar")
		Expect(os.WriteFile(p, []byte("data"), 0644)).To(Succeed())
		src, err := ResolveLocalSource(p, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(src.FileName).To(Equal("test-mod.jar"))
	})
	It("local source dir fails", func() {
		dir := GinkgoT().TempDir()
		_, err := ResolveLocalSource(dir, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("ResolveSource curseforge query needs key", func() {
		_, err := ResolveSource(domain.ModSource{Type: "curseforge", Query: "test"}, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
})
