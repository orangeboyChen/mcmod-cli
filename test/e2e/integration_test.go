// File: test/e2e/integration_test.go
// Created: 2026-06-20
// Description: End-to-end integration tests using real packspec/lock/build pipeline
// and exercising every CLI subcommand with a populated workspace.

package test

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

// readJSON is a tiny helper to parse JSON files inside tests.
func readJSON(t GinkgoTInterface, path string, out interface{}) {
	data, err := os.ReadFile(path)
	Expect(err).NotTo(HaveOccurred(), "read %s", path)
	Expect(json.Unmarshal(data, out)).To(Succeed(), "parse %s", path)
}

// writeJSON writes a struct as pretty JSON to path, creating parent dirs.
func writeJSON(path string, v interface{}) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		Fail(err.Error())
	}
	data, err := json.MarshalIndent(v, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(path, data, 0644)).To(Succeed())
}

// seedFakeJar writes a small zip file at the given path with a metadata entry
// and a dummy class file. Used to satisfy the build pipeline that needs jars
// in .cache/.
func seedFakeJar(path, metaPath, metaContent string) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		Fail(err.Error())
	}
	f, err := os.Create(path)
	Expect(err).NotTo(HaveOccurred())
	defer f.Close()
	w := zip.NewWriter(f)
	defer w.Close()
	entry, _ := w.Create(metaPath)
	_, _ = io.WriteString(entry, metaContent)
	class, _ := w.Create("com/example/Foo.class")
	_, _ = io.WriteString(class, "fake-class-data")
}

// seedCachedJar creates an empty placeholder jar in the cache directory tree.
func seedCachedJar(path string) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		Fail(err.Error())
	}
	Expect(os.WriteFile(path, []byte("placeholder-jar"), 0644)).To(Succeed())
}

// countZipEntries returns the number of file entries in a zip archive.
func countZipEntries(path string) int {
	r, err := zip.OpenReader(path)
	Expect(err).NotTo(HaveOccurred(), "open %s", path)
	defer r.Close()
	return len(r.File)
}

// zipHasEntry returns true if the zip contains an entry with the given name.
func zipHasEntry(path, name string) bool {
	r, err := zip.OpenReader(path)
	Expect(err).NotTo(HaveOccurred(), "open %s", path)
	defer r.Close()
	for _, f := range r.File {
		if f.Name == name {
			return true
		}
	}
	return false
}

// makeRealPackSpec writes a packspec.json that mirrors the project's real
// packspec.json schema (mods dict with per-mod scope, local+curseforge+github
// source variants) to the given directory.
func makeRealPackSpec(dir string) {
	spec := map[string]interface{}{
		"packName":         "integration-pack",
		"serverPackName":   "integration-server",
		"packVersion":      "0.1.0",
		"minecraftVersion": "1.21.1",
		"loaderName":       []string{"neoforge:21.1.219"},
		"author":           "integration-test",
		"mods": map[string]interface{}{
			"create": map[string]interface{}{
				"name":  "Create",
				"scope": "shared",
				"source": map[string]interface{}{
					"type":  "curseforge",
					"query": "Create",
				},
			},
			"jei": map[string]interface{}{
				"name":   "Just Enough Items",
				"scope":  "client",
				"loader": []string{"neoforge"},
				"source": map[string]interface{}{
					"type":  "curseforge",
					"query": "Just Enough Items",
				},
			},
			"server-enhanced": map[string]interface{}{
				"name":   "Server Enhanced Mod",
				"scope":  "server",
				"loader": []string{"neoforge"},
				"source": map[string]interface{}{
					"type":         "github-release",
					"repo":         "orangeboyChen/mc-server-enhanced-mod",
					"tag":          "v1.4.2",
					"assetPattern": "serverenhancedmod-1.21.1-*.jar",
				},
			},
			"asset-pattern-object": map[string]interface{}{
				"name":   "Asset Pattern Object",
				"scope":  "shared",
				"loader": []string{"neoforge"},
				"source": map[string]interface{}{
					"type": "github-release",
					"repo": "example/asset-pattern-object",
					"tag":  "v1.0.0",
					"assetPattern": map[string]string{
						"neoforge": "apobj-1.21.1-neoforge.jar",
					},
				},
			},
			"local-mod": map[string]interface{}{
				"name":   "Local Mod",
				"scope":  "client",
				"loader": []string{"neoforge"},
				"source": map[string]interface{}{
					"type": "local",
					"path": "./mods/local-mod.jar",
				},
			},
		},
	}
	data, _ := json.MarshalIndent(spec, "", "  ")
	Expect(os.WriteFile(filepath.Join(dir, "packspec.json"), data, 0644)).To(Succeed())
}

// makeRealLockForBuild writes a complete lock file referencing cached jars so
// that `mcmod build` can resolve every mod on disk.
func makeRealLockForBuild(dir string) {
	lock := &domain.PackLock{
		Loader: "neoforge", LoaderVersion: "21.1.219",
		MinecraftVersion: "1.21.1",
		Mods: map[string]domain.LockedMod{
			"create":               {Name: "Create", Version: "6.0.0", Scope: "shared", Source: domain.LockedSource{Type: "curseforge", ModID: 328085, FileID: 5812340, FileName: "create-1.21.1-neoforge.jar"}},
			"jei":                  {Name: "Just Enough Items", Version: "19.21.0.247", Scope: "client", Source: domain.LockedSource{Type: "curseforge", ModID: 238222, FileID: 5812400, FileName: "jei-1.21.1-neoforge.jar"}},
			"server-enhanced":      {Name: "Server Enhanced Mod", Version: "1.4.2", Scope: "server", Source: domain.LockedSource{Type: "github-release", Repo: "orangeboyChen/mc-server-enhanced-mod", Tag: "v1.4.2", AssetName: "serverenhancedmod-1.21.1-neoforge.jar", FileName: "serverenhancedmod-1.21.1-neoforge.jar"}},
			"asset-pattern-object": {Name: "Asset Pattern Object", Version: "1.0.0", Scope: "shared", Source: domain.LockedSource{Type: "github-release", Repo: "example/asset-pattern-object", Tag: "v1.0.0", AssetName: "apobj-1.21.1-neoforge.jar", FileName: "apobj-1.21.1-neoforge.jar"}},
			"local-mod":            {Name: "Local Mod", Version: "1.0.0", Scope: "client", Source: domain.LockedSource{Type: "local", Path: "./mods/local-mod.jar", FileName: "local-mod.jar"}},
		},
	}
	writeLockFile(dir, "1.21.1", "neoforge", lock)
}

// seedBuildCache creates the .cache/ jars required by makeRealLockForBuild so
// that build can complete without network access.
func seedBuildCache(dir string) {
	base := filepath.Join(dir, ".cache")
	seedCachedJar(filepath.Join(base, "curseforge", "328085", "5812340", "create-1.21.1-neoforge.jar"))
	seedCachedJar(filepath.Join(base, "curseforge", "238222", "5812400", "jei-1.21.1-neoforge.jar"))
	seedCachedJar(filepath.Join(base, "github-release", "orangeboyChen", "mc-server-enhanced-mod", "v1.4.2", "serverenhancedmod-1.21.1-neoforge.jar"))
	seedCachedJar(filepath.Join(base, "github-release", "example", "asset-pattern-object", "v1.0.0", "apobj-1.21.1-neoforge.jar"))
	localPath := filepath.Join(dir, "mods", "local-mod.jar")
	Expect(os.MkdirAll(filepath.Dir(localPath), 0755)).To(Succeed())
	seedCachedJar(localPath)
}

var _ = Describe("Integration: real packspec + build pipeline", func() {
	var d string

	BeforeEach(func() {
		d = GinkgoT().TempDir()
	})

	// ============== I01: read packspec + assetPattern tolerance ==============
	Describe("I01: read real packspec", func() {
		It("reads packspec with mods dict and assetPattern object", func() {
			makeRealPackSpec(d)
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("Create"))
			Expect(stdout).To(ContainSubstring("Just Enough Items"))
			Expect(stdout).To(ContainSubstring("Server Enhanced Mod"))
			Expect(stdout).To(ContainSubstring("Asset Pattern Object"))
			Expect(stdout).To(ContainSubstring("[Server]"))
			Expect(stdout).To(ContainSubstring("[Client]"))
			Expect(stdout).To(ContainSubstring("[Shared]"))
		})
		It("validate accepts real packspec", func() {
			makeRealPackSpec(d)
			stdout, _, err := runMcmod(d, "validate")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("valid"))
		})
		It("validate --spec accepts real packspec by path", func() {
			makeRealPackSpec(d)
			stdout, _, err := runMcmod(d, "validate", "--spec", filepath.Join(d, "packspec.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("valid"))
		})
	})

	// ============== I02: lock list/show with full real lockfile ==============
	Describe("I02: lock list/show with real lock", func() {
		BeforeEach(func() {
			makeRealPackSpec(d)
			makeRealLockForBuild(d)
		})
		It("lock list shows all scopes", func() {
			stdout, _, err := runMcmod(d, "lock", "list", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("[Server]"))
			Expect(stdout).To(ContainSubstring("[Client]"))
			Expect(stdout).To(ContainSubstring("[Shared]"))
			Expect(stdout).To(ContainSubstring("create"))
			Expect(stdout).To(ContainSubstring("jei"))
			Expect(stdout).To(ContainSubstring("server-enhanced"))
		})
		It("lock show full dumps JSON", func() {
			stdout, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			var l domain.PackLock
			Expect(json.Unmarshal([]byte(stdout), &l)).To(Succeed())
			Expect(l.Loader).To(Equal("neoforge"))
			Expect(l.Mods).To(HaveLen(5))
		})
		It("lock show single key prints curseforge fields", func() {
			stdout, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge", "create")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("modId: 328085"))
			Expect(stdout).To(ContainSubstring("fileId: 5812340"))
			Expect(stdout).To(ContainSubstring("fileName: create-1.21.1-neoforge.jar"))
		})
		It("lock show single key prints github-release fields", func() {
			stdout, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge", "server-enhanced")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("repo: orangeboyChen/mc-server-enhanced-mod"))
			Expect(stdout).To(ContainSubstring("tag: v1.4.2"))
		})
		It("tree alias reads the real lock", func() {
			stdout, _, err := runMcmod(d, "tree", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("dependency tree"))
			Expect(stdout).To(ContainSubstring("Create"))
		})
	})

	// ============== I03: lock add/update/delete round-trip ==============
	Describe("I03: lock add/update/delete round-trip", func() {
		BeforeEach(func() {
			makeRealPackSpec(d)
			makeRealLockForBuild(d)
		})
		It("lock add appends a new entry to the real lock", func() {
			_, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge", "extra-mod",
				"--name", "Extra Mod", "--version", "1.0.0", "--scope", "shared",
				"--source", "curseforge", "--mod-id", "1", "--file-id", "2", "--file-name", "extra.jar")
			Expect(err).NotTo(HaveOccurred())
			var l domain.PackLock
			readJSON(GinkgoT(), filepath.Join(d, "locks/dependencies/1.21.1-neoforge.json"), &l)
			Expect(l.Mods).To(HaveLen(6))
			Expect(l.Mods["extra-mod"].Source.ModID).To(Equal(1))
		})
		It("lock add local extends the real lock", func() {
			_, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge", "local-extra",
				"--name", "Local Extra", "--source", "local", "--path", "./mods/lx.jar", "--file-name", "lx.jar")
			Expect(err).NotTo(HaveOccurred())
			var l domain.PackLock
			readJSON(GinkgoT(), filepath.Join(d, "locks/dependencies/1.21.1-neoforge.json"), &l)
			Expect(l.Mods).To(HaveLen(6))
			Expect(l.Mods["local-extra"].Source.Path).To(Equal("./mods/lx.jar"))
		})
		It("lock update changes a single entry version", func() {
			_, _, err := runMcmod(d, "lock", "update", "1.21.1", "neoforge", "create", "--version", "7.0.0")
			Expect(err).NotTo(HaveOccurred())
			var l domain.PackLock
			readJSON(GinkgoT(), filepath.Join(d, "locks/dependencies/1.21.1-neoforge.json"), &l)
			Expect(l.Mods["create"].Version).To(Equal("7.0.0"))
		})
		It("lock delete removes a single entry but keeps the rest", func() {
			_, _, err := runMcmod(d, "lock", "delete", "1.21.1", "neoforge", "jei")
			Expect(err).NotTo(HaveOccurred())
			var l domain.PackLock
			readJSON(GinkgoT(), filepath.Join(d, "locks/dependencies/1.21.1-neoforge.json"), &l)
			Expect(l.Mods).To(HaveLen(4))
			Expect(l.Mods).To(HaveKey("create"))
			Expect(l.Mods).To(HaveKey("server-enhanced"))
			Expect(l.Mods).NotTo(HaveKey("jei"))
		})
	})

	// ============== I04: lock release set/list/show/delete ==============
	Describe("I04: lock release round-trip on real index", func() {
		It("release set then list then show then delete", func() {
			// Pre-create an existing index to exercise merging.
			existing := &domain.ReleaseIndex{
				Type: "package", PackName: "integration-pack", MinecraftVersion: "1.21.1",
				Releases: []domain.ReleaseRecord{
					{Version: "0.1.0", Type: "github-release",
						GitHub: domain.ReleaseGitHub{Repo: "owner/p", Tag: "v0.1.0", Name: "v0.1.0"}},
				},
			}
			os.MkdirAll(filepath.Join(d, "locks/releases"), 0755)
			writeReleaseIndexFile(d, "1.21.1", existing)

			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.2.0", "--repo", "owner/p", "--tag", "v0.2.0",
				"--name", "v0.2.0", "--body", "release notes", "--prerelease")
			Expect(err).NotTo(HaveOccurred())

			stdout, _, err := runMcmod(d, "lock", "release", "list", "1.21.1")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("0.1.0"))
			Expect(stdout).To(ContainSubstring("0.2.0"))

			out, _, err := runMcmod(d, "lock", "release", "show", "1.21.1", "0.2.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(ContainSubstring("owner/p"))
			Expect(out).To(ContainSubstring("v0.2.0"))
			Expect(out).To(ContainSubstring("release notes"))

			// delete only 0.2.0; 0.1.0 must remain
			_, _, err = runMcmod(d, "lock", "release", "delete", "1.21.1", "0.2.0")
			Expect(err).NotTo(HaveOccurred())
			var ri domain.ReleaseIndex
			readJSON(GinkgoT(), filepath.Join(d, "locks/releases/1.21.1.json"), &ri)
			Expect(ri.Releases).To(HaveLen(1))
			Expect(ri.Releases[0].Version).To(Equal("0.1.0"))
		})
		It("release delete removes single artifact for one loader", func() {
			existing := &domain.ReleaseIndex{
				Type: "package", PackName: "integration-pack", MinecraftVersion: "1.21.1",
				Releases: []domain.ReleaseRecord{{
					Version: "0.5.0", Type: "github-release",
					GitHub: domain.ReleaseGitHub{Repo: "owner/p", Tag: "v0.5.0"},
					Artifact: map[string]domain.ReleaseArtifactSet{
						"neoforge": {
							Client: "client.zip",
							Server: "server.zip",
						},
					},
				}},
			}
			os.MkdirAll(filepath.Join(d, "locks/releases"), 0755)
			writeReleaseIndexFile(d, "1.21.1", existing)

			_, _, err := runMcmod(d, "lock", "release", "delete", "1.21.1", "0.5.0", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			var ri domain.ReleaseIndex
			readJSON(GinkgoT(), filepath.Join(d, "locks/releases/1.21.1.json"), &ri)
			Expect(ri.Releases[0].Artifact["neoforge"].Client).To(BeEmpty())
			Expect(ri.Releases[0].Artifact["neoforge"].Server).To(Equal("server.zip"))
		})
		It("release delete with --target both removes all artifacts for the loader", func() {
			existing := &domain.ReleaseIndex{
				Type: "package", PackName: "integration-pack", MinecraftVersion: "1.21.1",
				Releases: []domain.ReleaseRecord{{
					Version: "0.6.0", Type: "github-release",
					Artifact: map[string]domain.ReleaseArtifactSet{
						"neoforge": {Client: "c.zip", Server: "s.zip"},
					},
				}},
			}
			os.MkdirAll(filepath.Join(d, "locks/releases"), 0755)
			writeReleaseIndexFile(d, "1.21.1", existing)

			_, _, err := runMcmod(d, "lock", "release", "delete", "1.21.1", "0.6.0", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			var ri domain.ReleaseIndex
			readJSON(GinkgoT(), filepath.Join(d, "locks/releases/1.21.1.json"), &ri)
			Expect(ri.Releases[0].Artifact["neoforge"].Client).To(BeEmpty())
			Expect(ri.Releases[0].Artifact["neoforge"].Server).To(BeEmpty())
		})
		It("release delete fails for unknown version", func() {
			_, _, err := runMcmod(d, "lock", "release", "delete", "1.21.1", "99.99.99")
			Expect(err).To(HaveOccurred())
		})
		It("release delete fails when index missing", func() {
			_, _, err := runMcmod(d, "lock", "release", "delete", "1.21.1", "0.1.0")
			Expect(err).To(HaveOccurred())
		})
	})

	// ============== I05: validate lock file with real lock ==============
	Describe("I05: validate lock with real lock", func() {
		It("accepts a real lock file", func() {
			makeRealPackSpec(d)
			makeRealLockForBuild(d)
			stdout, _, err := runMcmod(d, "validate", "--lock", filepath.Join(d, "locks/dependencies/1.21.1-neoforge.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("valid"))
		})
	})

	// ============== I06: validate release index with real index ==============
	Describe("I06: validate release index", func() {
		It("accepts a real release index", func() {
			ri := &domain.ReleaseIndex{
				Type: "package", PackName: "integration-pack", MinecraftVersion: "1.21.1",
				Releases: []domain.ReleaseRecord{{Version: "0.1.0", Type: "github-release"}},
			}
			os.MkdirAll(filepath.Join(d, "locks/releases"), 0755)
			writeReleaseIndexFile(d, "1.21.1", ri)
			stdout, _, err := runMcmod(d, "validate", "--release-index", filepath.Join(d, "locks/releases/1.21.1.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("valid"))
		})
	})

	// ============== I07: build with real lock + cache ==============
	Describe("I07: build with real lock + seeded cache", func() {
		BeforeEach(func() {
			makeRealPackSpec(d)
			makeRealLockForBuild(d)
			seedBuildCache(d)
		})
		It("build with default target produces both client and server zips", func() {
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			// Two zips must exist in releases/v0.1.0/
			files, _ := filepath.Glob(filepath.Join(d, "releases", "v0.1.0", "*.zip"))
			Expect(files).To(HaveLen(2))
			var hasClient, hasServer bool
			for _, f := range files {
				name := filepath.Base(f)
				if strings.Contains(name, "client") {
					hasClient = true
				}
				if strings.Contains(name, "server") {
					hasServer = true
				}
			}
			Expect(hasClient).To(BeTrue(), "client zip missing")
			Expect(hasServer).To(BeTrue(), "server zip missing")
		})
		It("build with --target client produces only client zip", func() {
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			files, _ := filepath.Glob(filepath.Join(d, "releases", "v0.1.0", "*.zip"))
			Expect(files).To(HaveLen(1))
			Expect(filepath.Base(files[0])).To(ContainSubstring("client"))
		})
		It("build with --target server produces only server zip", func() {
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "server")
			Expect(err).NotTo(HaveOccurred())
			files, _ := filepath.Glob(filepath.Join(d, "releases", "v0.1.0", "*.zip"))
			Expect(files).To(HaveLen(1))
			Expect(filepath.Base(files[0])).To(ContainSubstring("server"))
		})
		It("client zip contains shared + client mods", func() {
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			files, _ := filepath.Glob(filepath.Join(d, "releases", "v0.1.0", "*client.zip"))
			Expect(files).To(HaveLen(1))
			Expect(zipHasEntry(files[0], "mods/create-1.21.1-neoforge.jar")).To(BeTrue())
			Expect(zipHasEntry(files[0], "mods/jei-1.21.1-neoforge.jar")).To(BeTrue())
			Expect(zipHasEntry(files[0], "mods/serverenhancedmod-1.21.1-neoforge.jar")).To(BeFalse())
		})
		It("server zip contains shared + server mods", func() {
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "server")
			Expect(err).NotTo(HaveOccurred())
			files, _ := filepath.Glob(filepath.Join(d, "releases", "v0.1.0", "*server.zip"))
			Expect(files).To(HaveLen(1))
			Expect(zipHasEntry(files[0], "mods/create-1.21.1-neoforge.jar")).To(BeTrue())
			Expect(zipHasEntry(files[0], "mods/jei-1.21.1-neoforge.jar")).To(BeFalse())
			Expect(zipHasEntry(files[0], "mods/serverenhancedmod-1.21.1-neoforge.jar")).To(BeTrue())
		})
		It("build fails with hint when lock missing", func() {
			os.RemoveAll(filepath.Join(d, "locks"))
			_, stderr, err := runMcmod(d, "build", "1.21.1", "neoforge")
			_ = err
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("build fails with hint when spec missing", func() {
			os.Remove(filepath.Join(d, "packspec.json"))
			_, stderr, err := runMcmod(d, "build", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})

	// ============== I08: build with multi-loader spec ==============
	Describe("I08: build with multi-loader spec", func() {
		It("build iterates every loader in packspec", func() {
			makeRealPackSpec(d)
			// Add fabric loader + fabric lock + fabric cache
			specPath := filepath.Join(d, "packspec.json")
			data, _ := os.ReadFile(specPath)
			var spec map[string]interface{}
			Expect(json.Unmarshal(data, &spec)).To(Succeed())
			spec["loaderName"] = []string{"neoforge:21.1.219", "fabric:1.21.123"}
			writeJSON(specPath, spec)

			// fabric lock + cache
			fabricLock := &domain.PackLock{
				Loader: "fabric", LoaderVersion: "1.21.123",
				MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{
					"create": {Name: "Create", Scope: "shared", Source: domain.LockedSource{Type: "curseforge", ModID: 328085, FileID: 5812340, FileName: "create-1.21.1-fabric.jar"}},
				},
			}
			writeLockFile(d, "1.21.1", "fabric", fabricLock)
			makeRealLockForBuild(d)
			seedBuildCache(d)
			seedCachedJar(filepath.Join(d, ".cache/curseforge/328085/5812340/create-1.21.1-fabric.jar"))

			_, _, err := runMcmod(d, "build", "1.21.1")
			Expect(err).NotTo(HaveOccurred())
			files, _ := filepath.Glob(filepath.Join(d, "releases", "v0.1.0", "*.zip"))
			Expect(files).To(HaveLen(4), "expected 4 zips (2 loaders x 2 targets)")
		})
	})

	// ============== I09: config + set round-trip (isolated HOME) ==============
	Describe("I09: config + set isolation", func() {
		It("set cf-key --project writes .mcmod/config.json in temp dir only", func() {
			_, _, err := runMcmod(d, "set", "cf-key", "integration-test-key", "--project")
			Expect(err).NotTo(HaveOccurred())
			data, err := os.ReadFile(filepath.Join(d, ".mcmod/config.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(ContainSubstring("integration-test-key"))
		})
		It("set cf-key without --project does not pollute the temp dir", func() {
			_, _, err := runMcmod(d, "set", "cf-key", "userkey")
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(d, ".mcmod/config.json"))
			Expect(os.IsNotExist(err)).To(BeTrue(), "project config should not exist for user-level set")
		})
		It("config reflects the key set via set --project", func() {
			_, _, _ = runMcmod(d, "set", "cf-key", "reflected-key", "--project")
			stdout, _, err := runMcmod(d, "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("reflected-key"))
		})
	})

	// ============== I10: validate/release/--help end-to-end ==============
	Describe("I10: --help outputs for every subcommand", func() {
		It("all subcommands accept --help", func() {
			commands := [][]string{
				{"--help"},
				{"help"},
				{"version"},
				{"list", "--help"},
				{"validate", "--help"},
				{"build", "--help"},
				{"tree", "--help"},
				{"set", "--help"},
				{"config", "--help"},
				{"lock", "--help"},
				{"lock", "list", "--help"},
				{"lock", "show", "--help"},
				{"lock", "add", "--help"},
				{"lock", "update", "--help"},
				{"lock", "delete", "--help"},
				{"lock", "tree", "--help"},
				{"lock", "release", "--help"},
				{"lock", "release", "set", "--help"},
				{"lock", "release", "list", "--help"},
				{"lock", "release", "show", "--help"},
				{"lock", "release", "delete", "--help"},
			}
			for _, args := range commands {
				_, _, err := runMcmod(d, args...)
				Expect(err).NotTo(HaveOccurred(), "command %v", args)
			}
		})
	})

	// ============== I11: error paths end-to-end ==============
	Describe("I11: error paths end-to-end", func() {
		It("unknown command fails", func() {
			_, _, err := runMcmod(d, "unknown-command-xyz")
			Expect(err).To(HaveOccurred())
		})
		It("lock without spec fails", func() {
			_, _, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("validate in empty dir fails with hint", func() {
			_, stderr, err := runMcmod(d, "validate")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("lock release list with no index fails", func() {
			_, _, err := runMcmod(d, "lock", "release", "list", "1.21.1")
			Expect(err).To(HaveOccurred())
		})
	})
})
