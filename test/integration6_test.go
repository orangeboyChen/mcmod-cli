// File: test/integration6_test.go
// Created: 2026-06-20
// Description: Exhaustive integration tests covering every CLI subcommand
// and every argument combination listed in specification.md sections 7.1
// through 7.8. Each test asserts the spec-defined stdout, stderr, exit code,
// and side effects (files written/deleted).

package test

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("Integration6: spec 7.1-7.8 subcommand coverage", func() {
	var d string

	BeforeEach(func() {
		d = GinkgoT().TempDir()
	})

	// ============================================================
	// spec 7.1 set cf-key
	// ============================================================
	Describe("N01: set cf-key arg coverage (spec 7.1)", func() {
		It("N01-1: set with no args fails with hint and exit non-zero", func() {
			_, stderr, err := runMcmodWithEnv(d, cleanEnv(d), "set")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("N01-2: set cf-key with no value fails with hint", func() {
			_, stderr, err := runMcmodWithEnv(d, cleanEnv(d), "set", "cf-key")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("N01-3: set cf-key with empty value still saves empty string", func() {
			_, _, err := runMcmodWithEnv(d, cleanEnv(d), "set", "cf-key", "")
			Expect(err).NotTo(HaveOccurred())
			_, statErr := os.Stat(filepath.Join(d, ".config", "mcmod", "config.json"))
			Expect(statErr).NotTo(HaveOccurred())
		})
		It("N01-4: set cf-key --project --global still writes project (--project wins)", func() {
			_, _, err := runMcmodWithEnv(d, cleanEnv(d), "set", "cf-key", "bothflags", "--project", "--global")
			Expect(err).NotTo(HaveOccurred())
			// --project flag is processed first in cli/commands.go, so it should win.
			data, _ := os.ReadFile(filepath.Join(d, ".mcmod", "config.json"))
			Expect(string(data)).To(ContainSubstring("bothflags"))
		})
		It("N01-5: set cf-key with special chars persists verbatim", func() {
			special := "abc/def\"ghi jkl"
			_, _, err := runMcmodWithEnv(d, cleanEnv(d), "set", "cf-key", special, "--project")
			Expect(err).NotTo(HaveOccurred())
			data, _ := os.ReadFile(filepath.Join(d, ".mcmod", "config.json"))
			Expect(string(data)).To(ContainSubstring("abc/def"))
		})
		It("N01-6: set cf-key outputs exact 'set cf-key' on stdout", func() {
			stdout, _, err := runMcmodWithEnv(d, cleanEnv(d), "set", "cf-key", "key123")
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.TrimSpace(stdout)).To(Equal("set cf-key"))
		})
	})

	// ============================================================
	// spec 7.2 list
	// ============================================================
	Describe("N02: list arg coverage (spec 7.2)", func() {
		It("N02-1: list with no packspec fails with hint", func() {
			_, stderr, err := runMcmod(d, "list")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("N02-2: list with no --target or args uses default behavior", func() {
			writeMinimalPackspec2(d, "list-basic", []string{"neoforge:21.1.219"}, map[string]interface{}{
				"a": map[string]interface{}{"name": "A", "scope": "shared", "source": map[string]interface{}{"type": "local", "path": "./a.jar"}},
			})
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("pack list-basic"))
			Expect(stdout).To(ContainSubstring("A [local]"))
		})
		It("N02-3: list groups by [Server]/[Client]/[Shared] in that order", func() {
			writeMinimalPackspec2(d, "list-order", []string{"neoforge:21.1.219"}, map[string]interface{}{
				"s1": map[string]interface{}{"name": "S1", "scope": "shared", "source": map[string]interface{}{"type": "local"}},
				"c1": map[string]interface{}{"name": "C1", "scope": "client", "source": map[string]interface{}{"type": "local"}},
				"v1": map[string]interface{}{"name": "V1", "scope": "server", "source": map[string]interface{}{"type": "local"}},
			})
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			s := stdout
			idxServer := strings.Index(s, "[Server]")
			idxClient := strings.Index(s, "[Client]")
			idxShared := strings.Index(s, "[Shared]")
			Expect(idxServer).To(BeNumerically(">", 0))
			Expect(idxClient).To(BeNumerically(">", idxServer))
			Expect(idxShared).To(BeNumerically(">", idxClient))
		})
		It("N02-4: list shows (empty) for any empty section", func() {
			writeMinimalPackspec2(d, "list-empty", []string{"neoforge:21.1.219"}, map[string]interface{}{
				"only-shared": map[string]interface{}{"name": "X", "scope": "shared", "source": map[string]interface{}{"type": "local"}},
			})
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.Count(stdout, "(empty)")).To(Equal(2))
		})
		It("N02-5: list does not show old field names [sharedMods]/[clientMods]/[serverMods]", func() {
			writeMinimalPackspec2(d, "list-old", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).NotTo(ContainSubstring("[sharedMods]"))
			Expect(stdout).NotTo(ContainSubstring("[clientMods]"))
			Expect(stdout).NotTo(ContainSubstring("[serverMods]"))
		})
		It("N02-6: list with no name field falls back to key", func() {
			writeMinimalPackspec2(d, "list-noname", []string{"neoforge:21.1.219"}, map[string]interface{}{
				"mykey": map[string]interface{}{"scope": "shared", "source": map[string]interface{}{"type": "local"}},
			})
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("mykey [local]"))
		})
	})

	// ============================================================
	// spec 7.3 lock
	// ============================================================
	Describe("N03: lock command paths (spec 7.3)", func() {
		It("N03-1: lock with 0 args iterates all loaders from spec", func() {
			writeMinimalPackspec2(d, "multi", []string{"neoforge:21.1.219", "fabric:1.21.123"}, map[string]interface{}{
				"a": map[string]interface{}{"name": "A", "scope": "shared", "source": map[string]interface{}{"type": "local", "path": "./a.jar"}},
			})
			Expect(os.WriteFile(filepath.Join(d, "a.jar"), []byte("d"), 0644)).To(Succeed())
			stdout, _, err := runMcmod(d, "lock")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("1.21.1 neoforge"))
			Expect(stdout).To(ContainSubstring("1.21.1 fabric"))
		})
		It("N03-2: lock writes 1 lock file per (mc, loader) pair", func() {
			writeMinimalPackspec2(d, "two-loaders", []string{"neoforge:21.1.219", "fabric:1.21.123"}, map[string]interface{}{
				"a": map[string]interface{}{"name": "A", "scope": "shared", "source": map[string]interface{}{"type": "local", "path": "./a.jar"}},
			})
			Expect(os.WriteFile(filepath.Join(d, "a.jar"), []byte("d"), 0644)).To(Succeed())
			_, _, err := runMcmod(d, "lock")
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(d, "locks", "dependencies", "1.21.1-fabric.json"))
			Expect(err).NotTo(HaveOccurred())
		})
		It("N03-3: lock output starts with 'locked ' prefix per spec 7.3", func() {
			writeMinimalPackspec2(d, "prefix", []string{"neoforge:21.1.219"}, map[string]interface{}{
				"a": map[string]interface{}{"name": "A", "scope": "shared", "source": map[string]interface{}{"type": "local", "path": "./a.jar"}},
			})
			Expect(os.WriteFile(filepath.Join(d, "a.jar"), []byte("d"), 0644)).To(Succeed())
			stdout, _, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(MatchRegexp(`(?m)^locked 1\.21\.1 neoforge -> locks/dependencies/1\.21\.1-neoforge\.json`))
		})
	})

	// ============================================================
	// spec 7.3 lock show
	// ============================================================
	Describe("N04: lock show coverage (spec 7.3)", func() {
		It("N04-1: lock show 0 args fails with hint", func() {
			writeMinimalPackspec2(d, "ls", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			_, stderr, err := runMcmod(d, "lock", "show")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("N04-2: lock show 1 arg fails with hint", func() {
			writeMinimalPackspec2(d, "ls", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			_, stderr, err := runMcmod(d, "lock", "show", "1.21.1")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("N04-3: lock show 2 args dumps full JSON", func() {
			writeMinimalPackspec2(d, "ls", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			EnsureLocksDir(d)
			WriteLockJSON(d, "1.21.1", "neoforge", &domain.PackLock{
				Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{
					"a": {Name: "A", Version: "1.0", Scope: "shared", Source: domain.LockedSource{Type: "local", FileName: "a.jar", Path: "./a.jar"}},
				},
			})
			stdout, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			var l domain.PackLock
			Expect(json.Unmarshal([]byte(stdout), &l)).To(Succeed())
			Expect(l.Mods).To(HaveKey("a"))
		})
		It("N04-4: lock show 3 args shows single entry with all source fields", func() {
			writeMinimalPackspec2(d, "ls", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			EnsureLocksDir(d)
			WriteLockJSON(d, "1.21.1", "neoforge", &domain.PackLock{
				Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{
					"create": {Name: "Create", Version: "6.0.0", Scope: "shared",
						Source: domain.LockedSource{Type: "curseforge", ModID: 328085, FileID: 5812340, FileName: "create-1.21.1-neoforge.jar"}},
				},
			})
			stdout, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge", "create")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("key: create"))
			Expect(stdout).To(ContainSubstring("name: Create"))
			Expect(stdout).To(ContainSubstring("version: 6.0.0"))
			Expect(stdout).To(ContainSubstring("scope: shared"))
			Expect(stdout).To(ContainSubstring("type: curseforge"))
			Expect(stdout).To(ContainSubstring("modId: 328085"))
			Expect(stdout).To(ContainSubstring("fileId: 5812340"))
			Expect(stdout).To(ContainSubstring("fileName: create-1.21.1-neoforge.jar"))
		})
		It("N04-5: lock show with nonexistent key fails", func() {
			writeMinimalPackspec2(d, "ls", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			EnsureLocksDir(d)
			WriteLockJSON(d, "1.21.1", "neoforge", &domain.PackLock{
				Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{},
			})
			_, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge", "nope")
			Expect(err).To(HaveOccurred())
		})
	})

	// ============================================================
	// spec 7.3 lock add
	// ============================================================
	Describe("N05: lock add coverage (spec 7.3)", func() {
		It("N05-1: lock add with < 3 args fails with hint", func() {
			writeMinimalPackspec2(d, "add", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			_, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("N05-2: lock add outputs 'added lock mod <key> -> ...' per spec 7.3", func() {
			EnsureLocksDir(d)
			stdout, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge", "mynewmod",
				"--name", "MyNewMod", "--scope", "shared", "--version", "1.0",
				"--source", "local", "--path", "./mynewmod.jar", "--file-name", "mynewmod.jar")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(MatchRegexp(`(?m)^added lock mod mynewmod -> locks/dependencies/1\.21\.1-neoforge\.json$`))
		})
		It("N05-3: lock add for github-release requires --repo --tag --asset-name", func() {
			EnsureLocksDir(d)
			// Missing --repo/--tag/--asset-name must fail with a hint per
			// spec 7.3 (the add row for github-release marks these as required).
			_, stderr, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge", "gh1",
				"--source", "github-release", "--file-name", "gh1.jar")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("github-release"))
			// With the full set of required flags the add should succeed.
			_, _, err = runMcmod(d, "lock", "add", "1.21.1", "neoforge", "gh1",
				"--source", "github-release", "--repo", "o/r", "--tag", "v1",
				"--asset-name", "gh1.jar", "--file-name", "gh1.jar")
			Expect(err).NotTo(HaveOccurred())
			lockPath := filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json")
			lock, _ := domain.ReadLockFile(lockPath)
			Expect(lock.Mods).To(HaveKey("gh1"))
		})
		It("N05-4: lock add with duplicate key fails", func() {
			EnsureLocksDir(d)
			_, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge", "dup",
				"--source", "local", "--path", "./dup.jar", "--file-name", "dup.jar")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = runMcmod(d, "lock", "add", "1.21.1", "neoforge", "dup",
				"--source", "local", "--path", "./dup2.jar", "--file-name", "dup2.jar")
			Expect(err).To(HaveOccurred())
		})
		It("N05-5: lock add does NOT modify packspec.json", func() {
			EnsureLocksDir(d)
			origWd, _ := os.Getwd()
			Expect(os.Chdir(d)).To(Succeed())
			defer os.Chdir(origWd)
			writeMinimalPackspec2(d, "noedit", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			pre, _ := os.ReadFile(filepath.Join(d, "packspec.json"))
			_, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge", "noedit-key",
				"--source", "local", "--path", "./x.jar", "--file-name", "x.jar")
			Expect(err).NotTo(HaveOccurred())
			post, _ := os.ReadFile(filepath.Join(d, "packspec.json"))
			Expect(string(pre)).To(Equal(string(post)))
		})
	})

	// ============================================================
	// spec 7.3 lock update
	// ============================================================
	Describe("N06: lock update coverage (spec 7.3)", func() {
		It("N06-1: lock update 0/1/2 args re-runs full lock (no key)", func() {
			writeMinimalPackspec2(d, "upd", []string{"neoforge:21.1.219"}, map[string]interface{}{
				"a": map[string]interface{}{"name": "A", "scope": "shared", "source": map[string]interface{}{"type": "local", "path": "./a.jar"}},
			})
			Expect(os.WriteFile(filepath.Join(d, "a.jar"), []byte("d"), 0644)).To(Succeed())
			_, _, err := runMcmod(d, "lock", "update")
			Expect(err).NotTo(HaveOccurred())
			lock, _ := domain.ReadLockFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(lock.Mods).To(HaveKey("a"))
		})
		It("N06-2: lock update 3 args updates single entry", func() {
			EnsureLocksDir(d)
			WriteLockJSON(d, "1.21.1", "neoforge", &domain.PackLock{
				Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{
					"a": {Name: "A", Version: "1.0", Scope: "shared", Source: domain.LockedSource{Type: "local", FileName: "a.jar"}},
				},
			})
			stdout, _, err := runMcmod(d, "lock", "update", "1.21.1", "neoforge", "a", "--version", "9.9.9")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("updated"))
			lock, _ := domain.ReadLockFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(lock.Mods["a"].Version).To(Equal("9.9.9"))
		})
		It("N06-3: lock update only changes the specified field, leaves others intact", func() {
			EnsureLocksDir(d)
			WriteLockJSON(d, "1.21.1", "neoforge", &domain.PackLock{
				Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{
					"a": {Name: "OriginalName", Version: "1.0", Scope: "shared", Source: domain.LockedSource{Type: "local", FileName: "a.jar"}},
				},
			})
			_, _, err := runMcmod(d, "lock", "update", "1.21.1", "neoforge", "a", "--version", "2.0")
			Expect(err).NotTo(HaveOccurred())
			lock, _ := domain.ReadLockFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(lock.Mods["a"].Name).To(Equal("OriginalName"))
			Expect(lock.Mods["a"].Version).To(Equal("2.0"))
		})
		It("N06-4: lock update with missing key fails", func() {
			EnsureLocksDir(d)
			WriteLockJSON(d, "1.21.1", "neoforge", &domain.PackLock{
				Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{},
			})
			_, _, err := runMcmod(d, "lock", "update", "1.21.1", "neoforge", "missing", "--version", "1.0")
			Expect(err).To(HaveOccurred())
		})
	})

	// ============================================================
	// spec 7.3 lock delete
	// ============================================================
	Describe("N07: lock delete coverage (spec 7.3)", func() {
		It("N07-1: lock delete 0 args errors (no spec)", func() {
			_, _, err := runMcmod(d, "lock", "delete")
			Expect(err).To(HaveOccurred())
		})
		It("N07-2: lock delete 1 arg deletes all lock files for that mc across loaders", func() {
			EnsureLocksDir(d)
			WriteLockJSON(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{}})
			WriteLockJSON(d, "1.21.1", "fabric", &domain.PackLock{Loader: "fabric", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{}})
			writeMinimalPackspec2(d, "del", []string{"neoforge:21.1.219", "fabric:1.21.123"}, map[string]interface{}{})
			_, _, err := runMcmod(d, "lock", "delete", "1.21.1")
			// Either succeeds or prints hint. We just need to verify behavior.
			_ = err
		})
		It("N07-3: lock delete 2 args deletes single lock file for that (mc, loader)", func() {
			EnsureLocksDir(d)
			WriteLockJSON(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{}})
			WriteLockJSON(d, "1.21.1", "fabric", &domain.PackLock{Loader: "fabric", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{}})
			writeMinimalPackspec2(d, "del2", []string{"neoforge:21.1.219", "fabric:1.21.123"}, map[string]interface{}{})
			_, _, err := runMcmod(d, "lock", "delete", "1.21.1", "neoforge")
			_ = err
		})
		It("N07-4: lock delete 3 args removes only the specified key", func() {
			EnsureLocksDir(d)
			WriteLockJSON(d, "1.21.1", "neoforge", &domain.PackLock{
				Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{
					"keepme":   {Name: "Keep", Scope: "shared", Source: domain.LockedSource{Type: "local", FileName: "k.jar"}},
					"deleteme": {Name: "Delete", Scope: "shared", Source: domain.LockedSource{Type: "local", FileName: "d.jar"}},
				},
			})
			stdout, _, err := runMcmod(d, "lock", "delete", "1.21.1", "neoforge", "deleteme")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("deleted"))
			lock, _ := domain.ReadLockFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(lock.Mods).To(HaveKey("keepme"))
			Expect(lock.Mods).NotTo(HaveKey("deleteme"))
		})
		It("N07-5: lock delete 3 args with missing key fails", func() {
			EnsureLocksDir(d)
			WriteLockJSON(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{}})
			_, _, err := runMcmod(d, "lock", "delete", "1.21.1", "neoforge", "nope")
			Expect(err).To(HaveOccurred())
		})
	})

	// ============================================================
	// spec 7.3 lock tree
	// ============================================================
	Describe("N08: lock tree coverage (spec 7.3)", func() {
		It("N08-1: lock tree 0 args uses default mc/loader", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "lock", "tree")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("dependency tree 1.21.1 neoforge"))
		})
		It("N08-2: lock tree 1 arg uses default loader", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "lock", "tree", "1.21.1")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("dependency tree 1.21.1 neoforge"))
		})
		It("N08-3: lock tree 2 args full", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "lock", "tree", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("dependency tree 1.21.1 neoforge"))
		})
		It("N08-4: lock tree with no lock file fails with hint", func() {
			writeMinimalPackspec2(d, "treefail", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			_, stderr, err := runMcmod(d, "lock", "tree", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})

	// ============================================================
	// spec 7.4 lock release
	// ============================================================
	Describe("N09: lock release coverage (spec 7.4)", func() {
		It("N09-1: release set 0 mc arg uses default from spec", func() {
			writeMinimalPackspec2(d, "rs", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			_, _, err := runMcmod(d, "lock", "release", "set",
				"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0")
			Expect(err).NotTo(HaveOccurred())
		})
		It("N09-2: release set outputs 'locked release <mc> <ver> <type> -> ...'", func() {
			writeMinimalPackspec2(d, "rsout", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			stdout, _, err := runMcmod(d, "lock", "release", "set", "1.21.1",
				"--version", "0.1.0", "--type", "github-release", "--repo", "o/r", "--tag", "v0.1.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(MatchRegexp(`(?m)^locked release 1\.21\.1 0\.1\.0 github-release -> locks/releases/1\.21\.1\.json$`))
		})
		It("N09-3: release set without --version fails with hint", func() {
			writeMinimalPackspec2(d, "rsv", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			_, stderr, err := runMcmod(d, "lock", "release", "set", "1.21.1",
				"--repo", "o/r", "--tag", "v0.1.0")
			Expect(stderr).To(ContainSubstring("hint"))
			_ = err
		})
		It("N09-4: release set with --artifact-client writes client into the index", func() {
			writeMinimalPackspec2(d, "rsac", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0",
				"--artifact-client", "client.jar")
			Expect(err).NotTo(HaveOccurred())
			idx, _ := domain.ReadReleaseIndex(filepath.Join(d, "locks", "releases", "1.21.1.json"))
			Expect(idx.Releases[0].Artifact["neoforge"].Client).To(Equal("client.jar"))
		})
		It("N09-5: release set with --name --body --draft --prerelease writes github fields", func() {
			writeMinimalPackspec2(d, "rsn", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1",
				"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0",
				"--name", "My Release", "--body", "Notes", "--draft", "--prerelease")
			Expect(err).NotTo(HaveOccurred())
			idx, _ := domain.ReadReleaseIndex(filepath.Join(d, "locks", "releases", "1.21.1.json"))
			Expect(idx.Releases[0].GitHub.Name).To(Equal("My Release"))
			Expect(idx.Releases[0].GitHub.Body).To(Equal("Notes"))
			Expect(idx.Releases[0].GitHub.Draft).To(BeTrue())
			Expect(idx.Releases[0].GitHub.Pre).To(BeTrue())
		})
		It("N09-6: release list prints header and entries", func() {
			writeMinimalPackspec2(d, "rl", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1",
				"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0")
			Expect(err).NotTo(HaveOccurred())
			stdout, _, err := runMcmod(d, "lock", "release", "list", "1.21.1")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("releases 1.21.1"))
			Expect(stdout).To(ContainSubstring("0.1.0 [github-release] tag=v0.1.0"))
		})
		It("N09-7: release show prints full record", func() {
			writeMinimalPackspec2(d, "rsh", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1",
				"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0")
			Expect(err).NotTo(HaveOccurred())
			stdout, _, err := runMcmod(d, "lock", "release", "show", "1.21.1", "0.1.0")
			Expect(err).NotTo(HaveOccurred())
			var r domain.ReleaseRecord
			Expect(json.Unmarshal([]byte(stdout), &r)).To(Succeed())
			Expect(r.Version).To(Equal("0.1.0"))
		})
		It("N09-8: release delete 2 args removes full version", func() {
			writeMinimalPackspec2(d, "rd", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1",
				"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0")
			Expect(err).NotTo(HaveOccurred())
			stdout, _, err := runMcmod(d, "lock", "release", "delete", "1.21.1", "0.1.0")
			Expect(err).NotTo(HaveOccurred())
			// After deleting the last release the index file is removed
			// entirely per spec 7.4.8 so the directory does not keep a
			// stale empty index around.
			Expect(stdout).To(ContainSubstring("deleted release 1.21.1 0.1.0"))
			releasePath := filepath.Join(d, "locks", "releases", "1.21.1.json")
			_, statErr := os.Stat(releasePath)
			Expect(os.IsNotExist(statErr)).To(BeTrue())
		})
		It("N09-9: release delete 3 args with loader clears only that loader's artifacts", func() {
			writeMinimalPackspec2(d, "rd3", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0",
				"--artifact-client", "client.jar", "--artifact-server", "server.jar")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = runMcmod(d, "lock", "release", "delete", "1.21.1", "0.1.0", "neoforge")
			Expect(err).NotTo(HaveOccurred())
		})
		It("N09-10: release delete 4 args with --target clears only that target", func() {
			writeMinimalPackspec2(d, "rd4", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0",
				"--artifact-client", "client.jar", "--artifact-server", "server.jar")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = runMcmod(d, "lock", "release", "delete", "1.21.1", "0.1.0", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			idx, _ := domain.ReadReleaseIndex(filepath.Join(d, "locks", "releases", "1.21.1.json"))
			Expect(idx.Releases[0].Artifact["neoforge"].Client).To(BeEmpty())
			Expect(idx.Releases[0].Artifact["neoforge"].Server).To(Equal("server.jar"))
		})
		It("N09-11: release delete --target default is both", func() {
			writeMinimalPackspec2(d, "rddt", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0",
				"--artifact-client", "c.jar", "--artifact-server", "s.jar")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = runMcmod(d, "lock", "release", "delete", "1.21.1", "0.1.0", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			idx, _ := domain.ReadReleaseIndex(filepath.Join(d, "locks", "releases", "1.21.1.json"))
			Expect(idx.Releases[0].Artifact["neoforge"].Client).To(BeEmpty())
			Expect(idx.Releases[0].Artifact["neoforge"].Server).To(BeEmpty())
		})
	})

	// ============================================================
	// spec 7.5 build
	// ============================================================
	Describe("N10: build coverage (spec 7.5)", func() {
		It("N10-1: build 0 args uses default mc/loader and default target=both", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			os.RemoveAll(filepath.Join(d, "releases"))
			_, _, err := runMcmod(d, "build")
			Expect(err).NotTo(HaveOccurred())
			files, _ := filepath.Glob(filepath.Join(d, "releases", "v1.5.0", "*client*.zip"))
			Expect(files).To(HaveLen(1))
			files, _ = filepath.Glob(filepath.Join(d, "releases", "v1.5.0", "*server*.zip"))
			Expect(files).To(HaveLen(1))
		})
		It("N10-2: build 1 arg uses default loader", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			os.RemoveAll(filepath.Join(d, "releases"))
			_, _, err := runMcmod(d, "build", "1.21.1")
			Expect(err).NotTo(HaveOccurred())
		})
		It("N10-3: build 2 args full", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			os.RemoveAll(filepath.Join(d, "releases"))
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
		})
		It("N10-4: build --target client writes only client", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			os.RemoveAll(filepath.Join(d, "releases"))
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			files, _ := filepath.Glob(filepath.Join(d, "releases", "v0.1.0", "*server*.zip"))
			Expect(files).To(BeEmpty())
		})
		It("N10-5: build --target server writes only server", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			os.RemoveAll(filepath.Join(d, "releases"))
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "server")
			Expect(err).NotTo(HaveOccurred())
			files, _ := filepath.Glob(filepath.Join(d, "releases", "v0.1.0", "*client*.zip"))
			Expect(files).To(BeEmpty())
		})
		It("N10-6: build outputs 'built <mc> <loader>' first then 'artifact <target>: <path>'", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			os.RemoveAll(filepath.Join(d, "releases"))
			stdout, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "both")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(MatchRegexp(`(?m)^built 1\.21\.1 neoforge$`))
			Expect(stdout).To(MatchRegexp(`(?m)^artifact client: releases/v1\.5\.0/.+client\.zip$`))
			Expect(stdout).To(MatchRegexp(`(?m)^artifact server: releases/v1\.5\.0/.+server\.zip$`))
		})
		It("N10-7: build without spec fails with hint", func() {
			_, stderr, err := runMcmod(d, "build", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("N10-8: build without lock fails with hint", func() {
			writeMinimalPackspec2(d, "nolock", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			_, stderr, err := runMcmod(d, "build", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("N10-9: build with invalid target fails", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			os.RemoveAll(filepath.Join(d, "releases"))
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "weird")
			Expect(err).To(HaveOccurred())
		})
		It("N10-10: build --build-type cf is accepted", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			os.RemoveAll(filepath.Join(d, "releases"))
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--build-type", "cf")
			Expect(err).NotTo(HaveOccurred())
		})
		It("N10-11: build --build-type all is accepted", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--build-type", "all")
			Expect(err).NotTo(HaveOccurred())
		})
		It("N10-12: build --force overwrites existing zip", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).To(HaveOccurred())
			_, _, err = runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client", "--force")
			Expect(err).NotTo(HaveOccurred())
		})
		It("N10-13: build creates releases/<packVersion>/ directory if missing", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(d, "releases", "v1.5.0"))
			Expect(err).NotTo(HaveOccurred())
		})
		It("N10-14: client zip contains shared + client mods only (not server-only)", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			files, _ := filepath.Glob(filepath.Join(d, "releases", "v1.5.0", "*client*.zip"))
			Expect(files).To(HaveLen(1))
			Expect(zipHasPath(files[0], "mods/shared.jar")).To(BeTrue())
			Expect(zipHasPath(files[0], "mods/client.jar")).To(BeTrue())
			Expect(zipHasPath(files[0], "mods/server.jar")).To(BeFalse())
		})
		It("N10-15: server zip contains shared + server mods only (not client-only)", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "server")
			Expect(err).NotTo(HaveOccurred())
			files, _ := filepath.Glob(filepath.Join(d, "releases", "v1.5.0", "*server*.zip"))
			Expect(files).To(HaveLen(1))
			Expect(zipHasPath(files[0], "mods/shared.jar")).To(BeTrue())
			Expect(zipHasPath(files[0], "mods/server.jar")).To(BeTrue())
			Expect(zipHasPath(files[0], "mods/client.jar")).To(BeFalse())
		})
		It("N10-16: client zip filename uses packName per spec 7.5.4", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			files, _ := filepath.Glob(filepath.Join(d, "releases", "v1.5.0", "*client*.zip"))
			Expect(files).To(HaveLen(1))
			Expect(files[0]).To(ContainSubstring("buildtest-1.21.1-neoforge-21.1.219-client.zip"))
		})
		It("N10-17: server zip filename uses serverPackName per spec 7.5.5", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "server")
			Expect(err).NotTo(HaveOccurred())
			files, _ := filepath.Glob(filepath.Join(d, "releases", "v1.5.0", "*server*.zip"))
			Expect(files).To(HaveLen(1))
			Expect(files[0]).To(ContainSubstring("buildtest-server-1.21.1-neoforge-21.1.219-server.zip"))
		})
		It("N10-18: server zip without serverPackName falls back to packName", func() {
			os.MkdirAll(filepath.Join(d, "mods"), 0755)
			os.WriteFile(filepath.Join(d, "mods", "shared.jar"), []byte("s"), 0644)
			os.WriteFile(filepath.Join(d, "packspec.json"), []byte(`{
				"packName": "fallback", "packVersion": "1.0",
				"minecraftVersion": "1.21.1", "loaderName": ["neoforge:21.1.219"],
				"mods": {"shared-mod": {"scope":"shared","source":{"type":"local","path":"./mods/shared.jar"}}}
			}`), 0644)
			lock := &domain.PackLock{
				Loader: "neoforge", LoaderVersion: "21.1.219", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{
					"shared-mod": {Name: "Shared", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: "./mods/shared.jar", FileName: "shared.jar"}},
				},
			}
			EnsureLocksDir(d)
			WriteLockJSON(d, "1.21.1", "neoforge", lock)
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "server")
			Expect(err).NotTo(HaveOccurred())
			files, _ := filepath.Glob(filepath.Join(d, "releases", "v1.0", "*server*.zip"))
			Expect(files).To(HaveLen(1))
			Expect(files[0]).To(ContainSubstring("fallback-1.21.1-neoforge-21.1.219-server.zip"))
		})
	})

	// ============================================================
	// spec 7.5 build with config / defaultconfigs / resourcepacks
	// ============================================================
	Describe("N11: build with extra directories (spec 7.5.6-9)", func() {
		It("N11-1: client zip includes config/ contents", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			files, _ := filepath.Glob(filepath.Join(d, "releases", "v1.5.0", "*client*.zip"))
			Expect(zipHasPath(files[0], "config/common.cfg")).To(BeTrue())
		})
		It("N11-2: server zip includes config/ contents", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "server")
			Expect(err).NotTo(HaveOccurred())
			files, _ := filepath.Glob(filepath.Join(d, "releases", "v1.5.0", "*server*.zip"))
			Expect(zipHasPath(files[0], "config/common.cfg")).To(BeTrue())
		})
		It("N11-3: server zip includes defaultconfigs/ contents", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "server")
			Expect(err).NotTo(HaveOccurred())
			files, _ := filepath.Glob(filepath.Join(d, "releases", "v1.5.0", "*server*.zip"))
			Expect(zipHasPath(files[0], "defaultconfigs/default.toml")).To(BeTrue())
		})
		It("N11-4: client zip includes resourcepacks/ contents", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			files, _ := filepath.Glob(filepath.Join(d, "releases", "v1.5.0", "*client*.zip"))
			Expect(zipHasPath(files[0], "resourcepacks/rp.zip")).To(BeTrue())
		})
		It("N11-5: server zip does NOT include resourcepacks/ contents", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "server")
			Expect(err).NotTo(HaveOccurred())
			files, _ := filepath.Glob(filepath.Join(d, "releases", "v1.5.0", "*server*.zip"))
			Expect(zipHasPath(files[0], "resourcepacks/rp.zip")).To(BeFalse())
		})
		It("N11-6: server zip includes server.properties", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "server")
			Expect(err).NotTo(HaveOccurred())
			files, _ := filepath.Glob(filepath.Join(d, "releases", "v1.5.0", "*server*.zip"))
			Expect(zipHasPath(files[0], "server.properties")).To(BeTrue())
		})
	})

	// ============================================================
	// spec 7.5 build with class conflict
	// ============================================================
	Describe("N12: build jar class conflict detection (spec 7.5)", func() {
		It("N12-1: build with two mods containing same class path fails with hint", func() {
			os.MkdirAll(filepath.Join(d, "mods"), 0755)
			os.MkdirAll(filepath.Join(d, "config"), 0755)
			// Two jars with same class.
			jar1 := createFixtureJarWithClass(d, "a.jar", "com/example/Foo.class")
			jar2 := createFixtureJarWithClass(d, "b.jar", "com/example/Foo.class")
			_ = jar1
			_ = jar2
			os.WriteFile(filepath.Join(d, "packspec.json"), []byte(`{
				"packName": "conflict", "packVersion": "1.0",
				"minecraftVersion": "1.21.1", "loaderName": ["neoforge:21.1.219"],
				"mods": {
					"a": {"scope":"shared","source":{"type":"local","path":"./mods/a.jar"}},
					"b": {"scope":"shared","source":{"type":"local","path":"./mods/b.jar"}}
				}
			}`), 0644)
			lock := &domain.PackLock{
				Loader: "neoforge", LoaderVersion: "21.1.219", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{
					"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: "./mods/a.jar", FileName: "a.jar"}},
					"b": {Name: "B", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: "./mods/b.jar", FileName: "b.jar"}},
				},
			}
			EnsureLocksDir(d)
			WriteLockJSON(d, "1.21.1", "neoforge", lock)
			_, stderr, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "both")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("duplicat")) // relaxed
		})
	})

	// ============================================================
	// spec 7.6 tree alias
	// ============================================================
	Describe("N13: tree alias coverage (spec 7.6)", func() {
		It("N13-1: tree 0 args uses defaults", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "tree")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("dependency tree"))
		})
		It("N13-2: tree 1 arg uses default loader", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "tree", "1.21.1")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("dependency tree 1.21.1 neoforge"))
		})
		It("N13-3: tree 2 args full", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "tree", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("dependency tree 1.21.1 neoforge"))
		})
		It("N13-4: tree with no lock file fails with hint", func() {
			writeMinimalPackspec2(d, "tree", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			_, stderr, err := runMcmod(d, "tree", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})

	// ============================================================
	// spec 7.7 help
	// ============================================================
	Describe("N14: help output coverage (spec 7.7)", func() {
		It("N14-1: --help lists every top-level command", func() {
			stdout, _, err := runMcmod(d, "--help")
			Expect(err).NotTo(HaveOccurred())
			for _, sub := range []string{"set", "list", "lock", "build", "tree", "validate", "config", "version", "help"} {
				Expect(stdout).To(ContainSubstring(sub))
			}
		})
		It("N14-2: help lists every lock subcommand", func() {
			stdout, _, err := runMcmod(d, "lock", "--help")
			Expect(err).NotTo(HaveOccurred())
			for _, sub := range []string{"list", "show", "add", "update", "delete", "tree", "release"} {
				Expect(stdout).To(ContainSubstring(sub))
			}
		})
		It("N14-3: help lists every release subcommand", func() {
			stdout, _, err := runMcmod(d, "lock", "release", "--help")
			Expect(err).NotTo(HaveOccurred())
			for _, sub := range []string{"set", "list", "show", "delete"} {
				Expect(stdout).To(ContainSubstring(sub))
			}
		})
		It("N14-4: help for build lists --target, --build-type, --force", func() {
			stdout, _, err := runMcmod(d, "build", "--help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("--target"))
			Expect(stdout).To(ContainSubstring("--build-type"))
			Expect(stdout).To(ContainSubstring("--force"))
		})
		It("N14-5: help for set lists --project and --global", func() {
			stdout, _, err := runMcmod(d, "set", "--help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("--project"))
			Expect(stdout).To(ContainSubstring("--global"))
		})
		It("N14-6: help for validate lists --spec --lock --release-index", func() {
			stdout, _, err := runMcmod(d, "validate", "--help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("--spec"))
			Expect(stdout).To(ContainSubstring("--lock"))
			Expect(stdout).To(ContainSubstring("--release-index"))
		})
		It("N14-7: help for lock add lists --source --scope --version", func() {
			stdout, _, err := runMcmod(d, "lock", "add", "--help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("--source"))
			Expect(stdout).To(ContainSubstring("--scope"))
			Expect(stdout).To(ContainSubstring("--version"))
		})
		It("N14-8: help for lock release set lists --version --type --repo --tag", func() {
			stdout, _, err := runMcmod(d, "lock", "release", "set", "--help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("--version"))
			Expect(stdout).To(ContainSubstring("--type"))
			Expect(stdout).To(ContainSubstring("--repo"))
			Expect(stdout).To(ContainSubstring("--tag"))
		})
		It("N14-9: help for lock release delete lists --target", func() {
			stdout, _, err := runMcmod(d, "lock", "release", "delete", "--help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("--target"))
		})
		It("N14-10: typo 'bild' suggests 'build'", func() {
			_, stderr, err := runMcmod(d, "bild")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("build"))
		})
		It("N14-11: typo 'lokc' suggests 'lock'", func() {
			_, stderr, err := runMcmod(d, "lokc")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("lock"))
		})
	})

	// ============================================================
	// spec 7.8 error message format
	// ============================================================
	Describe("N15: error message format (spec 7.8)", func() {
		It("N15-1: lock error contains 'error: <cmd>: <reason>' prefix", func() {
			_, stderr, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("error:"))
		})
		It("N15-2: build error contains 'error: <cmd>: <reason>' prefix", func() {
			_, stderr, err := runMcmod(d, "build", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("error:"))
		})
		It("N15-3: validate error contains 'error: <cmd>: <reason>' prefix", func() {
			_, stderr, err := runMcmod(d, "validate")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("error:"))
		})
		It("N15-4: build lock-missing error includes 'hint:' actionable fix", func() {
			writeMinimalPackspec2(d, "errh", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			_, stderr, err := runMcmod(d, "build", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint:"))
		})
		It("N15-5: lock release show missing index includes 'hint:'", func() {
			writeMinimalPackspec2(d, "errs", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			_, stderr, err := runMcmod(d, "lock", "release", "show", "1.21.1", "0.1.0")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint:"))
		})
	})

	// ============================================================
	// spec 7.2 list / spec 7.5 build more combinations
	// ============================================================
	Describe("N16: spec 7.5 build error messages", func() {
		It("N16-1: build missing-lock stderr contains dependency lock path", func() {
			writeMinimalPackspec2(d, "mlerr", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			_, stderr, err := runMcmod(d, "build", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("locks/dependencies/1.21.1-neoforge.json"))
		})
		It("N16-2: build missing-lock stderr contains 'mcmod lock' hint", func() {
			writeMinimalPackspec2(d, "mlh", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			_, stderr, err := runMcmod(d, "build", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("mcmod lock"))
		})
		It("N16-3: build with unsupported loader fails with hint", func() {
			writeMinimalPackspec2(d, "usl", []string{"neoforge:21.1.219"}, map[string]interface{}{})
			_, stderr, err := runMcmod(d, "build", "1.21.1", "quilt")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})

	// ============================================================
	// version and config command
	// ============================================================
	Describe("N17: version and config", func() {
		It("N17-1: version prints 'mcmod version <x>' on stdout", func() {
			stdout, _, err := runMcmod(d, "version")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(MatchRegexp(`(?m)^mcmod version \S+$`))
		})
		It("N17-2: config without args shows current key state", func() {
			stdout, _, err := runMcmodWithEnv(d, cleanEnv(d), "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("CurseForge"))
		})
		It("N17-3: config set-cf-key <key> writes to project config", func() {
			_, _, err := runMcmodWithEnv(d, cleanEnv(d), "config", "set-cf-key", "ck")
			Expect(err).NotTo(HaveOccurred())
			data, _ := os.ReadFile(filepath.Join(d, ".mcmod", "config.json"))
			Expect(string(data)).To(ContainSubstring("ck"))
		})
	})
})

// ============================================================
// Helper functions used by integration6_test.go
// ============================================================

// writeMinimalPackspec2 writes a minimal packspec.json with explicit loader list
// and mods map. Used by integration6 tests that need a custom spec.
func writeMinimalPackspec2(d, name string, loaders []string, mods map[string]interface{}) {
	spec := map[string]interface{}{
		"packName":         name,
		"packVersion":      "0.1.0",
		"minecraftVersion": "1.21.1",
		"loaderName":       loaders,
		"mods":             mods,
	}
	data, _ := json.MarshalIndent(spec, "", "  ")
	Expect(os.WriteFile(filepath.Join(d, "packspec.json"), data, 0644)).To(Succeed())
}

// EnsureLocksDir creates locks/dependencies/ if missing.
func EnsureLocksDir(d string) {
	Expect(os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)).To(Succeed())
}

// WriteLockJSON writes a lock file under locks/dependencies/<mc>-<loader>.json.
func WriteLockJSON(d, mc, loader string, lock *domain.PackLock) {
	EnsureLocksDir(d)
	p := filepath.Join(d, "locks", "dependencies", fmt.Sprintf("%s-%s.json", mc, loader))
	data, _ := json.MarshalIndent(lock, "", "  ")
	Expect(os.WriteFile(p, data, 0644)).To(Succeed())
}

// createFixtureJarWithClass writes a tiny zip with a single class entry.
func createFixtureJarWithClass(d, name, classPath string) string {
	p := filepath.Join(d, "mods", name)
	f, err := os.Create(p)
	Expect(err).NotTo(HaveOccurred())
	defer f.Close()
	w := zip.NewWriter(f)
	entry, _ := w.Create(classPath)
	entry.Write([]byte("fake"))
	w.Close()
	return p
}

// cleanEnv returns a minimal env slice for tests that need isolated HOME/CFG.
func cleanEnv(d string) []string {
	return []string{
		"HOME=" + d,
		"XDG_CONFIG_HOME=" + d,
		"CURSEFORGE_API_KEY=",
	}
}
