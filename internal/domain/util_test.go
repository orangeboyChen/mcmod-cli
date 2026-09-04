// File: internal/domain/util_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/domain/util.go (ParseLoaderName, splitLoaderSpec, LoaderEntries, ArtifactBaseName, VariantKey, etc.).

package domain

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// --- from spec_test.go consolidated (Domain splitLoaderSpec) ---
var _ = Describe("Domain splitLoaderSpec", func() {
	It("splits name:version", func() {
		n, v := splitLoaderSpec("neoforge:21.1.219")
		Expect(n).To(Equal("neoforge"))
		Expect(v).To(Equal("21.1.219"))
	})
	It("returns whole string for no colon", func() {
		n, v := splitLoaderSpec("neoforge")
		Expect(n).To(Equal("neoforge"))
		Expect(v).To(Equal(""))
	})
	It("handles empty", func() {
		n, v := splitLoaderSpec("")
		Expect(n).To(Equal(""))
		Expect(v).To(Equal(""))
	})
})

// --- from spec_test.go consolidated (FinalEdge) ---
var _ = Describe("FinalEdge", func() {
	It("MarshalJSON with ClientMods+ServerMods builds mods", func() {
		spec := PackSpec{PackName: "e", MinecraftVersion: "1.21.1", LoaderName: []string{"fabric"}, PackVersion: "1",
			ClientMods: []ModSpec{{Name: "Client", Scope: "client", Source: ModSource{Type: SourceCurseForge, Query: "C"}}},
			ServerMods: []ModSpec{{Name: "Server", Scope: "server", Source: ModSource{Type: SourceCurseForge, Query: "S"}}},
		}
		data, err := spec.MarshalJSON()
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(MatchRegexp(`"mods"`))
	})
	It("MarshalJSON with only SharedMods", func() {
		spec := PackSpec{PackName: "s", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"}, PackVersion: "1",
			SharedMods: []ModSpec{{Name: "Shared", Source: ModSource{Type: SourceCurseForge, Query: "S"}}}}
		data, err := spec.MarshalJSON()
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(MatchRegexp(`"mods"`))
	})
	It("ValidateLock missing loader", func() {
		Expect(ValidateLock(PackLock{MinecraftVersion: "1.21.1", Mods: map[string]LockedMod{}})).To(HaveOccurred())
	})
	It("ValidateReleaseIndex missing release type", func() {
		Expect(ValidateReleaseIndex(ReleaseIndex{Type: "package", PackName: "p", MinecraftVersion: "1.21.1",
			Releases: []ReleaseRecord{{Version: "0.1.0"}}})).To(HaveOccurred())
	})
	It("AllEntriesForVariant empty entriesByMod", func() {
		spec := PackSpec{}
		Expect(AllEntriesForVariant(spec, "k")).To(BeEmpty())
	})
	It("EntriesForMod nil entriesByMod", func() {
		spec := PackSpec{}
		Expect(EntriesForMod(spec, "k")).To(BeNil())
	})
	It("AllMods empty", func() {
		Expect(AllMods(PackSpec{})).To(BeEmpty())
	})
	It("AllEntries empty", func() {
		Expect(AllEntries(PackSpec{})).To(BeEmpty())
	})
})

var _ = Describe("Loader entry helpers", func() {
	It("ParseLoaderName handles name and name:version", func() {
		n, v := ParseLoaderName("neoforge")
		Expect(n).To(Equal("neoforge"))
		Expect(v).To(BeEmpty())

		n, v = ParseLoaderName("neoforge:21.0.0")
		Expect(n).To(Equal("neoforge"))
		Expect(v).To(Equal("21.0.0"))
	})

	It("LoaderEntries returns one entry per loader", func() {
		entries := LoaderEntries(PackSpec{LoaderName: []string{"neoforge:21.0.0", "fabric:0.15"}})
		Expect(entries).To(HaveLen(2))
		Expect(entries[0].Name).To(Equal("neoforge"))
		Expect(entries[1].Name).To(Equal("fabric"))
	})

	It("LoaderEntries preserves all loader entries verbatim", func() {
		entries := LoaderEntries(PackSpec{LoaderName: []string{"neoforge:21.0.0", "neoforge:21.0.0"}})
		Expect(entries).To(HaveLen(2))
	})

	It("PrimaryLoaderName and PrimaryLoaderVersion return the first loader", func() {
		spec := PackSpec{LoaderName: []string{"neoforge:21.0.0"}}
		Expect(PrimaryLoaderName(spec)).To(Equal("neoforge"))
		Expect(PrimaryLoaderVersion(spec)).To(Equal("21.0.0"))
	})

	It("PrimaryLoaderName is empty when no loaders are configured", func() {
		Expect(PrimaryLoaderName(PackSpec{})).To(BeEmpty())
		Expect(PrimaryLoaderVersion(PackSpec{})).To(BeEmpty())
	})

	It("LoaderVersionFor returns the version matching the loader name", func() {
		spec := PackSpec{LoaderName: []string{"neoforge:21.0.0", "fabric:0.15"}}
		Expect(LoaderVersionFor(spec, "neoforge")).To(Equal("21.0.0"))
		Expect(LoaderVersionFor(spec, "fabric")).To(Equal("0.15"))
		Expect(LoaderVersionFor(spec, "quilt")).To(BeEmpty())
	})

	It("LoaderMatches matches either exact or prefixed-with-'-' entries", func() {
		spec := PackSpec{LoaderName: []string{"neoforge:21.0.0"}}
		Expect(LoaderMatches(spec, "neoforge")).To(BeTrue())
		Expect(LoaderMatches(spec, "fabric")).To(BeFalse())
	})

	It("VariantKey and DefaultVariantKey round-trip", func() {
		Expect(VariantKey("neoforge", "21.1")).To(Equal("neoforge:21.1"))
		Expect(DefaultVariantKey(PackSpec{LoaderName: []string{"neoforge:21.1"}})).To(Equal("neoforge:21.1"))
	})

	It("ArtifactBaseName joins pack name and version", func() {
		spec := PackSpec{PackName: "p"}
		Expect(ArtifactBaseName(spec, "1.21.1", "neoforge", "21.0.0", "client")).To(ContainSubstring("p"))
	})

	It("BaseName returns the right form for client and server", func() {
		spec := PackSpec{PackName: "p", ServerPackName: "ps"}
		Expect(BaseName(spec, "1.21.1", "1.0", "client")).To(ContainSubstring("p"))
		Expect(BaseName(spec, "1.21.1", "1.0", "server")).To(ContainSubstring("ps"))
	})
})

var _ = Describe("Mod collection helpers", func() {
	specWithMods := func() PackSpec {
		return PackSpec{Mods: map[string]ModSpec{
			"shared-mod": {Name: "shared-mod", Scope: ScopeShared},
			"client-mod": {Name: "client-mod", Scope: ScopeClient},
			"server-mod": {Name: "server-mod", Scope: ScopeServer},
		}}
	}

	It("ModsForScope returns mods in the requested scope", func() {
		spec := specWithMods()
		shared := ModsForScope(spec, ScopeShared)
		Expect(shared).To(HaveLen(1))
		Expect(shared[0].Name).To(Equal("shared-mod"))
	})

	It("Mods returns every mod across all scopes", func() {
		Expect(Mods(specWithMods())).To(HaveLen(3))
	})

	It("Dependencies returns the top-level spec.Dependencies", func() {
		spec := PackSpec{Dependencies: []ModSpec{{Name: "a"}, {Name: "b"}}}
		deps := Dependencies(spec)
		Expect(deps).To(HaveLen(2))
	})

	It("SetDependencies sets the top-level Dependencies field", func() {
		updated := SetDependencies(PackSpec{}, []ModSpec{{Name: "x"}})
		Expect(updated.Dependencies).To(HaveLen(1))
		Expect(updated.Dependencies[0].Name).To(Equal("x"))
	})

	It("SetModsForScope replaces the entire mod map with the new scope's mods", func() {
		spec := specWithMods()
		updated := SetModsForScope(spec, ScopeClient, []ModSpec{{Name: "new-client"}})
		// The set rewrites the mod map; old shared/server entries are dropped
		// from the map and the new client entry is normalized.
		Expect(updated.Mods).To(HaveLen(1))
		expectKey := NormalizeKey("new-client")
		_, ok := updated.Mods[expectKey]
		Expect(ok).To(BeTrue())
	})

	It("SetMods replaces the entire mod map", func() {
		spec := specWithMods()
		updated := SetMods(spec, []ModSpec{{Name: "only"}})
		Expect(Mods(updated)).To(HaveLen(1))
	})

	It("EntryIndex returns false when no variants are configured", func() {
		_, ok := EntryIndex(PackSpec{}, "jei")
		Expect(ok).To(BeFalse())
	})

	It("EntryIndex returns false for an unknown mod key", func() {
		_, ok := EntryIndex(PackSpec{}, "missing")
		Expect(ok).To(BeFalse())
	})

	It("FileNameForURL derives a filename from a URL", func() {
		Expect(FileNameForURL("https://x.example/mod.jar", "")).To(Equal("mod.jar"))
		Expect(FileNameForURL("https://x.example/path/mod.jar?ver=1", "")).To(ContainSubstring("mod.jar"))
	})

	It("FileNameForURL returns the fallback for an empty URL", func() {
		Expect(FileNameForURL("", "fallback.jar")).To(Equal("fallback.jar"))
	})

	It("ExpandURLPattern fills in placeholders", func() {
		got := ExpandURLPattern("mod-{modId}-{mcVersion}.jar", 1, 2, "m.jar", "1.0", "1.21.1")
		Expect(got).To(ContainSubstring("mod-1-1.21.1.jar"))
	})

	It("ExpandURLPattern produces a ForgeCDN fileId4 layout for long file ids", func() {
		got := ExpandURLPattern("files/{fileId4}/{fileName}", 1, 8240058, "m.jar", "1.0", "1.21.1")
		Expect(got).To(Equal("files/8240/58/m.jar"))
	})

	It("DefaultCurseForgeURL produces the standard edge CDN path", func() {
		Expect(DefaultCurseForgeURL(8263584, "sable-neoforge-1.21.1-2.0.3.jar")).To(Equal("https://edge.forgecdn.net/files/8263/584/sable-neoforge-1.21.1-2.0.3.jar"))
	})
})

var _ = Describe("EntryIndex and AllEntries via Variants", func() {
	It("EntryIndex finds the mod by name in a variant", func() {
		spec := PackSpec{
			Variants: map[string]PackVariantSpec{
				"neoforge:21.0.0": {
					LoaderName: []string{"neoforge:21.0.0"},
					Mods:       []ModSpec{{Name: "jei"}},
				},
			},
		}
		e, ok := EntryIndex(spec, "jei")
		Expect(ok).To(BeTrue())
		Expect(e.Name).To(Equal("jei"))
	})

	It("AllEntries returns all entries across variants when no entriesByMod or Mods are set", func() {
		spec := PackSpec{
			Variants: map[string]PackVariantSpec{
				"neoforge:21.0.0": {Mods: []ModSpec{{Name: "a"}, {Name: "b"}}},
			},
		}
		entries := AllEntries(spec)
		Expect(entries).To(HaveLen(2))
	})
})

var _ = Describe("AllMods and AllModsForVariant branches", func() {
	It("AllMods returns mods from spec.Mods", func() {
		spec := PackSpec{
			Mods: map[string]ModSpec{
				"a": {Name: "A"},
				"b": {Name: "B"},
			},
		}
		mods := AllMods(spec)
		Expect(mods).To(HaveLen(2))
	})

	It("AllMods falls back to variants when Mods is empty", func() {
		spec := PackSpec{
			Variants: map[string]PackVariantSpec{
				"v1": {LoaderName: []string{"neoforge"}, Mods: []ModSpec{{Name: "x"}, {Name: "y"}}},
			},
		}
		mods := AllMods(spec)
		Expect(mods).To(HaveLen(2))
	})

	It("AllModsForVariant returns main mods and dependencies", func() {
		spec := PackSpec{
			Mods:         map[string]ModSpec{"a": {Name: "A"}},
			Dependencies: []ModSpec{{Name: "D"}},
		}
		mods := AllModsForVariant(spec, "v1")
		Expect(len(mods)).To(BeNumerically(">=", 1))
	})

	It("AllModsForVariant returns nil for unknown variant", func() {
		spec := PackSpec{}
		Expect(AllModsForVariant(spec, "v1")).To(BeNil())
	})

	It("SetMods normalizes keys and falls back to Name when key is empty", func() {
		mods := []ModSpec{{Name: "Farmer's Delight"}, {Name: ""}}
		spec := SetMods(PackSpec{}, mods)
		Expect(spec.Mods).To(HaveKey("farmers-delight"))
	})

	It("SetModsForScope sets scope on each mod in the unified map", func() {
		spec := SetModsForScope(PackSpec{}, ScopeClient, []ModSpec{{Name: "B"}})
		Expect(spec.Mods).To(HaveKey("b"))
		Expect(spec.Mods["b"].Scope).To(Equal(ScopeClient))
	})

	It("ModsForScope returns empty slice for unknown scope", func() {
		spec := PackSpec{}
		Expect(ModsForScope(spec, "unknown")).To(BeEmpty())
	})

	It("FileNameForURL extracts last segment", func() {
		Expect(FileNameForURL("https://example.com/path/file.jar", "")).To(Equal("file.jar"))
	})

	It("FileNameForURL with empty input returns fallback", func() {
		Expect(FileNameForURL("", "fallback")).To(Equal("fallback"))
	})
})

var _ = Describe("EntryIndex and entriesByMod", func() {
	It("EntryIndex returns entry from entriesByMod when present", func() {
		spec := PackSpec{}
		spec = SetEntriesForMod(spec, "k", []EntrySpec{{Name: "k"}})
		e, ok := EntryIndex(spec, "k")
		Expect(ok).To(BeTrue())
		Expect(e.Name).To(Equal("k"))
	})

	It("EntryIndex returns false when no match", func() {
		spec := PackSpec{Mods: map[string]ModSpec{"x": {Name: "x"}}}
		_, ok := EntryIndex(spec, "missing")
		Expect(ok).To(BeFalse())
	})

	It("modMapFromSlice handles duplicates by keeping last", func() {
		mods := []ModSpec{{Name: "A", Scope: ScopeShared}, {Name: "A", Scope: ScopeClient}}
		m := modMapFromSlice(mods)
		Expect(m).To(HaveLen(1))
		Expect(m["a"].Scope).To(Equal(ScopeClient))
	})
})
