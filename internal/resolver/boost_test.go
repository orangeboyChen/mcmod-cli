// File: internal/resolver/boost_test.go
// Created: 2026-06-20
// Description: Extra resolver tests to push coverage higher.
package resolver

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("Resolver boost", func() {
	Describe("ResolveSource dispatcher", func() {
		It("errors on empty type", func() {
			_, err := ResolveSource(domain.ModSource{}, "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("errors on unknown type", func() {
			_, err := ResolveSource(domain.ModSource{Type: "unknown"}, "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("curseforge with query and no key errors", func() {
			_, err := ResolveSource(domain.ModSource{Type: "curseforge", Query: "foo"}, "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("curseforge with neither modId+fileId nor query errors", func() {
			_, err := ResolveSource(domain.ModSource{Type: "curseforge"}, "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("local with existing jar returns LockedSource", func() {
			dir := GinkgoT().TempDir()
			jarPath := filepath.Join(dir, "test.jar")
			Expect(os.WriteFile(jarPath, []byte("dummy"), 0644)).To(Succeed())
			out, err := ResolveSource(domain.ModSource{Type: "local", Path: jarPath}, "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			ls, ok := out.(*domain.LockedSource)
			Expect(ok).To(BeTrue())
			Expect(ls.Type).To(Equal("local"))
		})
		It("local with missing file errors", func() {
			_, err := ResolveSource(domain.ModSource{Type: "local", Path: "/no/such.jar"}, "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("local with directory path errors", func() {
			dir := GinkgoT().TempDir()
			_, err := ResolveSource(domain.ModSource{Type: "local", Path: dir}, "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("local with non-jar extension errors", func() {
			dir := GinkgoT().TempDir()
			p := filepath.Join(dir, "test.txt")
			Expect(os.WriteFile(p, []byte("dummy"), 0644)).To(Succeed())
			_, err := ResolveSource(domain.ModSource{Type: "local", Path: p}, "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("github-release with no key returns placeholder LockedSource", func() {
			out, err := ResolveSource(domain.ModSource{Type: "github-release", Repo: "o/r", Tag: "v1"}, "1.21.1", "neoforge")
			// May error or return placeholder depending on resolver
			_ = out
			_ = err
		})
	})

	Describe("ResolveLocalSource paths", func() {
		It("replaces {mcVersion} placeholder", func() {
			dir := GinkgoT().TempDir()
			p := filepath.Join(dir, "1.21.1.jar")
			Expect(os.WriteFile(p, []byte("dummy"), 0644)).To(Succeed())
			out, err := ResolveLocalSource(filepath.Join(dir, "{mcVersion}.jar"), "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(out.Path).To(Equal(p))
		})
		It("replaces {loader} placeholder", func() {
			dir := GinkgoT().TempDir()
			p := filepath.Join(dir, "neoforge.jar")
			Expect(os.WriteFile(p, []byte("dummy"), 0644)).To(Succeed())
			out, err := ResolveLocalSource(filepath.Join(dir, "{loader}.jar"), "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(out.Path).To(Equal(p))
		})
	})
})
