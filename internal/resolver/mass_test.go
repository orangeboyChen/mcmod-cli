// File: internal/resolver/mass_test.go
// Created: 2026-06-20
// Description: Mass resolver coverage.
package resolver

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("Resolver mass", func() {
	It("curseforge query with bad key calls API and fails", func() {
		os.Setenv("CURSEFORGE_API_KEY", "test")
		defer os.Unsetenv("CURSEFORGE_API_KEY")
		_, err := ResolveCurseForgeByQuery("test", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("curseforge by id fails without key", func() {
		_, err := ResolveCurseForgeByID(1, 2)
		Expect(err).To(HaveOccurred())
	})
	It("github release with mcVersion placeholder", func() {
		s, err := ResolveGitHubRelease("o/r", "v{mcVersion}", "a.jar", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(s.Tag).To(Equal("v1.21.1"))
	})
	It("github release with loader placeholder", func() {
		s, err := ResolveGitHubRelease("o/r", "v1", "m-{loader}.jar", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(s.AssetName).To(Equal("m-neoforge.jar"))
	})
	It("github release wildcard tries API", func() {
		_, err := ResolveGitHubRelease("o/nonexistent999", "*", "a.jar", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("local source existing file", func() {
		dir := GinkgoT().TempDir()
		p := filepath.Join(dir, "m.jar")
		Expect(os.WriteFile(p, []byte("data"), 0644)).To(Succeed())
		s, err := ResolveLocalSource(p, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(s.FileName).To(Equal("m.jar"))
	})
	It("local source dir fails", func() {
		dir := GinkgoT().TempDir()
		_, err := ResolveLocalSource(dir, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("local source missing file", func() {
		_, err := ResolveLocalSource("/nonexistent.jar", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("git package fails for bad repo", func() {
		_, err := ResolveGitPackage("o/bad-repo-99999-x", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("resolve source with type-local existing file", func() {
		dir := GinkgoT().TempDir()
		p := filepath.Join(dir, "lm.jar")
		Expect(os.WriteFile(p, []byte("d"), 0644)).To(Succeed())
		s, err := ResolveSource(domain.ModSource{Type: "local", Path: p}, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(s).NotTo(BeNil())
	})
	It("resolve source with curseforge needs both modId and fileId", func() {
		_, err := ResolveSource(domain.ModSource{Type: "curseforge", ModID: 1}, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
})
