// File: internal/domain/store_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/domain/store.go (LockFilePath, ReadLockFile, WriteLockFile, ReadReleaseIndex, WriteReleaseIndex, FileStore).

package domain

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// --- from spec_test.go consolidated (Store) ---
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

var _ = Describe("Direct file I/O helpers", func() {
	It("ReadLockFile/WriteLockFile round-trip", func() {
		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, "locks", "1.21.1", "neoforge", "dep.json")
		lock := &PackLock{
			Loader:           "neoforge",
			LoaderVersion:    "21.0.0",
			MinecraftVersion: "1.21.1",
			Mods:             map[string]LockedMod{"a": {Name: "a", Scope: "shared"}},
		}
		Expect(WriteLockFile(path, lock)).To(Succeed())
		got, err := ReadLockFile(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(got.MinecraftVersion).To(Equal("1.21.1"))
		Expect(got.Mods).To(HaveKey("a"))
	})

	It("ReadLockFile returns an error for a missing file", func() {
		_, err := ReadLockFile("/no/such/lock.json")
		Expect(err).To(HaveOccurred())
	})

	It("ReadLockFile returns an error for invalid JSON", func() {
		dir := GinkgoT().TempDir()
		bad := filepath.Join(dir, "bad.json")
		Expect(os.WriteFile(bad, []byte("not-json"), 0644)).To(Succeed())
		_, err := ReadLockFile(bad)
		Expect(err).To(HaveOccurred())
	})

	It("ReadReleaseIndex/WriteReleaseIndex round-trip", func() {
		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, "releases", "1.21.1.json")
		ri := &ReleaseIndex{
			Releases: []ReleaseRecord{
				{Version: "1.0.0", Type: "release"},
			},
		}
		Expect(WriteReleaseIndex(path, ri)).To(Succeed())
		got, err := ReadReleaseIndex(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Releases).To(HaveLen(1))
		Expect(got.Releases[0].Version).To(Equal("1.0.0"))
	})

	It("ReadReleaseIndex returns an error for a missing file", func() {
		_, err := ReadReleaseIndex("/no/such/release.json")
		Expect(err).To(HaveOccurred())
	})

	It("ReadReleaseIndex returns an error for invalid JSON", func() {
		dir := GinkgoT().TempDir()
		bad := filepath.Join(dir, "bad.json")
		Expect(os.WriteFile(bad, []byte("not-json"), 0644)).To(Succeed())
		_, err := ReadReleaseIndex(bad)
		Expect(err).To(HaveOccurred())
	})
})
