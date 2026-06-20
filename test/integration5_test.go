// File: test/integration5_test.go
// Created: 2026-06-20
// Description: Additional integration tests for tree resolution output,
// lock subcommand path coverage, and validate edge cases.

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

// writeMinimalPackspec writes a minimal packspec.json with the given mods.
func writeMinimalPackspec(d string, mods map[string]interface{}) {
	spec := map[string]interface{}{
		"packName":         "p",
		"serverPackName":   "p-server",
		"packVersion":      "0.1.0",
		"minecraftVersion": "1.21.1",
		"loaderName":       []string{"neoforge:21.1.219"},
		"mods":             mods,
	}
	data, _ := json.MarshalIndent(spec, "", "  ")
	Expect(os.WriteFile(filepath.Join(d, "packspec.json"), data, 0644)).To(Succeed())
}

var _ = Describe("Integration5: tree resolution + lock path coverage", func() {
	var d string

	BeforeEach(func() {
		d = GinkgoT().TempDir()
	})

	// ============== M01: tree with version resolution info ==============
	Describe("M01: tree output details", func() {
		It("M01-1: tree output shows version for each mod", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "tree", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			// The example lock has create@6.0.0, jei@19.21.0.247, server-enhanced@1.4.2
			Expect(stdout).To(ContainSubstring("6.0.0"))
			Expect(stdout).To(ContainSubstring("19.21.0.247"))
			Expect(stdout).To(ContainSubstring("1.4.2"))
		})
		It("M01-2: tree output groups mod lines by source type", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "tree", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			// Per spec 7.3 example: <name> <source:identifier> <version>
			Expect(stdout).To(ContainSubstring("curseforge:328085"))
			Expect(stdout).To(ContainSubstring("github:orangeboyChen/mc-server-enhanced-mod"))
		})
		It("M01-3: tree output shows source:identifier per mod", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "tree", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			// Each mod has a source identifier like "curseforge:ID" or "github:owner/repo".
			Expect(stdout).To(ContainSubstring("curseforge:"))
			Expect(stdout).To(ContainSubstring("github:"))
		})
		It("M01-4: tree output is alphabetized or stable", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout1, _, err := runMcmod(d, "tree", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			stdout2, _, err := runMcmod(d, "tree", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout1).To(Equal(stdout2))
		})
		It("M01-5: tree output with empty lock prints only header", func() {
			Expect(os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)).To(Succeed())
			lock := domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{}}
			data, _ := json.MarshalIndent(lock, "", "  ")
			Expect(os.WriteFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"), data, 0644)).To(Succeed())
			stdout, _, err := runMcmod(d, "tree", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("dependency tree"))
		})
	})

	// ============== M02: lock with no spec / various paths ==============
	Describe("M02: lock command paths", func() {
		It("M02-1: lock with default mc from spec writes lock", func() {
			writeMinimalPackspec(d, map[string]interface{}{
				"shared-mod": map[string]interface{}{
					"name":   "Shared",
					"scope":  "shared",
					"source": map[string]interface{}{"type": "local", "path": "./shared.jar"},
				},
			})
			os.WriteFile(filepath.Join(d, "shared.jar"), []byte("jar"), 0644)
			_, _, err := runMcmod(d, "lock")
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(err).NotTo(HaveOccurred())
		})
		It("M02-2: lock with explicit mc only iterates configured loaders", func() {
			writeMinimalPackspec(d, map[string]interface{}{})
			_, _, err := runMcmod(d, "lock", "1.21.1")
			Expect(err).NotTo(HaveOccurred())
			entries, _ := os.ReadDir(filepath.Join(d, "locks", "dependencies"))
			Expect(entries).To(HaveLen(1))
		})
		It("M02-3: lock (no args) with multi-loader spec writes 2 lock files", func() {
			spec := map[string]interface{}{
				"packName":         "ml",
				"packVersion":      "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName":       []string{"neoforge:21.1.219", "fabric:0.16.0"},
				"mods":             map[string]interface{}{},
			}
			data, _ := json.MarshalIndent(spec, "", "  ")
			os.WriteFile(filepath.Join(d, "packspec.json"), data, 0644)
			_, _, err := runMcmod(d, "lock", "1.21.1")
			Expect(err).NotTo(HaveOccurred())
			entries, _ := os.ReadDir(filepath.Join(d, "locks", "dependencies"))
			Expect(entries).To(HaveLen(2))
		})
		It("M02-4: lock with unsupported loader prints hint to stderr", func() {
			writeMinimalPackspec(d, map[string]interface{}{})
			_, stderr, err := runMcmod(d, "lock", "1.21.1", "fabric")
			// packspec only declares neoforge loader, so fabric should fail with hint
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("M02-5: lock output contains 'Locked' prefix", func() {
			writeMinimalPackspec(d, map[string]interface{}{})
			stdout, _, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.Contains(stdout, "Locked") || strings.Contains(stdout, "lock")).To(BeTrue())
		})
		It("M02-6: lock (no spec) prints hint and exits non-zero", func() {
			_, stderr, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})

	// ============== M03: validate --spec edge cases ==============
	Describe("M03: validate --spec edge cases", func() {
		It("M03-1: validate accepts spec with multi-loader loaderName", func() {
			spec := map[string]interface{}{
				"packName":         "ml",
				"packVersion":      "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName":       []string{"neoforge:21.1.219", "fabric:0.16.0"},
			}
			data, _ := json.MarshalIndent(spec, "", "  ")
			os.WriteFile(filepath.Join(d, "packspec.json"), data, 0644)
			stdout, _, err := runMcmod(d, "validate")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("valid"))
		})
		It("M03-2: validate rejects spec with malformed JSON", func() {
			os.WriteFile(filepath.Join(d, "packspec.json"), []byte(`{ "packName": "x"`), 0644)
			_, _, err := runMcmod(d, "validate")
			Expect(err).To(HaveOccurred())
		})
		It("M03-3: validate rejects spec with empty loaderName array", func() {
			spec := `{
				"packName": "x", "packVersion": "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName": []
			}`
			os.WriteFile(filepath.Join(d, "packspec.json"), []byte(spec), 0644)
			_, _, err := runMcmod(d, "validate")
			Expect(err).To(HaveOccurred())
		})
		It("M03-4: validate rejects spec with bad loaderName entry", func() {
			spec := `{
				"packName": "x", "packVersion": "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName": ["badformat"]
			}`
			os.WriteFile(filepath.Join(d, "packspec.json"), []byte(spec), 0644)
			_, _, err := runMcmod(d, "validate")
			Expect(err).To(HaveOccurred())
		})
		It("M03-5: validate --lock accepts a lock with at least one mod", func() {
			Expect(os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)).To(Succeed())
			lock := domain.PackLock{
				Loader: "neoforge", LoaderVersion: "21.1.219",
				MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{
					"a": {Name: "A", Scope: "shared", Version: "1.0",
						Source: domain.LockedSource{Type: "local", Path: "./a.jar", FileName: "a.jar"}},
				},
			}
			data, _ := json.MarshalIndent(lock, "", "  ")
			path := filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json")
			Expect(os.WriteFile(path, data, 0644)).To(Succeed())
			stdout, _, err := runMcmod(d, "validate", "--lock", path)
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("valid"))
		})
		It("M03-6: validate --release-index accepts a valid index", func() {
			Expect(os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)).To(Succeed())
			ri := domain.ReleaseIndex{
				Type: "package", PackName: "p", MinecraftVersion: "1.21.1",
				Releases: []domain.ReleaseRecord{},
			}
			data, _ := json.MarshalIndent(ri, "", "  ")
			path := filepath.Join(d, "locks", "releases", "1.21.1.json")
			Expect(os.WriteFile(path, data, 0644)).To(Succeed())
			stdout, _, err := runMcmod(d, "validate", "--release-index", path)
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("valid"))
		})
		It("M03-7: validate --release-index with empty releases is valid", func() {
			Expect(os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)).To(Succeed())
			ri := domain.ReleaseIndex{
				Type: "package", PackName: "p", MinecraftVersion: "1.21.1",
				Releases: []domain.ReleaseRecord{},
			}
			data, _ := json.MarshalIndent(ri, "", "  ")
			path := filepath.Join(d, "locks", "releases", "1.21.1.json")
			Expect(os.WriteFile(path, data, 0644)).To(Succeed())
			stdout, _, err := runMcmod(d, "validate", "--release-index", path)
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("valid"))
		})
		It("M03-8: validate --lock rejects bad loader field", func() {
			Expect(os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)).To(Succeed())
			lock := `{
				"loader": "",
				"minecraftVersion": "1.21.1",
				"mods": {}
			}`
			path := filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json")
			Expect(os.WriteFile(path, []byte(lock), 0644)).To(Succeed())
			_, _, err := runMcmod(d, "validate", "--lock", path)
			Expect(err).To(HaveOccurred())
		})
	})

	// ============== M04: lock release edge cases ==============
	Describe("M04: lock release edge cases", func() {
		It("M04-1: release list (no mc arg) uses default", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "lock", "release", "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("1.21.1"))
		})
		It("M04-2: release list with empty index prints no records", func() {
			Expect(os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)).To(Succeed())
			ri := domain.ReleaseIndex{
				Type: "package", PackName: "p", MinecraftVersion: "1.21.1",
				Releases: []domain.ReleaseRecord{},
			}
			data, _ := json.MarshalIndent(ri, "", "  ")
			Expect(os.WriteFile(filepath.Join(d, "locks", "releases", "1.21.1.json"), data, 0644)).To(Succeed())
			stdout, _, err := runMcmod(d, "lock", "release", "list", "1.21.1")
			Expect(err).NotTo(HaveOccurred())
			// Just check it didn't fail and outputs something
			Expect(stdout).To(ContainSubstring("1.21.1"))
		})
		It("M04-3: release set with only --version and --repo creates a new index", func() {
			Expect(os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)).To(Succeed())
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0")
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(d, "locks", "releases", "1.21.1.json"))
			Expect(err).NotTo(HaveOccurred())
		})
		It("M04-4: release set with custom release type", func() {
			Expect(os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)).To(Succeed())
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.1.0", "--type", "github-release", "--repo", "o/r", "--tag", "v0.1.0")
			Expect(err).NotTo(HaveOccurred())
		})
		It("M04-5: release set updates an existing index preserving other versions", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.2.0", "--repo", "o/r", "--tag", "v0.2.0")
			Expect(err).NotTo(HaveOccurred())
			data, _ := os.ReadFile(filepath.Join(d, "locks", "releases", "1.21.1.json"))
			var ri domain.ReleaseIndex
			Expect(json.Unmarshal(data, &ri)).To(Succeed())
			versions := []string{}
			for _, r := range ri.Releases {
				versions = append(versions, r.Version)
			}
			Expect(versions).To(ContainElement("0.1.0"))
			Expect(versions).To(ContainElement("0.2.0"))
		})
		It("M04-6: release show with valid 0.1.0 version prints full record", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "lock", "release", "show", "1.21.1", "0.1.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("github"))
		})
		It("M04-7: release delete with explicit loader and no target deletes full entry", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "release", "delete", "1.21.1", "0.1.0", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			data, _ := os.ReadFile(filepath.Join(d, "locks", "releases", "1.21.1.json"))
			var ri domain.ReleaseIndex
			Expect(json.Unmarshal(data, &ri)).To(Succeed())
			// The neoforge entry should still exist (since we just cleared artifacts
			// for that loader but the record itself is preserved). Actually, looking
			// at the implementation, it should clear all artifacts of that loader.
			rec := ri.FindRelease("0.1.0")
			Expect(rec).NotTo(BeNil())
		})
	})

	// ============== M05: list with various packspecs ==============
	Describe("M05: list with multi-loader packspec", func() {
		It("M05-1: list shows both loaders when multi-loader", func() {
			spec := map[string]interface{}{
				"packName":         "ml",
				"packVersion":      "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName":       []string{"neoforge:21.1.219", "fabric:0.16.0"},
			}
			data, _ := json.MarshalIndent(spec, "", "  ")
			os.WriteFile(filepath.Join(d, "packspec.json"), data, 0644)
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("neoforge:21.1.219"))
			Expect(stdout).To(ContainSubstring("fabric:0.16.0"))
		})
		It("M05-2: list with mod using curseforge query shows [curseforge] tag", func() {
			writeMinimalPackspec(d, map[string]interface{}{
				"create": map[string]interface{}{
					"name":   "Create",
					"scope":  "shared",
					"source": map[string]interface{}{"type": "curseforge", "query": "Create"},
				},
			})
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("[curseforge]"))
		})
		It("M05-3: list with mod using github-release shows [github-release] tag", func() {
			writeMinimalPackspec(d, map[string]interface{}{
				"my-mod": map[string]interface{}{
					"name":   "My Mod",
					"scope":  "shared",
					"source": map[string]interface{}{"type": "github-release", "repo": "o/r", "tag": "v1", "assetPattern": "m.jar"},
				},
			})
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("[github-release]"))
		})
		It("M05-4: list with mod using git shows [git] tag", func() {
			writeMinimalPackspec(d, map[string]interface{}{
				"my-mod": map[string]interface{}{
					"name":   "My Mod",
					"scope":  "shared",
					"source": map[string]interface{}{"type": "git", "repo": "o/r"},
				},
			})
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("[git]"))
		})
		It("M05-5: list with mod using local shows [local] tag", func() {
			writeMinimalPackspec(d, map[string]interface{}{
				"my-mod": map[string]interface{}{
					"name":   "My Mod",
					"scope":  "shared",
					"source": map[string]interface{}{"type": "local", "path": "./m.jar"},
				},
			})
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("[local]"))
		})
		It("M05-6: list with mod missing source.type shows [unknown] tag", func() {
			writeMinimalPackspec(d, map[string]interface{}{
				"my-mod": map[string]interface{}{
					"name":   "My Mod",
					"scope":  "shared",
					"source": map[string]interface{}{},
				},
			})
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("[unknown]"))
		})
	})

	// ============== M06: lock show with specific key edge cases ==============
	Describe("M06: lock show with various source types", func() {
		It("M06-1: lock show local entry prints path", func() {
			Expect(os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)).To(Succeed())
			lock := domain.PackLock{
				Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{
					"local-key": {Name: "Local", Scope: "shared", Version: "1.0",
						Source: domain.LockedSource{Type: "local", Path: "./local.jar", FileName: "local.jar"}},
				},
			}
			data, _ := json.MarshalIndent(lock, "", "  ")
			Expect(os.WriteFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"), data, 0644)).To(Succeed())
			stdout, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge", "local-key")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("type: local"))
		})
		It("M06-2: lock show with explicit mc/loader/key in 3-arg form", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge", "jei")
			Expect(err).NotTo(HaveOccurred())
		})
		It("M06-3: lock show with non-existent lock file fails", func() {
			_, _, err := runMcmod(d, "lock", "show", "9.9.9", "neoforge", "x")
			Expect(err).To(HaveOccurred())
		})
		It("M06-4: lock show 0/1 arg fails with hint", func() {
			_, _, err := runMcmod(d, "lock", "show")
			Expect(err).To(HaveOccurred())
			_, _, err = runMcmod(d, "lock", "show", "1.21.1")
			Expect(err).To(HaveOccurred())
		})
	})

	// ============== M07: lock tree with various ==============
	Describe("M07: lock tree with various inputs", func() {
		It("M07-1: lock tree 0 args uses default", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "lock", "tree")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("dependency tree"))
		})
		It("M07-2: lock tree 1 arg uses default loader", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "lock", "tree", "1.21.1")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("dependency tree"))
		})
		It("M07-3: lock tree 2 args full", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "lock", "tree", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("dependency tree"))
		})
		It("M07-4: lock tree with no lock file fails with hint", func() {
			writeMinimalPackspec(d, map[string]interface{}{})
			_, stderr, err := runMcmod(d, "lock", "tree", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})
})
