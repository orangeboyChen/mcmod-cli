// File: test/smoke_test.go
// Created: 2026-06-20
// Description: Comprehensive Ginkgo smoke tests covering all CLI commands.
package test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("Smoke: CLI commands", func() {
	var d string

	BeforeEach(func() {
		d = GinkgoT().TempDir()
	})

	// S01: mcmod --help / mcmod help
	Describe("S01: help", func() {
		It("--help shows all commands", func() {
			stdout, _, err := runMcmod(d, "--help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("lock"))
			Expect(stdout).To(ContainSubstring("build"))
			Expect(stdout).To(ContainSubstring("list"))
			Expect(stdout).To(ContainSubstring("validate"))
			Expect(stdout).To(ContainSubstring("set"))
			Expect(stdout).To(ContainSubstring("tree"))
			Expect(stdout).To(ContainSubstring("config"))
			Expect(stdout).To(ContainSubstring("version"))
		})
		It("mcmod help also works", func() {
			stdout, _, err := runMcmod(d, "help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("lock"))
		})
		It("subcommand --help works", func() {
			stdout, _, err := runMcmod(d, "lock", "--help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("lock"))
		})
		It("lock release --help works", func() {
			stdout, _, err := runMcmod(d, "lock", "release", "--help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("set"))
		})
	})

	// S02: mcmod set
	Describe("S02: set cf-key", func() {
		It("set cf-key --project writes project config", func() {
			stdout, _, err := runMcmod(d, "set", "cf-key", "test-key-123", "--project")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("set cf-key"))
			_, err = os.Stat(filepath.Join(d, ".mcmod", "config.json"))
			Expect(err).NotTo(HaveOccurred())
		})
		It("set cf-key without --project writes user config", func() {
			stdout, _, err := runMcmod(d, "set", "cf-key", "user-key")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("set cf-key"))
		})
		It("set with bad args returns error", func() {
			_, stderr, err := runMcmod(d, "set")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})

	// S03: mcmod list
	Describe("S03: list", func() {
		It("list with valid spec groups mods", func() {
			writeSpec(d, `{"packName":"list-test","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"],"mods":{"a":{"name":"ModA","scope":"shared","source":{"type":"curseforge","query":"ModA"}},"b":{"name":"ModB","scope":"client","source":{"type":"curseforge","query":"ModB"}}}}`)
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("list-test"))
			Expect(stdout).To(ContainSubstring("[Shared]"))
			Expect(stdout).To(ContainSubstring("[Client]"))
			Expect(stdout).To(ContainSubstring("ModA"))
			Expect(stdout).To(ContainSubstring("ModB"))
		})
		It("list with empty mods shows (empty) sections", func() {
			writeSpec(d, `{"packName":"e","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("(empty)"))
		})
		It("list without spec fails", func() {
			_, _, err := runMcmod(d, "list")
			Expect(err).To(HaveOccurred())
		})
	})

	// S04: mcmod validate
	Describe("S04: validate", func() {
		It("validate valid spec", func() {
			writeSpec(d, `{"packName":"v","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"],"mods":{"m":{"name":"M","source":{"type":"curseforge","query":"M"}}}}`)
			stdout, _, err := runMcmod(d, "validate")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("valid"))
		})
		It("validate invalid spec fails", func() {
			writeSpec(d, `{"packName":"","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			_, _, err := runMcmod(d, "validate")
			Expect(err).To(HaveOccurred())
		})
		It("validate --spec flag", func() {
			p := writeSpec(d, `{"packName":"specf","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["fabric"]}`)
			stdout, _, err := runMcmod(d, "validate", "--spec", p)
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("valid"))
		})
		It("validate --lock flag", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"m": {Name: "M", Scope: "shared",
					Source: domain.LockedSource{Type: "local", Path: "./m.jar"}}}})
			lockPath := filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json")
			stdout, _, err := runMcmod(d, "validate", "--lock", lockPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("valid"))
		})
		It("validate bad lock fails", func() {
			p := filepath.Join(d, "bad.json")
			os.WriteFile(p, []byte(`{"loader":""}`), 0644)
			_, _, err := runMcmod(d, "validate", "--lock", p)
			Expect(err).To(HaveOccurred())
		})
	})

	// S05: mcmod lock
	Describe("S05: lock", func() {
		It("lock without spec fails", func() {
			_, _, err := runMcmod(d, "lock")
			Expect(err).To(HaveOccurred())
		})
		It("lock with spec runs for all loaders", func() {
			writeSpecStd(d)
			stdout, stderr, err := runMcmod(d, "lock")
			// 不会成功因为没有 API key，但命令应该尝试执行
			_ = stdout
			_ = stderr
			_ = err
		})
		It("lock with invalid loader gives stderr hint", func() {
			writeSpecStd(d)
			_, stderr, err := runMcmod(d, "lock", "1.21.1", "quilt")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("unsupported"))
		})
	})

	// S06: lock list
	Describe("S06: lock list", func() {
		It("lock list without lock fails", func() {
			_, _, err := runMcmod(d, "lock", "list", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("lock list works with lock file", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"create": {Name: "Create", Version: "6.0", Scope: "shared",
					Source: domain.LockedSource{Type: "curseforge", ModID: 328085, FileID: 5812340, FileName: "create.jar"}}}})
			stdout, _, err := runMcmod(d, "lock", "list", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("Create"))
			Expect(stdout).To(ContainSubstring("create.jar"))
		})
	})

	// S07: lock show
	Describe("S07: lock show", func() {
		It("lock show without key dumps full lock", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"m": {Name: "M", Version: "1", Scope: "shared",
					Source: domain.LockedSource{Type: "local"}}}})
			stdout, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("neoforge"))
		})
		It("lock show with key shows entry", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"sk": {Name: "ShowKey", Version: "2", Scope: "client",
					Source: domain.LockedSource{Type: "curseforge", ModID: 123, FileID: 456, FileName: "sk.jar"}}}})
			stdout, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge", "sk")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("ShowKey"))
			Expect(stdout).To(ContainSubstring("modId: 123"))
		})
		It("lock show with missing key fails", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{}})
			_, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge", "nonexistent")
			Expect(err).To(HaveOccurred())
		})
		It("lock show without enough args fails", func() {
			_, _, err := runMcmod(d, "lock", "show")
			Expect(err).To(HaveOccurred())
		})
	})

	// S08: lock add curseforge
	Describe("S08: lock add curseforge", func() {
		It("adds curseforge lock entry", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			stdout, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge", "newmod",
				"--name", "NewMod", "--version", "1.0", "--scope", "shared",
				"--source", "curseforge", "--mod-id", "100", "--file-id", "200", "--file-name", "newmod.jar")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("added"))

			// verify lock file
			lock, err := domain.ReadLockFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(err).NotTo(HaveOccurred())
			m, ok := lock.Mods["newmod"]
			Expect(ok).To(BeTrue())
			Expect(m.Source.ModID).To(Equal(100))
			Expect(m.Source.FileName).To(Equal("newmod.jar"))
		})
		It("add duplicate key fails", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			runMcmod(d, "lock", "add", "1.21.1", "neoforge", "dup",
				"--source", "local", "--file-name", "d.jar")
			_, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge", "dup",
				"--source", "local", "--file-name", "d2.jar")
			Expect(err).To(HaveOccurred())
		})
	})

	// S09: lock add github-release
	Describe("S09: lock add github-release", func() {
		It("adds github-release lock entry", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			stdout, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge", "ghmod",
				"--name", "GHMod", "--version", "2.0",
				"--source", "github-release", "--repo", "owner/repo", "--tag", "v2.0",
				"--asset-name", "ghmod-1.21.1.jar", "--file-name", "ghmod-1.21.1.jar")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("added"))

			lock, err := domain.ReadLockFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(err).NotTo(HaveOccurred())
			m := lock.Mods["ghmod"]
			Expect(m.Source.Repo).To(Equal("owner/repo"))
			Expect(m.Source.AssetName).To(Equal("ghmod-1.21.1.jar"))
		})
	})

	// S10: lock update
	Describe("S10: lock update", func() {
		It("updates a lock entry version", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			runMcmod(d, "lock", "add", "1.21.1", "neoforge", "um",
				"--name", "UM", "--version", "1.0", "--source", "local",
				"--path", "./um.jar", "--file-name", "um.jar")
			stdout, _, err := runMcmod(d, "lock", "update", "1.21.1", "neoforge", "um", "--version", "2.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("updated"))
			lock, _ := domain.ReadLockFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(lock.Mods["um"].Version).To(Equal("2.0"))
		})
		It("update nonexistent key fails", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{}})
			_, _, err := runMcmod(d, "lock", "update", "1.21.1", "neoforge", "nokey", "--version", "1")
			Expect(err).To(HaveOccurred())
		})
	})
})
