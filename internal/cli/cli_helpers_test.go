// File: internal/cli/cli_helpers_test.go
// Created: 2026-06-20
// Description: Test helpers shared by cli/*_test.go files (GinkgoT temp dir, CLI execution capture, spec/lock writers).

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

// runCLI executes the cobra app with the given args and returns (stdout, stderr, err).
// It captures both the cobra buffers and direct os.Stdout/os.Stderr writes
// (some commands use fmt.Println directly rather than cmd.OutOrStderr()).
func runCLI(args ...string) (string, string, error) {
	cmd := NewApp()
	out := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(errBuf)
	cmd.SilenceErrors = false
	cmd.SilenceUsage = false
	cmd.SetArgs(args)
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
