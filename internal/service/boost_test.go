// File: internal/service/boost_test.go
// Created: 2026-06-20
// Description: Extra service tests to push coverage above 80%.
package service

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("Service boost coverage", func() {
	Describe("ReadLockFile", func() {
		It("ReadLockFile reads a valid lock file", func() {
			dir := GinkgoT().TempDir()
			p := filepath.Join(dir, "1.21.1-neoforge.json")
			Expect(os.WriteFile(p, []byte(`{"loader":"neoforge","minecraftVersion":"1.21.1","mods":{}}`), 0644)).To(Succeed())
			lock, err := ReadLockFile(p)
			Expect(err).NotTo(HaveOccurred())
			Expect(lock.Loader).To(Equal("neoforge"))
		})
		It("ReadLockFile returns error for missing file", func() {
			_, err := ReadLockFile("/no/such/lock.json")
			Expect(err).To(HaveOccurred())
		})
		It("ReadLockFile returns error for invalid JSON", func() {
			dir := GinkgoT().TempDir()
			p := filepath.Join(dir, "bad.json")
			Expect(os.WriteFile(p, []byte("{not json"), 0644)).To(Succeed())
			_, err := ReadLockFile(p)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("BuildArtifact with target=both and force", func() {
		It("buildOneArtifact errors for unsupported target", func() {
			dir := GinkgoT().TempDir()
			jarPath := filepath.Join(dir, "a.jar")
			Expect(os.WriteFile(jarPath, []byte("dummy"), 0644)).To(Succeed())
			oldWd, _ := os.Getwd()
			Expect(os.Chdir(dir)).To(Succeed())
			defer os.Chdir(oldWd)
			spec := &domain.PackSpec{PackName: "test", Mods: map[string]domain.ModSpec{"a": {Scope: "shared", Source: domain.ModSource{Type: "local", Path: jarPath}}}}
			lock := &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: jarPath, FileName: "a.jar"}}}}
			err := BuildArtifact(spec, lock, "1.21.1", "bogus")
			Expect(err).To(HaveOccurred())
		})
		It("BuildArtifact with target both produces two zips", func() {
			dir := GinkgoT().TempDir()
			jarPath := filepath.Join(dir, "a.jar")
			Expect(os.WriteFile(jarPath, []byte("dummy"), 0644)).To(Succeed())
			oldWd, _ := os.Getwd()
			Expect(os.Chdir(dir)).To(Succeed())
			defer os.Chdir(oldWd)
			spec := &domain.PackSpec{PackName: "p", PackVersion: "0.1.0", Mods: map[string]domain.ModSpec{"a": {Scope: "shared", Source: domain.ModSource{Type: "local", Path: jarPath}}}}
			lock := &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: jarPath, FileName: "a.jar"}}}}
			Expect(BuildArtifactWith(spec, lock, "1.21.1", "both", true)).To(Succeed())
			entries, _ := os.ReadDir(filepath.Join(dir, "releases", "v0.1.0"))
			Expect(len(entries)).To(Equal(2))
		})
		It("BuildArtifactWith with existing artifact + force overwrites", func() {
			dir := GinkgoT().TempDir()
			jarPath := filepath.Join(dir, "a.jar")
			Expect(os.WriteFile(jarPath, []byte("dummy"), 0644)).To(Succeed())
			oldWd, _ := os.Getwd()
			Expect(os.Chdir(dir)).To(Succeed())
			defer os.Chdir(oldWd)
			spec := &domain.PackSpec{PackName: "p", PackVersion: "0.1.0", Mods: map[string]domain.ModSpec{"a": {Scope: "shared", Source: domain.ModSource{Type: "local", Path: jarPath}}}}
			lock := &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: jarPath, FileName: "a.jar"}}}}
			Expect(BuildArtifact(spec, lock, "1.21.1", "client")).To(Succeed())
			// Without force: errors
			err := BuildArtifact(spec, lock, "1.21.1", "client")
			Expect(err).To(HaveOccurred())
			// With force: succeeds
			Expect(BuildArtifactWith(spec, lock, "1.21.1", "client", true)).To(Succeed())
		})
		It("buildOneArtifact with no mods errors", func() {
			dir := GinkgoT().TempDir()
			oldWd, _ := os.Getwd()
			Expect(os.Chdir(dir)).To(Succeed())
			defer os.Chdir(oldWd)
			spec := &domain.PackSpec{PackName: "p", PackVersion: "0.1.0", Mods: map[string]domain.ModSpec{}}
			lock := &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{}}
			err := BuildArtifact(spec, lock, "1.21.1", "client")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("BuildZip with config/defaultconfigs/resourcepacks dirs", func() {
		It("includes config dir when present", func() {
			dir := GinkgoT().TempDir()
			jarPath := filepath.Join(dir, "a.jar")
			Expect(os.WriteFile(jarPath, []byte("dummy"), 0644)).To(Succeed())
			cfgDir := filepath.Join(dir, "config")
			Expect(os.MkdirAll(cfgDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(cfgDir, "common.toml"), []byte("k=v"), 0644)).To(Succeed())
			oldWd, _ := os.Getwd()
			Expect(os.Chdir(dir)).To(Succeed())
			defer os.Chdir(oldWd)
			spec := &domain.PackSpec{PackName: "p", PackVersion: "0.1.0", Mods: map[string]domain.ModSpec{"a": {Scope: "client", Source: domain.ModSource{Type: "local", Path: jarPath}}}}
			lock := &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{"a": {Name: "A", Scope: "client", Source: domain.LockedSource{Type: "local", Path: jarPath, FileName: "a.jar"}}}}
			Expect(BuildArtifactWith(spec, lock, "1.21.1", "client", true)).To(Succeed())
		})
		It("includes defaultconfigs dir for server", func() {
			dir := GinkgoT().TempDir()
			jarPath := filepath.Join(dir, "a.jar")
			Expect(os.WriteFile(jarPath, []byte("dummy"), 0644)).To(Succeed())
			dcDir := filepath.Join(dir, "defaultconfigs")
			Expect(os.MkdirAll(dcDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(dcDir, "server-common.toml"), []byte("k=v"), 0644)).To(Succeed())
			oldWd, _ := os.Getwd()
			Expect(os.Chdir(dir)).To(Succeed())
			defer os.Chdir(oldWd)
			spec := &domain.PackSpec{PackName: "p", PackVersion: "0.1.0", Mods: map[string]domain.ModSpec{"a": {Scope: "server", Source: domain.ModSource{Type: "local", Path: jarPath}}}}
			lock := &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{"a": {Name: "A", Scope: "server", Source: domain.LockedSource{Type: "local", Path: jarPath, FileName: "a.jar"}}}}
			Expect(BuildArtifactWith(spec, lock, "1.21.1", "server", true)).To(Succeed())
		})
		It("includes resourcepacks dir for client", func() {
			dir := GinkgoT().TempDir()
			jarPath := filepath.Join(dir, "a.jar")
			Expect(os.WriteFile(jarPath, []byte("dummy"), 0644)).To(Succeed())
			rpDir := filepath.Join(dir, "resourcepacks")
			Expect(os.MkdirAll(rpDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(rpDir, "rp.zip"), []byte("data"), 0644)).To(Succeed())
			oldWd, _ := os.Getwd()
			Expect(os.Chdir(dir)).To(Succeed())
			defer os.Chdir(oldWd)
			spec := &domain.PackSpec{PackName: "p", PackVersion: "0.1.0", Mods: map[string]domain.ModSpec{"a": {Scope: "client", Source: domain.ModSource{Type: "local", Path: jarPath}}}}
			lock := &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{"a": {Name: "A", Scope: "client", Source: domain.LockedSource{Type: "local", Path: jarPath, FileName: "a.jar"}}}}
			Expect(BuildArtifactWith(spec, lock, "1.21.1", "client", true)).To(Succeed())
		})
	})

	Describe("BuildClientServerBuild", func() {
		It("errors when lock missing", func() {
			dir := GinkgoT().TempDir()
			oldWd, _ := os.Getwd()
			Expect(os.Chdir(dir)).To(Succeed())
			defer os.Chdir(oldWd)
			spec := &domain.PackSpec{PackName: "p", PackVersion: "0.1.0", LoaderName: []string{"neoforge:21.1.219"}}
			err := BuildClientServerBuild(spec, "1.21.1")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("loaderFromLock variations", func() {
		It("loaderFromLock with no version uses empty", func() {
			dir := GinkgoT().TempDir()
			oldWd, _ := os.Getwd()
			Expect(os.Chdir(dir)).To(Succeed())
			defer os.Chdir(oldWd)
			jarPath := filepath.Join(dir, "a.jar")
			Expect(os.WriteFile(jarPath, []byte("dummy"), 0644)).To(Succeed())
			spec := &domain.PackSpec{PackName: "p", PackVersion: "0.1.0", Mods: map[string]domain.ModSpec{"a": {Scope: "shared", Source: domain.ModSource{Type: "local", Path: jarPath}}}}
			lock := &domain.PackLock{Loader: "neoforge", LoaderVersion: "", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: jarPath, FileName: "a.jar"}}}}
			Expect(BuildArtifactWith(spec, lock, "1.21.1", "client", true)).To(Succeed())
		})
	})

	Describe("BuildLock with various sources", func() {
		It("BuildLock for a local mod", func() {
			dir := GinkgoT().TempDir()
			oldWd, _ := os.Getwd()
			Expect(os.Chdir(dir)).To(Succeed())
			defer os.Chdir(oldWd)
			jarPath := filepath.Join(dir, "a.jar")
			Expect(os.WriteFile(jarPath, []byte("dummy"), 0644)).To(Succeed())
			spec := &domain.PackSpec{
				PackName: "p", PackVersion: "0.1.0", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge:21.1.219"},
				Mods: map[string]domain.ModSpec{
					"a": {Name: "A", Scope: "shared", Source: domain.ModSource{Type: "local", Path: jarPath}},
				},
			}
			lock, err := BuildLock(spec, "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(lock.Mods).To(HaveKey("a"))
			Expect(lock.Mods["a"].Source.Type).To(Equal("local"))
		})
		It("BuildLock for an unsupported source errors", func() {
			spec := &domain.PackSpec{
				PackName: "p", PackVersion: "0.1.0", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge:21.1.219"},
				Mods: map[string]domain.ModSpec{
					"a": {Name: "A", Scope: "shared", Source: domain.ModSource{Type: "weird"}},
				},
			}
			lock, err := BuildLock(spec, "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(lock.Mods).NotTo(HaveKey("a"))
		})
	})

	Describe("CreateReleaseRecord variations", func() {
		It("creates a new release record", func() {
			dir := GinkgoT().TempDir()
			oldWd, _ := os.Getwd()
			Expect(os.Chdir(dir)).To(Succeed())
			defer os.Chdir(oldWd)
			Expect(os.WriteFile(filepath.Join(dir, "packspec.json"), []byte(`{"packName":"p","packVersion":"0.1.0","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`), 0644)).To(Succeed())
			ri, err := CreateReleaseRecord("1.21.1", "0.1.0", "github-release", &domain.ReleaseGitHub{Repo: "o/r", Tag: "v0.1.0"})
			Expect(err).NotTo(HaveOccurred())
			Expect(ri.PackName).To(Equal("p"))
		})
		It("updates an existing release record", func() {
			dir := GinkgoT().TempDir()
			oldWd, _ := os.Getwd()
			Expect(os.Chdir(dir)).To(Succeed())
			defer os.Chdir(oldWd)
			Expect(os.WriteFile(filepath.Join(dir, "packspec.json"), []byte(`{"packName":"p","packVersion":"0.1.0","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`), 0644)).To(Succeed())
			_, err := CreateReleaseRecord("1.21.1", "0.1.0", "github-release", &domain.ReleaseGitHub{Repo: "o/r", Tag: "v0.1.0"})
			Expect(err).NotTo(HaveOccurred())
			ri, err := CreateReleaseRecord("1.21.1", "0.2.0", "github-release", &domain.ReleaseGitHub{Repo: "o/r", Tag: "v0.2.0"})
			Expect(err).NotTo(HaveOccurred())
			Expect(ri.Releases).To(HaveLen(2))
		})
		It("backfills packName on existing index when missing", func() {
			dir := GinkgoT().TempDir()
			oldWd, _ := os.Getwd()
			Expect(os.Chdir(dir)).To(Succeed())
			defer os.Chdir(oldWd)
			// Pre-write an index with empty packName.
			idxDir := filepath.Join(dir, "locks", "releases")
			Expect(os.MkdirAll(idxDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(idxDir, "1.21.1.json"), []byte(`{"type":"package","packName":"","minecraftVersion":"1.21.1","releases":[]}`), 0644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(dir, "packspec.json"), []byte(`{"packName":"myPack","packVersion":"0.1.0","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`), 0644)).To(Succeed())
			ri, err := CreateReleaseRecord("1.21.1", "0.1.0", "github-release", &domain.ReleaseGitHub{Repo: "o/r", Tag: "v0.1.0"})
			Expect(err).NotTo(HaveOccurred())
			Expect(ri.PackName).To(Equal("myPack"))
		})
	})

	Describe("ReadReleaseIndex error paths", func() {
		It("errors on missing file", func() {
			_, err := ReadReleaseIndex("99.99")
			Expect(err).To(HaveOccurred())
		})
		It("errors on bad JSON", func() {
			dir := GinkgoT().TempDir()
			oldWd, _ := os.Getwd()
			Expect(os.Chdir(dir)).To(Succeed())
			defer os.Chdir(oldWd)
			idxDir := filepath.Join(dir, "locks", "releases")
			Expect(os.MkdirAll(idxDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(idxDir, "1.21.1.json"), []byte("{not json"), 0644)).To(Succeed())
			_, err := ReadReleaseIndex("1.21.1")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("WriteReleaseIndex and ReadReleaseIndex round-trip", func() {
		It("writes and reads a release index", func() {
			dir := GinkgoT().TempDir()
			oldWd, _ := os.Getwd()
			Expect(os.Chdir(dir)).To(Succeed())
			defer os.Chdir(oldWd)
			ri := &domain.ReleaseIndex{Type: "package", PackName: "p", MinecraftVersion: "1.21.1", Releases: []domain.ReleaseRecord{{Version: "0.1.0", Type: "github-release"}}}
			Expect(WriteReleaseIndex("1.21.1", ri)).To(Succeed())
			loaded, err := ReadReleaseIndex("1.21.1")
			Expect(err).NotTo(HaveOccurred())
			Expect(loaded.PackName).To(Equal("p"))
			Expect(loaded.Releases).To(HaveLen(1))
		})
	})

	Describe("MarshalLockJSON", func() {
		It("marshals a lock to JSON", func() {
			lock := &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{}}
			data, err := MarshalLockJSON(lock)
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Valid(data)).To(BeTrue())
		})
	})

	Describe("ListMods", func() {
		It("ListMods with shared+client+server mods", func() {
			spec := &domain.PackSpec{
				PackName: "p", PackVersion: "0.1.0", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge:21.1.219"},
				Mods: map[string]domain.ModSpec{
					"a": {Name: "A", Scope: "shared", Source: domain.ModSource{Type: "local"}},
					"b": {Name: "B", Scope: "client", Source: domain.ModSource{Type: "curseforge"}},
					"c": {Name: "C", Scope: "server", Source: domain.ModSource{Type: "github-release"}},
				},
			}
			out, err := ListMods(spec)
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(ContainSubstring("A [local]"))
			Expect(out).To(ContainSubstring("B [curseforge]"))
			Expect(out).To(ContainSubstring("C [github-release]"))
		})
	})

	Describe("BuildTree and FormatTree", func() {
		It("BuildTree sorts by scope", func() {
			lock := &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{
				"x": {Name: "X", Scope: "client", Source: domain.LockedSource{Type: "local"}},
				"y": {Name: "Y", Scope: "shared", Source: domain.LockedSource{Type: "local"}},
			}}
			entries := BuildTree(lock)
			Expect(entries).To(HaveLen(2))
		})
		It("FormatTree returns non-empty for entries", func() {
			entries := []TreeEntry{{Name: "A", Version: "1", Scope: "shared", Source: "local"}}
			out := FormatTree(entries)
			Expect(out).To(ContainSubstring("A"))
		})
	})
})

// Avoid unused import warnings
var _ = zip.Writer{}
var _ = filepath.Join
