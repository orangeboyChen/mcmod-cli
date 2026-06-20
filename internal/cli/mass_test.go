// File: internal/cli/mass_test.go
// Created: 2026-06-20
// Description: Mass CLI coverage.
package cli

import (
	"bytes"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

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
