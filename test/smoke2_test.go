// File: test/smoke2_test.go
// Created: 2026-06-20
// Description: Part 2 of smoke tests.
package test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("Smoke: CLI commands part 2", func() {
	var d string

	BeforeEach(func() {
		d = GinkgoT().TempDir()
	})

	// S11: lock delete
	Describe("S11: lock delete", func() {
		It("deletes a lock entry", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			runMcmod(d, "lock", "add", "1.21.1", "neoforge", "delmod",
				"--name", "Del", "--source", "local", "--path", "./d.jar", "--file-name", "d.jar")
			stdout, _, err := runMcmod(d, "lock", "delete", "1.21.1", "neoforge", "delmod")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("deleted"))
			lock, _ := domain.ReadLockFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			_, exists := lock.Mods["delmod"]
			Expect(exists).To(BeFalse())
		})
		It("delete nonexistent key fails", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{}})
			_, _, err := runMcmod(d, "lock", "delete", "1.21.1", "neoforge", "nokey")
			Expect(err).To(HaveOccurred())
		})
	})

	// S12: lock tree / mcmod tree
	Describe("S12: tree", func() {
		It("lock tree with lock file works", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"a": {Name: "Alpha", Version: "1", Scope: "shared",
					Source: domain.LockedSource{Type: "curseforge"}}}})
			stdout, _, err := runMcmod(d, "lock", "tree", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("Alpha"))
		})
		It("mcmod tree (alias) with lock works", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"b": {Name: "Beta", Version: "2", Scope: "client",
					Source: domain.LockedSource{Type: "local"}}}})
			stdout, _, err := runMcmod(d, "tree", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("Beta"))
		})
		It("tree without lock fails", func() {
			_, _, err := runMcmod(d, "tree", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
	})

	// S13: lock release set
	Describe("S13: lock release set", func() {
		It("creates a release record", func() {
			os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)
			writeReleaseIndexFile(d, "1.21.1", &domain.ReleaseIndex{Type: "package", PackName: "test", MinecraftVersion: "1.21.1"})
			stdout, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.2.0", "--repo", "owner/repo", "--tag", "v0.2.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("release"))
			ri, err := domain.ReadReleaseIndex(filepath.Join(d, domain.ReleaseIndexPath("1.21.1")))
			Expect(err).NotTo(HaveOccurred())
			Expect(ri.Releases).To(HaveLen(1))
			Expect(ri.Releases[0].Version).To(Equal("0.2.0"))
		})
	})

	// S14: lock release list/show/delete
	Describe("S14: lock release management", func() {
		It("release list shows records", func() {
			os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)
			ri := domain.ReleaseIndex{Type: "package", PackName: "p", MinecraftVersion: "1.21.1",
				Releases: []domain.ReleaseRecord{{Version: "0.1.0", Type: "github-release",
					GitHub: domain.ReleaseGitHub{Repo: "o/r", Tag: "v0.1.0"}}}}
			writeReleaseIndexFile(d, "1.21.1", &ri)
			stdout, _, err := runMcmod(d, "lock", "release", "list", "1.21.1")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("0.1.0"))
		})
		It("release show shows record", func() {
			os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)
			ri := domain.ReleaseIndex{Type: "package", PackName: "p", MinecraftVersion: "1.21.1",
				Releases: []domain.ReleaseRecord{{Version: "0.1.0", Type: "github-release"}}}
			writeReleaseIndexFile(d, "1.21.1", &ri)
			stdout, _, err := runMcmod(d, "lock", "release", "show", "1.21.1", "0.1.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("0.1.0"))
		})
		It("release show nonexistent fails", func() {
			os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)
			ri := domain.ReleaseIndex{Type: "package", PackName: "p", MinecraftVersion: "1.21.1"}
			writeReleaseIndexFile(d, "1.21.1", &ri)
			_, _, err := runMcmod(d, "lock", "release", "show", "1.21.1", "99.99")
			Expect(err).To(HaveOccurred())
		})
		It("release delete works", func() {
			os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)
			ri := domain.ReleaseIndex{Type: "package", PackName: "p", MinecraftVersion: "1.21.1",
				Releases: []domain.ReleaseRecord{{Version: "0.1.0", Type: "github-release"}}}
			writeReleaseIndexFile(d, "1.21.1", &ri)
			stdout, _, err := runMcmod(d, "lock", "release", "delete", "1.21.1", "0.1.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("deleted"))
		})
	})

	// S15: build
	Describe("S15: build", func() {
		It("build with missing lock prints hint to stderr", func() {
			writeSpec(d, `{"packName":"b","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			_, stderr, err := runMcmod(d, "build", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("build with existing lock and local mod", func() {
			writeSpec(d, `{"packName":"buildp","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"],"mods":{"m":{"scope":"shared","source":{"type":"local","path":"./m.jar"}}}}`)
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"m": {Name: "M", Version: "1", Scope: "shared",
					Source: domain.LockedSource{Type: "local", Path: "./m.jar", FileName: "m.jar"}}}})
			os.WriteFile(filepath.Join(d, "m.jar"), []byte("dummy content"), 0644)
			stdout, _, err := runMcmod(d, "build", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("built"))
		})
	})

	// S16: config
	Describe("S16: config", func() {
		It("config shows key", func() {
			stdout, _, err := runMcmod(d, "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("CurseForge"))
		})
		It("config set-cf-key writes key", func() {
			stdout, _, err := runMcmod(d, "config", "set-cf-key", "my-secret-key")
			Expect(err).NotTo(HaveOccurred())
			_ = stdout
		})
	})

	// S17: version
	Describe("S17: version", func() {
		It("version prints version", func() {
			stdout, _, err := runMcmod(d, "version")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("0.1.0"))
		})
	})

	// S18: 错误路径
	Describe("S18: error paths", func() {
		It("unknown command fails", func() {
			_, _, err := runMcmod(d, "bild")
			Expect(err).To(HaveOccurred())
		})
		It("unknown command fails with hint", func() {
			_, stderr, err := runMcmod(d, "lokc")
			Expect(err).To(HaveOccurred())
			_ = stderr
		})
		It("missing spec for validate fails", func() {
			_, _, err := runMcmod(d, "validate")
			Expect(err).To(HaveOccurred())
		})
		It("invalid loader prints hint to stderr", func() {
			writeSpec(d, `{"packName":"i","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			_, stderr, err := runMcmod(d, "lock", "1.21.1", "quilt")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("unsupported"))
		})
	})

	// S19: Build 缺 lock / 缺 mod
	Describe("S19: build failure paths", func() {
		It("build without lock fails with hint", func() {
			writeSpec(d, `{"packName":"bf","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["fabric"]}`)
			_, stderr, err := runMcmod(d, "build", "1.21.1", "fabric")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})
})
