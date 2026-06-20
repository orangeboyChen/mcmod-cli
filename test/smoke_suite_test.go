// File: test/smoke_suite_test.go
// Created: 2026-06-20
// Description: Ginkgo test suite for mcmod CLI smoke tests.

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

func TestSmoke(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Smoke Suite")
}

var smokeDir string
var mcmodBin string

var _ = BeforeSuite(func() {
	smokeDir = GinkgoT().TempDir()
	// Build the CLI binary
	mcmodBin = filepath.Join(smokeDir, "mcmod")
	cmd := exec.Command("go", "build", "-o", mcmodBin, "github.com/orangeboyChen/mcmod-cli/cmd/mcmod")
	output, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), "build mcmod: %s", string(output))
})

func writeSpec(dir string, content string) string {
	p := filepath.Join(dir, "packspec.json")
	err := os.WriteFile(p, []byte(content), 0644)
	Expect(err).NotTo(HaveOccurred())
	return p
}

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

func writeSpecStd(dir string) {
	writeSpec(dir, `{
		"packName": "test-pack",
		"packVersion": "0.1.0",
		"minecraftVersion": "1.21.1",
		"loaderName": ["neoforge:21.1.219", "fabric:1.21.123"],
		"author": "tester",
		"mods": {
			"jei": {
				"name": "Just Enough Items",
				"scope": "client",
				"source": {"type": "curseforge", "query": "Just Enough Items"}
			},
			"create": {
				"scope": "shared",
				"source": {"type": "curseforge", "query": "Create"}
			}
		}
	}`)
}

func writeLockFile(dir, mcVersion, loader string, lock *domain.PackLock) {
	p := filepath.Join(dir, "locks", "dependencies", fmt.Sprintf("%s-%s.json", mcVersion, loader))
	os.MkdirAll(filepath.Dir(p), 0755)
	data, _ := json.MarshalIndent(lock, "", "  ")
	err := os.WriteFile(p, data, 0644)
	Expect(err).NotTo(HaveOccurred())
}

func writeReleaseIndexFile(dir, mcVersion string, ri *domain.ReleaseIndex) {
	p := filepath.Join(dir, "locks", "releases", fmt.Sprintf("%s.json", mcVersion))
	os.MkdirAll(filepath.Dir(p), 0755)
	data, _ := json.MarshalIndent(ri, "", "  ")
	err := os.WriteFile(p, data, 0644)
	Expect(err).NotTo(HaveOccurred())
}

func createFixtureJar(dir, name, metaType, metaContent string) string {
	p := filepath.Join(dir, name)
	f, err := os.Create(p)
	Expect(err).NotTo(HaveOccurred())
	defer f.Close()
	w := zip.NewWriter(f)
	var metaPath string
	switch metaType {
	case "neoforge":
		metaPath = "META-INF/neoforge.mods.toml"
	case "fabric":
		metaPath = "fabric.mod.json"
	}
	entry, _ := w.Create(metaPath)
	entry.Write([]byte(metaContent))
	// Add a class file for conflict detection
	classEntry, _ := w.Create("com/example/Foo.class")
	classEntry.Write([]byte("fake class data"))
	w.Close()
	return p
}
