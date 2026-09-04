// File: internal/service/build_zip_cf_test.go
// Created: 2026-06-21
// Description: Ginkgo tests for internal/service/build_zip_cf.go (CurseForge modpack layout writer).

package service

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("buildCurseForgeManifest", func() {
	It("filters to shared+client curseforge mods with modId/fileId and sorts by projectID", func() {
		spec := &domain.PackSpec{
			PackName: "p", PackVersion: "0.1.0", MinecraftVersion: "1.21.1",
			Author: "tester", LoaderName: []string{"neoforge:1.0"},
		}
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "curseforge", ModID: 200, FileID: 11, FileName: "a.jar"}},
				"b": {Name: "B", Scope: "client", Source: domain.LockedSource{Type: "curseforge", ModID: 100, FileID: 22, FileName: "b.jar"}},
				"c": {Name: "C", Scope: "server", Source: domain.LockedSource{Type: "curseforge", ModID: 300, FileID: 33, FileName: "c.jar"}},
				"d": {Name: "D", Scope: "shared", Source: domain.LockedSource{Type: "github-release", Repo: "o/r", Tag: "v1", AssetName: "d.jar"}},
				"e": {Name: "E", Scope: "shared", Source: domain.LockedSource{Type: "curseforge", ModID: 0, FileID: 0, FileName: "e.jar"}},
			},
		}
		bc := &buildContext{Spec: spec, Lock: lock, McVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "1.0"}
		m, refs, skipped := bc.buildCurseForgeManifest()
		Expect(m.Name).To(Equal("p"))
		Expect(m.Version).To(Equal("0.1.0"))
		Expect(m.Author).To(Equal("tester"))
		Expect(m.Minecraft.Version).To(Equal("1.21.1"))
		Expect(m.Minecraft.ModLoaders).To(HaveLen(1))
		Expect(m.Minecraft.ModLoaders[0].ID).To(Equal("neoforge-1.0"))
		Expect(m.Minecraft.ModLoaders[0].Primary).To(BeTrue())
		Expect(m.Overrides).To(Equal("overrides"))
		Expect(m.ManifestType).To(Equal("minecraftModpack"))
		Expect(m.ManifestVersion).To(Equal(1))
		Expect(m.Files).To(HaveLen(2))
		// Sorted by projectID asc.
		Expect(m.Files[0].ProjectID).To(Equal(100))
		Expect(m.Files[1].ProjectID).To(Equal(200))
		Expect(refs).To(HaveLen(2))
		// "c" is server-scope and silently omitted (CF publishes client only).
		// "d" and "e" are reported as skipped because they are non-cf or
		// modid-less.
		Expect(skipped).To(ConsistOf("d", "e"))
	})

	It("uses '0' suffix for loader id when loader version is empty", func() {
		bc := &buildContext{
			Spec:      &domain.PackSpec{PackName: "p", PackVersion: "0.1.0"},
			Lock:      &domain.PackLock{Mods: map[string]domain.LockedMod{}},
			McVersion: "1.21.1", Loader: "fabric", LoaderVersion: "",
		}
		m, _, _ := bc.buildCurseForgeManifest()
		Expect(m.Minecraft.ModLoaders[0].ID).To(Equal("fabric-0"))
	})
})

var _ = Describe("renderCurseForgeModlistHTML", func() {
	It("escapes mod names and links by projectID", func() {
		m := curseforgeManifest{
			Name: "pack", Version: "1.2.3", Author: "a <b>",
			Minecraft: curseforgeMinecraft{Version: "1.21.1", ModLoaders: []curseforgeModLoaderID{{ID: "neoforge-1.0", Primary: true}}},
		}
		refs := []manifestModRef{
			{Key: "k1", Name: "Foo & <Bar>", Version: "1.0", ProjectID: 42},
			{Key: "k2", Name: "", Version: "", ProjectID: 7},
		}
		html := renderCurseForgeModlistHTML(m, refs)
		Expect(html).To(ContainSubstring("Foo &amp; &lt;Bar&gt;"))
		Expect(html).To(ContainSubstring("a &lt;b&gt;"))
		Expect(html).To(ContainSubstring("https://www.curseforge.com/minecraft/mc-mods/42"))
		Expect(html).To(ContainSubstring("https://www.curseforge.com/minecraft/mc-mods/7"))
		// Empty name falls back to key.
		Expect(html).To(ContainSubstring("k2"))
		// Empty version renders as "?".
		Expect(html).To(ContainSubstring("Foo &amp; &lt;Bar&gt;</a> (1.0)"))
		Expect(html).To(ContainSubstring("k2</a> (?)"))
	})
})

var _ = Describe("BuildArtifactCF end-to-end", func() {
	It("bundles curseforge mods into manifest and non-cf mods into overrides/mods/", func() {
		dir := GinkgoT().TempDir()
		wd, _ := os.Getwd()
		Expect(os.Chdir(dir)).To(Succeed())
		DeferCleanup(func() { _ = os.Chdir(wd) })

		// Pre-seed the github-release cache so resolveModJar finds the jar
		// without trying to download it (the test does not need network).
		ghDir := filepath.Join(".cache", "github-release", "o", "r", "v1")
		Expect(os.MkdirAll(ghDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(ghDir, "a.jar"), []byte("gh"), 0644)).To(Succeed())

		spec := &domain.PackSpec{
			PackName: "p", ServerPackName: "p-server", PackVersion: "0.4.0",
			MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge:1.0"}, Author: "a",
		}
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"shared-mod": {Name: "Shared", Scope: "shared", Source: domain.LockedSource{Type: "curseforge", ModID: 100, FileID: 1001, FileName: "a.jar"}},
				"client-mod": {Name: "Client", Scope: "client", Source: domain.LockedSource{Type: "curseforge", ModID: 200, FileID: 2002, FileName: "a.jar"}},
				"server-mod": {Name: "Server", Scope: "server", Source: domain.LockedSource{Type: "curseforge", ModID: 300, FileID: 3003, FileName: "a.jar"}},
				"github-mod": {Name: "Github", Scope: "shared", Source: domain.LockedSource{Type: "github-release", Repo: "o/r", Tag: "v1", AssetName: "a.jar"}},
			},
		}
		out, err := BuildArtifactCF(spec, lock, "1.21.1", true)
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring("releases"))
		Expect(out).To(HaveSuffix("-cf.zip"))

		r, err := zip.OpenReader(out)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { _ = r.Close() })

		// Top-level entries: manifest.json, modlist.html, overrides/...
		names := map[string]bool{}
		for _, f := range r.File {
			names[f.Name] = true
		}
		Expect(names).To(HaveKey("manifest.json"))
		Expect(names).To(HaveKey("modlist.html"))
		Expect(names).To(HaveKey("overrides/mods/github-mod/a.jar"))
		Expect(names).NotTo(HaveKey("mods/a.jar"))
		// Server-scope mod is omitted; github-source mod is omitted from
		// manifest.files[] (it lives in overrides/mods/ instead).
		manifestBytes := readZipEntry(r, "manifest.json")
		var m curseforgeManifest
		Expect(json.Unmarshal(manifestBytes, &m)).To(Succeed())
		Expect(m.Files).To(HaveLen(2))
		ids := []int{m.Files[0].ProjectID, m.Files[1].ProjectID}
		Expect(ids).To(ConsistOf(100, 200))

		// modlist.html lists every shared+client mod (cf and non-cf alike).
		// Server-scope mods are excluded.
		htmlBytes := readZipEntry(r, "modlist.html")
		html := string(htmlBytes)
		Expect(html).To(ContainSubstring("<!DOCTYPE html>"))
		Expect(html).To(ContainSubstring("Shared"))
		Expect(html).To(ContainSubstring("Client"))
		Expect(html).To(ContainSubstring("Github"))
		Expect(html).To(ContainSubstring("overrides/mods/github-mod/"))
		Expect(html).NotTo(ContainSubstring("Server"))
	})

	It("rejects a fresh build when the target zip already exists and force is false", func() {
		dir := GinkgoT().TempDir()
		spec := &domain.PackSpec{PackName: "p", PackVersion: "0.1.0", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge:1.0"}}
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "curseforge", ModID: 1, FileID: 2, FileName: "a.jar"}},
			},
		}
		wd, _ := os.Getwd()
		Expect(os.Chdir(dir)).To(Succeed())
		DeferCleanup(func() { _ = os.Chdir(wd) })

		_, err := BuildArtifactCF(spec, lock, "1.21.1", true)
		Expect(err).NotTo(HaveOccurred())
		// Second run without force should fail.
		_, err = BuildArtifactCF(spec, lock, "1.21.1", false)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("already exists"))
	})

	It("errors when no curseforge-sourced mods are present in the lock", func() {
		dir := GinkgoT().TempDir()
		jar := filepath.Join(dir, "a.jar")
		Expect(os.WriteFile(jar, []byte("x"), 0644)).To(Succeed())
		spec := &domain.PackSpec{PackName: "p", PackVersion: "0.1.0", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge:1.0"}}
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"gh": {Name: "GH", Scope: "shared", Source: domain.LockedSource{Type: "github-release", Repo: "o/r", Tag: "v1", AssetName: "a.jar"}},
			},
		}
		wd, _ := os.Getwd()
		Expect(os.Chdir(dir)).To(Succeed())
		DeferCleanup(func() { _ = os.Chdir(wd) })

		_, err := BuildArtifactCF(spec, lock, "1.21.1", true)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no curseforge-sourced mods in lock"))
	})

	It("returns an error for a nil spec or nil lock", func() {
		_, err := BuildArtifactCF(nil, &domain.PackLock{Mods: map[string]domain.LockedMod{}}, "1.21.1", true)
		Expect(err).To(HaveOccurred())
		_, err = BuildArtifactCF(&domain.PackSpec{}, nil, "1.21.1", true)
		Expect(err).To(HaveOccurred())
	})
})

// readZipEntry fetches the bytes of an entry from an open zip reader.
func readZipEntry(r *zip.ReadCloser, name string) []byte {
	for _, f := range r.File {
		if f.Name == name {
			rc, err := f.Open()
			Expect(err).NotTo(HaveOccurred())
			defer rc.Close()
			data, err := io.ReadAll(rc)
			Expect(err).NotTo(HaveOccurred())
			return data
		}
	}
	Fail("entry not found: " + name)
	return nil
}

var _ = Describe("BuildArtifactCF overrides", func() {
	It("copies project-root config/ and resourcepacks/ into overrides/", func() {
		dir := GinkgoT().TempDir()
		wd, _ := os.Getwd()
		Expect(os.Chdir(dir)).To(Succeed())
		DeferCleanup(func() { _ = os.Chdir(wd) })

		Expect(os.MkdirAll("config", 0755)).To(Succeed())
		Expect(os.WriteFile("config/common.cfg", []byte("cfg"), 0644)).To(Succeed())
		Expect(os.MkdirAll("resourcepacks", 0755)).To(Succeed())
		Expect(os.WriteFile("resourcepacks/pack.zip", []byte("rp"), 0644)).To(Succeed())

		spec := &domain.PackSpec{PackName: "p", PackVersion: "0.1.0", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge:1.0"}}
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "curseforge", ModID: 1, FileID: 2, FileName: "a.jar"}},
			},
		}
		out, err := BuildArtifactCF(spec, lock, "1.21.1", true)
		Expect(err).NotTo(HaveOccurred())

		r, err := zip.OpenReader(out)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { _ = r.Close() })
		names := map[string]bool{}
		for _, f := range r.File {
			names[f.Name] = true
		}
		Expect(names).To(HaveKey("overrides/config/common.cfg"))
		Expect(names).To(HaveKey("overrides/resourcepacks/pack.zip"))
	})
})

var _ = Describe("sortedLockKeys", func() {
	It("returns keys in alphabetical order regardless of map iteration order", func() {
		mods := map[string]domain.LockedMod{
			"zeta":  {},
			"alpha": {},
			"mu":    {},
		}
		keys := sortedLockKeys(mods)
		Expect(strings.Join(keys, ",")).To(Equal("alpha,mu,zeta"))
	})
})
