// File: internal/service/mass_test.go
// Created: 2026-06-20
// Description: Mass service coverage.
package service

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("Service mass", func() {
	It("BuildLock with empty mods", func() {
		spec := &domain.PackSpec{MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"}}
		lock, err := BuildLock(spec, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(lock.Mods).To(BeEmpty())
	})

	It("BuildLock curseforge without key", func() {
		spec := &domain.PackSpec{MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
			Mods: map[string]domain.ModSpec{"c": {Source: domain.ModSource{Type: "curseforge", Query: "create"}}}}
		lock, err := BuildLock(spec, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(lock.Mods).NotTo(HaveKey("c"))
	})

	It("LoadLock fails without lock file", func() {
		_, err := LoadLock("99.99", "neoforge")
		Expect(err).To(HaveOccurred())
	})

	It("SaveLock writes lock", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.MkdirAll("locks/dependencies", 0755)).To(Succeed())
		Expect(SaveLock("1.21.1", "neoforge", &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1"})).To(Succeed())
	})

	It("BuildArtifact nil lock", func() {
		Expect(BuildArtifact(nil, nil, "1.21.1", "client")).To(HaveOccurred())
	})

	It("BuildLock local missing file", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		spec := &domain.PackSpec{MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
			Mods: map[string]domain.ModSpec{"m": {Source: domain.ModSource{Type: "local", Path: "./nope.jar"}}}}
		lock, err := BuildLock(spec, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(lock.Mods).NotTo(HaveKey("m"))
	})

	It("CreateReleaseRecord new index", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.MkdirAll("locks/releases", 0755)).To(Succeed())
		index, err := CreateReleaseRecord("1.21.1", "0.1.0", "github-release", &domain.ReleaseGitHub{Repo: "o/r", Tag: "v0.1.0"})
		Expect(err).NotTo(HaveOccurred())
		Expect(index.Releases).To(HaveLen(1))
	})

	It("ReadReleaseIndex fails for missing", func() {
		_, err := ReadReleaseIndex("99.99")
		Expect(err).To(HaveOccurred())
	})

	It("BuildTree with mods", func() {
		lock := &domain.PackLock{Loader: "n", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{"a": {Name: "A", Version: "1", Scope: "shared", Source: domain.LockedSource{Type: "local"}}}}
		tree := BuildTree(lock)
		Expect(tree).To(HaveLen(1))
	})

	It("FormatTree with entries", func() {
		output := FormatTree([]TreeEntry{{Name: "A", Version: "1", Scope: "shared", Source: "local"}})
		Expect(output).To(ContainSubstring("A"))
	})

	It("ConfigureCFKey saves config", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(ConfigureCFKey("k")).To(Succeed())
	})

	It("ConfigureUserCFKey saves config", func() {
		Expect(ConfigureUserCFKey("uk")).To(Succeed())
	})

	It("ReadPackSpec fails in empty dir", func() {
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(GinkgoT().TempDir())
		_, err := ReadPackSpec(".")
		Expect(err).To(HaveOccurred())
	})

	It("BuildClientServerBuild missing lock", func() {
		spec := &domain.PackSpec{PackName: "t", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
			Mods: map[string]domain.ModSpec{"a": {Scope: "shared", Source: domain.ModSource{Type: "local"}}}}
		err := BuildClientServerBuild(spec, "1.21.1")
		Expect(err).To(HaveOccurred())
	})
})
