// File: test/smoke6_test.go
// Created: 2026-06-20
// Description: End-to-end integration tests for mcmod CLI. Each spec builds a
// realistic packspec.json, runs the real `mcmod` binary, and asserts against
// the exact stdout / stderr / exit code described in specification.md.

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

// writeSpecNewSchema writes a packspec.json that conforms to specification.md
// section 3 (unified `mods` dict, scope fields, serverPackName, mod-level loader).
func writeSpecNewSchema(dir string, spec string) {
	p := filepath.Join(dir, "packspec.json")
	Expect(os.WriteFile(p, []byte(spec), 0644)).To(Succeed())
}

var _ = Describe("Smoke6: end-to-end integration against real binary", func() {
	var d string

	BeforeEach(func() {
		d = GinkgoT().TempDir()
	})

	// ================== Spec 7.1: set ==================
	Describe("Spec 7.1: set cf-key", func() {
		It("set cf-key without flag writes user config and prints 'set cf-key'", func() {
			stdout, _, err := runMcmod(d, "set", "cf-key", "userkey")
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.TrimSpace(stdout)).To(Equal("set cf-key"))
		})
		It("set cf-key --project writes .mcmod/config.json and prints 'set cf-key'", func() {
			stdout, _, err := runMcmod(d, "set", "cf-key", "projkey", "--project")
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.TrimSpace(stdout)).To(Equal("set cf-key"))
			data, err := os.ReadFile(filepath.Join(d, ".mcmod", "config.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(ContainSubstring("projkey"))
		})
		It("set cf-key --global alias to user config", func() {
			stdout, _, err := runMcmod(d, "set", "cf-key", "globalkey", "--global")
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.TrimSpace(stdout)).To(Equal("set cf-key"))
		})
		It("set with insufficient args fails", func() {
			_, _, err := runMcmod(d, "set", "cf-key")
			Expect(err).To(HaveOccurred())
		})
	})

	// ================== Spec 7.2: list ==================
	Describe("Spec 7.2: list", func() {
		It("list groups by [Server], [Client], [Shared] with pack/loader headers", func() {
			writeSpecNewSchema(d, `{
				"packName": "demo", "packVersion": "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName": ["neoforge:21.1.219", "fabric:1.21.123"],
				"mods": {
					"shared-mod": {"name": "Shared", "scope": "shared", "source": {"type": "local", "path": "./s.jar"}},
					"client-mod": {"name": "Client", "scope": "client", "source": {"type": "local", "path": "./c.jar"}},
					"server-mod": {"name": "Server", "scope": "server", "source": {"type": "local", "path": "./v.jar"}}
				}
			}`)
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("pack demo (0.1.0)"))
			Expect(stdout).To(ContainSubstring("loader:"))
			Expect(stdout).To(ContainSubstring("  - neoforge:21.1.219"))
			Expect(stdout).To(ContainSubstring("  - fabric:1.21.123"))
			Expect(stdout).To(ContainSubstring("[Server]"))
			Expect(stdout).To(ContainSubstring("[Client]"))
			Expect(stdout).To(ContainSubstring("[Shared]"))
			Expect(stdout).To(ContainSubstring("Shared [local]"))
			Expect(stdout).To(ContainSubstring("Client [local]"))
			Expect(stdout).To(ContainSubstring("Server [local]"))
		})
		It("list shows (empty) for empty sections", func() {
			writeSpecNewSchema(d, `{
				"packName": "e", "packVersion": "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName": ["neoforge:21.1.219"],
				"mods": {}
			}`)
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("(empty)"))
		})
		It("list without packspec fails (spec 7.2 implicit)", func() {
			_, _, err := runMcmod(d, "list")
			Expect(err).To(HaveOccurred())
		})
	})

	// ================== Spec 7.3: lock ==================
	Describe("Spec 7.3: lock (all subcommands)", func() {
		setupLocalMods := func() {
			// Create local jar files
			os.MkdirAll(filepath.Join(d, "mods"), 0755)
			os.WriteFile(filepath.Join(d, "mods", "shared.jar"), []byte("a"), 0644)
			os.WriteFile(filepath.Join(d, "mods", "client.jar"), []byte("b"), 0644)
			os.WriteFile(filepath.Join(d, "mods", "server.jar"), []byte("c"), 0644)
			writeSpecNewSchema(d, `{
				"packName": "locktest", "packVersion": "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName": ["neoforge:21.1.219"],
				"mods": {
					"shared-mod": {"name": "Shared", "scope": "shared", "source": {"type": "local", "path": "./mods/shared.jar"}},
					"client-mod": {"name": "Client", "scope": "client", "source": {"type": "local", "path": "./mods/client.jar"}},
					"server-mod": {"name": "Server", "scope": "server", "source": {"type": "local", "path": "./mods/server.jar"}}
				}
			}`)
		}

		It("lock writes spec-format stdout for each (mc, loader) pair", func() {
			setupLocalMods()
			stdout, _, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("locked 1.21.1 neoforge -> locks/dependencies/1.21.1-neoforge.json"))
		})
		It("lock writes lock file with loaderVersion", func() {
			setupLocalMods()
			_, _, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			lock, err := domain.ReadLockFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(lock.LoaderVersion).To(Equal("21.1.219"))
		})
		It("lock with unsupported loader fails and exits non-zero", func() {
			writeSpecNewSchema(d, `{
				"packName": "p", "packVersion": "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName": ["neoforge:21.1.219"]
			}`)
			_, _, err := runMcmod(d, "lock", "1.21.1", "fabric")
			Expect(err).To(HaveOccurred())
		})
		It("lock list groups by [Server]/[Client]/[Shared]", func() {
			setupLocalMods()
			_, _, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			stdout, _, err := runMcmod(d, "lock", "list", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("lock 1.21.1 neoforge"))
			Expect(stdout).To(ContainSubstring("[Shared]"))
			Expect(stdout).To(ContainSubstring("[Client]"))
			Expect(stdout).To(ContainSubstring("[Server]"))
		})
		It("lock list on missing file fails with hint and non-zero exit", func() {
			_, stderr, err := runMcmod(d, "lock", "list", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("lock show <mc> <loader> without key dumps full lock as JSON", func() {
			setupLocalMods()
			_, _, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			stdout, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			var l domain.PackLock
			Expect(json.Unmarshal([]byte(stdout), &l)).To(Succeed())
			Expect(l.MinecraftVersion).To(Equal("1.21.1"))
		})
		It("lock show <mc> <loader> <key> prints key/name/version/scope/source", func() {
			setupLocalMods()
			_, _, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			stdout, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge", "shared-mod")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("key: shared-mod"))
			Expect(stdout).To(ContainSubstring("name: Shared"))
			Expect(stdout).To(ContainSubstring("scope: shared"))
			Expect(stdout).To(ContainSubstring("type: local"))
		})
		It("lock show <mc> <loader> <missing> fails", func() {
			setupLocalMods()
			_, _, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = runMcmod(d, "lock", "show", "1.21.1", "neoforge", "missing-mod")
			Expect(err).To(HaveOccurred())
		})
		It("lock show with 0-1 args fails", func() {
			_, _, err := runMcmod(d, "lock", "show")
			Expect(err).To(HaveOccurred())
		})
		It("lock add creates a local entry", func() {
			_, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge", "added-key",
				"--name", "Added", "--scope", "shared",
				"--source", "local", "--path", "./x.jar", "--file-name", "x.jar")
			Expect(err).NotTo(HaveOccurred())
			lock, err := domain.ReadLockFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(lock.Mods).To(HaveKey("added-key"))
		})
		It("lock add with duplicate key fails", func() {
			_, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge", "dupe",
				"--source", "local", "--path", "./x.jar", "--file-name", "x.jar")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = runMcmod(d, "lock", "add", "1.21.1", "neoforge", "dupe",
				"--source", "local", "--path", "./x.jar", "--file-name", "x.jar")
			Expect(err).To(HaveOccurred())
		})
		It("lock update single key with --version updates entry", func() {
			setupLocalMods()
			_, _, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = runMcmod(d, "lock", "update", "1.21.1", "neoforge", "shared-mod", "--version", "9.9.9")
			Expect(err).NotTo(HaveOccurred())
			lock, _ := domain.ReadLockFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(lock.Mods["shared-mod"].Version).To(Equal("9.9.9"))
		})
		It("lock update on missing lock fails", func() {
			_, _, err := runMcmod(d, "lock", "update", "1.21.1", "neoforge", "any")
			Expect(err).To(HaveOccurred())
		})
		It("lock delete with key removes entry from file", func() {
			setupLocalMods()
			_, _, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = runMcmod(d, "lock", "delete", "1.21.1", "neoforge", "shared-mod")
			Expect(err).NotTo(HaveOccurred())
			lock, _ := domain.ReadLockFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(lock.Mods).NotTo(HaveKey("shared-mod"))
		})
		It("lock delete without key removes the lock file (spec 7.3.8)", func() {
			setupLocalMods()
			_, _, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json")).To(BeAnExistingFile())
			_, _, err = runMcmod(d, "lock", "delete", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json")).NotTo(BeAnExistingFile())
		})
		It("lock tree prints dependency tree header and entry list", func() {
			setupLocalMods()
			_, _, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			stdout, _, err := runMcmod(d, "lock", "tree", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("dependency tree 1.21.1 neoforge"))
			Expect(stdout).To(ContainSubstring("Shared"))
			Expect(stdout).To(ContainSubstring("Client"))
		})
		It("tree alias works the same as lock tree", func() {
			setupLocalMods()
			_, _, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			stdout, _, err := runMcmod(d, "tree", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("dependency tree"))
		})
		It("tree on missing lock fails with hint and non-zero exit", func() {
			_, stderr, err := runMcmod(d, "tree", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})

	// ================== Spec 7.4: lock release ==================
	Describe("Spec 7.4: lock release", func() {
		setupPack := func() {
			writeSpecNewSchema(d, `{
				"packName": "releasetest", "serverPackName": "releasetest-server",
				"packVersion": "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName": ["neoforge:21.1.219"]
			}`)
		}
		It("release set writes spec-format stdout and backfills packName from spec", func() {
			setupPack()
			stdout, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("locked release 1.21.1 0.1.0 github-release -> locks/releases/1.21.1.json"))
			index, err := domain.ReadReleaseIndex(filepath.Join(d, "locks", "releases", "1.21.1.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(index.PackName).To(Equal("releasetest"))
			Expect(index.Releases).To(HaveLen(1))
			Expect(index.Releases[0].GitHub.Repo).To(Equal("o/r"))
		})
		It("release set without --version fails", func() {
			setupPack()
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--repo", "o/r", "--tag", "v0.1.0")
			Expect(err).To(HaveOccurred())
		})
		It("release set without --repo fails", func() {
			setupPack()
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.1.0", "--tag", "v0.1.0")
			Expect(err).To(HaveOccurred())
		})
		It("release set without --tag fails", func() {
			setupPack()
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.1.0", "--repo", "o/r")
			Expect(err).To(HaveOccurred())
		})
		It("release set with --name --body --draft --prerelease writes all fields", func() {
			setupPack()
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0",
				"--name", "Release Name", "--body", "Body", "--draft", "--prerelease")
			Expect(err).NotTo(HaveOccurred())
			index, _ := domain.ReadReleaseIndex(filepath.Join(d, "locks", "releases", "1.21.1.json"))
			Expect(index.Releases[0].GitHub.Name).To(Equal("Release Name"))
			Expect(index.Releases[0].GitHub.Body).To(Equal("Body"))
			Expect(index.Releases[0].GitHub.Draft).To(BeTrue())
			Expect(index.Releases[0].GitHub.Pre).To(BeTrue())
		})
		It("release list shows records with tag", func() {
			setupPack()
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0")
			Expect(err).NotTo(HaveOccurred())
			stdout, _, err := runMcmod(d, "lock", "release", "list", "1.21.1")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("releases 1.21.1"))
			Expect(stdout).To(ContainSubstring("0.1.0 [github-release] tag=v0.1.0"))
		})
		It("release list with no index fails", func() {
			setupPack()
			_, _, err := runMcmod(d, "lock", "release", "list", "1.21.1")
			Expect(err).To(HaveOccurred())
		})
		It("release show with valid version prints release", func() {
			setupPack()
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0")
			Expect(err).NotTo(HaveOccurred())
			stdout, _, err := runMcmod(d, "lock", "release", "show", "1.21.1", "0.1.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("o/r"))
		})
		It("release show with 0-1 args fails", func() {
			setupPack()
			_, _, err := runMcmod(d, "lock", "release", "show")
			Expect(err).To(HaveOccurred())
		})
		It("release delete removes the entry from the index", func() {
			setupPack()
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = runMcmod(d, "lock", "release", "delete", "1.21.1", "0.1.0")
			Expect(err).NotTo(HaveOccurred())
			// Per spec 7.4.8 the index file is removed when its last
			// release is deleted, so the file no longer exists.
			releasePath := filepath.Join(d, "locks", "releases", "1.21.1.json")
			_, statErr := os.Stat(releasePath)
			Expect(os.IsNotExist(statErr)).To(BeTrue())
		})
		It("release delete single client artifact keeps server", func() {
			setupPack()
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0",
				"--artifact-client", "client.zip", "--artifact-server", "server.zip")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = runMcmod(d, "lock", "release", "delete", "1.21.1", "0.1.0", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			index, _ := domain.ReadReleaseIndex(filepath.Join(d, "locks", "releases", "1.21.1.json"))
			Expect(index.Releases[0].Artifact["neoforge"].Client).To(BeEmpty())
			Expect(index.Releases[0].Artifact["neoforge"].Server).To(Equal("server.zip"))
		})
		It("release delete with 0-1 args fails", func() {
			setupPack()
			_, _, err := runMcmod(d, "lock", "release", "delete")
			Expect(err).To(HaveOccurred())
		})
	})

	// ================== Spec 7.5: build ==================
	Describe("Spec 7.5: build (real zip output)", func() {
		setupBuildable := func() {
			os.MkdirAll(filepath.Join(d, "mods"), 0755)
			os.MkdirAll(filepath.Join(d, "config"), 0755)
			os.MkdirAll(filepath.Join(d, "defaultconfigs"), 0755)
			os.MkdirAll(filepath.Join(d, "resourcepacks"), 0755)
			os.WriteFile(filepath.Join(d, "mods", "shared.jar"), []byte("shared-content"), 0644)
			os.WriteFile(filepath.Join(d, "mods", "client.jar"), []byte("client-content"), 0644)
			os.WriteFile(filepath.Join(d, "mods", "server.jar"), []byte("server-content"), 0644)
			os.WriteFile(filepath.Join(d, "config", "common.cfg"), []byte("cfg"), 0644)
			os.WriteFile(filepath.Join(d, "defaultconfigs", "default.toml"), []byte("d"), 0644)
			os.WriteFile(filepath.Join(d, "resourcepacks", "rp.zip"), []byte("rp"), 0644)
			os.WriteFile(filepath.Join(d, "server.properties"), []byte("server-port=25565"), 0644)
			writeSpecNewSchema(d, `{
				"packName": "buildtest", "serverPackName": "buildtest-server",
				"packVersion": "1.5.0",
				"minecraftVersion": "1.21.1",
				"loaderName": ["neoforge:21.1.219"],
				"mods": {
					"shared-mod": {"scope": "shared", "source": {"type": "local", "path": "./mods/shared.jar"}},
					"client-mod": {"scope": "client", "source": {"type": "local", "path": "./mods/client.jar"}},
					"server-mod": {"scope": "server", "source": {"type": "local", "path": "./mods/server.jar"}}
				}
			}`)
			// Pre-create the lock file with all three mods.
			lock := &domain.PackLock{
				Loader: "neoforge", LoaderVersion: "21.1.219",
				MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{
					"shared-mod": {Name: "Shared", Scope: "shared",
						Source: domain.LockedSource{Type: "local", Path: "./mods/shared.jar", FileName: "shared.jar"}},
					"client-mod": {Name: "Client", Scope: "client",
						Source: domain.LockedSource{Type: "local", Path: "./mods/client.jar", FileName: "client.jar"}},
					"server-mod": {Name: "Server", Scope: "server",
						Source: domain.LockedSource{Type: "local", Path: "./mods/server.jar", FileName: "server.jar"}},
				},
			}
			writeLockFile(d, "1.21.1", "neoforge", lock)
		}

		It("build --target both writes client and server zips with spec path format", func() {
			setupBuildable()
			stdout, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "both")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("built 1.21.1 neoforge"))
			Expect(filepath.Join(d, "releases", "v1.5.0", "buildtest-1.21.1-neoforge-21.1.219-client.zip")).To(BeAnExistingFile())
			Expect(filepath.Join(d, "releases", "v1.5.0", "buildtest-server-1.21.1-neoforge-21.1.219-server.zip")).To(BeAnExistingFile())
		})
		It("build --target client only writes client zip with shared+client mods", func() {
			setupBuildable()
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(d, "releases", "v1.5.0", "buildtest-1.21.1-neoforge-21.1.219-client.zip")).To(BeAnExistingFile())
			Expect(filepath.Join(d, "releases", "v1.5.0", "buildtest-server-1.21.1-neoforge-21.1.219-server.zip")).NotTo(BeAnExistingFile())
		})
		It("build --target server only writes server zip with shared+server mods", func() {
			setupBuildable()
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "server")
			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(d, "releases", "v1.5.0", "buildtest-server-1.21.1-neoforge-21.1.219-server.zip")).To(BeAnExistingFile())
		})
		It("build without --target defaults to both", func() {
			setupBuildable()
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(d, "releases", "v1.5.0", "buildtest-1.21.1-neoforge-21.1.219-client.zip")).To(BeAnExistingFile())
			Expect(filepath.Join(d, "releases", "v1.5.0", "buildtest-server-1.21.1-neoforge-21.1.219-server.zip")).To(BeAnExistingFile())
		})
		It("build on missing lock fails with hint and non-zero exit", func() {
			writeSpecNewSchema(d, `{
				"packName": "p", "packVersion": "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName": ["neoforge:21.1.219"]
			}`)
			_, stderr, err := runMcmod(d, "build", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("build without spec fails with hint and non-zero exit", func() {
			_, stderr, err := runMcmod(d, "build", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("build --force allows overwriting existing zip", func() {
			setupBuildable()
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).To(HaveOccurred())
			_, _, err = runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client", "--force")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	// ================== Spec 7 / 8 / general ==================
	Describe("Spec 7/8 general CLI", func() {
		It("no args prints help with all commands", func() {
			stdout, _, err := runMcmod(d)
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("Usage"))
			Expect(stdout).To(ContainSubstring("lock"))
			Expect(stdout).To(ContainSubstring("build"))
			Expect(stdout).To(ContainSubstring("list"))
			Expect(stdout).To(ContainSubstring("validate"))
			Expect(stdout).To(ContainSubstring("set"))
			Expect(stdout).To(ContainSubstring("tree"))
			Expect(stdout).To(ContainSubstring("config"))
			Expect(stdout).To(ContainSubstring("version"))
		})
		It("version prints to stdout", func() {
			stdout, _, err := runMcmod(d, "version")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("mcmod version"))
		})
		It("unknown command fails", func() {
			_, _, err := runMcmod(d, "totally-unknown-xyz")
			Expect(err).To(HaveOccurred())
		})
	})

	// ================== Spec 7.13: validate ==================
	Describe("Spec 7.13: validate", func() {
		It("validate valid spec", func() {
			writeSpecNewSchema(d, `{
				"packName": "v", "packVersion": "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName": ["neoforge:21.1.219"]
			}`)
			stdout, _, err := runMcmod(d, "validate")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("valid"))
		})
		It("validate --spec with bad path fails", func() {
			_, _, err := runMcmod(d, "validate", "--spec", "/nonexistent/spec.json")
			Expect(err).To(HaveOccurred())
		})
		It("validate --lock with valid file", func() {
			os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)
			writeLockFile(d, "1.21.1", "neoforge", &domain.PackLock{
				Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{
					"a": {Name: "A", Scope: "shared",
						Source: domain.LockedSource{Type: "local", Path: "./a.jar", FileName: "a.jar"}},
				},
			})
			stdout, _, err := runMcmod(d, "validate", "--lock", filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("valid"))
		})
		It("validate --release-index with valid file", func() {
			os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)
			writeReleaseIndexFile(d, "1.21.1", &domain.ReleaseIndex{
				Type: "package", PackName: "p", MinecraftVersion: "1.21.1",
				Releases: []domain.ReleaseRecord{{Version: "0.1.0", Type: "github-release"}},
			})
			stdout, _, err := runMcmod(d, "validate", "--release-index", filepath.Join(d, "locks", "releases", "1.21.1.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("valid"))
		})
	})
})
