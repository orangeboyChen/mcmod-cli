// File: internal/service/build_jar_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/service/build_jar.go (jar resolution and cache population).

package service

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"
	"path/filepath"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("Service resolveModJar", func() {
	It("resolves local mod by spec path", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		os.WriteFile(filepath.Join(dir, "local.jar"), []byte("data"), 0644)
		spec := &domain.PackSpec{
			PackName: "p", PackVersion: "0.1.0", MinecraftVersion: "1.21.1",
			LoaderName: []string{"neoforge:1.0"},
			Mods: map[string]domain.ModSpec{
				"a": {Source: domain.ModSource{Type: "local", Path: "./local.jar"}},
			},
		}
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Source: domain.LockedSource{Type: "local", FileName: "local.jar"}},
			},
		}
		bc := &buildContext{Spec: spec, Lock: lock, McVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "1.0", RootDir: "."}
		p, err := bc.resolveModJar("a", lock.Mods["a"])
		Expect(err).NotTo(HaveOccurred())
		Expect(p).To(ContainSubstring("local.jar"))
	})

	It("resolves local mod by cache fallback when path empty", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		os.MkdirAll(filepath.Join(dir, ".cache", "local"), 0755)
		os.WriteFile(filepath.Join(dir, ".cache", "local", "fb.jar"), []byte("data"), 0644)
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Source: domain.LockedSource{Type: "local", FileName: "fb.jar"}},
			},
		}
		bc := &buildContext{Lock: lock, McVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "1.0", RootDir: "."}
		p, err := bc.resolveModJar("a", lock.Mods["a"])
		Expect(err).NotTo(HaveOccurred())
		Expect(p).To(ContainSubstring("fb.jar"))
	})

	It("resolves local mod via project root fallback", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		os.WriteFile(filepath.Join(dir, "root.jar"), []byte("data"), 0644)
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Source: domain.LockedSource{Type: "local", FileName: "root.jar"}},
			},
		}
		bc := &buildContext{Lock: lock, McVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "1.0", RootDir: "."}
		p, err := bc.resolveModJar("a", lock.Mods["a"])
		Expect(err).NotTo(HaveOccurred())
		Expect(p).To(ContainSubstring("root.jar"))
	})

	It("local with missing path and missing file errors", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Source: domain.LockedSource{Type: "local", FileName: "missing.jar"}},
			},
		}
		bc := &buildContext{Lock: lock, McVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "1.0", RootDir: "."}
		_, err := bc.resolveModJar("a", lock.Mods["a"])
		Expect(err).To(HaveOccurred())
	})

	It("resolves local mod path with {mcVersion}/{loader} placeholders", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		os.WriteFile(filepath.Join(dir, "1.21.1-neoforge.jar"), []byte("data"), 0644)
		spec := &domain.PackSpec{
			PackName: "p", PackVersion: "0.1.0", MinecraftVersion: "1.21.1",
			LoaderName: []string{"neoforge:1.0"},
			Mods: map[string]domain.ModSpec{
				"a": {Source: domain.ModSource{Type: "local", Path: "./{mcVersion}-{loader}.jar"}},
			},
		}
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Source: domain.LockedSource{Type: "local", FileName: "1.21.1-neoforge.jar"}},
			},
		}
		bc := &buildContext{Spec: spec, Lock: lock, McVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "1.0", RootDir: "."}
		p, err := bc.resolveModJar("a", lock.Mods["a"])
		Expect(err).NotTo(HaveOccurred())
		Expect(p).To(ContainSubstring("1.21.1-neoforge.jar"))
	})

	It("curseforge source missing modId errors", func() {
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Source: domain.LockedSource{Type: "curseforge"}},
			},
		}
		bc := &buildContext{Lock: lock, McVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "1.0", RootDir: "."}
		_, err := bc.resolveModJar("a", lock.Mods["a"])
		Expect(err).To(HaveOccurred())
	})

	It("curseforge source with cache hit resolves path", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		os.MkdirAll(filepath.Join(dir, ".cache", "curseforge", "123", "456"), 0755)
		os.WriteFile(filepath.Join(dir, ".cache", "curseforge", "123", "456", "mod.jar"), []byte("data"), 0644)
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Source: domain.LockedSource{Type: "curseforge", ModID: 123, FileID: 456, FileName: "mod.jar"}},
			},
		}
		bc := &buildContext{Lock: lock, McVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "1.0", RootDir: "."}
		p, err := bc.resolveModJar("a", lock.Mods["a"])
		Expect(err).NotTo(HaveOccurred())
		Expect(p).To(ContainSubstring("mod.jar"))
	})

	It("github-release source missing fields errors", func() {
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Source: domain.LockedSource{Type: "github-release"}},
			},
		}
		bc := &buildContext{Lock: lock, McVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "1.0", RootDir: "."}
		_, err := bc.resolveModJar("a", lock.Mods["a"])
		Expect(err).To(HaveOccurred())
	})

	It("github-release source with cache hit resolves path", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		os.MkdirAll(filepath.Join(dir, ".cache", "github-release", "o", "r", "v1"), 0755)
		os.WriteFile(filepath.Join(dir, ".cache", "github-release", "o", "r", "v1", "asset.jar"), []byte("data"), 0644)
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Source: domain.LockedSource{Type: "github-release", Repo: "o/r", Tag: "v1", AssetName: "asset.jar"}},
			},
		}
		bc := &buildContext{Lock: lock, McVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "1.0", RootDir: "."}
		p, err := bc.resolveModJar("a", lock.Mods["a"])
		Expect(err).NotTo(HaveOccurred())
		Expect(p).To(ContainSubstring("asset.jar"))
	})

	It("unsupported source type errors", func() {
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Source: domain.LockedSource{Type: "wat"}},
			},
		}
		bc := &buildContext{Lock: lock, McVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "1.0", RootDir: "."}
		_, err := bc.resolveModJar("a", lock.Mods["a"])
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("Service build_service modsForTarget", func() {
	It("splits shared/client/server scopes", func() {
		bc := &buildContext{
			Lock: &domain.PackLock{Mods: map[string]domain.LockedMod{
				"s":  {Scope: "shared"},
				"c":  {Scope: "client"},
				"sv": {Scope: "server"},
				"x":  {Scope: ""},
			}},
		}
		client := bc.modsForTarget("client")
		Expect(client).To(ContainElement("s"))
		Expect(client).To(ContainElement("c"))
		Expect(client).To(ContainElement("x"))
		Expect(client).NotTo(ContainElement("sv"))
		server := bc.modsForTarget("server")
		Expect(server).To(ContainElement("s"))
		Expect(server).To(ContainElement("sv"))
		Expect(server).To(ContainElement("x"))
		Expect(server).NotTo(ContainElement("c"))
	})
})

var _ = Describe("populateCache", func() {
	It("is a no-op for local sources (no network call)", func() {
		dir := GinkgoT().TempDir()
		wd, _ := os.Getwd()
		Expect(os.Chdir(dir)).To(Succeed())
		DeferCleanup(func() { _ = os.Chdir(wd) })

		bc := &buildContext{RootDir: dir, Loader: "neoforge", McVersion: "1.21.1"}
		jar := filepath.Join(dir, "mod.jar")
		Expect(os.WriteFile(jar, []byte("x"), 0644)).To(Succeed())

		src := &domain.LockedSource{Type: "local", Path: jar, FileName: "mod.jar"}
		Expect(bc.populateCache("local-mod", src)).To(Succeed())
	})
})

var _ = Describe("resolveModJar local error paths", func() {
	It("returns an error for a local mod with no path", func() {
		dir := GinkgoT().TempDir()
		bc := &buildContext{RootDir: dir, Loader: "neoforge", McVersion: "1.21.1"}
		_, err := bc.resolveModJar("a", domain.LockedMod{Source: domain.LockedSource{Type: "local", FileName: "a.jar"}})
		Expect(err).To(HaveOccurred())
	})

	It("returns an error when the local file is missing", func() {
		dir := GinkgoT().TempDir()
		bc := &buildContext{RootDir: dir, Loader: "neoforge", McVersion: "1.21.1"}
		_, err := bc.resolveModJar("a", domain.LockedMod{Source: domain.LockedSource{Type: "local", Path: "/no/such/m.jar", FileName: "m.jar"}})
		Expect(err).To(HaveOccurred())
	})

	It("returns an error for a curseforge source missing modId/fileId/fileName", func() {
		dir := GinkgoT().TempDir()
		bc := &buildContext{RootDir: dir, Loader: "neoforge", McVersion: "1.21.1"}
		_, err := bc.resolveModJar("a", domain.LockedMod{Source: domain.LockedSource{Type: "curseforge"}})
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("populateCache non-local error path", func() {
	It("returns an error for a remote source that cannot be downloaded", func() {
		dir := GinkgoT().TempDir()
		wd, _ := os.Getwd()
		Expect(os.Chdir(dir)).To(Succeed())
		DeferCleanup(func() { _ = os.Chdir(wd) })

		bc := &buildContext{RootDir: dir, Loader: "neoforge", McVersion: "1.21.1"}
		// URL pointing at an unresolvable host; populateCache will call
		// downloader.Download which will fail to reach it.
		src := &domain.LockedSource{Type: "github-release", URL: "https://127.0.0.1:1/x.jar", FileName: "x.jar", Repo: "o/r", Tag: "v1", AssetName: "x.jar"}
		err := bc.populateCache("github-mod", src)
		Expect(err).To(HaveOccurred())
	})
})
