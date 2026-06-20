// File: internal/config/coverage_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for config package.
package config

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	It("GetCFKey relies on env var priority", func() {
		os.Setenv("CURSEFORGE_API_KEY", "env-key")
		defer os.Unsetenv("CURSEFORGE_API_KEY")
		Expect(GetCFKey()).To(Equal("env-key"))
	})

	It("uses env var when set", func() {
		os.Setenv("CURSEFORGE_API_KEY", "env-key")
		defer os.Unsetenv("CURSEFORGE_API_KEY")
		Expect(GetCFKey()).To(Equal("env-key"))
	})

	It("ReadEnvCFKey returns env value", func() {
		os.Setenv("CURSEFORGE_API_KEY", "direct")
		defer os.Unsetenv("CURSEFORGE_API_KEY")
		Expect(ReadEnvCFKey()).To(Equal("direct"))
	})

	It("WriteProjectConfig stores key", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(WriteProjectConfig("project-key")).To(Succeed())
		cfg, err := ReadProjectConfig()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg).NotTo(BeNil())
		Expect(cfg.CFKey).To(Equal("project-key"))
	})

	It("ReadProjectConfig returns nil for missing config", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		cfg, err := ReadProjectConfig()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg).To(BeNil())
	})

	It("WriteUserConfig writes config", func() {
		Expect(WriteUserConfig("user-key")).To(Succeed())
		usr, _ := os.UserHomeDir()
		path := filepath.Join(usr, ".config", "mcmod", "config.json")
		defer os.Remove(path)
		cfg, err := ReadUserConfig()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg).NotTo(BeNil())
	})
})

var _ = Describe("Config LoadDotEnv", func() {
	It("returns nil for missing file", func() {
		Expect(LoadDotEnv("/no/such/.env")).To(Succeed())
	})

	It("parses key=value pairs without quotes", func() {
		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, ".env")
		Expect(os.WriteFile(path, []byte("FOO=bar\n"), 0644)).To(Succeed())
		Expect(LoadDotEnv(path)).To(Succeed())
		Expect(os.Getenv("FOO")).To(Equal("bar"))
		os.Unsetenv("FOO")
	})

	It("strips double quotes", func() {
		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, ".env")
		Expect(os.WriteFile(path, []byte("FOO=\"bar\"\n"), 0644)).To(Succeed())
		Expect(LoadDotEnv(path)).To(Succeed())
		Expect(os.Getenv("FOO")).To(Equal("bar"))
		os.Unsetenv("FOO")
	})

	It("strips single quotes", func() {
		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, ".env")
		Expect(os.WriteFile(path, []byte("FOO='bar'\n"), 0644)).To(Succeed())
		Expect(LoadDotEnv(path)).To(Succeed())
		Expect(os.Getenv("FOO")).To(Equal("bar"))
		os.Unsetenv("FOO")
	})

	It("skips comments and blank lines", func() {
		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, ".env")
		Expect(os.WriteFile(path, []byte("# comment\n\nFOO=bar\n#another\n"), 0644)).To(Succeed())
		Expect(LoadDotEnv(path)).To(Succeed())
		Expect(os.Getenv("FOO")).To(Equal("bar"))
		os.Unsetenv("FOO")
	})

	It("does not override existing env", func() {
		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, ".env")
		os.Setenv("FOO", "keep")
		defer os.Unsetenv("FOO")
		Expect(os.WriteFile(path, []byte("FOO=overwritten\n"), 0644)).To(Succeed())
		Expect(LoadDotEnv(path)).To(Succeed())
		Expect(os.Getenv("FOO")).To(Equal("keep"))
	})
})
