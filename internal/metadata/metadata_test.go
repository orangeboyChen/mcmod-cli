// File: internal/metadata/metadata_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/metadata/* (jar, fabric, neoforge, identity).

package metadata

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// --- from boost_test.go (Metadata boost) ---
var _ = Describe("Metadata boost", func() {
	It("ReadJarMetadata on nonexistent file returns error", func() {
		_, err := ReadJarMetadata("/nonexistent/mod.jar")
		Expect(err).To(HaveOccurred())
	})

	It("DepInfoFromIdentity with colon", func() {
		info := DepInfoFromIdentity("curseforge:create")
		Expect(info.ModID).To(Equal("create"))
		Expect(info.Required).To(BeTrue())
	})

	It("DepInfoFromIdentity without colon", func() {
		info := DepInfoFromIdentity("simplemod")
		Expect(info.ModID).To(Equal("simplemod"))
	})

	It("SourceIdentity basic", func() {
		info := SourceIdentity("curseforge", "12345")
		Expect(info).To(Equal("curseforge:12345"))
	})

	It("SourceIdentity github returns github:", func() {
		info := SourceIdentity("github-release", "create")
		Expect(info).To(Equal("github:create"))
	})

	It("SourceIdentity default", func() {
		info := SourceIdentity("other", "modid")
		Expect(info).To(Equal("modid"))
	})

	It("InternalIdentity basic", func() {
		info := InternalIdentity("neoforge", "create")
		Expect(info).To(Equal("neoforge:create"))
	})
})

// --- from coverage_test.go (Metadata) ---
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

// writeJarEntry writes a jar containing a single entry with the given
// contents, so the specs below can build jars with one call.
func writeJarEntry(path, name string, contents []byte) {
	f, err := os.Create(path)
	Expect(err).NotTo(HaveOccurred())
	w := zip.NewWriter(f)
	wr, err := w.Create(name)
	Expect(err).NotTo(HaveOccurred())
	_, err = wr.Write(contents)
	Expect(err).NotTo(HaveOccurred())
	Expect(w.Close()).To(Succeed())
	Expect(f.Close()).To(Succeed())
}

// paddedNeoForgeTOML builds a neoforge.mods.toml whose dependency table is
// pushed past the 32KiB mark by `lines` comment lines.
func paddedNeoForgeTOML(lines int) []byte {
	var b strings.Builder
	b.WriteString("modLoader=\"javafml\"\nloaderVersion=\"[1,)\"\nlicense=\"MIT\"\n")
	b.WriteString("[[mods]]\nmodId=\"bigmod\"\nversion=\"1.0.0\"\ndisplayName=\"Big\"\n")
	for i := 0; i < lines; i++ {
		b.WriteString("# padding to push the dependency table past 32KiB\n")
	}
	b.WriteString("[[dependencies.bigmod]]\nmodId=\"jei\"\n" +
		"type=\"required\"\nversionRange=\"[19.0,)\"\n")
	return []byte(b.String())
}

// writeDualMetadataJar writes a jar carrying META-INF/mods.toml and, when
// withNeoForgeFile is set, META-INF/neoforge.mods.toml with identical
// contents, as Forge -> NeoForge migrations do.
func writeDualMetadataJar(path, modID string, withNeoForgeFile bool) {
	body := "modId=\"" + modID + "\"\nversion=\"1.0\"\n" +
		"[[dependencies." + modID + "]]\nmodId=\"missingdep\"\n" +
		"mandatory=true\nversionRange=\"[1.0,)\"\n"
	f, err := os.Create(path)
	Expect(err).NotTo(HaveOccurred())
	w := zip.NewWriter(f)
	names := []string{"META-INF/mods.toml"}
	if withNeoForgeFile {
		names = append(names, "META-INF/neoforge.mods.toml")
	}
	for _, n := range names {
		wr, err := w.Create(n)
		Expect(err).NotTo(HaveOccurred())
		_, err = wr.Write([]byte(body))
		Expect(err).NotTo(HaveOccurred())
	}
	Expect(w.Close()).To(Succeed())
	Expect(f.Close()).To(Succeed())
}

var _ = Describe("jar entry reading", func() {
	Describe("entries larger than 32KiB", func() {
		// A single io.Reader.Read on a deflated zip entry returns at most
		// 32KiB, so a larger entry used to be read only partially. The
		// dependency tables sit at the end of the file and were lost.
		It("reads a neoforge.mods.toml without dropping deps", func() {
			dir := GinkgoT().TempDir()
			jar := filepath.Join(dir, "big.jar")
			body := paddedNeoForgeTOML(2000)
			Expect(len(body)).To(BeNumerically(">", 32*1024))
			writeJarEntry(jar, "META-INF/neoforge.mods.toml", body)

			info, err := ReadNeoForgeMetadata(jar)
			Expect(err).NotTo(HaveOccurred())
			// ModID is deliberately not asserted: identity comes from
			// [[mods]] and is covered by the mod-identity specs.
			Expect(info.Dependencies).To(HaveLen(1))
			Expect(info.Dependencies[0].ModID).To(Equal("jei"))
			Expect(info.Dependencies[0].Ref).To(Equal("[19.0,)"))
		})

		It("reads a fabric.mod.json without dropping deps", func() {
			dir := GinkgoT().TempDir()
			jar := filepath.Join(dir, "big.jar")
			body := []byte(`{"id":"bigf","version":"1.0","pad":"` +
				strings.Repeat("x", 60000) + `","depends":{"jei":"*"}}`)
			Expect(len(body)).To(BeNumerically(">", 32*1024))
			writeJarEntry(jar, "fabric.mod.json", body)

			info, err := ReadFabricMetadata(jar)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.ModID).To(Equal("bigf"))
			Expect(info.Version).To(Equal("1.0"))
			Expect(info.Dependencies).To(HaveLen(1))
			Expect(info.Dependencies[0].ModID).To(Equal("jei"))
		})

		It("reads both small and large entries through the dispatcher", func() {
			dir := GinkgoT().TempDir()
			small := filepath.Join(dir, "small.jar")
			large := filepath.Join(dir, "large.jar")
			writeJarEntry(small, "META-INF/neoforge.mods.toml", paddedNeoForgeTOML(0))
			writeJarEntry(large, "META-INF/neoforge.mods.toml", paddedNeoForgeTOML(2000))

			for name, jar := range map[string]string{"small": small, "large": large} {
				info, err := ReadJarMetadata(jar)
				Expect(err).NotTo(HaveOccurred(), name)
				Expect(info.Dependencies).To(HaveLen(1), name)
			}
		})

		It("readZipEntry returns the full entry for a large file", func() {
			dir := GinkgoT().TempDir()
			jar := filepath.Join(dir, "blob.jar")
			payload := make([]byte, 200000)
			for i := range payload {
				payload[i] = byte(i % 251)
			}
			writeJarEntry(jar, "blob.bin", payload)

			r, err := zip.OpenReader(jar)
			Expect(err).NotTo(HaveOccurred())
			defer r.Close()
			Expect(r.File).To(HaveLen(1))
			got, err := readZipEntry(r.File[0])
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(payload))
		})

		It("readZipEntry handles an empty entry", func() {
			dir := GinkgoT().TempDir()
			jar := filepath.Join(dir, "empty.jar")
			writeJarEntry(jar, "empty.txt", nil)

			r, err := zip.OpenReader(jar)
			Expect(err).NotTo(HaveOccurred())
			defer r.Close()
			Expect(r.File).To(HaveLen(1))
			got, err := readZipEntry(r.File[0])
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(BeEmpty())
		})
	})

	Describe("jars carrying both metadata files", func() {
		// Forge -> NeoForge migrations often ship both files with identical
		// contents. Only one may be parsed, otherwise every dependency is
		// counted twice and each owner is listed twice in the report.
		It("counts each dependency once when both files are present", func() {
			jar := filepath.Join(GinkgoT().TempDir(), "dual.jar")
			writeDualMetadataJar(jar, "both", true)
			info, err := ReadNeoForgeMetadata(jar)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Dependencies).To(HaveLen(1))
			Expect(info.Dependencies[0].ModID).To(Equal("missingdep"))
		})

		It("still reads the legacy mods.toml when it is the only one", func() {
			jar := filepath.Join(GinkgoT().TempDir(), "legacy.jar")
			writeDualMetadataJar(jar, "legacy", false)
			info, err := ReadNeoForgeMetadata(jar)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Dependencies).To(HaveLen(1))
		})

		It("prefers neoforge.mods.toml when both are present but differ", func() {
			dir := GinkgoT().TempDir()
			jar := filepath.Join(dir, "pref.jar")
			f, err := os.Create(jar)
			Expect(err).NotTo(HaveOccurred())
			w := zip.NewWriter(f)
			legacy := "modid=\"from-legacy\"\nversion=\"1.0\"\n"
			modern := "modid=\"from-neoforge\"\nversion=\"2.0\"\n"
			wr, err := w.Create("META-INF/mods.toml")
			Expect(err).NotTo(HaveOccurred())
			_, err = wr.Write([]byte(legacy))
			Expect(err).NotTo(HaveOccurred())
			wr, err = w.Create("META-INF/neoforge.mods.toml")
			Expect(err).NotTo(HaveOccurred())
			_, err = wr.Write([]byte(modern))
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Close()).To(Succeed())
			Expect(f.Close()).To(Succeed())

			info, err := ReadNeoForgeMetadata(jar)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.ModID).To(Equal("from-neoforge"))
			Expect(info.Version).To(Equal("2.0"))
		})

		It("hasZipEntry matches exact names only", func() {
			jar := filepath.Join(GinkgoT().TempDir(), "names.jar")
			writeJarEntry(jar, "META-INF/neoforge.mods.toml", []byte("modid=\"x\"\n"))
			r, err := zip.OpenReader(jar)
			Expect(err).NotTo(HaveOccurred())
			defer r.Close()
			Expect(hasZipEntry(r.File, "META-INF/neoforge.mods.toml")).To(BeTrue())
			Expect(hasZipEntry(r.File, "META-INF/mods.toml")).To(BeFalse())
			Expect(hasZipEntry(r.File, "neoforge.mods.toml")).To(BeFalse())
		})
	})
})
