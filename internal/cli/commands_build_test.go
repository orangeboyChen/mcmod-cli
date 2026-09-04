// File: internal/cli/commands_build_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for `mcmod build` subcommand.

package cli

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

func writeCLIJar(path string) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	e, err := w.Create("META-INF/test.txt")
	Expect(err).NotTo(HaveOccurred())
	_, err = e.Write([]byte("test"))
	Expect(err).NotTo(HaveOccurred())
	Expect(w.Close()).To(Succeed())
	Expect(os.WriteFile(path, buf.Bytes(), 0644)).To(Succeed())
}

var _ = Describe("build", func() {
	It("build without spec errors", func() {
		chdirTemp("")
		_, _, err := runCLI("build")
		Expect(err).To(HaveOccurred())
	})
	It("build with missing lock prints hint", func() {
		chdirTemp(`{"packName":"b","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		_, stderr, err := runCLI("build", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
		Expect(stderr).To(ContainSubstring("hint"))
	})
	It("build with invalid target errors", func() {
		dir := chdirTemp(`{"packName":"b","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{
			Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{},
		})
		_, stderr, err := runCLI("build", "1.21.1", "neoforge", "--target", "wrong")
		Expect(err).To(HaveOccurred())
		Expect(stderr).To(ContainSubstring("invalid target"))
	})
	It("build with --build-type=cf produces a CurseForge layout zip", func() {
		dir := chdirTemp(`{"packName":"b","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		// Stage a cached jar at the path resolveModJar expects.
		cached := filepath.Join(dir, ".mcmod", "cache", "curseforge", "100", "200", "mod.jar")
		Expect(os.MkdirAll(filepath.Dir(cached), 0755)).To(Succeed())
		writeCLIJar(cached)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{
			Loader: "neoforge", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "curseforge", ModID: 100, FileID: 200, FileName: "mod.jar"}, Hash: "x"},
			},
		})
		stdout, _, err := runCLI("build", "1.21.1", "neoforge", "--build-type", "cf", "--force")
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("artifact cf:"))
		Expect(stdout).To(ContainSubstring("-cf.zip"))
	})

	It("build with --build-type=cf reports cf mods missing modid and bundles non-cf mods into overrides/mods/", func() {
		dir := chdirTemp(`{"packName":"b","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		// Seed a cached jar for the curseforge mod so resolveModJar succeeds.
		cached := filepath.Join(dir, ".mcmod", "cache", "curseforge", "100", "200", "mod.jar")
		Expect(os.MkdirAll(filepath.Dir(cached), 0755)).To(Succeed())
		writeCLIJar(cached)
		// Seed a cached jar for the github-release mod so its
		// overrides/mods/<key>/<jar> bundle succeeds without network.
		ghCached := filepath.Join(dir, ".mcmod", "cache", "github-release", "o", "r", "v1", "mod.jar")
		Expect(os.MkdirAll(filepath.Dir(ghCached), 0755)).To(Succeed())
		writeCLIJar(ghCached)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{
			Loader: "neoforge", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a":     {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "curseforge", ModID: 100, FileID: 200, FileName: "mod.jar"}, Hash: "x"},
				"gh":    {Name: "GH", Scope: "shared", Source: domain.LockedSource{Type: "github-release", Repo: "o/r", Tag: "v1", AssetName: "mod.jar"}, Hash: "x"},
				"cfbad": {Name: "NoID", Scope: "shared", Source: domain.LockedSource{Type: "curseforge", FileName: "mod.jar"}, Hash: "x"},
			},
		})
		_, stderr, err := runCLI("build", "1.21.1", "neoforge", "--build-type", "cf", "--force")
		Expect(err).NotTo(HaveOccurred())
		// cfbad (curseforge source but no modId/fileId) is reported as
		// omitted from the manifest.
		Expect(stderr).To(ContainSubstring("omitted from manifest"))
		Expect(stderr).To(ContainSubstring("cfbad"))
		// gh is non-cf and is bundled into overrides/mods/.
		Expect(stderr).To(ContainSubstring("bundled"))
		Expect(stderr).To(ContainSubstring("gh"))
	})

	It("build with --build-type=cf errors when no curseforge mods are present", func() {
		dir := chdirTemp(`{"packName":"b","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		jar := filepath.Join(dir, "mod.jar")
		writeCLIJar(jar)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{
			Loader: "neoforge", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", FileName: "mod.jar", Path: "mod.jar"}, Hash: "x"},
			},
		})
		_, stderr, err := runCLI("build", "1.21.1", "neoforge", "--build-type", "cf", "--force")
		Expect(err).To(HaveOccurred())
		Expect(stderr).To(ContainSubstring("error: build"))
		Expect(stderr).To(ContainSubstring("no curseforge-sourced mods"))
	})
	It("build with --build-type=all succeeds", func() {
		dir := chdirTemp(`{"packName":"b","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		jar := filepath.Join(dir, "mod.jar")
		writeCLIJar(jar)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{
			Loader: "neoforge", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", FileName: "mod.jar", Path: "mod.jar"}, Hash: "x"},
			},
		})
		_, _, err := runCLI("build", "1.21.1", "neoforge", "--build-type", "all", "--force")
		Expect(err).NotTo(HaveOccurred())
	})
	It("build with invalid --build-type errors", func() {
		chdirTemp(`{"packName":"b","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		_, stderr, err := runCLI("build", "1.21.1", "neoforge", "--build-type", "bogus")
		Expect(err).To(HaveOccurred())
		Expect(stderr).To(ContainSubstring("invalid --build-type"))
	})
})
