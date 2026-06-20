// File: internal/cache/boost_test.go
// Created: 2026-06-20
// Description: Boost coverage for cache package.

package cache

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

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
