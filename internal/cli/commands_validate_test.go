// File: internal/cli/commands_validate_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for `mcmod validate` subcommand.

package cli

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("validate", func() {
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

var _ = Describe("CLI extra4 - newValidateCmd paths", func() {
	It("validate with --spec and bad JSON fails", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		bad := filepath.Join(dir, "bad.json")
		Expect(os.WriteFile(bad, []byte("{not json"), 0644)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"validate", "--spec", bad})
		err := cmd.Execute()
		Expect(err).To(HaveOccurred())
	})

	It("validate with --lock and bad JSON fails", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		bad := filepath.Join(dir, "bad.json")
		Expect(os.WriteFile(bad, []byte("{not json"), 0644)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"validate", "--lock", bad})
		err := cmd.Execute()
		Expect(err).To(HaveOccurred())
	})

	It("validate with --release-index and bad JSON fails", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		bad := filepath.Join(dir, "bad.json")
		Expect(os.WriteFile(bad, []byte("{not json"), 0644)).To(Succeed())
		cmd := NewApp()
		cmd.SetArgs([]string{"validate", "--release-index", bad})
		err := cmd.Execute()
		Expect(err).To(HaveOccurred())
	})
})
