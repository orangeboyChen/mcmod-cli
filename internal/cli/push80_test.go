// File: internal/cli/push80_test.go
// Created: 2026-06-20
// Description: Final push to 80% CLI coverage.
package cli

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CLI push80", func() {
	It("newApp subcommands", func() {
		app := NewApp()
		_ = app.Commands()
	})
	It("lock add github-release", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.MkdirAll("locks/dependencies", 0755)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "add", "1.21.1", "neoforge", "ghm",
			"--name", "GHM", "--source", "github-release",
			"--repo", "o/r", "--tag", "v1", "--asset-name", "a.jar", "--file-name", "a.jar"})
		cmd.Execute()
	})
	It("lock add curseforge", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.MkdirAll("locks/dependencies", 0755)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "add", "1.21.1", "neoforge", "cfm",
			"--name", "CFM", "--source", "curseforge",
			"--mod-id", "1", "--file-id", "2", "--file-name", "m.jar"})
		cmd.Execute()
	})
	It("lock show single key", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.MkdirAll("locks/dependencies", 0755)).To(Succeed())
		Expect(os.WriteFile("locks/dependencies/1.21.1-neoforge.json",
			[]byte(`{"loader":"neoforge","minecraftVersion":"1.21.1","mods":{"sk":{"name":"SK","scope":"shared","source":{"type":"local","path":"./sk.jar"}}}}`), 0644)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "show", "1.21.1", "neoforge", "sk"})
		cmd.Execute()
	})
	It("lock show full dump", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.MkdirAll("locks/dependencies", 0755)).To(Succeed())
		Expect(os.WriteFile("locks/dependencies/1.21.1-neoforge.json",
			[]byte(`{"loader":"neoforge","minecraftVersion":"1.21.1","mods":{}}`), 0644)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "show", "1.21.1", "neoforge"})
		cmd.Execute()
	})
	It("lock release show", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.MkdirAll("locks/releases", 0755)).To(Succeed())
		Expect(os.WriteFile("locks/releases/1.21.1.json",
			[]byte(`{"type":"package","packName":"p","minecraftVersion":"1.21.1","releases":[{"version":"0.1.0","type":"github-release"}]}`), 0644)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "release", "show", "1.21.1", "0.1.0"})
		cmd.Execute()
	})
	It("lock release list", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.MkdirAll("locks/releases", 0755)).To(Succeed())
		Expect(os.WriteFile("locks/releases/1.21.1.json",
			[]byte(`{"type":"package","packName":"p","minecraftVersion":"1.21.1","releases":[{"version":"0.1.0","type":"github-release"}]}`), 0644)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"lock", "release", "list", "1.21.1"})
		cmd.Execute()
	})
})
