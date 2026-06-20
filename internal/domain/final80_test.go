// File: internal/domain/final80_test.go
// Created: 2026-06-20
// Description: Push total coverage over 80%.
package domain

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Final80", func() {
	It("VariantKey", func() {
		Expect(VariantKey("neoforge", "21.1")).To(Equal("neoforge:21.1"))
	})
	It("VariantKey no version", func() {
		Expect(VariantKey("fabric", "")).To(Equal("fabric"))
	})
	It("DefaultVariantKey empty", func() {
		Expect(DefaultVariantKey(PackSpec{})).To(Equal(""))
	})
	It("DefaultVariantKey first loader", func() {
		Expect(DefaultVariantKey(PackSpec{LoaderName: []string{"neo:1", "fab:2"}})).To(Equal("neo:1"))
	})
	It("ArtifactBaseName client", func() {
		s := PackSpec{PackName: "p"}
		Expect(ArtifactBaseName(s, "1.21.1", "neo", "21.1", "client")).To(Equal("p-1.21.1-neo-21.1-client"))
	})
	It("ArtifactBaseName server with server name", func() {
		s := PackSpec{PackName: "p", ServerPackName: "srv"}
		Expect(ArtifactBaseName(s, "1.21.1", "neo", "21.1", "server")).To(Equal("srv-1.21.1-neo-21.1-server"))
	})
	It("BaseName uses primary", func() {
		s := PackSpec{PackName: "p", LoaderName: []string{"neo:21.1"}}
		Expect(BaseName(s, "1.21.1", "1.0", "client")).To(Equal("p-1.21.1-neo-21.1-client"))
	})
	It("ParseLoaderName without version", func() {
		n, v := ParseLoaderName("neoforge")
		Expect(n).To(Equal("neoforge"))
		Expect(v).To(Equal(""))
	})
	It("ModsForScope empty returns empty", func() {
		Expect(ModsForScope(PackSpec{}, "shared")).To(BeEmpty())
	})
	It("SetModsForScope creates correct mods", func() {
		spec := SetModsForScope(PackSpec{}, "client", []ModSpec{{Name: "CMod"}})
		Expect(ModsForScope(spec, "client")).To(HaveLen(1))
	})
	It("AllModsForVariant with deps", func() {
		spec := PackSpec{Mods: map[string]ModSpec{"a": {Name: "A"}}}
		spec = SetDependencies(spec, []ModSpec{{Name: "dep"}})
		mods := AllModsForVariant(spec, "neo")
		Expect(mods).To(HaveLen(2))
	})
	It("ReleaseIndex Normalize already set", func() {
		ri := ReleaseIndex{Type: "custom"}
		ri.Normalize()
		Expect(ri.Type).To(Equal("custom"))
	})
	It("EnsureRelease increments", func() {
		ri := ReleaseIndex{}
		ri.EnsureRelease("1.0", "type-a")
		ri.EnsureRelease("2.0", "type-b")
		Expect(ri.Releases).To(HaveLen(2))
	})
	It("DeleteRelease works", func() {
		ri := ReleaseIndex{Releases: []ReleaseRecord{{Version: "1", Type: "t"}}}
		Expect(ri.DeleteRelease("1")).To(BeTrue())
		Expect(ri.DeleteRelease("1")).To(BeFalse())
	})
	It("SetArtifact both", func() {
		r := ReleaseRecord{}
		r.SetArtifact("l", "both", "path")
		Expect(r.ArtifactFor("l", "client")).To(Equal("path"))
		Expect(r.ArtifactFor("l", "server")).To(Equal("path"))
	})
	It("RemoveArtifact unknown loader", func() {
		r := ReleaseRecord{}
		r.RemoveArtifact("unknown", "client") // should not panic
	})
	It("FileNameForURL path with extension", func() {
		Expect(FileNameForURL("https://x.com/f.jar", "f")).To(Equal("f.jar"))
	})
	It("modMapFromSlice empty", func() {
		Expect(modMapFromSlice(nil)).To(BeEmpty())
	})
	It("cloneModMap nil", func() {
		Expect(cloneModMap(nil)).To(BeEmpty())
	})
})
