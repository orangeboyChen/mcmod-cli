// File: internal/cache/coverage_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for cache package.
package cache

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

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
