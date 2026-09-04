// File: test/e2e/e2e_suite_test.go
// Created: 2026-06-20
// Description: Ginkgo test suite for mcmod CLI end-to-end tests.

package test

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var mcmodBin string

var _ = BeforeSuite(func() {
	buildDir := GinkgoT().TempDir()
	// Build the CLI binary
	mcmodBin = filepath.Join(buildDir, "mcmod")
	cmd := exec.Command("go", "build", "-o", mcmodBin, "github.com/orangeboyChen/mcmod-cli/cmd/mcmod")
	output, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), "build mcmod: %s", string(output))
})

func runMcmod(dir string, args ...string) (string, string, error) {
	cmd := exec.Command(mcmodBin, args...)
	cmd.Dir = dir
	// Isolate HOME and CURSEFORGE_API_KEY so per-test state stays inside the
	// temp directory and never pollutes the host user config.
	cmd.Env = append(os.Environ(),
		"HOME="+dir,
		"XDG_CONFIG_HOME="+dir,
		"CURSEFORGE_API_KEY=",
	)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	// On non-zero exit, surface the captured stderr in the error message
	// so the failure mode is self-describing instead of a bare
	// "exit status 2" with an empty body.
	if err != nil && stderr.Len() > 0 {
		err = fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), stderr.String(), err
}

func writeLockFile(dir, mcVersion, loader string, lock *domain.PackLock) {
	p := filepath.Join(dir, "locks", "dependencies", fmt.Sprintf("%s-%s.json", mcVersion, loader))
	Expect(os.MkdirAll(filepath.Dir(p), 0755)).To(Succeed())
	data, err := json.MarshalIndent(lock, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(p, data, 0644)).To(Succeed())
}

func writeReleaseIndexFile(dir, mcVersion string, index *domain.ReleaseIndex) {
	p := filepath.Join(dir, "locks", "releases", fmt.Sprintf("%s.json", mcVersion))
	Expect(os.MkdirAll(filepath.Dir(p), 0755)).To(Succeed())
	data, err := json.MarshalIndent(index, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(p, data, 0644)).To(Succeed())
}

func createFixtureJar(dir, name, metaType, metaContent string) string {
	p := filepath.Join(dir, name)
	f, err := os.Create(p)
	Expect(err).NotTo(HaveOccurred())
	defer f.Close()
	w := zip.NewWriter(f)
	defer w.Close()
	metaPath := "fabric.mod.json"
	if metaType == "neoforge" {
		metaPath = "META-INF/neoforge.mods.toml"
	}
	entry, err := w.Create(metaPath)
	Expect(err).NotTo(HaveOccurred())
	_, err = entry.Write([]byte(metaContent))
	Expect(err).NotTo(HaveOccurred())
	classEntry, err := w.Create("com/example/Foo.class")
	Expect(err).NotTo(HaveOccurred())
	_, err = classEntry.Write([]byte("fake class data"))
	Expect(err).NotTo(HaveOccurred())
	return p
}
