// File: internal/cache/cache_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/cache/cache.go.

package cache

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// --- from boost_test.go (Cache boost) ---
var _ = Describe("Cache boost", func() {
	It("AtomicMove with nonexistent source fails", func() {
		err := AtomicMove("/nonexistent/src", "/tmp/dst")
		Expect(err).To(HaveOccurred())
	})

	It("AtomicMove creates destination directory", func() {
		d := GinkgoT().TempDir()
		src := filepath.Join(d, "src.txt")
		dst := filepath.Join(d, "subdir", "dst.txt")
		Expect(os.WriteFile(src, []byte("data"), 0644)).To(Succeed())
		err := AtomicMove(src, dst)
		Expect(err).NotTo(HaveOccurred())
		_, err = os.Stat(dst)
		Expect(err).NotTo(HaveOccurred())
	})

	It("CheckGitHubRelease returns false for missing file", func() {
		ok, size, err := CheckGitHubRelease("owner", "repo", "v1", "nonexistent.jar")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse())
		Expect(size).To(BeZero())
	})

	It("ComputeSHA256 on file works", func() {
		d := GinkgoT().TempDir()
		p := filepath.Join(d, "hash.txt")
		Expect(os.WriteFile(p, []byte("test"), 0644)).To(Succeed())
		hash, err := ComputeSHA256(p)
		Expect(err).NotTo(HaveOccurred())
		Expect(hash).To(HaveLen(64))
	})

	It("EnsureCacheDir works", func() {
		d := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(d)
		Expect(EnsureCacheDir()).To(Succeed())
		_, err := os.Stat(".cache")
		Expect(err).NotTo(HaveOccurred())
	})
})

// --- from coverage_test.go (Cache) ---
var _ = Describe("Cache", func() {
	It("CurseForgePath returns correct path", func() {
		Expect(CurseForgePath("1", "2", "test.jar")).To(ContainSubstring("curseforge"))
	})
	It("GitHubReleasePath returns correct path", func() {
		Expect(GitHubReleasePath("owner", "repo", "v1", "test.jar")).To(ContainSubstring("github-release"))
	})
	It("GitCachePath returns correct path", func() {
		Expect(GitCachePath("owner", "repo")).To(ContainSubstring("git"))
	})
	It("CheckCurseForge returns false for missing", func() {
		ok, size, err := CheckCurseForge("999", "888", "missing.jar")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse())
		Expect(size).To(BeZero())
	})
	It("CheckGitHubRelease returns false for missing", func() {
		ok, _, err := CheckGitHubRelease("o", "r", "v1", "missing.jar")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse())
	})
	It("ComputeSHA256 works on a file", func() {
		tmp := filepath.Join(os.TempDir(), "sha-test.txt")
		Expect(os.WriteFile(tmp, []byte("data"), 0644)).To(Succeed())
		defer os.Remove(tmp)
		hash, err := ComputeSHA256(tmp)
		Expect(err).NotTo(HaveOccurred())
		Expect(hash).To(HaveLen(64))
	})
	It("EnsureCacheDir creates dir", func() {
		Expect(EnsureCacheDir()).To(Succeed())
	})
	It("AtomicMove moves file", func() {
		src := filepath.Join(os.TempDir(), "atomic-src.txt")
		dst := filepath.Join(os.TempDir(), "atomic-dst.txt")
		Expect(os.WriteFile(src, []byte("data"), 0644)).To(Succeed())
		defer os.Remove(src)
		defer os.Remove(dst)
		Expect(AtomicMove(src, dst)).To(Succeed())
	})
})

var _ = Describe("Cache hash and check helpers", func() {
	It("CheckCurseForge returns true when the cache file exists", func() {
		dir := GinkgoT().TempDir()
		wd, _ := os.Getwd()
		Expect(os.Chdir(dir)).To(Succeed())
		DeferCleanup(func() { _ = os.Chdir(wd) })

		p := ".cache/curseforge/1/2/m.jar"
		Expect(os.MkdirAll(filepath.Dir(p), 0755)).To(Succeed())
		Expect(os.WriteFile(p, []byte("x"), 0644)).To(Succeed())
		ok, _, err := CheckCurseForge("1", "2", "m.jar")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
		ok, _, err = CheckCurseForge("99", "99", "nope.jar")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse())
	})

	It("CheckGitHubRelease returns true when the cache file exists", func() {
		dir := GinkgoT().TempDir()
		wd, _ := os.Getwd()
		Expect(os.Chdir(dir)).To(Succeed())
		DeferCleanup(func() { _ = os.Chdir(wd) })

		p := ".cache/github-release/o/r/v1/m.jar"
		Expect(os.MkdirAll(filepath.Dir(p), 0755)).To(Succeed())
		Expect(os.WriteFile(p, []byte("x"), 0644)).To(Succeed())
		ok, _, err := CheckGitHubRelease("o", "r", "v1", "m.jar")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
		ok, _, err = CheckGitHubRelease("nope", "nope", "v1", "m.jar")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse())
	})

	It("ComputeSHA256 hashes file contents", func() {
		dir := GinkgoT().TempDir()
		p := filepath.Join(dir, "h.txt")
		Expect(os.WriteFile(p, []byte("hello"), 0644)).To(Succeed())
		// sha256("hello") = 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
		hash, err := ComputeSHA256(p)
		Expect(err).NotTo(HaveOccurred())
		Expect(hash).To(Equal("2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"))
	})

	It("ComputeSHA256 returns an error for a missing file", func() {
		_, err := ComputeSHA256("/no/such/file")
		Expect(err).To(HaveOccurred())
	})

	It("AtomicMove overwrites an existing destination", func() {
		dir := GinkgoT().TempDir()
		src := filepath.Join(dir, "src.txt")
		dst := filepath.Join(dir, "dst.txt")
		Expect(os.WriteFile(src, []byte("new"), 0644)).To(Succeed())
		Expect(os.WriteFile(dst, []byte("old"), 0644)).To(Succeed())
		Expect(AtomicMove(src, dst)).To(Succeed())
		data, err := os.ReadFile(dst)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(Equal("new"))
	})
})
