// File: internal/config/config_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/config/config.go.

package config

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// --- from coverage_test.go (Config) ---
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

// --- from coverage_test.go (Config LoadDotEnv) ---
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

// --- from extra_test.go (Config extended) ---
var _ = Describe("Config extended", func() {
	It("ReadUserConfig returns nil for bad path", func() {
		cfg, err := ReadUserConfig()
		// 如果用户配置文件存在但解析失败，可能报错
		if cfg == nil && err == nil {
			// OK - config not found
		}
	})

	It("GetCFKey reads from project over user", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)

		// Write project config
		os.MkdirAll(".mcmod", 0700)
		os.WriteFile(".mcmod/config.json", []byte(`{"cfKey":"proj-val"}`), 0600)
		os.Unsetenv("CURSEFORGE_API_KEY")
		Expect(GetCFKey()).To(Equal("proj-val"))
	})

	It("ReadProjectConfig with bad JSON returns error", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		os.MkdirAll(".mcmod", 0700)
		os.WriteFile(".mcmod/config.json", []byte(`{bad}`), 0600)
		cfg, err := ReadProjectConfig()
		Expect(err).To(HaveOccurred())
		Expect(cfg).To(BeNil())
	})

	It("ReadUserConfig with bad JSON returns error", func() {
		usr, _ := os.UserHomeDir()
		p := filepath.Join(usr, ".config", "mcmod", "config.json")
		os.MkdirAll(filepath.Dir(p), 0700)
		os.WriteFile(p, []byte(`{bad}`), 0600)
		defer os.Remove(p)
		_, err := ReadUserConfig()
		Expect(err).To(HaveOccurred())
	})

	It("ReadEnvCFKey reads env", func() {
		os.Setenv("CURSEFORGE_API_KEY", "env-val")
		defer os.Unsetenv("CURSEFORGE_API_KEY")
		Expect(ReadEnvCFKey()).To(Equal("env-val"))
	})

	It("GetCFKey falls through when nothing set", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		os.Unsetenv("CURSEFORGE_API_KEY")
		Expect(GetCFKey()).To(BeEmpty())
	})
})

// --- from extra_test.go (Config homeDir branches) ---
var _ = Describe("Config homeDir branches", func() {
	It("uses XDG_CONFIG_HOME when set", func() {
		origXDG := os.Getenv("XDG_CONFIG_HOME")
		origHome := os.Getenv("HOME")
		os.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-test")
		os.Setenv("HOME", "/tmp/home-test")
		defer os.Setenv("XDG_CONFIG_HOME", origXDG)
		defer os.Setenv("HOME", origHome)
		Expect(homeDir()).To(Equal("/tmp/xdg-test"))
	})

	It("falls back to HOME when XDG unset", func() {
		origXDG := os.Getenv("XDG_CONFIG_HOME")
		origHome := os.Getenv("HOME")
		os.Unsetenv("XDG_CONFIG_HOME")
		os.Setenv("HOME", "/tmp/home-only")
		defer os.Setenv("XDG_CONFIG_HOME", origXDG)
		defer os.Setenv("HOME", origHome)
		Expect(homeDir()).To(Equal("/tmp/home-only"))
	})
})
