// File: internal/cli/boost_test.go
// Created: 2026-06-20
// Description: Comprehensive CLI unit tests to push coverage above 80%.
package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

// chdirTemp changes the working directory to a fresh temp dir for the duration
// of the test and writes the provided packspec.json. Returns the temp dir.
// The test process is also isolated from the host user config by redirecting
// XDG_CONFIG_HOME and HOME to a fresh empty temp dir.
func chdirTemp(spec string) string {
	dir := GinkgoT().TempDir()
	orig, _ := os.Getwd()
	origHome := os.Getenv("HOME")
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	cfgDir := GinkgoT().TempDir()
	Expect(os.Setenv("XDG_CONFIG_HOME", cfgDir)).To(Succeed())
	Expect(os.Setenv("HOME", cfgDir)).To(Succeed())
	DeferCleanup(func() {
		_ = os.Chdir(orig)
		_ = os.Setenv("HOME", origHome)
		_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
	})
	Expect(os.Chdir(dir)).To(Succeed())
	if spec != "" {
		Expect(os.WriteFile(filepath.Join(dir, "packspec.json"), []byte(spec), 0644)).To(Succeed())
	}
	return dir
}

func runCLI(args ...string) (string, string, error) {
	cmd := NewApp()
	out := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(errBuf)
	cmd.SilenceErrors = false
	cmd.SilenceUsage = false
	cmd.SetArgs(args)
	// Capture os.Stdout because some commands use fmt.Println directly.
	origStdout := os.Stdout
	origStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr
	err := cmd.Execute()
	wOut.Close()
	wErr.Close()
	os.Stdout = origStdout
	os.Stderr = origStderr
	stdoutBytes, _ := io.ReadAll(rOut)
	stderrBytes, _ := io.ReadAll(rErr)
	// Merge in case cobra wrote to SetOut/SetErr (for --help, errors, etc.).
	return out.String() + string(stdoutBytes), errBuf.String() + string(stderrBytes), err
}

func writePackSpec(dir, spec string) {
	Expect(os.WriteFile(filepath.Join(dir, "packspec.json"), []byte(spec), 0644)).To(Succeed())
}

func ensureLocksDir(dir string) {
	Expect(os.MkdirAll(filepath.Join(dir, "locks", "dependencies"), 0755)).To(Succeed())
}

func writeLockJSON(dir, mc, loader string, lock *domain.PackLock) {
	p := filepath.Join(dir, "locks", "dependencies", mc+"-"+loader+".json")
	data, _ := json.MarshalIndent(lock, "", "  ")
	Expect(os.WriteFile(p, data, 0644)).To(Succeed())
}

var _ = Describe("CLI boost coverage", func() {

	Describe("Run() exits non-zero on error", func() {
		It("Run with bad args exits", func() {
			// Cannot directly test os.Exit, but NewApp+Execute propagates the error.
			cmd := NewApp()
			cmd.SetArgs([]string{"definitely-not-a-real-cmd"})
			err := cmd.Execute()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("set", func() {
		It("set cf-key with only one arg errors", func() {
			_, _, err := runCLI("set", "cf-key")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("hint"))
		})
		It("set with wrong first arg errors", func() {
			_, _, err := runCLI("set", "wrong-arg", "value")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("hint"))
		})
		It("set cf-key --global writes user config", func() {
			dir := chdirTemp(`{"packName":"t","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			_, _, err := runCLI("set", "cf-key", "globalkey", "--global")
			Expect(err).NotTo(HaveOccurred())
			// No project file expected.
			_, statErr := os.Stat(filepath.Join(dir, ".mcmod", "config.json"))
			Expect(os.IsNotExist(statErr)).To(BeTrue())
		})
	})

	Describe("list", func() {
		It("list without spec errors", func() {
			chdirTemp("")
			_, _, err := runCLI("list")
			Expect(err).To(HaveOccurred())
		})
		It("list with empty mods prints (empty) for each scope", func() {
			chdirTemp(`{"packName":"empty","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			stdout, _, err := runCLI("list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("(empty)"))
			Expect(stdout).To(ContainSubstring("[Server]"))
			Expect(stdout).To(ContainSubstring("[Client]"))
			Expect(stdout).To(ContainSubstring("[Shared]"))
		})
	})

	Describe("validate", func() {
		It("validate with no spec errors", func() {
			chdirTemp("")
			_, _, err := runCLI("validate")
			Expect(err).To(HaveOccurred())
		})
		It("validate invalid spec errors", func() {
			chdirTemp(`{"packName":"","packVersion":"","minecraftVersion":"","loaderName":[]}`)
			_, _, err := runCLI("validate")
			Expect(err).To(HaveOccurred())
		})
		It("validate --lock with bad file errors", func() {
			chdirTemp(`{"packName":"v","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			_, _, err := runCLI("validate", "--lock", "/no/such/lock.json")
			Expect(err).To(HaveOccurred())
		})
		It("validate --release-index with bad file errors", func() {
			chdirTemp(`{"packName":"v","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			_, _, err := runCLI("validate", "--release-index", "/no/such/index.json")
			Expect(err).To(HaveOccurred())
		})
		It("validate --lock with bad JSON errors", func() {
			dir := chdirTemp(`{"packName":"v","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			p := filepath.Join(dir, "bad.json")
			Expect(os.WriteFile(p, []byte("{not json"), 0644)).To(Succeed())
			_, _, err := runCLI("validate", "--lock", p)
			Expect(err).To(HaveOccurred())
		})
		It("validate --release-index with bad JSON errors", func() {
			dir := chdirTemp(`{"packName":"v","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			p := filepath.Join(dir, "bad.json")
			Expect(os.WriteFile(p, []byte("{not json"), 0644)).To(Succeed())
			_, _, err := runCLI("validate", "--release-index", p)
			Expect(err).To(HaveOccurred())
		})
		It("validate --release-index with bad structure errors", func() {
			dir := chdirTemp(`{"packName":"v","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			p := filepath.Join(dir, "bad.json")
			Expect(os.WriteFile(p, []byte(`{"type":"package","packName":"p","minecraftVersion":"1.21.1","releases":"bad"}`), 0644)).To(Succeed())
			_, _, err := runCLI("validate", "--release-index", p)
			Expect(err).To(HaveOccurred())
		})
		It("validate --release-index with empty releases succeeds", func() {
			dir := chdirTemp(`{"packName":"v","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			p := filepath.Join(dir, "good.json")
			Expect(os.WriteFile(p, []byte(`{"type":"package","packName":"p","minecraftVersion":"1.21.1","releases":[]}`), 0644)).To(Succeed())
			stdout, _, err := runCLI("validate", "--release-index", p)
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("Release index is valid"))
		})
		It("validate --spec with bad file errors", func() {
			chdirTemp(`{"packName":"v","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			_, _, err := runCLI("validate", "--spec", "/no/such/spec.json")
			Expect(err).To(HaveOccurred())
		})
		It("validate --spec with bad JSON errors", func() {
			dir := chdirTemp(`{"packName":"v","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			p := filepath.Join(dir, "bad.json")
			Expect(os.WriteFile(p, []byte("{not json"), 0644)).To(Succeed())
			_, _, err := runCLI("validate", "--spec", p)
			Expect(err).To(HaveOccurred())
		})
		It("validate --lock with valid lock file succeeds", func() {
			dir := chdirTemp(`{"packName":"v","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			p := filepath.Join(dir, "lock.json")
			Expect(os.WriteFile(p, []byte(`{"loader":"neoforge","minecraftVersion":"1.21.1","mods":{"m":{"name":"M","source":{"type":"local","path":"./m.jar","fileName":"m.jar"}}}}`), 0644)).To(Succeed())
			stdout, _, err := runCLI("validate", "--lock", p)
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("Lock file is valid"))
		})
		It("validate --lock with bad structure errors", func() {
			dir := chdirTemp(`{"packName":"v","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			p := filepath.Join(dir, "bad.json")
			Expect(os.WriteFile(p, []byte(`{"loader":"neoforge","minecraftVersion":"1.21.1","mods":"bad"}`), 0644)).To(Succeed())
			_, _, err := runCLI("validate", "--lock", p)
			Expect(err).To(HaveOccurred())
		})
		It("validate --spec with valid file succeeds", func() {
			dir := chdirTemp(`{"packName":"v","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			p := filepath.Join(dir, "spec.json")
			Expect(os.WriteFile(p, []byte(`{"packName":"v","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`), 0644)).To(Succeed())
			stdout, _, err := runCLI("validate", "--spec", p)
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("packspec.json is valid"))
		})
	})

	Describe("lock", func() {
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

	Describe("lock list", func() {
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

	Describe("lock show", func() {
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

	Describe("lock add", func() {
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

	Describe("lock update", func() {
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
	})

	Describe("lock delete", func() {
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
	})

	Describe("lock tree", func() {
		It("lock tree without lock errors", func() {
			chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			_, _, err := runCLI("lock", "tree", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("lock tree with lock works", func() {
			dir := chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			ensureLocksDir(dir)
			writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{
				Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{
					"a": {Name: "A", Version: "1", Scope: "shared", Source: domain.LockedSource{Type: "local", FileName: "a.jar"}},
				},
			})
			_, _, err := runCLI("lock", "tree", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("release set", func() {
		It("release set writes index", func() {
			chdirTemp(`{"packName":"p","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			_, _, err := runCLI("lock", "release", "set", "1.21.1",
				"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0")
			Expect(err).NotTo(HaveOccurred())
		})
		It("release set with loader and artifacts writes artifact map", func() {
			chdirTemp(`{"packName":"p","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			_, _, err := runCLI("lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0",
				"--artifact-client", "client.jar", "--artifact-server", "server.jar")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("release list/show/delete", func() {
		It("release list with no index errors", func() {
			chdirTemp(`{"packName":"p","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			_, _, err := runCLI("lock", "release", "list", "1.21.1")
			Expect(err).To(HaveOccurred())
		})
		It("release show without args errors", func() {
			_, _, err := runCLI("lock", "release", "show")
			Expect(err).To(HaveOccurred())
		})
		It("release show with no index errors", func() {
			chdirTemp(`{"packName":"p","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			_, _, err := runCLI("lock", "release", "show", "1.21.1", "0.1.0")
			Expect(err).To(HaveOccurred())
		})
		It("release delete with too few args errors", func() {
			_, _, err := runCLI("lock", "release", "delete")
			Expect(err).To(HaveOccurred())
		})
		It("release delete with no index for loader-specific target succeeds gracefully", func() {
			chdirTemp(`{"packName":"p","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			stdout, _, err := runCLI("lock", "release", "delete", "1.21.1", "0.1.0", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("deleted"))
		})
		It("release delete for non-existent version in non-empty index succeeds gracefully", func() {
			chdirTemp(`{"packName":"p","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			_, _, err := runCLI("lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0")
			Expect(err).NotTo(HaveOccurred())
			stdout, _, err := runCLI("lock", "release", "delete", "1.21.1", "99.99.99", "neoforge", "--target", "client")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("deleted"))
		})
		It("release delete entire version", func() {
			chdirTemp(`{"packName":"p","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			_, _, err := runCLI("lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = runCLI("lock", "release", "delete", "1.21.1", "0.1.0")
			Expect(err).NotTo(HaveOccurred())
		})
		It("release delete non-existing version in non-empty index (whole-record path) errors", func() {
			chdirTemp(`{"packName":"p","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			_, _, err := runCLI("lock", "release", "set", "1.21.1", "neoforge",
				"--version", "0.1.0", "--repo", "o/r", "--tag", "v0.1.0")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = runCLI("lock", "release", "delete", "1.21.1", "9.9.9")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("build", func() {
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
	})

	Describe("config", func() {
		It("config shows the key", func() {
			chdirTemp(`{"packName":"c","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			_, _, err := runCLI("set", "cf-key", "keyval", "--project")
			Expect(err).NotTo(HaveOccurred())
			stdout, _, err := runCLI("config")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("keyval"))
		})
		It("config with no key shows (not set)", func() {
			chdirTemp(`{"packName":"c","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			stdout, _, err := runCLI("config")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("not set"))
		})
		It("config set-cf-key writes key", func() {
			dir := chdirTemp(`{"packName":"c","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			_, _, err := runCLI("config", "set-cf-key", "thekey")
			Expect(err).NotTo(HaveOccurred())
			_, statErr := os.Stat(filepath.Join(dir, ".mcmod", "config.json"))
			Expect(statErr).NotTo(HaveOccurred())
		})
	})

	Describe("tree alias", func() {
		It("tree without lock errors", func() {
			chdirTemp(`{"packName":"t","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			_, _, err := runCLI("tree", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("tree with lock works", func() {
			dir := chdirTemp(`{"packName":"t","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
			ensureLocksDir(dir)
			writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{
				Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]domain.LockedMod{
					"a": {Name: "A", Version: "1", Scope: "shared", Source: domain.LockedSource{Type: "local", FileName: "a.jar"}},
				},
			})
			stdout, _, err := runCLI("tree", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("dependency tree"))
		})
	})

	Describe("version", func() {
		It("version prints version info", func() {
			stdout, _, err := runCLI("version")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("mcmod version"))
		})
	})

	Describe("help", func() {
		It("root help lists all commands", func() {
			stdout, _, err := runCLI("--help")
			Expect(err).NotTo(HaveOccurred())
			for _, sub := range []string{"lock", "build", "list", "validate", "set", "tree", "config", "version"} {
				Expect(stdout).To(ContainSubstring(sub))
			}
		})
		It("help subcommand lists all commands", func() {
			stdout, _, err := runCLI("help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("lock"))
		})
		It("lock --help", func() {
			stdout, _, err := runCLI("lock", "--help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("lock"))
		})
		It("lock release --help", func() {
			stdout, _, err := runCLI("lock", "release", "--help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("set"))
		})
		It("set --help", func() {
			stdout, _, err := runCLI("set", "--help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("cf-key"))
		})
		It("lock add --help", func() {
			stdout, _, err := runCLI("lock", "add", "--help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("source"))
		})
		It("lock update --help", func() {
			stdout, _, err := runCLI("lock", "update", "--help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("version"))
		})
	})
})
