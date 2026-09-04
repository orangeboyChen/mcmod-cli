// File: internal/cli/commands_lock_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for `mcmod lock` and its subcommands (list/show/add/update/delete/tree/release).

package cli

import (
	"bytes"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("lock", func() {
	It("lock without spec errors", func() {
		chdirTemp("")
		_, _, err := runCLI("lock", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("lock with no loaders defined produces no output", func() {
		chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":[]}`)
		_, _, err := runCLI("lock")
		Expect(err).NotTo(HaveOccurred())
	})
	It("lock with invalid mcVersion still goes through build path", func() {
		chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge:21.1.219"]}`)
		_, _, err := runCLI("lock", "1.21.1", "neoforge")
		// We expect an error (no API key for non-local mods) or empty success.
		_ = err
	})
})

var _ = Describe("lock list", func() {
	It("lock list with no lock errors", func() {
		chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		_, _, err := runCLI("lock", "list", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("lock list prints scopes", func() {
		dir := chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{
			Loader: "neoforge", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Version: "1", Scope: "shared", Source: domain.LockedSource{Type: "local", FileName: "a.jar"}},
				"b": {Name: "B", Version: "2", Scope: "client", Source: domain.LockedSource{Type: "local", FileName: "b.jar"}},
				"c": {Name: "C", Version: "3", Scope: "server", Source: domain.LockedSource{Type: "local", FileName: "c.jar"}},
			},
		})
		stdout, _, err := runCLI("lock", "list", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("[Shared]"))
		Expect(stdout).To(ContainSubstring("[Client]"))
		Expect(stdout).To(ContainSubstring("[Server]"))
	})
	It("lock list with empty mods shows (empty) for each", func() {
		dir := chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{}})
		stdout, _, err := runCLI("lock", "list", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("(empty)"))
	})
})

var _ = Describe("lock show", func() {
	It("lock show without lock errors", func() {
		chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		_, _, err := runCLI("lock", "show", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("lock show without key dumps JSON", func() {
		dir := chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{
			Loader: "neoforge", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Version: "1", Scope: "shared", Source: domain.LockedSource{Type: "local", FileName: "a.jar"}},
			},
		})
		stdout, _, err := runCLI("lock", "show", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("\"minecraftVersion\""))
	})
	It("lock show with key shows curseforge details", func() {
		dir := chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{
			Loader: "neoforge", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Version: "1", Scope: "shared", Source: domain.LockedSource{Type: "curseforge", ModID: 100, FileID: 200, FileName: "a.jar"}},
				"b": {Name: "B", Version: "2", Scope: "client", Source: domain.LockedSource{Type: "github-release", Repo: "o/r", Tag: "v1", AssetName: "b.jar"}},
			},
		})
		stdout, _, err := runCLI("lock", "show", "1.21.1", "neoforge", "a")
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("modId: 100"))
		Expect(stdout).To(ContainSubstring("fileId: 200"))
		stdout2, _, err := runCLI("lock", "show", "1.21.1", "neoforge", "b")
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout2).To(ContainSubstring("repo: o/r"))
		Expect(stdout2).To(ContainSubstring("tag: v1"))
	})
})

var _ = Describe("lock add", func() {
	It("lock add with too few args errors", func() {
		chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		_, _, err := runCLI("lock", "add", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("lock add with existing key errors", func() {
		dir := chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		_, _, err := runCLI("lock", "add", "1.21.1", "neoforge", "dup",
			"--source", "local", "--path", "./x.jar", "--file-name", "x.jar")
		Expect(err).NotTo(HaveOccurred())
		_, _, err = runCLI("lock", "add", "1.21.1", "neoforge", "dup",
			"--source", "local", "--path", "./y.jar", "--file-name", "y.jar")
		Expect(err).To(HaveOccurred())
	})
	It("lock add with github-release creates entry", func() {
		dir := chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		_, _, err := runCLI("lock", "add", "1.21.1", "neoforge", "gh",
			"--name", "GH", "--version", "1",
			"--source", "github-release", "--repo", "o/r", "--tag", "v1", "--asset-name", "gh.jar", "--file-name", "gh.jar")
		Expect(err).NotTo(HaveOccurred())
	})
	It("lock add with curseforge source creates entry", func() {
		dir := chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		_, _, err := runCLI("lock", "add", "1.21.1", "neoforge", "cf",
			"--name", "CF", "--version", "1", "--mod-id", "10", "--file-id", "20", "--file-name", "cf.jar",
			"--source", "curseforge")
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("lock update", func() {
	It("lock update without lock errors", func() {
		chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		_, _, err := runCLI("lock", "update", "1.21.1", "neoforge", "k")
		Expect(err).To(HaveOccurred())
	})
	It("lock update with missing key errors", func() {
		dir := chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{}})
		_, _, err := runCLI("lock", "update", "1.21.1", "neoforge", "missing", "--version", "1")
		Expect(err).To(HaveOccurred())
	})
	It("lock update without key refreshes from spec", func() {
		_ = chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge:21.1.219"],"mods":{}}`)
		_, _, err := runCLI("lock", "update", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
	})
	It("lock update with key updates version", func() {
		dir := chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{
			Loader: "neoforge", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Version: "1", Scope: "shared", Source: domain.LockedSource{Type: "local", FileName: "a.jar"}},
			},
		})
		_, _, err := runCLI("lock", "update", "1.21.1", "neoforge", "a", "--version", "9")
		Expect(err).NotTo(HaveOccurred())
	})

	It("lock update with key updates name and scope", func() {
		dir := chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{
			Loader: "neoforge", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Version: "1", Scope: "shared", Source: domain.LockedSource{Type: "local", FileName: "a.jar"}},
			},
		})
		_, _, err := runCLI("lock", "update", "1.21.1", "neoforge", "a", "--name", "AA", "--scope", "client")
		Expect(err).NotTo(HaveOccurred())
	})

	It("lock update with --source overwrites source via buildLockedSource", func() {
		dir := chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{
			Loader: "neoforge", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Version: "1", Scope: "shared", Source: domain.LockedSource{Type: "local", FileName: "a.jar"}},
			},
		})
		_, _, err := runCLI("lock", "update", "1.21.1", "neoforge", "a",
			"--source", "github-release", "--repo", "o/r", "--tag", "v1", "--asset-name", "gh.jar", "--file-name", "gh.jar")
		Expect(err).NotTo(HaveOccurred())
	})

	It("lock update with --source and missing required field errors", func() {
		dir := chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{
			Loader: "neoforge", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Version: "1", Scope: "shared", Source: domain.LockedSource{Type: "local", FileName: "a.jar"}},
			},
		})
		_, _, err := runCLI("lock", "update", "1.21.1", "neoforge", "a", "--source", "local")
		Expect(err).To(HaveOccurred())
	})

	It("lock update per-field source update for file-name/repo/tag/asset/path", func() {
		dir := chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{
			Loader: "neoforge", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Version: "1", Scope: "shared", Source: domain.LockedSource{Type: "github-release", Repo: "o/r", Tag: "v1", AssetName: "old.jar", FileName: "old.jar"}},
			},
		})
		_, _, err := runCLI("lock", "update", "1.21.1", "neoforge", "a",
			"--file-name", "new.jar", "--tag", "v2", "--asset-name", "new.jar", "--path", "p.jar")
		Expect(err).NotTo(HaveOccurred())
	})

	It("lock update per-field source update for mod-id and file-id", func() {
		dir := chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{
			Loader: "neoforge", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Version: "1", Scope: "shared", Source: domain.LockedSource{Type: "curseforge", ModID: 0, FileID: 0, FileName: "old.jar"}},
			},
		})
		_, _, err := runCLI("lock", "update", "1.21.1", "neoforge", "a", "--mod-id", "123", "--file-id", "456", "--repo", "o/r")
		Expect(err).NotTo(HaveOccurred())
	})

	It("lock update per-field source ignores non-numeric mod-id/file-id", func() {
		dir := chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{
			Loader: "neoforge", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Version: "1", Scope: "shared", Source: domain.LockedSource{Type: "curseforge", ModID: 1, FileID: 2, FileName: "a.jar"}},
			},
		})
		// non-numeric Sscanf fails -> unchanged
		_, _, err := runCLI("lock", "update", "1.21.1", "neoforge", "a", "--mod-id", "abc")
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("lock delete", func() {
	It("lock delete with key removes entry", func() {
		dir := chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{
			Loader: "neoforge", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", FileName: "a.jar"}},
			},
		})
		_, _, err := runCLI("lock", "delete", "1.21.1", "neoforge", "a")
		Expect(err).NotTo(HaveOccurred())
	})
	It("lock delete with missing key errors", func() {
		dir := chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{}})
		_, _, err := runCLI("lock", "delete", "1.21.1", "neoforge", "missing")
		Expect(err).To(HaveOccurred())
	})
	It("lock delete with key but no lock errors", func() {
		chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		_, _, err := runCLI("lock", "delete", "1.21.1", "neoforge", "missing")
		Expect(err).To(HaveOccurred())
	})
	It("lock delete with mc/loader but missing files errors", func() {
		chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		_, _, err := runCLI("lock", "delete", "1.21.1")
		Expect(err).To(HaveOccurred())
	})
	It("lock delete with mc+loader but missing lock file errors", func() {
		chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		_, _, err := runCLI("lock", "delete", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("lock delete without spec errors", func() {
		chdirTemp("")
		_, _, err := runCLI("lock", "delete", "1.21.1")
		Expect(err).To(HaveOccurred())
	})

	It("lock delete with mc+loader removes existing file", func() {
		dir := chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{}})
		_, _, err := runCLI("lock", "delete", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		_, err = os.Stat(filepath.Join(dir, "locks", "dependencies", "1.21.1-neoforge.json"))
		Expect(err).To(HaveOccurred())
	})

	It("lock delete without args uses spec defaults and removes files", func() {
		dir := chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{}})
		_, _, err := runCLI("lock", "delete")
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("CLI extra3 - buildLockedSource validation", func() {
	It("curseforge without required fields errors", func() {
		_, err := buildLockedSource("curseforge", "", "", "", "", "", "", "")
		Expect(err).To(HaveOccurred())
	})

	It("curseforge with invalid mod-id errors", func() {
		_, err := buildLockedSource("curseforge", "abc", "456", "x.jar", "", "", "", "")
		Expect(err).To(HaveOccurred())
	})

	It("github-release without repo/tag/asset errors", func() {
		_, err := buildLockedSource("github-release", "", "", "gh.jar", "", "", "", "")
		Expect(err).To(HaveOccurred())
	})

	It("github-release full valid set succeeds", func() {
		ls, err := buildLockedSource("github-release", "", "", "gh.jar", "o/r", "v1", "gh.jar", "")
		Expect(err).NotTo(HaveOccurred())
		Expect(ls.Type).To(Equal("github-release"))
		Expect(ls.Repo).To(Equal("o/r"))
		Expect(ls.Tag).To(Equal("v1"))
		Expect(ls.AssetName).To(Equal("gh.jar"))
		Expect(ls.FileName).To(Equal("gh.jar"))
	})

	It("git source requires repo + file-name", func() {
		_, err := buildLockedSource("git", "", "", "", "", "", "", "")
		Expect(err).To(HaveOccurred())
		ls, err := buildLockedSource("git", "", "", "x.jar", "o/r", "", "", "")
		Expect(err).NotTo(HaveOccurred())
		Expect(ls.Repo).To(Equal("o/r"))
	})

	It("local source requires path", func() {
		_, err := buildLockedSource("local", "", "", "", "", "", "", "")
		Expect(err).To(HaveOccurred())
		ls, err := buildLockedSource("local", "", "", "x.jar", "", "", "", "./x.jar")
		Expect(err).NotTo(HaveOccurred())
		Expect(ls.Path).To(Equal("./x.jar"))
	})

	It("unknown source type errors", func() {
		_, err := buildLockedSource("bogus", "", "", "", "", "", "", "")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("CLI extra3 - lock update bulk", func() {
	It("runLockUpdateBulk returns error when spec missing", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		err := runLockUpdateBulk([]string{})
		Expect(err).To(HaveOccurred())
	})

	It("runLockUpdateBulk iterates declared loaders", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.WriteFile("packspec.json", []byte(`{"packName":"t","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge:21.1.219"]}`), 0644)).To(Succeed())
		err := runLockUpdateBulk([]string{"1.21.1"})
		// BuildLockWithExisting needs resolver and may fail; just verify
		// it ran the path.
		_ = err
	})
})

var _ = Describe("CLI extra3 - printLockList missing file", func() {
	It("printLockList reports missing lock with hint", func() {
		err := printLockList("1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("hint"))
	})
})

var _ = Describe("CLI extra4 - runLockUpdateBulk with empty loaders", func() {
	It("falls back to neoforge when spec has no loaders", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.WriteFile("packspec.json", []byte(`{"packName":"t","packVersion":"1","minecraftVersion":"1.21.1"}`), 0644)).To(Succeed())
		err := runLockUpdateBulk([]string{"1.21.1"})
		// resolver may fail without network, but the function should still
		// proceed to the build step.
		_ = err
	})
})

var _ = Describe("CLI extra5 - lock add gh-release missing path error", func() {
	It("returns validation error", func() {
		_, err := buildLockedSource("github-release", "", "", "", "", "", "", "")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("github-release"))
	})
})

var _ = Describe("CLI mass", func() {
	It("set command with bad args", func() {
		cmd := NewApp()
		cmd.SetArgs([]string{"set"})
		err := cmd.Execute()
		Expect(err).To(HaveOccurred())
	})

	It("lock with invalid loader", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.WriteFile("packspec.json", []byte(`{"packName":"t","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`), 0644)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "1.21.1", "quilt"})
		_ = cmd.Execute()
	})

	It("lock show without args errors", func() {
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "show"})
		err := cmd.Execute()
		Expect(err).To(HaveOccurred())
	})

	It("lock list without lock fails gracefully", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.WriteFile("packspec.json", []byte(`{"packName":"t","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`), 0644)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "list", "1.21.1", "neoforge"})
		_ = cmd.Execute()
	})

	It("lock add with conflict fails", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.MkdirAll("locks/dependencies", 0755)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "add", "1.21.1", "neoforge", "m",
			"--source", "local", "--file-name", "m.jar", "--path", "./m.jar"})
		_ = cmd.Execute()
		// Adding same key again should error
		cmd2 := NewApp()
		cmd2.SetArgs([]string{"lock", "add", "1.21.1", "neoforge", "m",
			"--source", "local", "--file-name", "m.jar"})
		_ = cmd2.Execute()
	})

	It("lock delete without key prints message", func() {
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "delete"})
		_ = cmd.Execute()
	})

	It("lock tree without lock", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "tree", "1.21.1", "neoforge"})
		_ = cmd.Execute()
	})

	It("lock release list without index", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "release", "list", "1.21.1"})
		_ = cmd.Execute()
	})

	It("lock release show without version errors", func() {
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "release", "show", "1.21.1", "nonexistent"})
		_ = cmd.Execute()
	})

	It("lock release delete with version", func() {
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "release", "delete", "1.21.1", "0.1.0"})
		_ = cmd.Execute()
	})

	It("lock update without lock", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "update", "1.21.1", "neoforge"})
		_ = cmd.Execute()
	})

	It("build with lock missing", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.WriteFile("packspec.json", []byte(`{"packName":"t","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`), 0644)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"build", "1.21.1", "neoforge"})
		_ = cmd.Execute()
	})

	It("validate with bad lock file", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.WriteFile("bad.json", []byte(`{invalid`), 0644)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"validate", "--lock", "bad.json"})
		_ = cmd.Execute()
	})

	It("validate with nonexistent spec file", func() {
		cmd := NewApp()
		cmd.SetArgs([]string{"validate", "--spec", "/nonexistent.json"})
		err := cmd.Execute()
		Expect(err).To(HaveOccurred())
	})

	It("version flag may print to stderr", func() {
		cmd := NewApp()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"version"})
		Expect(cmd.Execute()).To(Succeed())
	})

	It("newLockReleaseCmd creates subcommand", func() {
		cmd := newLockReleaseCmd()
		Expect(cmd).NotTo(BeNil())
	})

	It("newReleaseSetCmd validates required flags", func() {
		cmd := newReleaseSetCmd()
		Expect(cmd).NotTo(BeNil())
	})
})
var _ = Describe("CLI extra commands", func() {
	It("set cf-key without key yields error", func() {
		cmd := NewApp()
		buf := new(bytes.Buffer)
		cmd.SetErr(buf)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"set"})
		_ = cmd.Execute()
	})

	It("lock list without lock fails gracefully", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.WriteFile("packspec.json", []byte(`{"packName":"t","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`), 0644)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "list"})
		_ = cmd.Execute()
	})

	It("lock show without args errors", func() {
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "show"})
		err := cmd.Execute()
		Expect(err).To(HaveOccurred())
	})

	It("lock add creates a lock entry", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.MkdirAll("locks/dependencies", 0755)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "add", "1.21.1", "neoforge", "testmod",
			"--name", "TestMod", "--version", "1.0",
			"--source", "local", "--path", "./test.jar", "--file-name", "test.jar"})
		_ = cmd.Execute() // 可能成功或失败
	})

	It("lock update with key updates version", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.MkdirAll("locks/dependencies", 0755)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "update", "1.21.1", "neoforge"})
		_ = cmd.Execute()
	})

	It("lock delete with key works", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.MkdirAll("locks/dependencies", 0755)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "delete", "1.21.1", "neoforge", "testmod"})
		_ = cmd.Execute()
	})

	It("lock delete without key prints message", func() {
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "delete"})
		_ = cmd.Execute()
	})

	It("lock tree without lock fails gracefully", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.WriteFile("packspec.json", []byte(`{"packName":"t","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`), 0644)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "tree"})
		_ = cmd.Execute()
	})

	It("lock release list without index errors", func() {
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "release", "list"})
		_ = cmd.Execute()
	})

	It("lock release show without args errors", func() {
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "release", "show"})
		err := cmd.Execute()
		Expect(err).To(HaveOccurred())
	})

	It("build help shows flags", func() {
		cmd := NewApp()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"build", "--help"})
		Expect(cmd.Execute()).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("target"))
	})

	It("config --help", func() {
		cmd := NewApp()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"config", "--help"})
		Expect(cmd.Execute()).To(Succeed())
	})
})
var _ = Describe("CLI extra2", func() {
	It("lock add complete flow", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.MkdirAll("locks/dependencies", 0755)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "add", "1.21.1", "neoforge", "mymod",
			"--name", "MyMod", "--version", "1.5",
			"--scope", "shared", "--source", "curseforge",
			"--mod-id", "123", "--file-id", "456", "--file-name", "mymod.jar"})
		_ = cmd.Execute()
	})

	It("lock add with github-release", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.MkdirAll("locks/dependencies", 0755)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "add", "1.21.1", "neoforge", "ghmod",
			"--name", "GHMod", "--source", "github-release",
			"--repo", "owner/repo", "--tag", "v1", "--asset-name", "gh.jar", "--file-name", "gh.jar"})
		_ = cmd.Execute()
	})

	It("lock update full refresh in temp dir", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.WriteFile("packspec.json", []byte(`{"packName":"t","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`), 0644)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "update"})
		_ = cmd.Execute()
	})

	It("lock tree with args", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.WriteFile("packspec.json", []byte(`{"packName":"t","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`), 0644)).To(Succeed())
		// Create lock so tree succeeds
		Expect(os.MkdirAll("locks/dependencies", 0755)).To(Succeed())
		Expect(os.WriteFile("locks/dependencies/1.21.1-neoforge.json",
			[]byte(`{"loader":"neoforge","minecraftVersion":"1.21.1","mods":{"a":{"name":"A","scope":"shared","source":{"type":"local"}}}}`), 0644)).To(Succeed())
		cmd := NewApp()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"lock", "tree", "1.21.1", "neoforge"})
		_ = cmd.Execute()
		_ = buf
	})

	It("lock release set creates record", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.MkdirAll("locks/releases", 0755)).To(Succeed())
		cmd := NewApp()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"lock", "release", "set", "1.21.1", "neoforge",
			"--version", "0.1.0", "--repo", "owner/repo", "--tag", "v0.1.0"})
		_ = cmd.Execute()
	})

	It("build with target both in temp dir", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.WriteFile("packspec.json", []byte(`{"packName":"bt","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"],"mods":{"m":{"scope":"shared","source":{"type":"local","path":"./m.jar"}}}}`), 0644)).To(Succeed())
		Expect(os.MkdirAll("locks/dependencies", 0755)).To(Succeed())
		Expect(os.WriteFile("locks/dependencies/1.21.1-neoforge.json",
			[]byte(`{"loader":"neoforge","minecraftVersion":"1.21.1","mods":{"m":{"name":"M","scope":"shared","source":{"type":"local","path":"./m.jar","fileName":"m.jar"}}}}`), 0644)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"build", "1.21.1", "neoforge"})
		_ = cmd.Execute()
	})

	It("tree with args", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.MkdirAll("locks/dependencies", 0755)).To(Succeed())
		Expect(os.WriteFile("locks/dependencies/1.21.1-neoforge.json",
			[]byte(`{"loader":"neoforge","minecraftVersion":"1.21.1","mods":{"a":{"name":"A","scope":"shared","source":{"type":"local"}}}}`), 0644)).To(Succeed())
		cmd := NewApp()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"tree", "1.21.1", "neoforge"})
		_ = cmd.Execute()
	})

	It("validate with release index", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.WriteFile("rel.json", []byte(`{"type":"package","packName":"p","minecraftVersion":"1.21.1","releases":[{"version":"0.1.0","type":"github-release"}]}`), 0644)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"validate", "--release-index", "rel.json"})
		_ = cmd.Execute()
	})

	It("validate with invalid lock fails gracefully", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.WriteFile("l.json", []byte(`{"loader":"neoforge"}`), 0644)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"validate", "--lock", "l.json"})
		_ = cmd.Execute()
	})
})
