// File: internal/domain/validate_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for validation and store operations.

package domain

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

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

var _ = Describe("Store", func() {
	var tmpDir string
	var store *FileStore

	BeforeEach(func() {
		tmpDir = GinkgoT().TempDir()
		store = DefaultFileStore(tmpDir)
	})

	Describe("FileStore.SaveSpec/LoadSpec", func() {
		It("saves and loads a spec", func() {
			spec := PackSpec{PackName: "test", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"}, PackVersion: "0.1.0"}
			Expect(store.SaveSpec(spec)).To(Succeed())
			loaded, err := store.LoadSpec()
			Expect(err).NotTo(HaveOccurred())
			Expect(loaded.PackName).To(Equal("test"))
		})

		It("fails to load from empty dir", func() {
			_, err := store.LoadSpec()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("FileStore.SaveLock/LoadLock", func() {
		It("saves and loads a lock", func() {
			lock := PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]LockedMod{"create": {Name: "Create", Scope: ScopeShared, Source: LockedSource{Type: SourceCurseForge, ModID: 1, FileID: 2, FileName: "create.jar"}}}}
			Expect(store.SaveLock("1.21.1", "neoforge", lock)).To(Succeed())
			loaded, err := store.LoadLock("1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(loaded.Loader).To(Equal("neoforge"))
			Expect(loaded.Mods).To(HaveLen(1))
		})

		It("falls back to legacy root lock", func() {
			Expect(os.WriteFile(filepath.Join(tmpDir, "1.21.1.json"),
				[]byte(`{"loader":"neoforge","minecraftVersion":"1.21.1","mods":{"a":{"scope":"shared","source":{"type":"local","path":"./a.jar","fileName":"a.jar"}}}}`), 0644)).To(Succeed())
			loaded, err := store.LoadLock("1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(loaded.Mods).To(HaveLen(1))
		})

		It("fails for missing lock", func() {
			_, err := store.LoadLock("99.99", "neoforge")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("FileStore.SaveReleaseIndex/LoadReleaseIndex", func() {
		It("saves and loads a release index", func() {
			ri := ReleaseIndex{Type: "package", PackName: "test", MinecraftVersion: "1.21.1"}
			Expect(store.SaveReleaseIndex("1.21.1", ri)).To(Succeed())
			loaded, err := store.LoadReleaseIndex("1.21.1")
			Expect(err).NotTo(HaveOccurred())
			Expect(loaded.PackName).To(Equal("test"))
		})

		It("fails for missing index", func() {
			_, err := store.LoadReleaseIndex("99.99")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("unmarshalPackLock legacy", func() {
		It("handles legacy versions format", func() {
			data := []byte(`{"versions":{"1.21.1":{"loaderName":"fabric","sharedMods":[{"name":"S","source":{"type":"local","path":"./s.jar","fileName":"s.jar"}}]}}}`)
			lock, err := unmarshalPackLock(data)
			Expect(err).NotTo(HaveOccurred())
			Expect(lock.Loader).To(Equal("fabric"))
			Expect(lock.Mods).To(HaveLen(1))
		})

		It("fails for truly invalid data", func() {
			_, err := unmarshalPackLock([]byte("not json"))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("WritePackSpec/ReadPackSpec", func() {
		It("round-trips packspec.json", func() {
			Expect(WritePackSpec(tmpDir, &PackSpec{PackName: "rt", MinecraftVersion: "1.21.1", LoaderName: []string{"fabric"}, PackVersion: "1.0"})).To(Succeed())
			spec, err := ReadPackSpec(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(spec.PackName).To(Equal("rt"))
		})

		It("ReadPackSpec fails on bad dir", func() {
			_, err := ReadPackSpec("/nonexistent")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("LockFilePath/ReleaseIndexPath", func() {
		It("returns correct paths", func() {
			Expect(LockFilePath("1.21.1", "neoforge")).To(Equal("locks/dependencies/1.21.1-neoforge.json"))
			Expect(ReleaseIndexPath("1.21.1")).To(Equal("locks/releases/1.21.1.json"))
		})
	})
})
