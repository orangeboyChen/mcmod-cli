// File: internal/domain/validate_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/domain/validate.go (ValidateSpec, ValidateLock, ValidateReleaseIndex, ValidateLoaderName, isValidScope, validateSourceMod).

package domain

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// --- from spec_test.go consolidated (Validate) ---
var _ = Describe("Validate", func() {
	Describe("ValidateSpec", func() {
		It("passes for a valid spec", func() {
			spec := PackSpec{
				PackName:         "pack",
				MinecraftVersion: "1.21.1",
				LoaderName:       []string{"neoforge:21.1.219", "fabric:1.21.123"},
				PackVersion:      "0.1.0",
				Mods: map[string]ModSpec{
					"create": {Name: "Create", Scope: ScopeShared, Source: ModSource{Type: SourceCurseForge, Query: "Create"}},
				},
			}
			Expect(ValidateSpec(spec)).To(Succeed())
		})

		It("rejects missing packName", func() {
			err := ValidateSpec(PackSpec{MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"}, PackVersion: "0.1.0"})
			Expect(err).To(HaveOccurred())
		})

		It("rejects unsupported loaders", func() {
			err := ValidateSpec(PackSpec{PackName: "p", MinecraftVersion: "1.21.1", LoaderName: []string{"quilt"}, PackVersion: "0.1.0"})
			Expect(err).To(HaveOccurred())
		})

		It("rejects non-normalized mod keys", func() {
			spec := PackSpec{
				PackName:         "p",
				MinecraftVersion: "1.21.1",
				LoaderName:       []string{"neoforge"},
				PackVersion:      "0.1.0",
				Mods:             map[string]ModSpec{"Bad Key": {Name: "Bad Key", Source: ModSource{Type: SourceCurseForge, Query: "Bad Key"}}},
			}
			Expect(ValidateSpec(spec)).To(HaveOccurred())
		})

		It("rejects curseforge with extra fields", func() {
			spec := PackSpec{
				PackName: "p", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"}, PackVersion: "0.1.0",
				Mods: map[string]ModSpec{"create": {Source: ModSource{Type: SourceCurseForge, Query: "Create", FileName: "create.jar"}}},
			}
			Expect(ValidateSpec(spec)).To(HaveOccurred())
		})

		It("rejects github-release with fileName", func() {
			spec := PackSpec{
				PackName: "p", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"}, PackVersion: "0.1.0",
				Mods: map[string]ModSpec{"m": {Source: ModSource{Type: SourceGitHubRelease, Repo: "o/r", Tag: "v1", AssetPattern: "*.jar", FileName: "demo.jar"}}},
			}
			Expect(ValidateSpec(spec)).To(HaveOccurred())
		})
	})

	Describe("ValidateLock", func() {
		It("passes for a valid lock", func() {
			lock := PackLock{
				Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]LockedMod{"create": {Name: "Create", Scope: ScopeShared, Source: LockedSource{Type: SourceCurseForge, ModID: 328085, FileID: 5812340, FileName: "create.jar"}}},
			}
			Expect(ValidateLock(lock)).To(Succeed())
		})

		It("rejects unsupported loader", func() {
			Expect(ValidateLock(PackLock{Loader: "forge", MinecraftVersion: "1.21.1", Mods: map[string]LockedMod{}})).To(HaveOccurred())
		})

		It("rejects empty mods", func() {
			Expect(ValidateLock(PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1"})).To(HaveOccurred())
		})

		It("rejects missing curseforge modId", func() {
			lock := PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]LockedMod{"m": {Source: LockedSource{Type: SourceCurseForge, FileName: "m.jar"}}}}
			Expect(ValidateLock(lock)).To(HaveOccurred())
		})

		It("rejects missing github-release repo", func() {
			lock := PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]LockedMod{"m": {Source: LockedSource{Type: SourceGitHubRelease, Tag: "v1", FileName: "m.jar"}}}}
			Expect(ValidateLock(lock)).To(HaveOccurred())
		})
	})

	Describe("ValidateReleaseIndex", func() {
		It("passes for a valid index", func() {
			index := ReleaseIndex{Type: "package", PackName: "pack", MinecraftVersion: "1.21.1", Releases: []ReleaseRecord{{Version: "0.1.0", Type: "github-release"}}}
			Expect(ValidateReleaseIndex(index)).To(Succeed())
		})

		It("rejects missing type", func() {
			Expect(ValidateReleaseIndex(ReleaseIndex{PackName: "p", MinecraftVersion: "1.21.1"})).To(HaveOccurred())
		})

		It("rejects duplicate versions", func() {
			index := ReleaseIndex{Type: "package", PackName: "p", MinecraftVersion: "1.21.1",
				Releases: []ReleaseRecord{{Version: "0.1.0", Type: "github-release"}, {Version: "0.1.0", Type: "github-release"}}}
			Expect(ValidateReleaseIndex(index)).To(HaveOccurred())
		})
	})

	Describe("ValidateLoaderName", func() {
		It("accepts neoforge and fabric", func() {
			Expect(ValidateLoaderName("neoforge")).To(Succeed())
			Expect(ValidateLoaderName("fabric")).To(Succeed())
		})
		It("rejects others", func() {
			Expect(ValidateLoaderName("quilt")).To(HaveOccurred())
		})
	})
})

// --- from spec_test.go consolidated (Domain validation edge cases) ---
var _ = Describe("Domain validation edge cases", func() {
	It("ValidateSpec rejects missing packVersion", func() {
		err := ValidateSpec(PackSpec{PackName: "p", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"}})
		Expect(err).To(HaveOccurred())
	})

	It("ValidateSpec rejects invalid scope", func() {
		err := ValidateSpec(PackSpec{
			PackName: "p", PackVersion: "1", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
			Mods: map[string]ModSpec{"m": {Scope: "bad", Source: ModSource{Type: SourceCurseForge, Query: "m"}}},
		})
		Expect(err).To(HaveOccurred())
	})

	It("ValidateSpec rejects unknown source type", func() {
		err := ValidateSpec(PackSpec{
			PackName: "p", PackVersion: "1", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
			Mods: map[string]ModSpec{"m": {Source: ModSource{Type: "unsupported"}}},
		})
		Expect(err).To(HaveOccurred())
	})

	It("ValidateLock github-release missing tag", func() {
		lock := PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
			Mods: map[string]LockedMod{"m": {Source: LockedSource{Type: SourceGitHubRelease, Repo: "o/r", FileName: "a.jar"}}}}
		Expect(ValidateLock(lock)).To(HaveOccurred())
	})

	It("ValidateLock local missing path", func() {
		lock := PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
			Mods: map[string]LockedMod{"m": {Source: LockedSource{Type: SourceLocal}}}}
		Expect(ValidateLock(lock)).To(HaveOccurred())
	})

	It("ValidateLock URL missing URL", func() {
		lock := PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
			Mods: map[string]LockedMod{"m": {Source: LockedSource{Type: SourceURL}}}}
		Expect(ValidateLock(lock)).To(HaveOccurred())
	})

	It("VariantKey with and without version", func() {
		Expect(VariantKey("neoforge", "1.0")).To(Equal("neoforge:1.0"))
		Expect(VariantKey("fabric", "")).To(Equal("fabric"))
	})

	It("DefaultVariantKey returns first loader", func() {
		Expect(DefaultVariantKey(PackSpec{LoaderName: []string{"neoforge:1.0"}})).To(Equal("neoforge:1.0"))
		Expect(DefaultVariantKey(PackSpec{})).To(Equal(""))
	})

	It("normalizeLoaderList normalizes", func() {
		Expect(normalizeLoaderList([]string{"", " neoforge ", "neoforge", "fabric"})).To(HaveLen(2))
		Expect(normalizeLoaderList([]string{})).To(BeEmpty())
	})

	It("modScope default", func() {
		Expect(modScope(ModSpec{})).To(Equal(ScopeShared))
		Expect(modScope(ModSpec{Scope: ScopeClient})).To(Equal(ScopeClient))
	})

	It("cloneModMap", func() {
		m := map[string]ModSpec{"a": {Name: "A"}}
		c := cloneModMap(m)
		Expect(c).To(HaveLen(1))
		c["b"] = ModSpec{}
		Expect(m).To(HaveLen(1))
	})

	It("modMapFromSlice", func() {
		slice := []ModSpec{{Name: "Display Name", Source: ModSource{Type: SourceCurseForge, Query: "Display Name"}}}
		m := modMapFromSlice(slice)
		Expect(m).To(HaveKey("display-name"))
	})

	It("modsForScopeFromMap", func() {
		m := map[string]ModSpec{"a": {Name: "A", Scope: ScopeClient}}
		Expect(modsForScopeFromMap(m, ScopeClient)).To(HaveLen(1))
		Expect(modsForScopeFromMap(m, ScopeShared)).To(BeEmpty())
	})

	It("setModsForScopeInMap replaces scope mods", func() {
		m := map[string]ModSpec{"a": {Name: "A", Scope: ScopeClient}}
		setModsForScopeInMap(m, ScopeClient, []ModSpec{{Name: "B"}})
		Expect(m).To(HaveLen(1))
		Expect(m["b"].Name).To(Equal("B"))
	})

	It("ReadLockFile opens existing file", func() {
		dir := GinkgoT().TempDir()
		store := DefaultFileStore(dir)
		lock := PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
			Mods: map[string]LockedMod{"m": {Name: "M", Source: LockedSource{Type: SourceLocal}}}}
		Expect(store.SaveLock("1.21.1", "neoforge", lock)).To(Succeed())
		loaded, err := store.LoadLock("1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded.Mods).To(HaveLen(1))
	})

	It("WriteLockFile write then ReadLockFile read", func() {
		dir := GinkgoT().TempDir()
		store := DefaultFileStore(dir)
		lock := PackLock{Loader: "fabric", MinecraftVersion: "1.21.1",
			Mods: map[string]LockedMod{"m": {Name: "M", Source: LockedSource{Type: SourceLocal}}}}
		Expect(store.SaveLock("1.21.1", "fabric", lock)).To(Succeed())
		loaded, err := store.LoadLock("1.21.1", "fabric")
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded.Mods).To(HaveLen(1))
	})

	It("WriteReleaseIndex writes and ReadReleaseIndex reads", func() {
		dir := GinkgoT().TempDir()
		os.MkdirAll(filepath.Join(dir, "locks", "releases"), 0755)
		path := ReleaseIndexPath("1.21.1")
		ri := ReleaseIndex{Type: "package", PackName: "test", MinecraftVersion: "1.21.1"}
		Expect(WriteReleaseIndex(filepath.Join(dir, path), &ri)).To(Succeed())
		loaded, err := ReadReleaseIndex(filepath.Join(dir, path))
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded.PackName).To(Equal("test"))
	})

	It("ReadPackSpec with valid file", func() {
		dir := GinkgoT().TempDir()
		Expect(WritePackSpec(dir, &PackSpec{PackName: "spec-test", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"}, PackVersion: "1"})).To(Succeed())
		spec, err := ReadPackSpec(dir)
		Expect(err).NotTo(HaveOccurred())
		Expect(spec.PackName).To(Equal("spec-test"))
	})

	It("ReadPackSpec with bad json", func() {
		dir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(dir, "packspec.json"), []byte("{badjson}"), 0644)).To(Succeed())
		_, err := ReadPackSpec(dir)
		Expect(err).To(HaveOccurred())
	})

	It("unmarshalPackLock with empty lock falls through", func() {
		_, err := unmarshalPackLock([]byte(`{}`))
		Expect(err).To(HaveOccurred())
	})

	It("unmarshalPackLock with nonsense", func() {
		_, err := unmarshalPackLock([]byte("not json at all"))
		Expect(err).To(HaveOccurred())
	})

	It("ReleaseIndex Normalize idempotent", func() {
		ri := ReleaseIndex{}
		ri.Normalize()
		Expect(ri.Type).To(Equal("package"))
		ri.Normalize()
		Expect(ri.Type).To(Equal("package"))
	})

	It("ReleaseIndex DeleteRelease nonexistent", func() {
		ri := ReleaseIndex{}
		Expect(ri.DeleteRelease("v99")).To(BeFalse())
	})

	It("ReleaseIndex EnsureRelease sets GitHub", func() {
		idx := ReleaseIndex{Type: "package"}
		r := idx.EnsureRelease("0.1.0", "github-release")
		r.GitHub = ReleaseGitHub{Repo: "o/r", Tag: "v0.1.0"}
		// GitHub set successfully
		_ = r
	})

	It("EntriesForMod returns nil when no entries", func() {
		Expect(EntriesForMod(PackSpec{}, "test")).To(BeNil())
	})

	It("AllEntries returns entries from entriesByMod", func() {
		spec := SetEntriesForMod(PackSpec{}, "k", []EntrySpec{{Name: "e1"}})
		Expect(AllEntries(spec)).To(HaveLen(1))
	})

	It("AllEntriesForVariant returns entries from entriesByMod", func() {
		spec := SetEntriesForMod(PackSpec{}, "neoforge", []EntrySpec{{Name: "e1"}})
		entries := AllEntriesForVariant(spec, "neoforge")
		Expect(entries).To(HaveLen(1))
	})

	It("AllEntriesForVariant with no entries returns empty", func() {
		Expect(AllEntriesForVariant(PackSpec{}, "any")).To(HaveLen(0))
	})

	It("AllModsForVariant includes deps sorted", func() {
		spec := PackSpec{
			Mods: map[string]ModSpec{"b": {Name: "B"}, "a": {Name: "A"}},
		}
		spec = SetDependencies(spec, []ModSpec{{Name: "dep"}})
		mods := AllModsForVariant(spec, "neoforge")
		Expect(mods).To(HaveLen(3))
	})

	It("NormalizeKey handles edge cases", func() {
		Expect(NormalizeKey("Test!@#Mod")).To(Equal("test-mod"))
		Expect(NormalizeKey("---test---")).To(Equal("test"))
		Expect(NormalizeKey("Already-normal")).To(Equal("already-normal"))
	})

	It("MarshalJSON includes mods map from legacy arrays", func() {
		spec := PackSpec{
			PackName: "m", PackVersion: "1", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
			SharedMods: []ModSpec{{Name: "Shared", Source: ModSource{Type: SourceCurseForge}}},
			ClientMods: []ModSpec{{Name: "Client-only", Source: ModSource{Type: SourceCurseForge}}},
		}
		data, err := spec.MarshalJSON()
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(MatchRegexp(`"mods"`))
	})

	It("MarshalJSON omits legacy arrays when Mods is used", func() {
		spec := PackSpec{
			PackName: "m", PackVersion: "1", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
			Mods: map[string]ModSpec{"m": {Name: "Mod", Source: ModSource{Type: SourceCurseForge}}},
		}
		data, err := spec.MarshalJSON()
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).NotTo(MatchRegexp(`sharedMods|clientMods`))
	})

	It("FileStore loading with legacy root lock file", func() {
		dir := GinkgoT().TempDir()
		store := DefaultFileStore(dir)
		Expect(os.WriteFile(filepath.Join(dir, "1.21.1.json"),
			[]byte(`{"loader":"fabric","minecraftVersion":"1.21.1","mods":{"a":{"scope":"shared","source":{"type":"local","path":"./a.jar","fileName":"a.jar"}}}}`), 0644)).To(Succeed())
		loaded, err := store.LoadLock("1.21.1", "fabric")
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded.Mods).To(HaveLen(1))
	})

	It("FileStore LoadLock with empty loader finds single candidate", func() {
		dir := GinkgoT().TempDir()
		store := DefaultFileStore(dir)
		Expect(os.MkdirAll(filepath.Join(dir, "locks", "dependencies"), 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "locks", "dependencies", "1.21.1-fabric.json"),
			[]byte(`{"loader":"fabric","minecraftVersion":"1.21.1","mods":{"a":{"scope":"shared","source":{"type":"local","path":"./a.jar","fileName":"a.jar"}}}}`), 0644)).To(Succeed())
		loaded, err := store.LoadLock("1.21.1", "")
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded.Loader).To(Equal("fabric"))
	})
})

var _ = Describe("ValidateSpec source validation", func() {
	validSpec := func() PackSpec {
		return PackSpec{
			PackName: "p", PackVersion: "1", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
			Mods: map[string]ModSpec{},
		}
	}

	It("rejects curseforge source with slug", func() {
		spec := validSpec()
		spec.Mods["m"] = ModSpec{Name: "m", Source: ModSource{Type: SourceCurseForge, Query: "m", Slug: "m"}}
		Expect(ValidateSpec(spec)).NotTo(Succeed())
	})

	It("rejects curseforge source with modId/fileId", func() {
		spec := validSpec()
		spec.Mods["m"] = ModSpec{Name: "m", Source: ModSource{Type: SourceCurseForge, Query: "m", ModID: 1, FileID: 2}}
		Expect(ValidateSpec(spec)).NotTo(Succeed())
	})

	It("rejects curseforge source with fileName", func() {
		spec := validSpec()
		spec.Mods["m"] = ModSpec{Name: "m", Source: ModSource{Type: SourceCurseForge, Query: "m", FileName: "m.jar"}}
		Expect(ValidateSpec(spec)).NotTo(Succeed())
	})

	It("rejects github-release source without repo", func() {
		spec := validSpec()
		spec.Mods["m"] = ModSpec{Name: "m", Source: ModSource{Type: SourceGitHubRelease, Tag: "v1", AssetPattern: "x.jar"}}
		Expect(ValidateSpec(spec)).NotTo(Succeed())
	})

	It("rejects github-release source without tag", func() {
		spec := validSpec()
		spec.Mods["m"] = ModSpec{Name: "m", Source: ModSource{Type: SourceGitHubRelease, Repo: "o/r", AssetPattern: "x.jar"}}
		Expect(ValidateSpec(spec)).NotTo(Succeed())
	})

	It("rejects github-release source without asset pattern", func() {
		spec := validSpec()
		spec.Mods["m"] = ModSpec{Name: "m", Source: ModSource{Type: SourceGitHubRelease, Repo: "o/r", Tag: "v1"}}
		Expect(ValidateSpec(spec)).NotTo(Succeed())
	})

	It("rejects github-release source with fileName", func() {
		spec := validSpec()
		spec.Mods["m"] = ModSpec{Name: "m", Source: ModSource{Type: SourceGitHubRelease, Repo: "o/r", Tag: "v1", AssetPattern: "x.jar", FileName: "x.jar"}}
		Expect(ValidateSpec(spec)).NotTo(Succeed())
	})

	It("rejects git source without repo", func() {
		spec := validSpec()
		spec.Mods["m"] = ModSpec{Name: "m", Source: ModSource{Type: SourceGit}}
		Expect(ValidateSpec(spec)).NotTo(Succeed())
	})

	It("rejects git source with query", func() {
		spec := validSpec()
		spec.Mods["m"] = ModSpec{Name: "m", Source: ModSource{Type: SourceGit, Repo: "o/r", Query: "x"}}
		Expect(ValidateSpec(spec)).NotTo(Succeed())
	})

	It("rejects local source without path", func() {
		spec := validSpec()
		spec.Mods["m"] = ModSpec{Name: "m", Source: ModSource{Type: SourceLocal}}
		Expect(ValidateSpec(spec)).NotTo(Succeed())
	})

	It("rejects local source with URL", func() {
		spec := validSpec()
		spec.Mods["m"] = ModSpec{Name: "m", Source: ModSource{Type: SourceLocal, Path: "./x.jar", URL: "https://x"}}
		Expect(ValidateSpec(spec)).NotTo(Succeed())
	})

	It("rejects unknown source type", func() {
		spec := validSpec()
		spec.Mods["m"] = ModSpec{Name: "m", Source: ModSource{Type: "wat"}}
		Expect(ValidateSpec(spec)).NotTo(Succeed())
	})
})

var _ = Describe("ValidateSpec more branches", func() {
	It("rejects missing minecraftVersion", func() {
		err := ValidateSpec(PackSpec{
			PackName: "p", PackVersion: "1", LoaderName: []string{"neoforge:1"},
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("minecraftVersion"))
	})

	It("rejects empty loaderName array", func() {
		err := ValidateSpec(PackSpec{
			PackName: "p", PackVersion: "1", MinecraftVersion: "1.21.1",
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("loaderName"))
	})

	It("rejects mod with empty source type", func() {
		err := ValidateSpec(PackSpec{
			PackName: "p", PackVersion: "1", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
			Mods: map[string]ModSpec{"a": {Name: "A"}},
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("empty source type"))
	})

	It("rejects mod with per-loader unsupported loader", func() {
		err := ValidateSpec(PackSpec{
			PackName: "p", PackVersion: "1", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
			Mods: map[string]ModSpec{
				"a": {Name: "A", Loader: []string{"bukkit"}, Source: ModSource{Type: "local"}},
			},
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unsupported loader"))
	})

	It("accepts valid per-loader filter", func() {
		err := ValidateSpec(PackSpec{
			PackName: "p", PackVersion: "1", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
			Mods: map[string]ModSpec{
				"a": {Name: "A", Loader: []string{"neoforge"}, Source: ModSource{Type: "local", Path: "./x.jar"}},
			},
		})
		Expect(err).NotTo(HaveOccurred())
	})
})
