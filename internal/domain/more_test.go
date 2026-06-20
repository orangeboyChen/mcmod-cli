// File: internal/domain/more_test.go
// Created: 2026-06-20
// Description: Push domain coverage past 80%.
package domain

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("More domain", func() {
	It("DefaultVariantKey for single loader", func() {
		Expect(DefaultVariantKey(PackSpec{LoaderName: []string{"n:1"}})).To(Equal("n:1"))
	})

	It("LoaderEntries with no versions", func() {
		e := LoaderEntries(PackSpec{LoaderName: []string{"neoforge", "fabric"}})
		Expect(e).To(HaveLen(2))
		Expect(e[0].Name).To(Equal("neoforge"))
		Expect(e[0].Version).To(BeEmpty())
	})

	It("Single ModSpec scope default via ModsForScope", func() {
		spec := PackSpec{Mods: map[string]ModSpec{"a": {Name: "A"}}}
		mods := ModsForScope(spec, ScopeShared)
		Expect(mods).To(HaveLen(1))
		Expect(mods[0].Name).To(Equal("A"))
	})

	It("SetModsForScope replaces scoped mods", func() {
		spec := PackSpec{Mods: map[string]ModSpec{"a": {Name: "A", Scope: ScopeClient}}}
		spec = SetModsForScope(spec, ScopeClient, []ModSpec{{Name: "B"}})
		Expect(ModsForScope(spec, ScopeClient)).To(HaveLen(1))
		Expect(ModsForScope(spec, ScopeClient)[0].Name).To(Equal("B"))
	})

	It("SetMods with name creates expected key", func() {
		spec := SetMods(PackSpec{}, []ModSpec{{Name: "Test"}})
		Expect(spec.Mods["test"].Name).To(Equal("Test"))
	})

	It("AllModsForVariant with Variant mods", func() {
		spec := PackSpec{
			Mods: map[string]ModSpec{"a": {Name: "A"}},
			Variants: map[string]PackVariantSpec{
				"neo": {Mods: []ModSpec{{Name: "VA"}}},
			},
		}
		mods := AllModsForVariant(spec, "neo")
		Expect(mods).To(HaveLen(1))
		Expect(mods[0].Name).To(Equal("A"))
	})

	It("AllEntriesForVariant returns entries from entriesByMod", func() {
		spec := PackSpec{}
		spec.entriesByMod = map[string][]EntrySpec{"k": {{Name: "e1"}}}
		Expect(AllEntriesForVariant(spec, "k")).To(HaveLen(1))
	})

	It("FileNameForURL with URL containing path", func() {
		fn := FileNameForURL("https://example.com/path/to/file.jar", "fall")
		Expect(fn).To(Equal("file.jar"))
	})

	It("UnmarshalJSON handles bad data", func() {
		var s PackSpec
		err := s.UnmarshalJSON([]byte("not json"))
		Expect(err).To(HaveOccurred())
	})

	It("MarshalJSON with no Mods omits field", func() {
		spec := PackSpec{PackName: "n", PackVersion: "1", MinecraftVersion: "1.21.1", LoaderName: []string{"fabric"}}
		data, err := spec.MarshalJSON()
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).NotTo(ContainSubstring(`"mods"`))
	})

	It("SelectedAssetPattern with AssetPatternByLoader returns first for unknown", func() {
		src := ModSource{
			AssetPatternByLoader: map[string]string{
				"neoforge": "neo.jar",
				"fabric":   "fabric.jar",
			},
		}
		Expect(src.SelectedAssetPattern("quilt")).NotTo(BeEmpty())
	})

	It("EnsureRelease sets release type", func() {
		ri := ReleaseIndex{}
		r := ri.EnsureRelease("1.0", "github-release")
		Expect(r.Type).To(Equal("github-release"))
	})

	It("SetArtifact with TargetBoth", func() {
		r := ReleaseRecord{}
		r.SetArtifact("neo", "both", "both.zip")
		Expect(r.ArtifactFor("neo", "client")).To(Equal("both.zip"))
		Expect(r.ArtifactFor("neo", "server")).To(Equal("both.zip"))
	})

	It("ArtifactFor unknown loader returns empty", func() {
		r := ReleaseRecord{}
		Expect(r.ArtifactFor("unk", "client")).To(BeEmpty())
	})

	It("RemoveArtifact removes server only", func() {
		r := ReleaseRecord{Artifact: map[string]ReleaseArtifactSet{"l": {Client: "c", Server: "s"}}}
		r.RemoveArtifact("l", "server")
		Expect(r.ArtifactFor("l", "server")).To(BeEmpty())
		Expect(r.ArtifactFor("l", "client")).To(Equal("c"))
	})

	It("EnsureRelease appends when version new", func() {
		ri := ReleaseIndex{Releases: []ReleaseRecord{{Version: "1.0", Type: "release"}}}
		r := ri.EnsureRelease("2.0", "release")
		Expect(r.Version).To(Equal("2.0"))
		Expect(ri.Releases).To(HaveLen(2))
	})

	It("ReleaseIndex DeleteRelease returns false for missing", func() {
		ri := ReleaseIndex{}
		Expect(ri.DeleteRelease("missing")).To(BeFalse())
	})

	It("EntryIndex searches entriesByMod for name", func() {
		spec := PackSpec{}
		spec.entriesByMod = map[string][]EntrySpec{"a": {{Name: "entry1"}}}
		e, ok := EntryIndex(spec, "entry1")
		Expect(ok).To(BeTrue())
		Expect(e.Name).To(Equal("entry1"))
	})

	It("AllEntries returns entries from entriesByMod", func() {
		spec := PackSpec{}
		spec.entriesByMod = map[string][]EntrySpec{"k": {{Name: "e1"}}}
		Expect(AllEntries(spec)).To(HaveLen(1))
	})

	It("AllMods with only variants returns variant mods", func() {
		spec := PackSpec{
			Variants: map[string]PackVariantSpec{"v": {Mods: []ModSpec{{Name: "VM"}}}},
		}
		mods := AllMods(spec)
		Expect(mods).To(HaveLen(1))
		Expect(mods[0].Name).To(Equal("VM"))
	})
})
