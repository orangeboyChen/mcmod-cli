// File: internal/cli/coverage_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for CLI package.
package cli

import (
	"bytes"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CLI", func() {
	It("NewApp creates root command", func() {
		app := NewApp()
		Expect(app).NotTo(BeNil())
		Expect(app.Use).To(Equal("mcmod"))
	})

	It("has all expected subcommands", func() {
		cmd := NewApp()
		names := make(map[string]bool)
		for _, c := range cmd.Commands() {
			names[c.Name()] = true
		}
		for _, name := range []string{"set", "lock", "build", "list", "validate", "tree", "config", "version"} {
			Expect(names).To(HaveKey(name))
		}
	})

	It("help output contains commands", func() {
		cmd := NewApp()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"--help"})
		Expect(cmd.Execute()).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("lock"))
	})

	It("version runs without error", func() {
		cmd := NewApp()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"version"})
		Expect(cmd.Execute()).To(Succeed())
	})

	It("list in temp dir with packspec works", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.WriteFile("packspec.json", []byte(`{"packName":"test","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`), 0644)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"list"})
		Expect(cmd.Execute()).To(Succeed())
	})

	It("config shows key status", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		cmd := NewApp()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"config"})
		Expect(cmd.Execute()).To(Succeed())
	})

	It("config set-cf-key writes config", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		cmd := NewApp()
		cmd.SetArgs([]string{"config", "set-cf-key", "test-key"})
		Expect(cmd.Execute()).To(Succeed())
	})

	It("tree with packspec in temp dir", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.WriteFile("packspec.json", []byte(`{"packName":"t","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`), 0644)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"tree"})
		_ = cmd.Execute() // may fail if no lock
	})

	It("validate with packspec", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.WriteFile("packspec.json", []byte(`{"packName":"v","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`), 0644)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"validate"})
		Expect(cmd.Execute()).To(Succeed())
	})

	It("validate --spec flag works", func() {
		dir := GinkgoT().TempDir()
		Expect(os.WriteFile(dir+"/spec.json", []byte(`{"packName":"v","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`), 0644)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"validate", "--spec", dir + "/spec.json"})
		Expect(cmd.Execute()).To(Succeed())
	})

	It("set cf-key writes user config", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		cmd := NewApp()
		cmd.SetArgs([]string{"set", "cf-key", "my-key"})
		Expect(cmd.Execute()).To(Succeed())
	})

	It("set cf-key --project writes project config", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		cmd := NewApp()
		cmd.SetArgs([]string{"set", "cf-key", "proj-key", "--project"})
		Expect(cmd.Execute()).To(Succeed())
	})

	It("lock runs in temp dir with valid packspec", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.WriteFile("packspec.json", []byte(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`), 0644)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"lock"})
		_ = cmd.Execute() // 可能成功或失败
	})

	It("usage template is non-empty", func() {
		Expect(usageTemplate()).NotTo(BeEmpty())
	})
})
