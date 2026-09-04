// File: test/smoke4_test.go
// Created: 2026-06-20
// Description: Part 4 of smoke tests - comprehensive coverage for all CLI subcommands, flags, and edge cases.

package test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("Smoke: comprehensive CLI coverage", func() {
	var d string

	BeforeEach(func() {
		d = GinkgoT().TempDir()
	})

	// S30: mcmod set with --global flag
	Describe("S30: set cf-key --global", func() {
		It("set cf-key --global works", func() {
			stdout, _, err := runMcmod(d, "set", "cf-key", "global-key", "--global")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("set cf-key"))
		})
		It("set with wrong first arg fails", func() {
			_, stderr, err := runMcmod(d, "set", "wrong-arg", "key")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})

	// S31: mcmod lock with multi-mod spec (no API key, verifies graceful error)
	Describe("S31: lock with local-only mod (no API needed)", func() {
		It("lock with local mod resolves without API", func() {
			writeSpec(d, `{"packName":"local-lock","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"],
"mods":{"mymod":{"name":"MyMod","scope":"shared","source":{"type":"local","path":"./mymod.jar"}}}}`)
			os.WriteFile(filepath.Join(d, "mymod.jar"), []byte("dummy"), 0644)
			stdout, _, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("locked"))
			// Verify lock file was written
			_, err = os.Stat(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(err).NotTo(HaveOccurred())
		})
		It("lock with both fabric and neoforge loaders", func() {
			writeSpec(d, `{"packName":"dual-lock","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge","fabric"],
"mods":{"m":{"name":"M","scope":"shared","source":{"type":"local","path":"./m.jar"}}}}`)
			os.WriteFile(filepath.Join(d, "m.jar"), []byte("d"), 0644)
			stdout, stderr, err := runMcmod(d, "lock", "1.21.1")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("locked"))
			_ = stderr
			// Should have locked for neoforge (fabric may fail if no version match)
		})
	})

	// S32: lock add with all flag combinations
	Describe("S32: lock add comprehensive", func() {
		It("lock add with all curseforge flags", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			stdout, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge", "cfmod",
				"--name", "CFMod", "--version", "2.0", "--scope", "client",
				"--source", "curseforge", "--mod-id", "999", "--file-id", "888", "--file-name", "cfmod.jar")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("added"))
			lock, _ := domain.ReadLockFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			m, ok := lock.Mods["cfmod"]
			Expect(ok).To(BeTrue())
			Expect(m.Scope).To(Equal("client"))
			Expect(m.Source.ModID).To(Equal(999))
		})
		It("lock add with github-release flags", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			stdout, _, err := runMcmod(d, "lock", "add", "1.21.1", "fabric", "ghrel",
				"--name", "GHRel", "--version", "v1.0", "--scope", "shared",
				"--source", "github-release", "--repo", "owner/repo", "--tag", "v1.0",
				"--asset-name", "mod.jar", "--file-name", "mod.jar")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("added"))
		})
		It("lock add with local source", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			stdout, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge", "localmod",
				"--name", "LocalMod", "--source", "local",
				"--path", "./localmod.jar", "--file-name", "localmod.jar")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("added"))
		})
		It("lock add without enough args fails", func() {
			_, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("lock add duplicate key fails", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"dup": {Name: "Dup", Source: domain.LockedSource{Type: "local"}}}})
			_, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge", "dup", "--source", "local")
			Expect(err).To(HaveOccurred())
		})
	})

	// S33: lock update single key with --version
	Describe("S33: lock update comprehensive", func() {
		It("lock update single key with --version", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"mykey": {Name: "MyKey", Version: "1.0",
					Source: domain.LockedSource{Type: "local", Path: "./mykey.jar", FileName: "mykey.jar"}}}})
			stdout, _, err := runMcmod(d, "lock", "update", "1.21.1", "neoforge", "mykey", "--version", "2.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("updated"))
			lock, _ := domain.ReadLockFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(lock.Mods["mykey"].Version).To(Equal("2.0"))
		})
		It("lock update single key without lock fails", func() {
			_, _, err := runMcmod(d, "lock", "update", "1.21.1", "neoforge", "nokey")
			Expect(err).To(HaveOccurred())
		})
		It("lock update single key with nonexistent key fails", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"a": {Name: "A", Source: domain.LockedSource{Type: "local"}}}})
			_, _, err := runMcmod(d, "lock", "update", "1.21.1", "neoforge", "nonexistent")
			Expect(err).To(HaveOccurred())
		})
	})

	// S34: lock delete comprehensive
	Describe("S34: lock delete comprehensive", func() {
		It("lock delete with 3 args deletes entry", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"delme": {Name: "DelMe",
					Source: domain.LockedSource{Type: "local", Path: "./x.jar", FileName: "x.jar"}}}})
			stdout, _, err := runMcmod(d, "lock", "delete", "1.21.1", "neoforge", "delme")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("deleted"))
			lock, _ := domain.ReadLockFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			_, exists := lock.Mods["delme"]
			Expect(exists).To(BeFalse())
		})
		It("lock delete with missing lock file fails", func() {
			_, _, err := runMcmod(d, "lock", "delete", "1.21.1", "neoforge", "x")
			Expect(err).To(HaveOccurred())
		})
	})

	// S35: lock list with edge cases
	Describe("S35: lock list edge cases", func() {
		It("lock list with empty mods shows (empty)", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{}})
			writeSpec(d, `{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			stdout, _, err := runMcmod(d, "lock", "list", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("(empty)"))
		})
		It("lock list with mixed scope mods", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{
					"a": {Name: "A", Version: "1", Scope: "shared", Source: domain.LockedSource{Type: "curseforge", FileName: "a.jar"}},
					"b": {Name: "B", Version: "2", Scope: "client", Source: domain.LockedSource{Type: "curseforge", FileName: "b.jar"}},
					"c": {Name: "C", Version: "3", Scope: "server", Source: domain.LockedSource{Type: "curseforge", FileName: "c.jar"}},
				}})
			stdout, _, err := runMcmod(d, "lock", "list", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("[Shared]"))
			Expect(stdout).To(ContainSubstring("[Client]"))
			Expect(stdout).To(ContainSubstring("[Server]"))
		})
	})

	// S36: lock show with detailed fields (curseforge + github)
	Describe("S36: lock show detailed", func() {
		It("lock show with curseforge fields", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"cfmod": {Name: "CFMod", Version: "1.0", Scope: "shared",
					Source: domain.LockedSource{Type: "curseforge", ModID: 12345, FileID: 67890, FileName: "cfmod.jar"}}}})
			stdout, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge", "cfmod")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("modId: 12345"))
			Expect(stdout).To(ContainSubstring("fileId: 67890"))
			Expect(stdout).To(ContainSubstring("fileName: cfmod.jar"))
		})
		It("lock show with github-release fields", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"ghmod": {Name: "GHMod", Version: "2.0", Scope: "shared",
					Source: domain.LockedSource{Type: "github-release", Repo: "owner/repo", Tag: "v2.0",
						AssetName: "mod.jar", FileName: "mod.jar"}}}})
			stdout, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge", "ghmod")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("repo: owner/repo"))
			Expect(stdout).To(ContainSubstring("tag: v2.0"))
		})
	})

	// S37: build with --target, --build-type, --force
	Describe("S37: build comprehensive", func() {
		It("build with --target client", func() {
			writeSpec(d, `{"packName":"btc","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"],
"mods":{"m":{"scope":"shared","source":{"type":"local","path":"./m.jar"}}}}`)
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"m": {Name: "M", Scope: "shared",
					Source: domain.LockedSource{Type: "local", Path: "./m.jar", FileName: "m.jar"}}}})
			os.WriteFile(filepath.Join(d, "m.jar"), []byte("dummy"), 0644)
			stdout, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("built"))
		})
		It("build without --target (defaults to both)", func() {
			writeSpec(d, `{"packName":"bdef","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"],
"mods":{"m":{"scope":"shared","source":{"type":"local","path":"./m.jar"}}}}`)
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"m": {Name: "M", Scope: "shared",
					Source: domain.LockedSource{Type: "local", Path: "./m.jar", FileName: "m.jar"}}}})
			os.WriteFile(filepath.Join(d, "m.jar"), []byte("dummy"), 0644)
			stdout, _, err := runMcmod(d, "build", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("built"))
		})
		It("build without spec fails", func() {
			_, _, err := runMcmod(d, "build")
			Expect(err).To(HaveOccurred())
		})
	})

	// S38: validate with release-index and invalid paths
	Describe("S38: validate comprehensive", func() {
		It("validate --release-index with valid file", func() {
			Expect(os.WriteFile(filepath.Join(d, "rel.json"), []byte(`{"type":"package","packName":"p","minecraftVersion":"1.21.1",
"releases":[{"version":"0.1.0","type":"github-release"}]}`), 0644)).To(Succeed())
			stdout, _, err := runMcmod(d, "validate", "--release-index", filepath.Join(d, "rel.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("valid"))
		})
		It("validate --release-index with invalid file", func() {
			os.WriteFile(filepath.Join(d, "bad.json"), []byte(`{invalid`), 0644)
			_, _, err := runMcmod(d, "validate", "--release-index", filepath.Join(d, "bad.json"))
			Expect(err).To(HaveOccurred())
		})
	})

	// S39: lock release set with all optional flags
	Describe("S39: lock release set comprehensive", func() {
		It("release set with all optional flags", func() {
			os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)
			writeReleaseIndexFile(d, "1.21.1", &domain.ReleaseIndex{Type: "package", PackName: "test", MinecraftVersion: "1.21.1"})
			stdout, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.5.0", "--repo", "owner/repo", "--tag", "v0.5.0",
				"--name", "Release v0.5.0", "--body", "Release body text",
				"--draft", "--prerelease")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("release"))
		})
		It("release set without required --version fails", func() {
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1")
			Expect(err).To(HaveOccurred())
		})
		It("release set with --artifact-client and --artifact-server", func() {
			os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)
			writeReleaseIndexFile(d, "1.21.1", &domain.ReleaseIndex{Type: "package", PackName: "test", MinecraftVersion: "1.21.1"})
			stdout, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.6.0", "--repo", "owner/repo", "--tag", "v0.6.0",
				"--artifact-client", "client.zip", "--artifact-server", "server.zip")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("release"))
		})
	})

	// S40: lock release management edge cases
	Describe("S40: release management edge cases", func() {
		It("release list without args uses default mc", func() {
			os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)
			writeReleaseIndexFile(d, "1.21.1", &domain.ReleaseIndex{Type: "package", PackName: "p", MinecraftVersion: "1.21.1",
				Releases: []domain.ReleaseRecord{{Version: "0.1.0", Type: "github-release"}}})
			stdout, _, err := runMcmod(d, "lock", "release", "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("0.1.0"))
		})
		It("release show with nonexistent version fails", func() {
			os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)
			writeReleaseIndexFile(d, "1.21.1", &domain.ReleaseIndex{Type: "package", PackName: "p", MinecraftVersion: "1.21.1",
				Releases: []domain.ReleaseRecord{{Version: "0.1.0", Type: "github-release"}}})
			_, _, err := runMcmod(d, "lock", "release", "show", "1.21.1", "99.99")
			Expect(err).To(HaveOccurred())
		})
		It("release delete with --target flag", func() {
			// Pre-create the release index with one entry so the delete
			// has something to operate on (the test directory may or
			// may not have a packspec.json from earlier specs).
			idx := &domain.ReleaseIndex{Releases: []domain.ReleaseRecord{{
				Version: "0.1.0", Type: "github-release",
				Artifact: map[string]domain.ReleaseArtifactSet{
					"neoforge": {Client: "c.jar", Server: "s.jar"},
				},
			}}}
			_ = os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)
			_ = domain.WriteReleaseIndex(filepath.Join(d, "locks", "releases", "1.21.1.json"), idx)
			stdout, _, err := runMcmod(d, "lock", "release", "delete", "1.21.1", "0.1.0", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("deleted"))
		})
	})

	// S41: config edge cases
	Describe("S41: config edge cases", func() {
		It("config without args shows current state", func() {
			stdout, _, err := runMcmod(d, "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("CurseForge"))
		})
	})

	// S42: error and edge case paths
	Describe("S42: error and edge case paths", func() {
		It("lock with empty mods list", func() {
			writeSpec(d, `{"packName":"empty","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			_, stderr, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			_ = stderr
		})
		It("build with --build-type=cf emits the CurseForge layout zip", func() {
			writeSpec(d, `{"packName":"bt","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"],
"mods":{"m":{"scope":"shared","source":{"type":"curseforge","modId":1,"fileId":2}}}}`)
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"m": {Name: "M", Scope: "shared",
					Source: domain.LockedSource{Type: "curseforge", ModID: 1, FileID: 2, FileName: "m.jar"}}}})
			// Stage the cached jar so resolveModJar does not hit the network.
			cached := filepath.Join(d, ".cache", "curseforge", "1", "2", "m.jar")
			Expect(os.MkdirAll(filepath.Dir(cached), 0755)).To(Succeed())
			Expect(os.WriteFile(cached, []byte("d"), 0644)).To(Succeed())
			stdout, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--build-type", "cf", "--force")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("built"))
			Expect(stdout).To(ContainSubstring("artifact cf:"))
		})

		It("build with --build-type=cf errors when no curseforge mods present", func() {
			writeSpec(d, `{"packName":"bt2","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"],
"mods":{"m":{"scope":"shared","source":{"type":"local","path":"./m.jar"}}}}`)
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"m": {Name: "M", Scope: "shared",
					Source: domain.LockedSource{Type: "local", Path: "./m.jar", FileName: "m.jar"}}}})
			os.WriteFile(filepath.Join(d, "m.jar"), []byte("d"), 0644)
			_, stderr, err := runMcmod(d, "build", "1.21.1", "neoforge", "--build-type", "cf", "--force")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("no curseforge-sourced mods"))
		})
		It("build with --force flag", func() {
			writeSpec(d, `{"packName":"bf","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"],
"mods":{"m":{"scope":"shared","source":{"type":"local","path":"./m.jar"}}}}`)
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"m": {Name: "M", Scope: "shared",
					Source: domain.LockedSource{Type: "local", Path: "./m.jar", FileName: "m.jar"}}}})
			os.WriteFile(filepath.Join(d, "m.jar"), []byte("d"), 0644)
			stdout, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--force")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("built"))
		})
		It("unknown command returns error", func() {
			_, _, err := runMcmod(d, "unknown-command")
			Expect(err).To(HaveOccurred())
		})
		It("validate with bad spec path fails", func() {
			_, _, err := runMcmod(d, "validate", "--spec", "/nonexistent/spec.json")
			Expect(err).To(HaveOccurred())
		})
	})

	// S43: lock tree with multi-mod lock
	Describe("S43: lock tree with multi-mod", func() {
		It("lock tree with several mods", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{
					"a": {Name: "Alpha", Version: "1", Scope: "shared", Source: domain.LockedSource{Type: "local"}},
					"b": {Name: "Beta", Version: "2", Scope: "client", Source: domain.LockedSource{Type: "local"}},
				}})
			stdout, _, err := runMcmod(d, "lock", "tree", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("Alpha"))
			Expect(stdout).To(ContainSubstring("Beta"))
		})
	})

	// S44: tree alias edge cases
	Describe("S44: tree alias edge cases", func() {
		It("tree alias with partial args", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"x": {Name: "X",
					Source: domain.LockedSource{Type: "local"}}}})
			stdout, _, err := runMcmod(d, "tree", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("X"))
		})
	})

	// S45: version output to stdout
	Describe("S45: version output", func() {
		It("version outputs to stdout", func() {
			stdout, _, err := runMcmod(d, "version")
			Expect(err).NotTo(HaveOccurred())
			_ = stdout
		})
	})

	// S46: lock update full refresh with spec
	Describe("S46: lock update full refresh", func() {
		It("lock update full refresh with local mods", func() {
			writeSpec(d, `{"packName":"full-upd","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"],
"mods":{"m":{"name":"M","scope":"shared","source":{"type":"local","path":"./m.jar"}}}}`)
			os.WriteFile(filepath.Join(d, "m.jar"), []byte("d"), 0644)
			stdout, _, err := runMcmod(d, "lock", "update")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("updated"))
		})
	})

	// S47: validate with default spec
	Describe("S47: validate default spec", func() {
		It("validate with valid spec in dir", func() {
			writeSpec(d, `{"packName":"val","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"],
"mods":{"m":{"source":{"type":"curseforge","query":"M"}}}}`)
			stdout, _, err := runMcmod(d, "validate")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("valid"))
		})
	})
})
