// File: internal/config/config_test.go
// Created: 2026-09-04
// Description: Ginkgo tests for project configuration behavior.

package config

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func inTempProject() string {
	dir := GinkgoT().TempDir()
	old, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	Expect(os.Chdir(dir)).To(Succeed())
	DeferCleanup(func() { Expect(os.Chdir(old)).To(Succeed()) })
	return dir
}

var _ = Describe("Project configuration", func() {
	It("writes and reads .mcmod/config.json", func() {
		dir := inTempProject()
		Expect(WriteProjectConfig("project-key")).To(Succeed())
		Expect(filepath.Join(dir, ".mcmod", "config.json")).To(BeAnExistingFile())
		cfg, err := ReadProjectConfig()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg).NotTo(BeNil())
		Expect(cfg.CFKey).To(Equal("project-key"))
	})

	It("keeps compatibility aliases in the project directory", func() {
		inTempProject()
		Expect(WriteUserConfig("alias-key")).To(Succeed())
		cfg, err := ReadUserConfig()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.CFKey).To(Equal("alias-key"))
	})

	It("returns nil for a missing project config", func() {
		inTempProject()
		cfg, err := ReadProjectConfig()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg).To(BeNil())
	})

	It("returns an error for malformed JSON", func() {
		inTempProject()
		Expect(os.MkdirAll(".mcmod", 0700)).To(Succeed())
		Expect(os.WriteFile(".mcmod/config.json", []byte(`{bad}`), 0600)).To(Succeed())
		cfg, err := ReadProjectConfig()
		Expect(err).To(HaveOccurred())
		Expect(cfg).To(BeNil())
	})

	It("returns an error when the config directory is a file", func() {
		inTempProject()
		Expect(os.WriteFile(".mcmod", []byte("not a directory"), 0600)).To(Succeed())
		Expect(WriteProjectConfig("key")).To(HaveOccurred())
	})

	It("prioritizes the environment key over project config", func() {
		inTempProject()
		Expect(WriteProjectConfig("project-key")).To(Succeed())
		Expect(os.Setenv("CURSEFORGE_API_KEY", "env-key")).To(Succeed())
		DeferCleanup(func() { _ = os.Unsetenv("CURSEFORGE_API_KEY") })
		Expect(GetCFKey()).To(Equal("env-key"))
	})

	It("reads the project key when the environment is empty", func() {
		inTempProject()
		Expect(os.Unsetenv("CURSEFORGE_API_KEY")).To(Succeed())
		Expect(WriteProjectConfig("project-key")).To(Succeed())
		Expect(GetCFKey()).To(Equal("project-key"))
	})

	It("returns an empty key when no source is configured", func() {
		inTempProject()
		Expect(os.Unsetenv("CURSEFORGE_API_KEY")).To(Succeed())
		Expect(GetCFKey()).To(BeEmpty())
	})

	It("reads the environment key directly", func() {
		Expect(os.Setenv("CURSEFORGE_API_KEY", "direct")).To(Succeed())
		DeferCleanup(func() { _ = os.Unsetenv("CURSEFORGE_API_KEY") })
		Expect(ReadEnvCFKey()).To(Equal("direct"))
	})
})

var _ = Describe("LoadDotEnv", func() {
	It("ignores missing files and comments", func() {
		Expect(LoadDotEnv("/no/such/.env")).To(Succeed())
		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, ".env")
		Expect(os.WriteFile(path, []byte("# comment\n\nFOO=bar\n"), 0600)).To(Succeed())
		Expect(LoadDotEnv(path)).To(Succeed())
		Expect(os.Getenv("FOO")).To(Equal("bar"))
		DeferCleanup(func() { _ = os.Unsetenv("FOO") })
	})

	It("strips matching quotes and preserves existing values", func() {
		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, ".env")
		Expect(os.WriteFile(path, []byte("QUOTED=\"bar\"\nSINGLE='baz'\nKEEP=replace\nINVALID\n"), 0600)).To(Succeed())
		Expect(os.Setenv("KEEP", "original")).To(Succeed())
		DeferCleanup(func() { _ = os.Unsetenv("QUOTED"); _ = os.Unsetenv("SINGLE"); _ = os.Unsetenv("KEEP") })
		Expect(LoadDotEnv(path)).To(Succeed())
		Expect(os.Getenv("QUOTED")).To(Equal("bar"))
		Expect(os.Getenv("SINGLE")).To(Equal("baz"))
		Expect(os.Getenv("KEEP")).To(Equal("original"))
	})
})
