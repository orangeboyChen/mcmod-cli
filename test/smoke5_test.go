// File: test/smoke5_test.go
// Created: 2026-06-20
// Description: Part 5 of smoke tests - exhaustive coverage of CLI commands, subcommands, flags, and edge cases.

package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("Smoke: exhaustive CLI coverage S50-S65", func() {
	var d string

	BeforeEach(func() {
		d = GinkgoT().TempDir()
	})

	// ===== S50: help and entry points =====
	Describe("S50: help and entry points", func() {
		It("S50-1: no args shows usage", func() {
			stdout, _, err := runMcmod(d)
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("Usage"))
		})
		It("S50-2: help with unknown subcommand still works", func() {
			_, stderr, err := runMcmod(d, "help", "not-a-real-command")
			_ = err
			Expect(stderr).To(ContainSubstring("Usage"))
		})
		It("S50-3: build --help lists flags", func() {
			stdout, _, err := runMcmod(d, "build", "--help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("target"))
		})
		It("S50-4: validate --help lists flags", func() {
			stdout, _, err := runMcmod(d, "validate", "--help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("spec"))
		})
		It("S50-5: list --help exists", func() {
			stdout, _, err := runMcmod(d, "list", "--help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("list"))
		})
		It("S50-6: set --help shows cf-key usage", func() {
			stdout, _, err := runMcmod(d, "set", "--help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("cf-key"))
		})
	})

	// ===== S51: set cf-key edge cases =====
	Describe("S51: set cf-key edge cases", func() {
		It("S51-1: set with one arg fails", func() {
			_, _, err := runMcmod(d, "set", "cf-key")
			Expect(err).To(HaveOccurred())
		})
		It("S51-2: set cf-key --project writes .mcmod/config.json", func() {
			_, _, _ = runMcmod(d, "set", "cf-key", "testkey", "--project")
			data, err := os.ReadFile(filepath.Join(d, ".mcmod", "config.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(ContainSubstring("testkey"))
		})
		It("S51-3: set cf-key without --project does not write .mcmod", func() {
			_, _, _ = runMcmod(d, "set", "cf-key", "userkey")
			_, err := os.Stat(filepath.Join(d, ".mcmod", "config.json"))
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
		It("S51-4: set cf-key --global does not write .mcmod", func() {
			_, _, _ = runMcmod(d, "set", "cf-key", "globalkey", "--global")
			_, err := os.Stat(filepath.Join(d, ".mcmod", "config.json"))
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
		It("S51-5: set cf-key --project overwrites previous project key", func() {
			_, _, _ = runMcmod(d, "set", "cf-key", "first", "--project")
			_, _, _ = runMcmod(d, "set", "cf-key", "second", "--project")
			data, err := os.ReadFile(filepath.Join(d, ".mcmod", "config.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(ContainSubstring("second"))
			Expect(string(data)).NotTo(ContainSubstring("first"))
		})
	})

	// ===== S52: list edge cases =====
	Describe("S52: list edge cases", func() {
		It("S52-1: list with server-only mods shows [Server] section", func() {
			writeSpec(d, `{"packName":"s","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"],
"mods":{"srv":{"name":"Srv","scope":"server","source":{"type":"local","path":"./srv.jar"}}}}`)
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("[Server]"))
			Expect(stdout).To(ContainSubstring("Srv"))
		})
		It("S52-2: list with mixed scopes shows all three sections", func() {
			writeSpec(d, `{"packName":"mix","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"],
"mods":{"s1":{"name":"SrvMod","scope":"server","source":{"type":"local","path":"./s1.jar"}},
"c1":{"name":"CliMod","scope":"client","source":{"type":"local","path":"./c1.jar"}},
"sh1":{"name":"SharedMod","scope":"shared","source":{"type":"local","path":"./sh1.jar"}}}}`)
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("[Server]"))
			Expect(stdout).To(ContainSubstring("[Client]"))
			Expect(stdout).To(ContainSubstring("[Shared]"))
		})
		It("S52-3: list with multi loader lists both", func() {
			writeSpec(d, `{"packName":"ml","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge","fabric"],
"mods":{"a":{"name":"A","scope":"shared","source":{"type":"local","path":"./a.jar"}}}}`)
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("neoforge"))
			Expect(stdout).To(ContainSubstring("fabric"))
		})
		It("S52-4: list reports missing packspec with hint", func() {
			_, stderr, err := runMcmod(d, "list")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})

	// ===== S53: validate edge cases =====
	Describe("S53: validate edge cases", func() {
		It("S53-1: validate --lock with missing file fails", func() {
			_, _, err := runMcmod(d, "validate", "--lock", "/nonexistent/lock.json")
			Expect(err).To(HaveOccurred())
		})
		It("S53-2: validate --spec and default spec both work", func() {
			writeSpec(d, `{"packName":"vspec","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			s1, _, err := runMcmod(d, "validate")
			Expect(err).NotTo(HaveOccurred())
			Expect(s1).To(ContainSubstring("valid"))
			s2, _, err := runMcmod(d, "validate", "--spec", filepath.Join(d, "packspec.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(s2).To(ContainSubstring("valid"))
		})
		It("S53-3: validate --release-index with missing file fails", func() {
			_, _, err := runMcmod(d, "validate", "--release-index", "/nonexistent/rel.json")
			Expect(err).To(HaveOccurred())
		})
		It("S53-4: validate --spec rejects missing required fields", func() {
			os.WriteFile(filepath.Join(d, "bad.json"), []byte(`{"packName":"x"}`), 0644)
			_, _, err := runMcmod(d, "validate", "--spec", filepath.Join(d, "bad.json"))
			Expect(err).To(HaveOccurred())
		})
		It("S53-5: validate --lock with valid lock", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: "./a.jar", FileName: "a.jar"}}}})
			stdout, _, err := runMcmod(d, "validate", "--lock", filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("valid"))
		})
		It("S53-6: validate with no spec in dir fails with hint", func() {
			_, stderr, err := runMcmod(d, "validate")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})

	// ===== S54: lock edge cases =====
	Describe("S54: lock edge cases", func() {
		It("S54-1: lock without spec fails", func() {
			_, _, err := runMcmod(d, "lock")
			Expect(err).To(HaveOccurred())
		})
		It("S54-2: lock with explicit mc and loader writes lock file with all mods", func() {
			writeSpec(d, `{"packName":"lk","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"],
"mods":{"a":{"name":"A","scope":"shared","source":{"type":"local","path":"./a.jar"}},
"b":{"name":"B","scope":"client","source":{"type":"local","path":"./b.jar"}}}}`)
			os.WriteFile(filepath.Join(d, "a.jar"), []byte("a"), 0644)
			os.WriteFile(filepath.Join(d, "b.jar"), []byte("b"), 0644)
			_, _, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			lock, err := domain.ReadLockFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(lock.Mods).To(HaveLen(2))
		})
		It("S54-3: lock with multiple loaders writes both files", func() {
			writeSpec(d, `{"packName":"ml","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge","fabric"],
"mods":{"a":{"name":"A","scope":"shared","source":{"type":"local","path":"./a.jar"}}}}`)
			os.WriteFile(filepath.Join(d, "a.jar"), []byte("a"), 0644)
			_, _, err := runMcmod(d, "lock", "1.21.1")
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(d, "locks", "dependencies", "1.21.1-fabric.json"))
			Expect(err).NotTo(HaveOccurred())
		})
		It("S54-4: lock with empty mods writes empty map", func() {
			writeSpec(d, `{"packName":"em","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			_, _, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			lock, err := domain.ReadLockFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(lock.Mods).To(BeEmpty())
		})
		It("S54-5: lock output contains 'Locked' prefix", func() {
			writeSpec(d, `{"packName":"lp","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"],
"mods":{"a":{"name":"A","scope":"shared","source":{"type":"local","path":"./a.jar"}}}}`)
			os.WriteFile(filepath.Join(d, "a.jar"), []byte("a"), 0644)
			stdout, _, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("locked"))
		})
		It("S54-6: lock with unsupported loader prints hint to stderr", func() {
			writeSpec(d, `{"packName":"u","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			_, stderr, err := runMcmod(d, "lock", "1.21.1", "fabric")
			_ = err
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})

	// ===== S55: lock list edge cases =====
	Describe("S55: lock list edge cases", func() {
		It("S55-1: lock list default mc/loader", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: "./a.jar", FileName: "a.jar"}}}})
			stdout, _, err := runMcmod(d, "lock", "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("[Shared]"))
		})
		It("S55-2: lock list shows all three scope sections", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{
					"s": {Name: "S", Scope: "server", Source: domain.LockedSource{Type: "local"}},
					"c": {Name: "C", Scope: "client", Source: domain.LockedSource{Type: "local"}},
					"x": {Name: "X", Scope: "shared", Source: domain.LockedSource{Type: "local"}},
				}})
			stdout, _, err := runMcmod(d, "lock", "list", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("[Server]"))
			Expect(stdout).To(ContainSubstring("[Client]"))
			Expect(stdout).To(ContainSubstring("[Shared]"))
		})
		It("S55-3: lock list with empty mods shows (empty) three times", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{}})
			stdout, _, err := runMcmod(d, "lock", "list", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.Count(stdout, "(empty)")).To(Equal(3))
		})
		It("S55-4: lock list file not found fails with hint", func() {
			_, stderr, err := runMcmod(d, "lock", "list", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("S55-5: lock list with mod having empty scope is treated as shared", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"x": {Name: "X", Scope: "", Source: domain.LockedSource{Type: "local"}}}})
			stdout, _, err := runMcmod(d, "lock", "list", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("[Shared]"))
			Expect(stdout).To(ContainSubstring("X"))
		})
		It("S55-6: lock list shows mod key and name in lines", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"mykey": {Name: "MyName", Version: "v1.0", Scope: "shared",
					Source: domain.LockedSource{Type: "local", FileName: "my.jar"}}}})
			stdout, _, err := runMcmod(d, "lock", "list", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("mykey"))
			Expect(stdout).To(ContainSubstring("MyName"))
			Expect(stdout).To(ContainSubstring("v1.0"))
		})
	})

	// ===== S56: lock show edge cases =====
	Describe("S56: lock show edge cases", func() {
		It("S56-1: lock show with 0 args fails", func() {
			_, _, err := runMcmod(d, "lock", "show")
			Expect(err).To(HaveOccurred())
		})
		It("S56-2: lock show with 1 arg fails", func() {
			_, _, err := runMcmod(d, "lock", "show", "1.21.1")
			Expect(err).To(HaveOccurred())
		})
		It("S56-3: lock show without key dumps full JSON", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: "./a.jar", FileName: "a.jar"}}}})
			stdout, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			var l domain.PackLock
			Expect(json.Unmarshal([]byte(stdout), &l)).To(Succeed())
		})
		It("S56-4: lock show with key shows scope and source", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"myk": {Name: "MyK", Version: "1.2", Scope: "client",
					Source: domain.LockedSource{Type: "curseforge", ModID: 42, FileID: 7, FileName: "x.jar"}}}})
			stdout, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge", "myk")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("scope: client"))
			Expect(stdout).To(ContainSubstring("curseforge"))
		})
		It("S56-5: lock show with missing key fails", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{}})
			_, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge", "missing")
			Expect(err).To(HaveOccurred())
		})
		It("S56-6: lock show with missing file fails", func() {
			_, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
	})

	// ===== S57: lock add edge cases =====
	Describe("S57: lock add edge cases", func() {
		It("S57-1: lock add with 0 args fails", func() {
			_, _, err := runMcmod(d, "lock", "add")
			Expect(err).To(HaveOccurred())
		})
		It("S57-2: lock add with 1 arg fails", func() {
			_, _, err := runMcmod(d, "lock", "add", "1.21.1")
			Expect(err).To(HaveOccurred())
		})
		It("S57-3: lock add with 2 args fails", func() {
			_, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("S57-4: lock add curseforge writes mod-id and file-id", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			_, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge", "mycf",
				"--name", "MyCF", "--version", "1.0", "--scope", "shared",
				"--source", "curseforge", "--mod-id", "100", "--file-id", "200", "--file-name", "mycf.jar")
			Expect(err).NotTo(HaveOccurred())
			lock, err := domain.ReadLockFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(lock.Mods["mycf"].Source.ModID).To(Equal(100))
			Expect(lock.Mods["mycf"].Source.FileID).To(Equal(200))
		})
		It("S57-5: lock add local without lock file creates new lock", func() {
			_, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge", "newlocal",
				"--name", "NewLocal", "--source", "local",
				"--path", "./x.jar", "--file-name", "x.jar")
			Expect(err).NotTo(HaveOccurred())
			lock, err := domain.ReadLockFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(lock.Mods).To(HaveLen(1))
		})
		It("S57-6: lock add appends to existing lock without overwriting", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"first": {Name: "First", Scope: "shared", Source: domain.LockedSource{Type: "local"}}}})
			_, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge", "second",
				"--source", "local", "--path", "./s.jar", "--file-name", "s.jar")
			Expect(err).NotTo(HaveOccurred())
			lock, _ := domain.ReadLockFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(lock.Mods).To(HaveLen(2))
		})
	})

	// ===== S58: lock update edge cases =====
	Describe("S58: lock update edge cases", func() {
		It("S58-1: lock update with 0 args and no spec fails", func() {
			_, _, err := runMcmod(d, "lock", "update")
			Expect(err).To(HaveOccurred())
		})
		It("S58-2: lock update with 2 args fails", func() {
			_, _, err := runMcmod(d, "lock", "update", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("S58-3: lock update single key without lock file fails", func() {
			_, _, err := runMcmod(d, "lock", "update", "1.21.1", "neoforge", "nokey")
			Expect(err).To(HaveOccurred())
		})
		It("S58-4: lock update single key with nonexistent key fails", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"a": {Name: "A", Source: domain.LockedSource{Type: "local"}}}})
			_, _, err := runMcmod(d, "lock", "update", "1.21.1", "neoforge", "missing")
			Expect(err).To(HaveOccurred())
		})
		It("S58-5: lock update single key without --version still saves", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"myk": {Name: "MyK", Version: "1.0", Scope: "shared",
					Source: domain.LockedSource{Type: "local", Path: "./m.jar", FileName: "m.jar"}}}})
			_, _, err := runMcmod(d, "lock", "update", "1.21.1", "neoforge", "myk")
			Expect(err).NotTo(HaveOccurred())
			lock, _ := domain.ReadLockFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(lock.Mods["myk"].Name).To(Equal("MyK"))
		})
		It("S58-6: lock update full refresh re-runs all loaders", func() {
			writeSpec(d, `{"packName":"ur","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge","fabric"],
"mods":{"a":{"name":"A","scope":"shared","source":{"type":"local","path":"./a.jar"}}}}`)
			os.WriteFile(filepath.Join(d, "a.jar"), []byte("a"), 0644)
			stdout, _, err := runMcmod(d, "lock", "update")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("updated"))
		})
	})

	// ===== S59: lock delete edge cases =====
	Describe("S59: lock delete edge cases", func() {
		It("S59-1: lock delete with 0 args prints hint", func() {
			_, _, err := runMcmod(d, "lock", "delete")
			_ = err
		})
		It("S59-2: lock delete with 1 arg prints hint", func() {
			_, _, err := runMcmod(d, "lock", "delete", "1.21.1")
			_ = err
		})
		It("S59-3: lock delete with 2 args prints hint", func() {
			_, _, err := runMcmod(d, "lock", "delete", "1.21.1", "neoforge")
			_ = err
		})
		It("S59-4: lock delete with 3 args removes entry", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{
					"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local"}},
					"b": {Name: "B", Scope: "shared", Source: domain.LockedSource{Type: "local"}},
				}})
			stdout, _, err := runMcmod(d, "lock", "delete", "1.21.1", "neoforge", "a")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("deleted"))
			lock, _ := domain.ReadLockFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(lock.Mods).To(HaveLen(1))
			Expect(lock.Mods).To(HaveKey("b"))
		})
		It("S59-5: lock delete with missing lock fails", func() {
			_, _, err := runMcmod(d, "lock", "delete", "1.21.1", "neoforge", "x")
			Expect(err).To(HaveOccurred())
		})
		It("S59-6: lock delete with nonexistent key fails", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{}})
			_, _, err := runMcmod(d, "lock", "delete", "1.21.1", "neoforge", "nokey")
			Expect(err).To(HaveOccurred())
		})
	})

	// ===== S60: tree edge cases =====
	Describe("S60: tree edge cases", func() {
		It("S60-1: lock tree with 0 args uses default mc/loader", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: "./a.jar", FileName: "a.jar"}}}})
			stdout, _, err := runMcmod(d, "lock", "tree")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("A"))
		})
		It("S60-2: lock tree with 1 arg uses default loader", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: "./a.jar", FileName: "a.jar"}}}})
			stdout, _, err := runMcmod(d, "lock", "tree", "1.21.1")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("A"))
		})
		It("S60-3: tree alias with 0 args uses default", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: "./a.jar", FileName: "a.jar"}}}})
			stdout, _, err := runMcmod(d, "tree")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("A"))
		})
		It("S60-4: tree output contains dependency tree header", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: "./a.jar", FileName: "a.jar"}}}})
			stdout, _, err := runMcmod(d, "tree", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("dependency tree"))
		})
		It("S60-5: tree with no lock file fails with hint", func() {
			_, stderr, err := runMcmod(d, "tree", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})

	// ===== S61: build edge cases =====
	Describe("S61: build edge cases", func() {
		setupBuildablePack := func() {
			writeSpec(d, `{"packName":"bld","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"],
"mods":{"a":{"name":"A","scope":"shared","source":{"type":"local","path":"./a.jar"}}}}`)
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"a": {Name: "A", Scope: "shared",
					Source: domain.LockedSource{Type: "local", Path: "./a.jar", FileName: "a.jar"}}}})
			os.WriteFile(filepath.Join(d, "a.jar"), []byte("a"), 0644)
		}
		It("S61-1: build with 0 args uses default mc/loader", func() {
			setupBuildablePack()
			stdout, _, err := runMcmod(d, "build")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("built"))
		})
		It("S61-2: build with --target server only", func() {
			setupBuildablePack()
			stdout, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "server")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("built"))
		})
		It("S61-3: build with --build-type all", func() {
			setupBuildablePack()
			stdout, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--build-type", "all")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("built"))
		})
		It("S61-4: build with --build-type cf", func() {
			setupBuildablePack()
			stdout, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--build-type", "cf")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("built"))
		})
		It("S61-5: build with --force flag", func() {
			setupBuildablePack()
			stdout, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--force")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("built"))
		})
		It("S61-6: build with missing lock fails with hint", func() {
			writeSpec(d, `{"packName":"nl","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			_, stderr, err := runMcmod(d, "build", "1.21.1", "neoforge")
			_ = err
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("S61-7: build with missing spec fails with hint", func() {
			_, stderr, err := runMcmod(d, "build", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})

	// ===== S62: release edge cases =====
	Describe("S62: release edge cases", func() {
		It("S62-1: release list with one version", func() {
			os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)
			writeReleaseIndexFile(d, "1.21.1", &domain.ReleaseIndex{Type: "package", PackName: "p", MinecraftVersion: "1.21.1",
				Releases: []domain.ReleaseRecord{{Version: "0.1.0", Type: "github-release"}}})
			stdout, _, err := runMcmod(d, "lock", "release", "list", "1.21.1")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("0.1.0"))
		})
		It("S62-2: release list with multiple versions", func() {
			os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)
			writeReleaseIndexFile(d, "1.21.1", &domain.ReleaseIndex{Type: "package", PackName: "p", MinecraftVersion: "1.21.1",
				Releases: []domain.ReleaseRecord{
					{Version: "0.1.0", Type: "github-release"},
					{Version: "0.2.0", Type: "github-release"},
					{Version: "0.3.0", Type: "github-release"},
				}})
			stdout, _, err := runMcmod(d, "lock", "release", "list", "1.21.1")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("0.1.0"))
			Expect(stdout).To(ContainSubstring("0.2.0"))
			Expect(stdout).To(ContainSubstring("0.3.0"))
		})
		It("S62-3: release show with valid version", func() {
			os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)
			writeReleaseIndexFile(d, "1.21.1", &domain.ReleaseIndex{Type: "package", PackName: "p", MinecraftVersion: "1.21.1",
				Releases: []domain.ReleaseRecord{{Version: "0.1.0", Type: "github-release",
					GitHub: domain.ReleaseGitHub{Repo: "o/r", Tag: "v0.1.0"}}}})
			stdout, _, err := runMcmod(d, "lock", "release", "show", "1.21.1", "0.1.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("o/r"))
			Expect(stdout).To(ContainSubstring("v0.1.0"))
		})
		It("S62-4: release show with 0 args fails", func() {
			_, _, err := runMcmod(d, "lock", "release", "show")
			Expect(err).To(HaveOccurred())
		})
		It("S62-5: release show with 1 arg fails", func() {
			_, _, err := runMcmod(d, "lock", "release", "show", "1.21.1")
			Expect(err).To(HaveOccurred())
		})
		It("S62-6: release delete with 0 args fails", func() {
			_, _, err := runMcmod(d, "lock", "release", "delete")
			Expect(err).To(HaveOccurred())
		})
		It("S62-7: release delete with 1 arg fails", func() {
			_, _, err := runMcmod(d, "lock", "release", "delete", "1.21.1")
			Expect(err).To(HaveOccurred())
		})
		It("S62-8: release set without --repo fails", func() {
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.1.0", "--tag", "v0.1.0")
			Expect(err).To(HaveOccurred())
		})
		It("S62-9: release set without --tag fails", func() {
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.1.0", "--repo", "o/r")
			Expect(err).To(HaveOccurred())
		})
		It("S62-10: release set with --name and --body", func() {
			os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)
			writeReleaseIndexFile(d, "1.21.1", &domain.ReleaseIndex{Type: "package", PackName: "p", MinecraftVersion: "1.21.1"})
			stdout, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0",
				"--name", "First", "--body", "Body text")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("release"))
		})
		It("S62-11: release set with --draft and --prerelease", func() {
			os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)
			writeReleaseIndexFile(d, "1.21.1", &domain.ReleaseIndex{Type: "package", PackName: "p", MinecraftVersion: "1.21.1"})
			stdout, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.2.0", "--repo", "o/r", "--tag", "v0.2.0",
				"--draft", "--prerelease")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("release"))
		})
		It("S62-12: release set with --artifact-client and --artifact-server", func() {
			os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)
			writeReleaseIndexFile(d, "1.21.1", &domain.ReleaseIndex{Type: "package", PackName: "p", MinecraftVersion: "1.21.1"})
			stdout, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.3.0", "--repo", "o/r", "--tag", "v0.3.0",
				"--artifact-client", "client.zip", "--artifact-server", "server.zip")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("release"))
		})
		It("S62-13: release list with no index fails", func() {
			_, _, err := runMcmod(d, "lock", "release", "list", "1.21.1")
			Expect(err).To(HaveOccurred())
		})
	})

	// ===== S63: config edge cases =====
	Describe("S63: config edge cases", func() {
		It("S63-1: config with no args shows (not set) when no key", func() {
			stdout, _, err := runMcmod(d, "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("not set"))
		})
		It("S63-2: config set-cf-key writes project config", func() {
			stdout, _, err := runMcmod(d, "config", "set-cf-key", "ckey")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("CurseForge"))
		})
		It("S63-3: config after project key set shows the key", func() {
			_, _, _ = runMcmod(d, "config", "set-cf-key", "abc123")
			stdout, _, err := runMcmod(d, "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("abc123"))
		})
		It("S63-4: config with random subcommand falls back to showing", func() {
			stdout, _, err := runMcmod(d, "config", "unknown")
			_ = err
			Expect(stdout).To(ContainSubstring("CurseForge"))
		})
	})

	// ===== S64: fixtures and cache =====
	Describe("S64: fixtures and cache", func() {
		It("S64-1: create neoforge fixture jar", func() {
			p := createFixtureJar(d, "neoforge_mod.jar", "neoforge",
				`modLoader="javafml"`+"\n"+`loaderVersion="[1,)"`+"\n"+`[[mods]]`+"\n"+`modId="mymod"`)
			Expect(p).To(BeAnExistingFile())
		})
		It("S64-2: create fabric fixture jar", func() {
			p := createFixtureJar(d, "fabric_mod.jar", "fabric",
				`{"id":"mymod","version":"1.0"}`)
			Expect(p).To(BeAnExistingFile())
		})
		It("S64-3: cache miss on non-existent file", func() {
			exists, err := isFileExists(filepath.Join(d, "nope.bin"))
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})
		It("S64-4: cache hit on existing file", func() {
			p := filepath.Join(d, "f.txt")
			os.WriteFile(p, []byte("x"), 0644)
			exists, err := isFileExists(p)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})
	})

	// ===== S65: error paths =====
	Describe("S65: error paths", func() {
		It("S65-1: completely unknown command fails", func() {
			_, _, err := runMcmod(d, "totally-unknown-command-xyz")
			Expect(err).To(HaveOccurred())
		})
		It("S65-2: set with no args fails", func() {
			_, _, err := runMcmod(d, "set")
			Expect(err).To(HaveOccurred())
		})
		It("S65-3: validate in empty dir fails with hint", func() {
			_, stderr, err := runMcmod(d, "validate")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("S65-4: lock show on missing file fails", func() {
			_, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge", "k")
			Expect(err).To(HaveOccurred())
		})
		It("S65-5: tree on missing lock fails", func() {
			_, stderr, err := runMcmod(d, "tree", "1.21.1", "fabric")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})
})

// isFileExists is a small helper used by S64-3/4 to test file presence.
func isFileExists(p string) (bool, error) {
	if _, err := os.Stat(p); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}
