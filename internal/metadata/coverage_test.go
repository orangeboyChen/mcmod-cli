// File: internal/metadata/coverage_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for metadata package.
package metadata

import (
	"archive/zip"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metadata", func() {
	Describe("SourceIdentity", func() {
		It("formats curseforge identity", func() {
			Expect(SourceIdentity("curseforge", "12345")).To(Equal("curseforge:12345"))
		})
		It("formats github identity", func() {
			Expect(SourceIdentity("github-release", "o/r")).To(Equal("github:o/r"))
		})
	})

	Describe("InternalIdentity", func() {
		It("formats correctly", func() {
			Expect(InternalIdentity("neoforge", "test_mod")).To(Equal("neoforge:test_mod"))
		})
	})

	Describe("Confidence constants", func() {
		It("are defined", func() {
			Expect(ConfidenceMetadata).To(Equal(IdentityConfidence("metadata")))
		})
	})

	Describe("parseSimpleTOML", func() {
		It("parses simple key=value", func() {
			data := []byte("modid=\"test\"\nversion=\"1.0\"\n[section]\nk=v\n")
			result := parseSimpleTOML(data)
			Expect(result["modid"]).To(Equal("test"))
			Expect(result["version"]).To(Equal("1.0"))
		})
	})

	Describe("DepInfoFromIdentity", func() {
		It("splits on colon", func() {
			Expect(DepInfoFromIdentity("cf:123").ModID).To(Equal("123"))
		})
		It("handles bare id", func() {
			Expect(DepInfoFromIdentity("bare").ModID).To(Equal("bare"))
		})
	})

	Describe("ParseFabricDepends", func() {
		It("parses dependencies", func() {
			deps := ParseFabricDepends([]byte(`{"fapi":"*"}`))
			Expect(deps).To(HaveLen(1))
			Expect(deps[0].ModID).To(Equal("fapi"))
		})
		It("returns empty for nil", func() {
			Expect(ParseFabricDepends(nil)).To(BeEmpty())
		})
	})

	Describe("ReadNeoForgeMetadata", func() {
		It("reads from a real zip", func() {
			dir := GinkgoT().TempDir()
			jar := filepath.Join(dir, "neo.jar")
			f, _ := os.Create(jar)
			w := zip.NewWriter(f)
			e, _ := w.Create("META-INF/neoforge.mods.toml")
			e.Write([]byte("modid=\"neotest\"\nversion=\"2.0\"\n"))
			w.Close()
			f.Close()
			info, err := ReadNeoForgeMetadata(jar)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.ModID).To(Equal("neotest"))
			Expect(info.Version).To(Equal("2.0"))
		})
		It("reads from alt mods.toml path", func() {
			dir := GinkgoT().TempDir()
			jar := filepath.Join(dir, "neo-alt.jar")
			f, _ := os.Create(jar)
			w := zip.NewWriter(f)
			e, _ := w.Create("META-INF/mods.toml")
			e.Write([]byte("modid=\"altmod\"\n"))
			w.Close()
			f.Close()
			info, err := ReadNeoForgeMetadata(jar)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.ModID).To(Equal("altmod"))
		})
		It("fails on non-zip file", func() {
			_, err := ReadNeoForgeMetadata("/nonexistent.jar")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ReadFabricMetadata", func() {
		It("reads from a real zip", func() {
			dir := GinkgoT().TempDir()
			jar := filepath.Join(dir, "fabric.jar")
			f, _ := os.Create(jar)
			w := zip.NewWriter(f)
			e, _ := w.Create("fabric.mod.json")
			e.Write([]byte(`{"id":"fabtest","version":"3.0","depends":{"lib-a":"*"}}`))
			w.Close()
			f.Close()
			info, err := ReadFabricMetadata(jar)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.ModID).To(Equal("fabtest"))
			Expect(info.Version).To(Equal("3.0"))
			Expect(info.Dependencies).To(HaveLen(1))
		})
		It("fails on non-zip file", func() {
			_, err := ReadFabricMetadata("/nonexistent.jar")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ReadJarMetadata", func() {
		It("auto-detects neoforge", func() {
			dir := GinkgoT().TempDir()
			jar := filepath.Join(dir, "auto.jar")
			f, _ := os.Create(jar)
			w := zip.NewWriter(f)
			e, _ := w.Create("META-INF/neoforge.mods.toml")
			e.Write([]byte("modid=\"auto_mod\"\nversion=\"4.0\"\n"))
			w.Close()
			f.Close()
			info, err := ReadJarMetadata(jar)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.ModID).To(Equal("auto_mod"))
		})
	})
})
