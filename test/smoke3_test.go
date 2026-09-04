// File: test/smoke3_test.go
// Created: 2026-06-20
// Description: Part 3 of smoke tests - structure verification, cache, fixtures.
package test

import (
	"archive/zip"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/orangeboyChen/mcmod-cli/internal/cache"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
	"github.com/orangeboyChen/mcmod-cli/internal/metadata"
)

var _ = Describe("Smoke: structure and fixtures", func() {
	var d string

	BeforeEach(func() {
		d = GinkgoT().TempDir()
	})

	// S20: Release index结构校验
	Describe("S20: release index structure", func() {
		It("has correct type/fields", func() {
			ri := domain.ReleaseIndex{Type: "package", PackName: "mypack", MinecraftVersion: "1.21.1",
				Releases: []domain.ReleaseRecord{{
					Version: "0.1.0",
					Type:    "github-release",
					GitHub:  domain.ReleaseGitHub{Repo: "o/r", Tag: "v0.1.0"},
					Artifact: map[string]domain.ReleaseArtifactSet{
						"neoforge": {Client: "releases/v0.1.0/client.zip", Server: "releases/v0.1.0/server.zip"},
					},
				}},
			}
			Expect(ri.Type).To(Equal("package"))
			Expect(ri.Releases[0].Artifact["neoforge"].Client).To(ContainSubstring("client.zip"))
			Expect(ri.Releases[0].Artifact["neoforge"].Server).To(ContainSubstring("server.zip"))
		})
	})

	// S21: Lock文件结构校验
	Describe("S21: lock file structure", func() {
		It("has required fields", func() {
			lock := domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{
					"create": {
						Name: "Create", Version: "6.0", Scope: "shared",
						Identity: &domain.Identity{Source: "curseforge:328085", Internal: "neoforge:create", Confidence: "metadata"},
						Source:   domain.LockedSource{Type: "curseforge", ModID: 328085, FileID: 5812340, FileName: "create-1.21.1-neoforge.jar"},
					},
				},
			}
			Expect(domain.ValidateLock(lock)).To(Succeed())
			m := lock.Mods["create"]
			Expect(m.Source.ModID).To(Equal(328085))
			Expect(m.Source.FileName).To(Equal("create-1.21.1-neoforge.jar"))
			Expect(m.Identity.Confidence).To(Equal("metadata"))
		})
		It("rejects query in lock source", func() {
			lock := domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{"m": {Name: "M",
					Source: domain.LockedSource{Type: "curseforge", ModID: 1, FileID: 2, FileName: "m.jar"}}}}
			Expect(domain.ValidateLock(lock)).To(Succeed())
		})
	})

	// S22: 本地fixture jar + jar metadata解析
	Describe("S22: fixture jar metadata", func() {
		It("reads neoforge metadata from fixture jar", func() {
			jar := createFixtureJar(d, "neoforge-mod.jar", "neoforge",
				"modid=\"neotest\"\nversion=\"3.0.0\"\n")
			info, err := metadata.ReadNeoForgeMetadata(jar)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.ModID).To(Equal("neotest"))
			Expect(info.Version).To(Equal("3.0.0"))
		})
		It("reads fabric metadata from fixture jar", func() {
			jar := createFixtureJar(d, "fabric-mod.jar", "fabric",
				`{"id":"fabtest","version":"4.0.0","depends":{"fabric-api":"*"}}`)
			info, err := metadata.ReadFabricMetadata(jar)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.ModID).To(Equal("fabtest"))
			Expect(info.Version).To(Equal("4.0.0"))
			Expect(info.Dependencies).To(HaveLen(1))
		})
		It("auto-detects neoforge jar", func() {
			jar := createFixtureJar(d, "auto.jar", "neoforge",
				"modid=\"auto_mod\"\nversion=\"1.0\"\n")
			info, err := metadata.ReadJarMetadata(jar)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.ModID).To(Equal("auto_mod"))
		})
		It("reads class path from fixture jar", func() {
			jar := createFixtureJar(d, "class-check.jar", "fabric",
				`{"id":"cc","version":"1.0"}`)
			r, err := zip.OpenReader(jar)
			Expect(err).NotTo(HaveOccurred())
			defer r.Close()
			var foundClass bool
			for _, f := range r.File {
				if f.Name == "com/example/Foo.class" {
					foundClass = true
					break
				}
			}
			Expect(foundClass).To(BeTrue())
		})
	})

	// S23: 缓存hit/miss + 原子移动
	Describe("S23: cache operations", func() {
		It("cache miss properly detected", func() {
			ok, _, err := cache.CheckCurseForge("999", "888", "missing.jar")
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
		})
		It("cache hit detected after file created", func() {
			cp := cache.CurseForgePath("1", "2", "mod.jar")
			os.MkdirAll(filepath.Dir(cp), 0755)
			os.WriteFile(cp, []byte("cached data"), 0644)
			ok, size, err := cache.CheckCurseForge("1", "2", "mod.jar")
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(size).To(BeNumerically(">", 0))
		})
		It("atomic move works", func() {
			src := filepath.Join(d, "src.tmp")
			dst := filepath.Join(d, "dst.txt")
			os.WriteFile(src, []byte("data"), 0644)
			err := cache.AtomicMove(src, dst)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(dst)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(src)
			Expect(err).To(HaveOccurred())
		})
		It("SHA256 computation", func() {
			p := filepath.Join(d, "hash-test.txt")
			os.WriteFile(p, []byte("test data"), 0644)
			hash, err := cache.ComputeSHA256(p)
			Expect(err).NotTo(HaveOccurred())
			Expect(hash).To(HaveLen(64))
		})
	})

	// S24: 重复class冲突检测
	Describe("S24: class conflict detection", func() {
		It("two jars with same class path detected", func() {
			j1 := createFixtureJar(d, "mod-a.jar", "fabric", `{"id":"mod_a","version":"1.0"}`)
			j2 := createFixtureJar(d, "mod-b.jar", "fabric", `{"id":"mod_b","version":"1.0"}`)

			paths1 := extractClassPaths(j1)
			paths2 := extractClassPaths(j2)
			conflicts := intersectStrings(paths1, paths2)
			Expect(conflicts).To(HaveLen(1))
			Expect(conflicts[0]).To(Equal("com/example/Foo.class"))
		})
	})
})

func extractClassPaths(jarPath string) []string {
	r, err := zip.OpenReader(jarPath)
	Expect(err).NotTo(HaveOccurred())
	defer r.Close()
	var paths []string
	for _, f := range r.File {
		if len(f.Name) > 6 && f.Name[len(f.Name)-6:] == ".class" {
			paths = append(paths, f.Name)
		}
	}
	return paths
}

func intersectStrings(a, b []string) []string {
	set := make(map[string]bool)
	for _, s := range a {
		set[s] = true
	}
	var result []string
	for _, s := range b {
		if set[s] {
			result = append(result, s)
		}
	}
	return result
}
