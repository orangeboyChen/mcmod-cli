// File: test/e2e/integration4_test.go
// Created: 2026-06-20
// Description: End-to-end build tests that inspect the produced zip
// artifacts in detail — file contents, scope separation, and override
// directories.

package test

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

// setupBuildableProject creates a project with packspec + lock + jars +
// overrides in dir. Returns the path to the dir.
func setupBuildableProject(d string, mcVersion, loader, loaderVersion string) {
	os.MkdirAll(filepath.Join(d, "mods"), 0755)
	os.MkdirAll(filepath.Join(d, "config"), 0755)
	os.MkdirAll(filepath.Join(d, "defaultconfigs"), 0755)
	os.MkdirAll(filepath.Join(d, "resourcepacks"), 0755)
	for _, name := range []string{"shared.jar", "client.jar", "server.jar"} {
		var buf bytes.Buffer
		w := zip.NewWriter(&buf)
		e, _ := w.Create("META-INF/test.txt")
		_, _ = e.Write([]byte("test"))
		_ = w.Close()
		_ = os.WriteFile(filepath.Join(d, "mods", name), buf.Bytes(), 0644)
	}
	os.WriteFile(filepath.Join(d, "config", "common.cfg"), []byte("cfg"), 0644)
	os.WriteFile(filepath.Join(d, "defaultconfigs", "default.toml"), []byte("d"), 0644)
	os.WriteFile(filepath.Join(d, "resourcepacks", "rp.zip"), []byte("rp"), 0644)
	os.WriteFile(filepath.Join(d, "server.properties"), []byte("server-port=25565"), 0644)
	os.WriteFile(filepath.Join(d, "packspec.json"), []byte(`{
		"packName": "buildtest", "serverPackName": "buildtest-server",
		"packVersion": "1.5.0",
		"minecraftVersion": "`+mcVersion+`",
		"loaderName": ["`+loader+`:`+loaderVersion+`"],
		"mods": {
			"shared-mod": {"scope": "shared", "source": {"type": "local", "path": "./mods/shared.jar"}},
			"client-mod": {"scope": "client", "source": {"type": "local", "path": "./mods/client.jar"}},
			"server-mod": {"scope": "server", "source": {"type": "local", "path": "./mods/server.jar"}}
		}
	}`), 0644)
	lock := &domain.PackLock{
		Loader: loader, LoaderVersion: loaderVersion,
		MinecraftVersion: mcVersion,
		Mods: map[string]domain.LockedMod{
			"shared-mod": {Name: "Shared", Scope: "shared", Version: "1.0",
				Source: domain.LockedSource{Type: "local", Path: "./mods/shared.jar", FileName: "shared.jar"}},
			"client-mod": {Name: "Client", Scope: "client", Version: "1.0",
				Source: domain.LockedSource{Type: "local", Path: "./mods/client.jar", FileName: "client.jar"}},
			"server-mod": {Name: "Server", Scope: "server", Version: "1.0",
				Source: domain.LockedSource{Type: "local", Path: "./mods/server.jar", FileName: "server.jar"}},
		},
	}
	writeLockFile(d, mcVersion, loader, lock)
}

// setupCfBuildableProject writes a minimal spec/lock whose mods are all
// curseforge-sourced with modId/fileId set, so `mcmod build --build-type cf`
// has at least one eligible mod to emit in the manifest. Jars are staged in
// the expected .mcmod/cache/curseforge/<modId>/<fileId>/<fileName> tree so
// resolveModJar does not try to download.
func setupCfBuildableProject(d string, mcVersion, loader, loaderVersion string) {
	os.MkdirAll(filepath.Join(d, "config"), 0755)
	os.WriteFile(filepath.Join(d, "config", "common.cfg"), []byte("cfg"), 0644)
	os.WriteFile(filepath.Join(d, "packspec.json"), []byte(`{
		"packName": "buildtest", "packVersion": "1.5.0",
		"minecraftVersion": "`+mcVersion+`",
		"loaderName": ["`+loader+`:`+loaderVersion+`"],
		"mods": {
			"cf-mod": {"scope": "shared", "source": {"type": "curseforge", "modId": 111, "fileId": 222}}
		}
	}`), 0644)
	// Stage the cached jar.
	cached := filepath.Join(d, ".mcmod", "cache", "curseforge", "111", "222", "cf-mod.jar")
	Expect(os.MkdirAll(filepath.Dir(cached), 0755)).To(Succeed())
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	e, _ := w.Create("META-INF/test.txt")
	_, _ = e.Write([]byte("test"))
	_ = w.Close()
	Expect(os.WriteFile(cached, buf.Bytes(), 0644)).To(Succeed())
	lock := &domain.PackLock{
		Loader: loader, LoaderVersion: loaderVersion,
		MinecraftVersion: mcVersion,
		Mods: map[string]domain.LockedMod{
			"cf-mod": {Name: "CfMod", Scope: "shared", Version: "1.0",
				Source: domain.LockedSource{Type: "curseforge", ModID: 111, FileID: 222, FileName: "cf-mod.jar"}},
		},
	}
	writeLockFile(d, mcVersion, loader, lock)
}

// zipEntries returns the list of names in a zip file.
func zipEntries(path string) []string {
	r, err := zip.OpenReader(path)
	Expect(err).NotTo(HaveOccurred(), "open %s", path)
	defer r.Close()
	names := make([]string, 0, len(r.File))
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	return names
}

// zipHasPath returns true if the zip contains an entry with the given name.
func zipHasPath(path, name string) bool {
	for _, n := range zipEntries(path) {
		if n == name {
			return true
		}
	}
	return false
}

// zipContainsAny returns true if the zip contains any entry whose name starts with prefix.
func zipContainsAny(path, prefix string) bool {
	for _, n := range zipEntries(path) {
		if strings.HasPrefix(n, prefix) {
			return true
		}
	}
	return false
}

var _ = Describe("Integration4: build zip content", func() {
	var d string

	BeforeEach(func() {
		d = GinkgoT().TempDir()
	})

	// ============== L01: build target flags ==============
	Describe("L01: build --target flags", func() {
		It("L01-1: build --target both writes both zips", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "both")
			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(d, "releases", "v1.5.0", "buildtest-1.21.1-neoforge-21.1.219-client.zip")).To(BeAnExistingFile())
			Expect(filepath.Join(d, "releases", "v1.5.0", "buildtest-server-1.21.1-neoforge-21.1.219-server.zip")).To(BeAnExistingFile())
		})
		It("L01-2: build --target client writes only client zip", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(d, "releases", "v1.5.0", "buildtest-1.21.1-neoforge-21.1.219-client.zip")).To(BeAnExistingFile())
			Expect(filepath.Join(d, "releases", "v1.5.0", "buildtest-server-1.21.1-neoforge-21.1.219-server.zip")).NotTo(BeAnExistingFile())
		})
		It("L01-3: build --target server writes only server zip", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "server")
			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(d, "releases", "v1.5.0", "buildtest-server-1.21.1-neoforge-21.1.219-server.zip")).To(BeAnExistingFile())
			Expect(filepath.Join(d, "releases", "v1.5.0", "buildtest-1.21.1-neoforge-21.1.219-client.zip")).NotTo(BeAnExistingFile())
		})
		It("L01-4: build without --target defaults to both", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(d, "releases", "v1.5.0", "buildtest-1.21.1-neoforge-21.1.219-client.zip")).To(BeAnExistingFile())
			Expect(filepath.Join(d, "releases", "v1.5.0", "buildtest-server-1.21.1-neoforge-21.1.219-server.zip")).To(BeAnExistingFile())
		})
		It("L01-5: build with --build-type cf is accepted", func() {
			setupCfBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			stdout, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--build-type", "cf", "--target", "client", "--force")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("artifact cf:"))
		})
		It("L01-6: build with --build-type all is accepted", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--build-type", "all", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
		})
		It("L01-7: build --force allows overwriting an existing zip", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).To(HaveOccurred())
			_, _, err = runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client", "--force")
			Expect(err).NotTo(HaveOccurred())
		})
		It("L01-8: build with invalid target fails", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "wrong")
			Expect(err).To(HaveOccurred())
		})
		It("L01-9: build without spec fails with hint", func() {
			_, stderr, err := runMcmod(d, "build", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("L01-10: build without lock fails with hint", func() {
			os.WriteFile(filepath.Join(d, "packspec.json"), []byte(`{
				"packName": "p", "packVersion": "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName": ["neoforge:21.1.219"]
			}`), 0644)
			_, stderr, err := runMcmod(d, "build", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
	})

	// ============== L02: zip content verification ==============
	Describe("L02: zip content verification", func() {
		It("L02-1: client zip contains mods/shared.jar and mods/client.jar", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			zp := filepath.Join(d, "releases", "v1.5.0", "buildtest-1.21.1-neoforge-21.1.219-client.zip")
			Expect(zipHasPath(zp, "mods/shared.jar")).To(BeTrue())
			Expect(zipHasPath(zp, "mods/client.jar")).To(BeTrue())
		})
		It("L02-2: client zip does NOT contain mods/server.jar", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			zp := filepath.Join(d, "releases", "v1.5.0", "buildtest-1.21.1-neoforge-21.1.219-client.zip")
			Expect(zipHasPath(zp, "mods/server.jar")).To(BeFalse())
		})
		It("L02-3: server zip contains mods/shared.jar and mods/server.jar", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "server")
			Expect(err).NotTo(HaveOccurred())
			zp := filepath.Join(d, "releases", "v1.5.0", "buildtest-server-1.21.1-neoforge-21.1.219-server.zip")
			Expect(zipHasPath(zp, "mods/shared.jar")).To(BeTrue())
			Expect(zipHasPath(zp, "mods/server.jar")).To(BeTrue())
		})
		It("L02-4: server zip does NOT contain mods/client.jar", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "server")
			Expect(err).NotTo(HaveOccurred())
			zp := filepath.Join(d, "releases", "v1.5.0", "buildtest-server-1.21.1-neoforge-21.1.219-server.zip")
			Expect(zipHasPath(zp, "mods/client.jar")).To(BeFalse())
		})
		It("L02-5: client zip contains resourcepacks/", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			zp := filepath.Join(d, "releases", "v1.5.0", "buildtest-1.21.1-neoforge-21.1.219-client.zip")
			Expect(zipContainsAny(zp, "resourcepacks/")).To(BeTrue())
		})
		It("L02-6: server zip contains defaultconfigs/", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "server")
			Expect(err).NotTo(HaveOccurred())
			zp := filepath.Join(d, "releases", "v1.5.0", "buildtest-server-1.21.1-neoforge-21.1.219-server.zip")
			Expect(zipContainsAny(zp, "defaultconfigs/")).To(BeTrue())
		})
		It("L02-7: server zip contains server.properties", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "server")
			Expect(err).NotTo(HaveOccurred())
			zp := filepath.Join(d, "releases", "v1.5.0", "buildtest-server-1.21.1-neoforge-21.1.219-server.zip")
			Expect(zipHasPath(zp, "server.properties")).To(BeTrue())
		})
		It("L02-8: both zips contain config/", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "both")
			Expect(err).NotTo(HaveOccurred())
			client := filepath.Join(d, "releases", "v1.5.0", "buildtest-1.21.1-neoforge-21.1.219-client.zip")
			server := filepath.Join(d, "releases", "v1.5.0", "buildtest-server-1.21.1-neoforge-21.1.219-server.zip")
			Expect(zipContainsAny(client, "config/")).To(BeTrue())
			Expect(zipContainsAny(server, "config/")).To(BeTrue())
		})
		It("L02-9: zip filename format uses loaderVersion in middle", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(d, "releases", "v1.5.0", "buildtest-1.21.1-neoforge-21.1.219-client.zip")).To(BeAnExistingFile())
		})
		It("L02-10: zip uses serverPackName for server zip", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "server")
			Expect(err).NotTo(HaveOccurred())
			zp := filepath.Join(d, "releases", "v1.5.0", "buildtest-server-1.21.1-neoforge-21.1.219-server.zip")
			Expect(zp).To(BeAnExistingFile())
		})
		It("L02-11: zip uses packName for client zip", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			zp := filepath.Join(d, "releases", "v1.5.0", "buildtest-1.21.1-neoforge-21.1.219-client.zip")
			Expect(zp).To(BeAnExistingFile())
		})
		It("L02-12: zip filename embeds packVersion in path", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			// packVersion is 1.5.0, so the file should be under releases/v1.5.0/
			entries, _ := os.ReadDir(filepath.Join(d, "releases"))
			Expect(entries).To(HaveLen(1))
			Expect(entries[0].Name()).To(Equal("v1.5.0"))
		})
	})

	// ============== L03: build output messages ==============
	Describe("L03: build output messages", func() {
		It("L03-1: build success prints 'built' on stdout or stderr", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			stdout, stderr, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			combined := stdout + stderr
			Expect(combined).To(ContainSubstring("built"))
		})
		It("L03-2: build with no second build (different target) writes only that one", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = runMcmod(d, "build", "1.21.1", "neoforge", "--target", "server")
			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(d, "releases", "v1.5.0", "buildtest-1.21.1-neoforge-21.1.219-client.zip")).To(BeAnExistingFile())
			Expect(filepath.Join(d, "releases", "v1.5.0", "buildtest-server-1.21.1-neoforge-21.1.219-server.zip")).To(BeAnExistingFile())
		})
		It("L03-3: build 0 args uses default mc/loader", func() {
			setupBuildableProject(d, "1.21.1", "neoforge", "21.1.219")
			_, _, err := runMcmod(d, "build")
			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(d, "releases", "v1.5.0", "buildtest-1.21.1-neoforge-21.1.219-client.zip")).To(BeAnExistingFile())
		})
	})

	// ============== L04: build with example seed (3 mods: create, jei, server-enhanced) ==============
	Describe("L04: build with generated e2e workspace", func() {
		It("L04-1: example seed build --target both creates 2 zips", func() {
			copyExampleWorkspace(GinkgoT(), d)
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "both")
			Expect(err).NotTo(HaveOccurred())
			entries, _ := os.ReadDir(filepath.Join(d, "releases", "v0.1.0"))
			Expect(entries).To(HaveLen(2))
		})
		It("L04-2: example seed build --target client creates 1 zip", func() {
			copyExampleWorkspace(GinkgoT(), d)
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			entries, _ := os.ReadDir(filepath.Join(d, "releases", "v0.1.0"))
			Expect(entries).To(HaveLen(1))
		})
		It("L04-3: example seed build --target server creates 1 zip", func() {
			copyExampleWorkspace(GinkgoT(), d)
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "server")
			Expect(err).NotTo(HaveOccurred())
			entries, _ := os.ReadDir(filepath.Join(d, "releases", "v0.1.0"))
			Expect(entries).To(HaveLen(1))
		})
		It("L04-4: example seed client zip contains create.jar and jei.jar but not server-enhanced.jar", func() {
			copyExampleWorkspace(GinkgoT(), d)
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			entries, _ := os.ReadDir(filepath.Join(d, "releases", "v0.1.0"))
			Expect(entries).To(HaveLen(1))
			zp := filepath.Join(d, "releases", "v0.1.0", entries[0].Name())
			Expect(zipHasPath(zp, "mods/create-1.21.1-neoforge.jar")).To(BeTrue())
			Expect(zipHasPath(zp, "mods/jei-1.21.1-neoforge.jar")).To(BeTrue())
			Expect(zipHasPath(zp, "mods/serverenhancedmod-1.21.1-neoforge.jar")).To(BeFalse())
		})
		It("L04-5: example seed server zip contains create.jar and server-enhanced.jar but not jei.jar", func() {
			copyExampleWorkspace(GinkgoT(), d)
			_, _, err := runMcmod(d, "build", "1.21.1", "neoforge", "--target", "server")
			Expect(err).NotTo(HaveOccurred())
			entries, _ := os.ReadDir(filepath.Join(d, "releases", "v0.1.0"))
			Expect(entries).To(HaveLen(1))
			zp := filepath.Join(d, "releases", "v0.1.0", entries[0].Name())
			Expect(zipHasPath(zp, "mods/create-1.21.1-neoforge.jar")).To(BeTrue())
			Expect(zipHasPath(zp, "mods/serverenhancedmod-1.21.1-neoforge.jar")).To(BeTrue())
			Expect(zipHasPath(zp, "mods/jei-1.21.1-neoforge.jar")).To(BeFalse())
		})
	})
})
