// File: internal/domain/spec_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/domain/spec.go (PackSpec, ModSpec, ModSource, PackVariantSpec).

package domain

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// --- from spec_test_58.go (Model) ---
var _ = Describe("Model", func() {
	Describe("NormalizeModKey", func() {
		It("converts Farmer's Delight to farmers-delight", func() {
			Expect(NormalizeModKey("Farmer's Delight")).To(Equal("farmers-delight"))
		})
		It("converts Brewin' And Chewin' to brewin-and-chewin", func() {
			Expect(NormalizeModKey("Brewin' And Chewin'")).To(Equal("brewin-and-chewin"))
		})
		It("converts Create Crafts & Additions to create-crafts-additions", func() {
			Expect(NormalizeModKey("Create Crafts & Additions")).To(Equal("create-crafts-additions"))
		})
		It("converts Greenhouse Config to greenhouse-config", func() {
			Expect(NormalizeModKey("Greenhouse Config")).To(Equal("greenhouse-config"))
		})
	})

	Describe("NormalizeKey", func() {
		It("also converts properly", func() {
			Expect(NormalizeKey("Farmer's Delight")).To(Equal("farmers-delight"))
		})
	})

	Describe("PackSpec JSON round-trip", func() {
		It("preserves loaderName array", func() {
			data := []byte(`{"packName":"demo","minecraftVersion":"1.21.1","loaderName":["neoforge","fabric"],"loaderVersion":"21.1.219","packVersion":"0.1.0"}`)
			var spec PackSpec
			err := json.Unmarshal(data, &spec)
			Expect(err).NotTo(HaveOccurred())
			Expect(spec.LoaderName).To(HaveLen(2))
			Expect(spec.LoaderName[0]).To(Equal("neoforge"))
			Expect(spec.LoaderNameIsArray).To(BeTrue())
		})

		It("marshal produces unified mods map", func() {
			spec := PackSpec{
				PackName:         "demo",
				MinecraftVersion: "1.21.1",
				LoaderName:       []string{"neoforge"},
				PackVersion:      "0.1.0",
				SharedMods: []ModSpec{{
					Name:  "Farmer's Delight",
					Scope: ScopeShared,
					Source: ModSource{
						Type:  SourceCurseForge,
						Query: "Farmer's Delight",
					},
				}},
			}
			data, err := json.Marshal(spec)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(ContainSubstring(`"mods":{"farmers-delight"`))
			Expect(string(data)).NotTo(ContainSubstring("sharedMods"))
		})

		It("handles ModSource AssetPatternByLoader", func() {
			src := ModSource{
				Type: SourceGitHubRelease,
				AssetPatternByLoader: map[string]string{
					"fabric":   "demo-fabric.jar",
					"neoforge": "demo-neoforge.jar",
				},
			}
			data, err := json.Marshal(src)
			Expect(err).NotTo(HaveOccurred())

			var decoded ModSource
			err = json.Unmarshal(data, &decoded)
			Expect(err).NotTo(HaveOccurred())
			Expect(decoded.SelectedAssetPattern("neoforge")).To(Equal("demo-neoforge.jar"))
			Expect(decoded.SelectedAssetPattern("fabric")).To(Equal("demo-fabric.jar"))
			Expect(decoded.SelectedAssetPattern("quilt")).NotTo(BeEmpty())
		})
	})

	Describe("Loader helpers", func() {
		It("LoaderEntries returns parsed entries", func() {
			spec := PackSpec{LoaderName: []string{"neoforge:21.1.219", "fabric:1.21.123"}}
			entries := LoaderEntries(spec)
			Expect(entries).To(HaveLen(2))
			Expect(entries[0].Name).To(Equal("neoforge"))
			Expect(entries[0].Version).To(Equal("21.1.219"))
			Expect(entries[1].Name).To(Equal("fabric"))
			Expect(entries[1].Version).To(Equal("1.21.123"))
		})

		It("PrimaryLoaderName and version", func() {
			spec := PackSpec{LoaderName: []string{"neoforge:21.1.219", "fabric:1.21.123"}}
			Expect(PrimaryLoaderName(spec)).To(Equal("neoforge"))
			Expect(PrimaryLoaderVersion(spec)).To(Equal("21.1.219"))
			Expect(PrimaryLoaderName(PackSpec{})).To(Equal(""))
		})

		It("LoaderVersionFor", func() {
			spec := PackSpec{LoaderName: []string{"neoforge:21.1.219"}}
			Expect(LoaderVersionFor(spec, "fabric")).To(Equal(""))
			Expect(LoaderVersionFor(spec, "neoforge")).To(Equal("21.1.219"))
		})

		It("LoaderMatches", func() {
			spec := PackSpec{LoaderName: []string{"neoforge"}}
			Expect(LoaderMatches(spec, "neoforge")).To(BeTrue())
			Expect(LoaderMatches(spec, "quilt")).To(BeFalse())
		})

		It("ParseLoaderName splits correctly", func() {
			n, v := ParseLoaderName("neoforge:21.1.219")
			Expect(n).To(Equal("neoforge"))
			Expect(v).To(Equal("21.1.219"))
			n, v = ParseLoaderName("fabric")
			Expect(n).To(Equal("fabric"))
			Expect(v).To(Equal(""))
		})
	})

	Describe("ModsForScope and Mods helpers", func() {
		It("ModsForScope returns mods by scope", func() {
			spec := PackSpec{
				Mods: map[string]ModSpec{
					"a": {Name: "A", Scope: ScopeShared},
					"b": {Name: "B", Scope: ScopeClient},
				},
			}
			shared := ModsForScope(spec, ScopeShared)
			Expect(shared).To(HaveLen(1))
			Expect(shared[0].Name).To(Equal("A"))
		})

		It("Mods returns all mods", func() {
			spec := PackSpec{Mods: map[string]ModSpec{"a": {Name: "A"}}}
			Expect(Mods(spec)).To(HaveLen(1))
		})

		It("SetMods replaces all mods", func() {
			spec := SetMods(PackSpec{}, []ModSpec{{Name: "One", Scope: ScopeShared, Source: ModSource{Type: SourceCurseForge, Query: "One"}}})
			Expect(spec.Mods).To(HaveLen(1))
			Expect(spec.Mods["one"].Name).To(Equal("One"))
		})
	})

	Describe("Release index", func() {
		It("Normalize sets type to package", func() {
			idx := ReleaseIndex{}
			idx.Normalize()
			Expect(idx.Type).To(Equal("package"))
		})

		It("EnsureRelease creates or finds records", func() {
			idx := ReleaseIndex{Type: "package"}
			r := idx.EnsureRelease("0.1.0", "github-release")
			Expect(r.Version).To(Equal("0.1.0"))
			Expect(idx.Releases).To(HaveLen(1))

			// EnsureRelease of existing returns same version
			r2 := idx.EnsureRelease("0.1.0", "snapshot")
			Expect(r2.Version).To(Equal("0.1.0"))
			Expect(idx.Releases).To(HaveLen(1))
		})

		It("SetArtifact and ArtifactFor work correctly", func() {
			idx := ReleaseIndex{}
			r := idx.EnsureRelease("1.0", "release")
			r.SetArtifact("neoforge", "client", "releases/v1.0/client.zip")
			r.SetArtifact("neoforge", "server", "releases/v1.0/server.zip")
			Expect(r.ArtifactFor("neoforge", "client")).To(Equal("releases/v1.0/client.zip"))
			Expect(r.ArtifactFor("neoforge", "server")).To(Equal("releases/v1.0/server.zip"))
			Expect(r.ArtifactFor("neoforge", "both")).To(Equal("releases/v1.0/client.zip"))
		})

		It("RemoveArtifact clears a target", func() {
			idx := ReleaseIndex{}
			r := idx.EnsureRelease("1.0", "release")
			r.SetArtifact("neoforge", "client", "client.zip")
			r.RemoveArtifact("neoforge", "client")
			Expect(r.ArtifactFor("neoforge", "client")).To(Equal(""))
		})

		It("DeleteRelease removes a release", func() {
			idx := ReleaseIndex{Releases: []ReleaseRecord{{Version: "0.1.0", Type: "release"}}}
			Expect(idx.DeleteRelease("0.1.0")).To(BeTrue())
			Expect(idx.Releases).To(BeEmpty())
			Expect(idx.DeleteRelease("0.1.0")).To(BeFalse())
		})
	})

	Describe("VariantKey, ArtifactBaseName, BaseName", func() {
		It("VariantKey includes version", func() {
			Expect(VariantKey("neoforge", "21.1.219")).To(Equal("neoforge:21.1.219"))
		})
		It("VariantKey without version", func() {
			Expect(VariantKey("neoforge", "")).To(Equal("neoforge"))
		})
		It("ArtifactBaseName builds correct name", func() {
			spec := PackSpec{PackName: "pack", ServerPackName: "server-pack"}
			name := ArtifactBaseName(spec, "1.21.1", "neoforge", "21.1", "client")
			Expect(name).To(Equal("pack-1.21.1-neoforge-21.1-client"))
		})
		It("ArtifactBaseName uses server name for server target", func() {
			spec := PackSpec{PackName: "pack", ServerPackName: "srv-pack"}
			name := ArtifactBaseName(spec, "1.21.1", "neoforge", "21.1", "server")
			Expect(name).To(Equal("srv-pack-1.21.1-neoforge-21.1-server"))
		})
		It("BaseName uses primary loader", func() {
			spec := PackSpec{PackName: "test", LoaderName: []string{"fabric:1.21.123"}}
			Expect(BaseName(spec, "1.21.1", "0.1.0", "client")).To(Equal("test-1.21.1-fabric-1.21.123-client"))
		})
	})

	Describe("Dependencies helpers", func() {
		It("SetDependencies updates dependencies", func() {
			spec := SetDependencies(PackSpec{}, []ModSpec{{Name: "dep", Source: ModSource{Type: SourceCurseForge, Query: "dep"}}})
			Expect(spec.Dependencies).To(HaveLen(1))
			Expect(Dependencies(spec)).To(HaveLen(1))
		})
	})

	Describe("EntriesForMod", func() {
		It("returns entries by mod key", func() {
			spec := SetEntriesForMod(PackSpec{}, "mod1", []EntrySpec{{Name: "e1", ArtifactName: "e1.zip", Target: "both"}})
			entries := EntriesForMod(spec, "mod1")
			Expect(entries).To(HaveLen(1))
			Expect(entries[0].Name).To(Equal("e1"))
		})
	})

	Describe("EntryIndex", func() {
		It("finds entry in entriesByMod", func() {
			spec := SetEntriesForMod(PackSpec{}, "mod1", []EntrySpec{{Name: "entry", ArtifactName: "entry.zip"}})
			e, ok := EntryIndex(spec, "entry")
			Expect(ok).To(BeTrue())
			Expect(e.ArtifactName).To(Equal("entry.zip"))
		})
		It("returns false for missing", func() {
			_, ok := EntryIndex(PackSpec{}, "nonexistent")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("AllMods", func() {
		It("returns mods when Mods is populated", func() {
			spec := PackSpec{Mods: map[string]ModSpec{"a": {Name: "A"}}}
			Expect(AllMods(spec)).To(HaveLen(1))
		})
	})

	Describe("AllEntries", func() {
		It("returns all entries", func() {
			spec := SetEntriesForMod(PackSpec{}, "k", []EntrySpec{{Name: "e1"}})
			Expect(AllEntries(spec)).To(HaveLen(1))
		})
	})

	Describe("AllModsForVariant", func() {
		It("includes deps and mods ordered", func() {
			spec := PackSpec{
				Mods: map[string]ModSpec{
					"alpha": {Name: "Alpha", Scope: ScopeShared},
					"beta":  {Name: "Beta", Scope: ScopeShared},
				},
			}
			mods := AllModsForVariant(spec, "neoforge")
			Expect(mods).To(HaveLen(2))
		})
	})

	Describe("FileNameForURL", func() {
		It("extracts file name and query stripping", func() {
			Expect(FileNameForURL("https://ex.com/file.jar?q=1", "fall")).To(Equal("file.jar"))
		})
		It("returns fallback for empty URL", func() {
			Expect(FileNameForURL("", "fallback")).To(Equal("fallback"))
		})
	})

	Describe("UnmarshalJSON legacy loaderName", func() {
		It("handles string loaderName", func() {
			data := []byte(`{"packName":"test","packVersion":"1","minecraftVersion":"1.21.1","loaderName":"neoforge:21.1"}`)
			var spec PackSpec
			err := spec.UnmarshalJSON(data)
			Expect(err).NotTo(HaveOccurred())
			Expect(spec.LoaderName).To(HaveLen(1))
			Expect(spec.LoaderName[0]).To(Equal("neoforge:21.1"))
			Expect(spec.LoaderNameIsArray).To(BeFalse())
		})
		It("handles no loaderName field", func() {
			data := []byte(`{"packName":"test","packVersion":"1","minecraftVersion":"1.21.1"}`)
			var spec PackSpec
			err := spec.UnmarshalJSON(data)
			Expect(err).NotTo(HaveOccurred())
			Expect(spec.LoaderName).To(BeEmpty())
		})
	})
})

// --- from spec_test_58.go (Domain ModSource UnmarshalJSON additional cases) ---
var _ = Describe("Domain ModSource UnmarshalJSON additional cases", func() {
	It("accepts assetPattern as object", func() {
		data := []byte(`{"type":"github-release","repo":"o/r","tag":"v1","assetPattern":{"neoforge":"x.jar"}}`)
		var ms ModSource
		Expect(json.Unmarshal(data, &ms)).To(Succeed())
		Expect(ms.AssetPattern).To(Equal(""))
		Expect(ms.AssetPatternByLoader).To(HaveKeyWithValue("neoforge", "x.jar"))
	})
	It("accepts assetPattern as string", func() {
		data := []byte(`{"type":"github-release","repo":"o/r","tag":"v1","assetPattern":"x.jar"}`)
		var ms ModSource
		Expect(json.Unmarshal(data, &ms)).To(Succeed())
		Expect(ms.AssetPattern).To(Equal("x.jar"))
	})
	It("parses with fileName field", func() {
		data := []byte(`{"type":"curseforge","fileName":"a.jar","modId":1,"fileId":2}`)
		var ms ModSource
		Expect(json.Unmarshal(data, &ms)).To(Succeed())
		Expect(ms.FileName).To(Equal("a.jar"))
		Expect(ms.ModID).To(Equal(1))
	})
	It("falls through when assetPattern key absent", func() {
		data := []byte(`{"type":"local","path":"./x.jar"}`)
		var ms ModSource
		Expect(json.Unmarshal(data, &ms)).To(Succeed())
		Expect(ms.Path).To(Equal("./x.jar"))
	})
})

var _ = Describe("ModSource URL helpers", func() {
	It("SelectedAssetPattern returns the per-loader override when present", func() {
		s := ModSource{AssetPattern: "default.jar", AssetPatternByLoader: map[string]string{
			"neoforge": "neoforge.jar",
		}}
		Expect(s.SelectedAssetPattern("neoforge")).To(Equal("neoforge.jar"))
	})

	It("SelectedAssetPattern falls back to the first map entry for unknown loaders", func() {
		s := ModSource{AssetPatternByLoader: map[string]string{
			"neoforge": "neoforge.jar",
		}}
		Expect(s.SelectedAssetPattern("fabric")).To(Equal("neoforge.jar"))
	})

	It("SelectedAssetPattern returns AssetPattern when no map is set", func() {
		s := ModSource{AssetPattern: "default.jar"}
		Expect(s.SelectedAssetPattern("neoforge")).To(Equal("default.jar"))
	})

	It("RenderURL returns the literal URL when set", func() {
		s := ModSource{URL: "https://literal/x.jar"}
		Expect(s.RenderURL(0, 0, "", "", "")).To(Equal("https://literal/x.jar"))
	})

	It("RenderURL expands the URLPattern when no literal URL is set", func() {
		s := ModSource{URLPattern: "https://x/{mcVersion}/m.jar"}
		Expect(s.RenderURL(0, 0, "m.jar", "1.0", "1.21.1")).To(Equal("https://x/1.21.1/m.jar"))
	})

	It("RenderURL returns empty when neither URL nor URLPattern is set", func() {
		s := ModSource{}
		Expect(s.RenderURL(0, 0, "", "", "")).To(BeEmpty())
	})
})

var _ = Describe("ModSource UnmarshalJSON", func() {
	It("unmarshals with no assetPattern via the primary path", func() {
		data := []byte(`{"type":"local","path":"./x.jar","fileName":"x.jar"}`)
		var s ModSource
		Expect(json.Unmarshal(data, &s)).To(Succeed())
		Expect(s.Type).To(Equal("local"))
		Expect(s.Path).To(Equal("./x.jar"))
	})

	It("unmarshals assetPattern as a string fallback", func() {
		data := []byte(`{"type":"github-release","repo":"o/r","tag":"v1","assetPattern":"single.jar"}`)
		var s ModSource
		Expect(json.Unmarshal(data, &s)).To(Succeed())
		Expect(s.AssetPattern).To(Equal("single.jar"))
	})

	It("unmarshals assetPatternByLoader object fallback", func() {
		data := []byte(`{"type":"github-release","repo":"o/r","tag":"v1","assetPattern":{"fabric":"f.jar","neoforge":"n.jar"}}`)
		var s ModSource
		Expect(json.Unmarshal(data, &s)).To(Succeed())
		Expect(s.AssetPatternByLoader["fabric"]).To(Equal("f.jar"))
		Expect(s.AssetPatternByLoader["neoforge"]).To(Equal("n.jar"))
	})
})
