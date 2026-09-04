// File: test/integration2_test.go
// Created: 2026-06-20
// Description: End-to-end integration tests that use the real project
// packspec.json (21 mods across curseforge / github-release / local) and the
// real mcmod binary to exercise every command and subcommand.

package test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

// copyProjectPackSpec copies the real project packspec.json into a temp
// directory so tests can run against the real modlist.
func copyProjectPackSpec(t GinkgoTInterface, dst string) {
	root, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	src := filepath.Join(root, "packspec.json")
	if _, err := os.Stat(src); err != nil {
		src = filepath.Join(root, "..", "packspec.json")
	}
	in, err := os.ReadFile(src)
	Expect(err).NotTo(HaveOccurred(), "read project packspec.json")
	Expect(os.WriteFile(filepath.Join(dst, "packspec.json"), in, 0644)).To(Succeed())
}

// copyDir recursively copies src to dst.
func copyDir(t GinkgoTInterface, src, dst string) {
	err := os.MkdirAll(dst, 0755)
	Expect(err).NotTo(HaveOccurred())
	entries, err := os.ReadDir(src)
	if err != nil {
		return
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if e.IsDir() {
			copyDir(t, s, d)
		} else {
			data, err := os.ReadFile(s)
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(d, data, 0644)).To(Succeed())
		}
	}
}

// copyExampleSeed copies the example smoke workspace (packspec + locks + .cache)
// into a temp directory so tests can build against cached jars.
func copyExampleSeed(t GinkgoTInterface, dst string) {
	root, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	exampleDir := filepath.Join(root, "..", "examples", "smoke")
	if _, err := os.Stat(exampleDir); err != nil {
		exampleDir = filepath.Join(root, "examples", "smoke")
	}
	data, err := os.ReadFile(filepath.Join(exampleDir, "packspec.json"))
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(filepath.Join(dst, "packspec.json"), data, 0644)).To(Succeed())
	for _, sub := range []string{"locks/dependencies", "locks/releases"} {
		src := filepath.Join(exampleDir, sub)
		dstDir := filepath.Join(dst, sub)
		Expect(os.MkdirAll(dstDir, 0755)).To(Succeed())
		entries, err := os.ReadDir(src)
		Expect(err).NotTo(HaveOccurred())
		for _, e := range entries {
			b, err := os.ReadFile(filepath.Join(src, e.Name()))
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(filepath.Join(dstDir, e.Name()), b, 0644)).To(Succeed())
		}
	}
	copyDir(t, filepath.Join(exampleDir, ".cache"), filepath.Join(dst, ".cache"))
}

// runMcmodWithEnv runs mcmod with explicit env overrides (used to test
// CURSEFORGE_API_KEY resolution order).
func runMcmodWithEnv(dir string, env []string, args ...string) (string, string, error) {
	cmd := exec.Command(mcmodBin, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

var _ = Describe("Integration2: real packspec coverage", func() {
	var d string

	BeforeEach(func() {
		d = GinkgoT().TempDir()
	})

	// ============== J01: list against real 21-mod packspec ==============
	Describe("J01: list against real packspec", func() {
		It("J01-1: list prints all 21 mods from real packspec", func() {
			copyProjectPackSpec(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			expectedNames := []string{
				"Farmer's Delight", "Create", "Create Crafts",
				"Create Aeronautics", "Brewin", "My Nether",
				"Barbeque", "CasualnessDelight", "Corn Delight",
				"Cuisine Delight", "Curtain", "End's Delight", "Exposure",
				"Kotlin for Forge", "Mysterious Mountain Lib", "Pineapple Delight",
				"Sable", "Ube's Delight", "Greenhouse Config", "Just Enough Items", "MC Server Enhanced Mod",
			}
			for _, k := range expectedNames {
				Expect(stdout).To(ContainSubstring(k), "missing mod display name %s in list", k)
			}
		})
		It("J01-2: list groups by scope with shared, client, server", func() {
			copyProjectPackSpec(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("[Server]"))
			Expect(stdout).To(ContainSubstring("[Client]"))
			Expect(stdout).To(ContainSubstring("[Shared]"))
		})
		It("J01-3: list shows github-release and curseforge source tags", func() {
			copyProjectPackSpec(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("[github-release]"))
			Expect(stdout).To(ContainSubstring("[curseforge]"))
		})
		It("J01-4: list reports missing packspec with hint", func() {
			_, stderr, err := runMcmod(d, "list")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})

	// ============== J02: validate real packspec ==============
	Describe("J02: validate real packspec", func() {
		It("J02-1: validate accepts real packspec via default path", func() {
			copyProjectPackSpec(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "validate")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("valid"))
		})
		It("J02-2: validate --spec accepts real packspec by path", func() {
			copyProjectPackSpec(GinkgoT(), d)
			sp := filepath.Join(d, "packspec.json")
			stdout, _, err := runMcmod(d, "validate", "--spec", sp)
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("valid"))
		})
		It("J02-3: validate --spec rejects missing path with hint", func() {
			_, stderr, err := runMcmod(d, "validate", "--spec", "/no/such/file.json")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})

	// ============== J03: validate --lock with various lock files ==============
	Describe("J03: validate --lock with various lock files", func() {
		It("J03-1: validate --lock accepts a valid lock", func() {
			copyProjectPackSpec(GinkgoT(), d)
			Expect(os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)).To(Succeed())
			lock := domain.PackLock{
				Loader: "neoforge", LoaderVersion: "21.1.219",
				MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{
					"a": {Name: "A", Version: "1.0", Scope: "shared",
						Source: domain.LockedSource{Type: "local", Path: "./a.jar", FileName: "a.jar"}},
				},
			}
			path := filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json")
			data, _ := json.MarshalIndent(lock, "", "  ")
			Expect(os.WriteFile(path, data, 0644)).To(Succeed())
			stdout, _, err := runMcmod(d, "validate", "--lock", path)
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("valid"))
		})
		It("J03-2: validate --lock rejects malformed JSON", func() {
			path := filepath.Join(d, "broken.json")
			Expect(os.WriteFile(path, []byte("not-json"), 0644)).To(Succeed())
			_, _, err := runMcmod(d, "validate", "--lock", path)
			Expect(err).To(HaveOccurred())
		})
		It("J03-3: validate --lock rejects missing file", func() {
			_, stderr, err := runMcmod(d, "validate", "--lock", "/no/such/lock.json")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})

	// ============== J04: validate --release-index ==============
	Describe("J04: validate --release-index", func() {
		It("J04-1: validate --release-index accepts valid file", func() {
			Expect(os.MkdirAll(filepath.Join(d, "locks", "releases"), 0755)).To(Succeed())
			ri := domain.ReleaseIndex{
				Type: "package", PackName: "p", MinecraftVersion: "1.21.1",
				Releases: []domain.ReleaseRecord{{Version: "0.1.0", Type: "github-release"}},
			}
			path := filepath.Join(d, "locks", "releases", "1.21.1.json")
			data, _ := json.MarshalIndent(ri, "", "  ")
			Expect(os.WriteFile(path, data, 0644)).To(Succeed())
			stdout, _, err := runMcmod(d, "validate", "--release-index", path)
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("valid"))
		})
		It("J04-2: validate --release-index rejects malformed JSON", func() {
			path := filepath.Join(d, "broken.json")
			Expect(os.WriteFile(path, []byte("{not-json"), 0644)).To(Succeed())
			_, _, err := runMcmod(d, "validate", "--release-index", path)
			Expect(err).To(HaveOccurred())
		})
		It("J04-3: validate --release-index rejects missing file", func() {
			_, stderr, err := runMcmod(d, "validate", "--release-index", "/no/such/ri.json")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})

	// ============== J05: validate --spec with bad fields ==============
	Describe("J05: validate --spec with bad fields", func() {
		It("J05-1: validate --spec rejects missing packName", func() {
			Expect(os.WriteFile(filepath.Join(d, "packspec.json"), []byte(`{
				"packVersion": "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName": ["neoforge:21.1.219"]
			}`), 0644)).To(Succeed())
			_, _, err := runMcmod(d, "validate")
			Expect(err).To(HaveOccurred())
		})
		It("J05-2: validate --spec rejects missing minecraftVersion", func() {
			Expect(os.WriteFile(filepath.Join(d, "packspec.json"), []byte(`{
				"packName": "p", "packVersion": "0.1.0",
				"loaderName": ["neoforge:21.1.219"]
			}`), 0644)).To(Succeed())
			_, _, err := runMcmod(d, "validate")
			Expect(err).To(HaveOccurred())
		})
		It("J05-3: validate --spec rejects missing loaderName", func() {
			Expect(os.WriteFile(filepath.Join(d, "packspec.json"), []byte(`{
				"packName": "p", "packVersion": "0.1.0",
				"minecraftVersion": "1.21.1"
			}`), 0644)).To(Succeed())
			_, _, err := runMcmod(d, "validate")
			Expect(err).To(HaveOccurred())
		})
	})

	// ============== J06: lock with real 21-mod packspec ==============
	Describe("J06: lock writes real 21-mod spec", func() {
		It("J06-1: lock without spec fails with hint", func() {
			_, stderr, err := runMcmod(d, "lock")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("J06-2: lock with empty mods writes empty map", func() {
			Expect(os.WriteFile(filepath.Join(d, "packspec.json"), []byte(`{
				"packName": "e", "packVersion": "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName": ["neoforge:21.1.219"]
			}`), 0644)).To(Succeed())
			_, _, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			data, err := os.ReadFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(ContainSubstring(`"mods": {}`))
		})
		It("J06-3: lock with unsupported loader prints hint to stderr", func() {
			copyProjectPackSpec(GinkgoT(), d)
			_, stderr, err := runMcmod(d, "lock", "1.21.1", "unsupported-loader")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})

	// ============== J07: tree against real packspec + real lock ==============
	Describe("J07: tree against real workspace", func() {
		It("J07-1: tree alias reads example lock and prints all mods", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "tree")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("dependency tree"))
			Expect(stdout).To(ContainSubstring("Create"))
			Expect(stdout).To(ContainSubstring("Just Enough Items"))
			Expect(stdout).To(ContainSubstring("Server Enhanced"))
		})
		It("J07-2: lock tree with explicit args", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "lock", "tree", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("dependency tree"))
		})
		It("J07-3: tree with no lock fails with hint", func() {
			Expect(os.WriteFile(filepath.Join(d, "packspec.json"), []byte(`{
				"packName": "p", "packVersion": "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName": ["neoforge:21.1.219"]
			}`), 0644)).To(Succeed())
			_, stderr, err := runMcmod(d, "tree")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("J07-4: lock tree with 0 args uses default mc/loader", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "lock", "tree")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("1.21.1"))
		})
		It("J07-5: lock tree with 1 arg uses default loader", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "lock", "tree", "1.21.1")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("1.21.1"))
		})
	})

	// ============== J08: lock list with example lock ==============
	Describe("J08: lock list with example workspace", func() {
		It("J08-1: lock list shows all three scope sections", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "lock", "list", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("[Server]"))
			Expect(stdout).To(ContainSubstring("[Client]"))
			Expect(stdout).To(ContainSubstring("[Shared]"))
		})
		It("J08-2: lock list (no args) uses default mc/loader", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "lock", "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("lock 1.21.1 neoforge"))
		})
		It("J08-3: lock list with empty mods prints (empty) three times", func() {
			Expect(os.MkdirAll(filepath.Join(d, "locks", "dependencies"), 0755)).To(Succeed())
			lock := domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{}}
			data, _ := json.MarshalIndent(lock, "", "  ")
			Expect(os.WriteFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"), data, 0644)).To(Succeed())
			stdout, _, err := runMcmod(d, "lock", "list", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			occurrences := strings.Count(stdout, "(empty)")
			Expect(occurrences).To(Equal(3))
		})
		It("J08-4: lock list fails with hint when file is missing", func() {
			_, stderr, err := runMcmod(d, "lock", "list", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})

	// ============== J09: lock show with example lock ==============
	Describe("J09: lock show with example workspace", func() {
		It("J09-1: lock show with no key dumps full JSON", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			var l domain.PackLock
			Expect(json.Unmarshal([]byte(stdout), &l)).To(Succeed())
			Expect(l.Loader).To(Equal("neoforge"))
		})
		It("J09-2: lock show with key prints scope/source", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge", "create")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("scope: shared"))
			Expect(stdout).To(ContainSubstring("source:"))
			Expect(stdout).To(ContainSubstring("type: curseforge"))
		})
		It("J09-3: lock show with missing key fails", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge", "no-such-key")
			Expect(err).To(HaveOccurred())
		})
		It("J09-4: lock show with 0 or 1 arg fails", func() {
			_, _, err := runMcmod(d, "lock", "show")
			Expect(err).To(HaveOccurred())
			_, _, err = runMcmod(d, "lock", "show", "1.21.1")
			Expect(err).To(HaveOccurred())
		})
		It("J09-5: lock show with missing lock file fails", func() {
			_, _, err := runMcmod(d, "lock", "show", "9.9.9", "neoforge", "k")
			Expect(err).To(HaveOccurred())
		})
	})

	// ============== J10: lock add/update/delete round-trip ==============
	Describe("J10: lock add/update/delete with example workspace", func() {
		It("J10-1: lock add curseforge writes mod-id and file-id", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge", "cfkey",
				"--name", "CF", "--version", "1.0", "--scope", "shared",
				"--source", "curseforge",
				"--mod-id", "111111", "--file-id", "222222", "--file-name", "cf.jar")
			Expect(err).NotTo(HaveOccurred())
			data, _ := os.ReadFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			var l domain.PackLock
			Expect(json.Unmarshal(data, &l)).To(Succeed())
			Expect(l.Mods["cfkey"].Source.ModID).To(Equal(111111))
			Expect(l.Mods["cfkey"].Source.FileID).To(Equal(222222))
		})
		It("J10-2: lock add github-release writes repo/tag/assetName", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge", "ghkey",
				"--name", "GH", "--version", "1.0", "--scope", "shared",
				"--source", "github-release",
				"--repo", "o/r", "--tag", "v1.0", "--asset-name", "gh.jar", "--file-name", "gh.jar")
			Expect(err).NotTo(HaveOccurred())
			data, _ := os.ReadFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			var l domain.PackLock
			Expect(json.Unmarshal(data, &l)).To(Succeed())
			Expect(l.Mods["ghkey"].Source.Repo).To(Equal("o/r"))
			Expect(l.Mods["ghkey"].Source.Tag).To(Equal("v1.0"))
			Expect(l.Mods["ghkey"].Source.AssetName).To(Equal("gh.jar"))
		})
		It("J10-3: lock add local writes path and fileName", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge", "lckey",
				"--name", "LC", "--version", "1.0", "--scope", "shared",
				"--source", "local",
				"--path", "./mods/lc.jar", "--file-name", "lc.jar")
			Expect(err).NotTo(HaveOccurred())
			data, _ := os.ReadFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			var l domain.PackLock
			Expect(json.Unmarshal(data, &l)).To(Succeed())
			Expect(l.Mods["lckey"].Source.Path).To(Equal("./mods/lc.jar"))
			Expect(l.Mods["lckey"].Source.FileName).To(Equal("lc.jar"))
		})
		It("J10-4: lock add duplicate key fails", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge", "create",
				"--source", "curseforge", "--mod-id", "1", "--file-id", "1", "--file-name", "x.jar")
			Expect(err).To(HaveOccurred())
		})
		It("J10-5: lock add creates new lock file when missing", func() {
			Expect(os.WriteFile(filepath.Join(d, "packspec.json"), []byte(`{
				"packName": "p", "packVersion": "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName": ["neoforge:21.1.219"]
			}`), 0644)).To(Succeed())
			_, _, err := runMcmod(d, "lock", "add", "1.21.1", "neoforge", "first",
				"--source", "local", "--path", "./a.jar", "--file-name", "a.jar")
			Expect(err).NotTo(HaveOccurred())
			data, _ := os.ReadFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			var l domain.PackLock
			Expect(json.Unmarshal(data, &l)).To(Succeed())
			Expect(l.Mods).To(HaveLen(1))
		})
		It("J10-6: lock update single key changes version", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "update", "1.21.1", "neoforge", "create", "--version", "2.0")
			Expect(err).NotTo(HaveOccurred())
			data, _ := os.ReadFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			var l domain.PackLock
			Expect(json.Unmarshal(data, &l)).To(Succeed())
			Expect(l.Mods["create"].Version).To(Equal("2.0"))
		})
		It("J10-7: lock update single key without --version still saves", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "update", "1.21.1", "neoforge", "create")
			Expect(err).NotTo(HaveOccurred())
		})
		It("J10-8: lock update missing key fails", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "update", "1.21.1", "neoforge", "no-such")
			Expect(err).To(HaveOccurred())
		})
		It("J10-9: lock delete single key keeps other entries", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "delete", "1.21.1", "neoforge", "create")
			Expect(err).NotTo(HaveOccurred())
			data, _ := os.ReadFile(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			var l domain.PackLock
			Expect(json.Unmarshal(data, &l)).To(Succeed())
			Expect(l.Mods).NotTo(HaveKey("create"))
			Expect(l.Mods).To(HaveKey("jei"))
		})
		It("J10-10: lock delete missing key fails", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "delete", "1.21.1", "neoforge", "nope")
			Expect(err).To(HaveOccurred())
		})
		It("J10-11: lock delete with no key removes lock file", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "delete", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json"))
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
		It("J10-12: lock delete with no key/no loader removes all loaders", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "delete", "1.21.1")
			Expect(err).NotTo(HaveOccurred())
			entries, _ := os.ReadDir(filepath.Join(d, "locks", "dependencies"))
			Expect(entries).To(BeEmpty())
		})
		It("J10-13: lock add with 0/1/2 args fails", func() {
			_, _, err := runMcmod(d, "lock", "add")
			Expect(err).To(HaveOccurred())
			_, _, err = runMcmod(d, "lock", "add", "1.21.1")
			Expect(err).To(HaveOccurred())
			_, _, err = runMcmod(d, "lock", "add", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("J10-14: lock update with 0/1/2 args fails", func() {
			_, _, err := runMcmod(d, "lock", "update")
			Expect(err).To(HaveOccurred())
			_, _, err = runMcmod(d, "lock", "update", "1.21.1")
			Expect(err).To(HaveOccurred())
			_, _, err = runMcmod(d, "lock", "update", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("J10-15: lock update single key without lock file fails", func() {
			_, _, err := runMcmod(d, "lock", "update", "9.9.9", "neoforge", "k")
			Expect(err).To(HaveOccurred())
		})
	})

	// ============== J11: lock release set/list/show/delete ==============
	Describe("J11: lock release round-trip", func() {
		It("J11-1: release set writes GitHub metadata to release index", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.2.0", "--repo", "o/r", "--tag", "v0.2.0",
				"--name", "R0.2", "--body", "Body")
			Expect(err).NotTo(HaveOccurred())
			data, _ := os.ReadFile(filepath.Join(d, "locks", "releases", "1.21.1.json"))
			var ri domain.ReleaseIndex
			Expect(json.Unmarshal(data, &ri)).To(Succeed())
			Expect(ri.Releases).To(HaveLen(2))
			rec := ri.FindRelease("0.2.0")
			Expect(rec).NotTo(BeNil())
			Expect(rec.GitHub.Repo).To(Equal("o/r"))
			Expect(rec.GitHub.Name).To(Equal("R0.2"))
			Expect(rec.GitHub.Body).To(Equal("Body"))
		})
		It("J11-2: release list prints both versions", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.2.0", "--repo", "o/r", "--tag", "v0.2.0")
			Expect(err).NotTo(HaveOccurred())
			stdout, _, err := runMcmod(d, "lock", "release", "list", "1.21.1")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("0.1.0"))
			Expect(stdout).To(ContainSubstring("0.2.0"))
		})
		It("J11-3: release show with valid version prints full record", func() {
			copyExampleSeed(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "lock", "release", "show", "1.21.1", "0.1.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("github"))
			Expect(stdout).To(ContainSubstring("orangeboyChen/mc-smoke"))
		})
		It("J11-4: release show with missing version fails", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "release", "show", "1.21.1", "9.9.9")
			Expect(err).To(HaveOccurred())
		})
		It("J11-5: release delete full record removes the index file", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "release", "delete", "1.21.1", "0.1.0")
			Expect(err).NotTo(HaveOccurred())
			// Per spec 7.4.8 the index file is removed when the last
			// release is deleted; no empty file should remain behind.
			releasePath := filepath.Join(d, "locks", "releases", "1.21.1.json")
			_, statErr := os.Stat(releasePath)
			Expect(os.IsNotExist(statErr)).To(BeTrue())
		})
		It("J11-6: release delete with 0 args fails", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "release", "delete")
			Expect(err).To(HaveOccurred())
		})
		It("J11-7: release delete with 1 arg fails", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "release", "delete", "1.21.1")
			Expect(err).To(HaveOccurred())
		})
		It("J11-8: release delete single client artifact keeps server", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "release", "delete", "1.21.1", "0.1.0", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			data, _ := os.ReadFile(filepath.Join(d, "locks", "releases", "1.21.1.json"))
			var ri domain.ReleaseIndex
			Expect(json.Unmarshal(data, &ri)).To(Succeed())
			Expect(ri.Releases[0].Artifact["neoforge"].Client).To(BeEmpty())
			Expect(ri.Releases[0].Artifact["neoforge"].Server).NotTo(BeEmpty())
		})
		It("J11-9: release delete single server artifact keeps client", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "release", "delete", "1.21.1", "0.1.0", "neoforge", "--target", "server")
			Expect(err).NotTo(HaveOccurred())
			data, _ := os.ReadFile(filepath.Join(d, "locks", "releases", "1.21.1.json"))
			var ri domain.ReleaseIndex
			Expect(json.Unmarshal(data, &ri)).To(Succeed())
			Expect(ri.Releases[0].Artifact["neoforge"].Server).To(BeEmpty())
			Expect(ri.Releases[0].Artifact["neoforge"].Client).NotTo(BeEmpty())
		})
		It("J11-10: release delete with --target both clears both", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "release", "delete", "1.21.1", "0.1.0", "neoforge", "--target", "both")
			Expect(err).NotTo(HaveOccurred())
			data, _ := os.ReadFile(filepath.Join(d, "locks", "releases", "1.21.1.json"))
			var ri domain.ReleaseIndex
			Expect(json.Unmarshal(data, &ri)).To(Succeed())
			Expect(ri.Releases[0].Artifact["neoforge"].Client).To(BeEmpty())
			Expect(ri.Releases[0].Artifact["neoforge"].Server).To(BeEmpty())
		})
		It("J11-11: release list with no index fails", func() {
			_, _, err := runMcmod(d, "lock", "release", "list", "1.21.1")
			Expect(err).To(HaveOccurred())
		})
		It("J11-12: release set with --draft and --prerelease writes flags", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.3.0", "--repo", "o/r", "--tag", "v0.3.0",
				"--draft", "--prerelease")
			Expect(err).NotTo(HaveOccurred())
			data, _ := os.ReadFile(filepath.Join(d, "locks", "releases", "1.21.1.json"))
			var ri domain.ReleaseIndex
			Expect(json.Unmarshal(data, &ri)).To(Succeed())
			rec := ri.FindRelease("0.3.0")
			Expect(rec).NotTo(BeNil())
			Expect(rec.GitHub.Draft).To(BeTrue())
			Expect(rec.GitHub.Pre).To(BeTrue())
		})
		It("J11-13: release set with --artifact-client/--artifact-server writes paths", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.4.0", "--repo", "o/r", "--tag", "v0.4.0",
				"--artifact-client", "client-x.zip", "--artifact-server", "server-x.zip")
			Expect(err).NotTo(HaveOccurred())
			data, _ := os.ReadFile(filepath.Join(d, "locks", "releases", "1.21.1.json"))
			var ri domain.ReleaseIndex
			Expect(json.Unmarshal(data, &ri)).To(Succeed())
			rec := ri.FindRelease("0.4.0")
			Expect(rec).NotTo(BeNil())
			Expect(rec.Artifact["neoforge"].Client).To(Equal("client-x.zip"))
			Expect(rec.Artifact["neoforge"].Server).To(Equal("server-x.zip"))
		})
		It("J11-14: release set without --repo fails", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.5.0", "--tag", "v0.5.0")
			Expect(err).To(HaveOccurred())
		})
		It("J11-15: release set without --tag fails", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.6.0", "--repo", "o/r")
			Expect(err).To(HaveOccurred())
		})
		It("J11-16: release set without --version fails", func() {
			copyExampleSeed(GinkgoT(), d)
			_, _, err := runMcmod(d, "lock", "release", "set", "1.21.1", "neoforge",
				"--repo", "o/r", "--tag", "v1")
			Expect(err).To(HaveOccurred())
		})
		It("J11-17: release show with 0 or 1 arg fails", func() {
			_, _, err := runMcmod(d, "lock", "release", "show")
			Expect(err).To(HaveOccurred())
			_, _, err = runMcmod(d, "lock", "release", "show", "1.21.1")
			Expect(err).To(HaveOccurred())
		})
	})
})
