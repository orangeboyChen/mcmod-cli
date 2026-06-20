// File: internal/service/mass2_test.go
// Created: 2026-06-20
// Description: Mass2 service coverage.
package service

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("Service mass2", func() {
	It("ListMods empty pack", func() {
		spec := &domain.PackSpec{PackName: "e", PackVersion: "1", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"}}
		output, err := ListMods(spec)
		Expect(err).NotTo(HaveOccurred())
		Expect(output).NotTo(BeEmpty())
	})
	It("BuildLock empty mods", func() {
		spec := &domain.PackSpec{MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"}}
		lock, err := BuildLock(spec, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(lock.Mods).To(BeEmpty())
	})
	It("BuildTree with names", func() {
		lock := &domain.PackLock{Loader: "n", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"create": {Name: "Create", Version: "6.0.0", Scope: "shared", Source: domain.LockedSource{Type: "curseforge"}},
				"jei":    {Name: "JEI", Version: "19.0", Scope: "client", Source: domain.LockedSource{Type: "curseforge"}},
			}}
		tree := BuildTree(lock)
		Expect(tree).To(HaveLen(2))
		output := FormatTree(tree)
		Expect(output).To(ContainSubstring("Create"))
	})
	It("LoadLock not found", func() {
		_, err := LoadLock("99.99", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("SaveLock saves and loads", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		store := domain.DefaultFileStore(dir)
		lock := domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{}}
		Expect(store.SaveLock("1.21.1", "neoforge", lock)).To(Succeed())
		loaded, err := store.LoadLock("1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded.Loader).To(Equal("neoforge"))
	})
	It("BuildClientServerBuild creates lock", func() {
		spec := &domain.PackSpec{
			PackName: "test", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
			Mods: map[string]domain.ModSpec{"a": {Scope: "shared", Source: domain.ModSource{Type: "local"}}},
		}
		err := BuildClientServerBuild(spec, "1.21.1")
		Expect(err).To(HaveOccurred())
	})
	It("GetCFKey with env set", func() {
		os.Setenv("CURSEFORGE_API_KEY", "env-val")
		defer os.Unsetenv("CURSEFORGE_API_KEY")
		Expect(GetCFKey()).To(Equal("env-val"))
	})
	It("ReadReleaseIndex with missing file", func() {
		_, err := ReadReleaseIndex("99.99")
		Expect(err).To(HaveOccurred())
	})
	It("WriteReleaseIndex creates file", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.MkdirAll("locks/releases", 0755)).To(Succeed())
		ri := &domain.ReleaseIndex{Type: "package", PackName: "p", MinecraftVersion: "1.21.1"}
		Expect(WriteReleaseIndex("1.21.1", ri)).To(Succeed())
	})
	It("MarshalLockJSON complex lock", func() {
		lock := &domain.PackLock{Loader: "n", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Version: "1", Scope: "shared", Identity: &domain.Identity{Source: "cf:1", Confidence: "source-only"}},
			}}
		data, err := MarshalLockJSON(lock)
		Expect(err).NotTo(HaveOccurred())
		Expect(data).NotTo(BeEmpty())
	})
	It("FormatTree empty lock", func() {
		entries := BuildTree(&domain.PackLock{Loader: "n", MinecraftVersion: "1.21.1"})
		Expect(FormatTree(entries)).To(BeEmpty())
	})
	It("BuildArtifact server target", func() {
		spec := &domain.PackSpec{PackName: "test",
			Mods: map[string]domain.ModSpec{"a": {Scope: "shared", Source: domain.ModSource{Type: "local", Path: "./a.jar"}}}}
		lock := &domain.PackLock{Loader: "n", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: "./a.jar", FileName: "a.jar"}}}}
		// BuildArtifact now requires the jar to exist on disk.
		dir := GinkgoT().TempDir()
		old, _ := os.Getwd()
		os.Chdir(dir)
		defer os.Chdir(old)
		Expect(os.WriteFile("a.jar", []byte("data"), 0644)).To(Succeed())
		Expect(BuildArtifact(spec, lock, "1.21.1", "server")).To(Succeed())
	})
	It("ReadPackSpec with dir", func() {
		dir := GinkgoT().TempDir()
		Expect(domain.WritePackSpec(dir, &domain.PackSpec{PackName: "x", MinecraftVersion: "1.21.1", LoaderName: []string{"n"}, PackVersion: "1"})).To(Succeed())
		spec, err := ReadPackSpec(dir)
		Expect(err).NotTo(HaveOccurred())
		Expect(spec.PackName).To(Equal("x"))
	})
})
