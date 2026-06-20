// File: internal/resolver/local_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/resolver/local.go (ResolveLocalSource).

package resolver

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ResolveLocalSource", func() {
	It("returns a LockedSource for an existing jar file", func() {
		dir := GinkgoT().TempDir()
		jar := filepath.Join(dir, "mod.jar")
		Expect(os.WriteFile(jar, []byte("x"), 0644)).To(Succeed())

		src, err := ResolveLocalSource(jar, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(src.Type).To(Equal("local"))
		Expect(src.Path).To(Equal(jar))
		Expect(src.FileName).To(Equal("mod.jar"))
	})

	It("expands {mcVersion} and {loader} placeholders", func() {
		dir := GinkgoT().TempDir()
		// File name must match the expanded pattern exactly.
		target := filepath.Join(dir, "1.21.1-neoforge-mod.jar")
		Expect(os.WriteFile(target, []byte("x"), 0644)).To(Succeed())

		pattern := filepath.Join(dir, "{mcVersion}-{loader}-mod.jar")
		src, err := ResolveLocalSource(pattern, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(src.Path).To(Equal(target))
	})

	It("returns an error when the file is missing", func() {
		_, err := ResolveLocalSource("/no/such/mod.jar", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})

	It("returns an error when the path is a directory", func() {
		_, err := ResolveLocalSource(GinkgoT().TempDir(), "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})

	It("returns an error when the file is not a .jar", func() {
		dir := GinkgoT().TempDir()
		notJar := filepath.Join(dir, "mod.zip")
		Expect(os.WriteFile(notJar, []byte("x"), 0644)).To(Succeed())
		_, err := ResolveLocalSource(notJar, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
})
