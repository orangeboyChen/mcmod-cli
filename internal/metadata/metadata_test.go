// File: internal/metadata/metadata_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/metadata/* (jar, fabric, neoforge, identity).

package metadata

import (
	"archive/zip"
	"os"
	"path/filepath"

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

	Describe("firstTOMLTableValue", func() {
		// neooforge.mods.toml declares the mod's own id under [[mods]] and
		// every dependency's id under [[dependencies.<modId>]], so both share
		// the key name modId. Only the [[mods]] value identifies the jar.
		const realistic = `modLoader="javafml"
loaderVersion="[1,)"
license="MIT"
[[mods]]
modId="examplemod"
version="1.0.0"
displayName="Example"
[[dependencies.examplemod]]
modId="neoforge"
type="required"
versionRange="[21.0.0,)"
[[dependencies.examplemod]]
modId="jei"
type="required"
versionRange="[19.0.0,)"
`

		It("reads the mod id from [[mods]] and ignores dependency ids", func() {
			value, ok := firstTOMLTableValue([]byte(realistic), "modid", "modId")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("examplemod"))
		})

		It("matches the lowercase modid spelling", func() {
			value, ok := firstTOMLTableValue([]byte("[[mods]]\nmodid=\"lower\"\n"), "modid", "modId")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("lower"))
		})

		It("accepts a single-bracket table header", func() {
			value, ok := firstTOMLTableValue([]byte("[mods]\nmodId=\"single\"\n"), "modId")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("single"))
		})

		It("matches a nested table name", func() {
			value, ok := firstTOMLTableValue([]byte("[[mods.sub]]\nmodId=\"nested\"\n"), "modId")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("nested"))
		})

		It("returns false when the table is absent", func() {
			_, ok := firstTOMLTableValue([]byte("[[other]]\nmodId=\"nope\"\n"), "modId")
			Expect(ok).To(BeFalse())
		})

		It("returns false when the key is absent from the table", func() {
			_, ok := firstTOMLTableValue([]byte("[[mods]]\nversion=\"1.0\"\n"), "modId")
			Expect(ok).To(BeFalse())
		})

		It("ignores the key once a later table begins", func() {
			// The dependency table also sets modId; it must not be picked up
			// once we have left [[mods]].
			body := "[[mods]]\nversion=\"1.0\"\n[[dependencies.x]]\nmodId=\"dep\"\n"
			_, ok := firstTOMLTableValue([]byte(body), "modId")
			Expect(ok).To(BeFalse())
		})

		It("skips comment lines inside the table", func() {
			body := "[[mods]]\n# modId=\"commented\"\nmodId=\"real\"\n"
			value, ok := firstTOMLTableValue([]byte(body), "modId")
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("real"))
		})
	})

	Describe("ReadNeoForgeMetadata with dependencies", func() {
		It("keeps the [[mods]] id and reports every dependency", func() {
			dir := GinkgoT().TempDir()
			jar := filepath.Join(dir, "with-deps.jar")
			f, err := os.Create(jar)
			Expect(err).NotTo(HaveOccurred())
			w := zip.NewWriter(f)
			body := `modLoader="javafml"
loaderVersion="[1,)"
[[mods]]
modId="examplemod"
version="1.0.0"
[[dependencies.examplemod]]
modId="neoforge"
type="required"
versionRange="[21.0.0,)"
[[dependencies.examplemod]]
modId="jei"
type="required"
versionRange="[19.0.0,)"
`
			wr, err := w.Create("META-INF/neoforge.mods.toml")
			Expect(err).NotTo(HaveOccurred())
			_, err = wr.Write([]byte(body))
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Close()).To(Succeed())
			Expect(f.Close()).To(Succeed())

			info, err := ReadNeoForgeMetadata(jar)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.ModID).To(Equal("examplemod"))
			Expect(info.Version).To(Equal("1.0.0"))
			Expect(info.Dependencies).To(HaveLen(2))
			Expect(info.Dependencies[0].ModID).To(Equal("neoforge"))
			Expect(info.Dependencies[0].Required).To(BeTrue())
			Expect(info.Dependencies[1].ModID).To(Equal("jei"))
			Expect(info.Dependencies[1].Ref).To(Equal("[19.0.0,)"))
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
