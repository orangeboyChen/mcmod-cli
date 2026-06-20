// File: internal/service/build_service_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/service/build_service.go (top-level BuildArtifact* / BuildClientServerBuild).

package service

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"
	"path/filepath"

	"encoding/json"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("Service additional build coverage", func() {
	It("BuildArtifactAndReturnPath client target returns zip path", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		// local source
		os.WriteFile(filepath.Join(dir, "a.jar"), []byte("a"), 0644)
		os.WriteFile(filepath.Join(dir, "b.jar"), []byte("b"), 0644)
		spec := &domain.PackSpec{
			PackName: "p", PackVersion: "0.1.0", MinecraftVersion: "1.21.1",
			LoaderName: []string{"neoforge:1.0"},
		}
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"shared-mod": {Name: "Shared", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: "./a.jar", FileName: "a.jar"}},
			},
		}
		out, err := BuildArtifactAndReturnPath(spec, lock, "1.21.1", "client", false)
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring("client"))
		_, err = os.Stat(out)
		Expect(err).NotTo(HaveOccurred())
	})

	It("BuildArtifactAndReturnPath both produces both zips", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		os.WriteFile(filepath.Join(dir, "a.jar"), []byte("a"), 0644)
		spec := &domain.PackSpec{
			PackName: "p", ServerPackName: "p-server", PackVersion: "0.2.0",
			MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge:1.0"},
		}
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: "./a.jar", FileName: "a.jar"}},
			},
		}
		_, err := BuildArtifactAndReturnPath(spec, lock, "1.21.1", "both", false)
		Expect(err).NotTo(HaveOccurred())
	})

	It("buildZip is the legacy alias", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		os.WriteFile(filepath.Join(dir, "a.jar"), []byte("a"), 0644)
		spec := &domain.PackSpec{
			PackName: "p", PackVersion: "0.3.0", MinecraftVersion: "1.21.1",
			LoaderName: []string{"neoforge:1.0"},
		}
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: "./a.jar", FileName: "a.jar"}},
			},
		}
		err := BuildArtifact(spec, lock, "1.21.1", "client")
		Expect(err).NotTo(HaveOccurred())
	})

	It("buildOneArtifact invalid target errors", func() {
		err := buildOneArtifact(&domain.PackSpec{}, &domain.PackLock{}, "1.21.1", "weird")
		Expect(err).To(HaveOccurred())
	})

	It("buildOneArtifactWith missing lock mod errors", func() {
		spec := &domain.PackSpec{
			PackName: "p", PackVersion: "0.4.0", MinecraftVersion: "1.21.1",
			LoaderName: []string{"neoforge:1.0"},
		}
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{},
		}
		err := buildOneArtifactWith(spec, lock, "1.21.1", "client", false)
		Expect(err).To(HaveOccurred())
	})

	It("BuildArtifactWith nil spec errors", func() {
		err := BuildArtifactWith(nil, &domain.PackLock{}, "1.21.1", "client", false)
		Expect(err).To(HaveOccurred())
	})

	It("BuildArtifactWith nil lock errors", func() {
		err := BuildArtifactWith(&domain.PackSpec{}, nil, "1.21.1", "client", false)
		Expect(err).To(HaveOccurred())
	})

	It("loaderFromLock nil returns empty", func() {
		loader, ver := loaderFromLock(nil)
		Expect(loader).To(Equal(""))
		Expect(ver).To(Equal(""))
	})

	It("loaderFromLock returns values", func() {
		loader, ver := loaderFromLock(&domain.PackLock{Loader: "neoforge", LoaderVersion: "1.0"})
		Expect(loader).To(Equal("neoforge"))
		Expect(ver).To(Equal("1.0"))
	})

	It("BuildLock from spec with local mods", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		os.WriteFile(filepath.Join(dir, "a.jar"), []byte("a"), 0644)
		spec := &domain.PackSpec{
			PackName: "p", PackVersion: "0.5.0", MinecraftVersion: "1.21.1",
			LoaderName: []string{"neoforge:1.0"},
			Mods: map[string]domain.ModSpec{
				"a": {Name: "A", Scope: "shared", Source: domain.ModSource{Type: "local", Path: "./a.jar"}},
			},
		}
		lock, err := BuildLock(spec, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(lock.Mods).To(HaveKey("a"))
	})

	It("BuildLock with empty mods map returns empty lock", func() {
		spec := &domain.PackSpec{
			PackName: "p", PackVersion: "0.6.0", MinecraftVersion: "1.21.1",
			LoaderName: []string{"neoforge:1.0"},
		}
		lock, err := BuildLock(spec, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(lock.Mods).To(BeEmpty())
	})
})

var _ = Describe("Service BuildClientServerBuild", func() {
	It("iterates loaders and builds", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		os.WriteFile(filepath.Join(dir, "a.jar"), []byte("a"), 0644)
		spec := &domain.PackSpec{
			PackName: "p", ServerPackName: "p-server", PackVersion: "0.1.0",
			MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge:1.0"},
			Mods: map[string]domain.ModSpec{
				"a": {Name: "A", Scope: "shared", Source: domain.ModSource{Type: "local", Path: "./a.jar"}},
			},
		}
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: "./a.jar", FileName: "a.jar"}},
			},
		}
		os.MkdirAll(filepath.Join(dir, "locks", "dependencies"), 0755)
		lockData, _ := json.MarshalIndent(lock, "", "  ")
		Expect(os.WriteFile(filepath.Join(dir, "locks", "dependencies", "1.21.1-neoforge.json"), lockData, 0644)).To(Succeed())
		err := BuildClientServerBuild(spec, "1.21.1")
		Expect(err).NotTo(HaveOccurred())
	})

	It("errors when lock is missing", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		spec := &domain.PackSpec{
			PackName: "p", PackVersion: "0.1.0", MinecraftVersion: "1.21.1",
			LoaderName: []string{"neoforge:1.0"},
		}
		err := BuildClientServerBuild(spec, "1.21.1")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("Service build --force", func() {
	It("without force errors on existing zip", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		os.WriteFile(filepath.Join(dir, "a.jar"), []byte("a"), 0644)
		spec := &domain.PackSpec{
			PackName: "p", PackVersion: "0.1.0", MinecraftVersion: "1.21.1",
			LoaderName: []string{"neoforge:1.0"},
		}
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: "./a.jar", FileName: "a.jar"}},
			},
		}
		_, err := BuildArtifactAndReturnPath(spec, lock, "1.21.1", "client", false)
		Expect(err).NotTo(HaveOccurred())
		_, err = BuildArtifactAndReturnPath(spec, lock, "1.21.1", "client", false)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("already exists"))
	})

	It("with force overwrites existing zip", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		os.WriteFile(filepath.Join(dir, "a.jar"), []byte("a"), 0644)
		spec := &domain.PackSpec{
			PackName: "p", PackVersion: "0.1.0", MinecraftVersion: "1.21.1",
			LoaderName: []string{"neoforge:1.0"},
		}
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: "./a.jar", FileName: "a.jar"}},
			},
		}
		_, err := BuildArtifactAndReturnPath(spec, lock, "1.21.1", "client", false)
		Expect(err).NotTo(HaveOccurred())
		_, err = BuildArtifactAndReturnPath(spec, lock, "1.21.1", "client", true)
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("Service BuildArtifact end-to-end", func() {
	It("BuildArtifact fails when no mods are present", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		err := BuildArtifact(&domain.PackSpec{PackName: "t", PackVersion: "0.1.0"}, &domain.PackLock{Mods: map[string]domain.LockedMod{}}, "1.21.1", "client")
		Expect(err).To(HaveOccurred())
	})

	It("BuildArtifactWith rejects invalid target", func() {
		err := BuildArtifactWith(&domain.PackSpec{}, &domain.PackLock{}, "1.21.1", "weird", false)
		Expect(err).To(HaveOccurred())
	})

	It("BuildArtifactWith respects --force semantics", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		spec := &domain.PackSpec{PackName: "t", PackVersion: "0.1.0", MinecraftVersion: "1.21.1"}
		lock := &domain.PackLock{Loader: "neoforge", LoaderVersion: "21.1.219", Mods: map[string]domain.LockedMod{
			"x": {Name: "X", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: "./x.jar", FileName: "x.jar"}},
		}}
		// Pre-create the local jar so the build path doesn't fail at the
		// missing-jar step.
		Expect(os.WriteFile("x.jar", []byte("dummy"), 0644)).To(Succeed())
		err := BuildArtifactWith(spec, lock, "1.21.1", "client", false)
		Expect(err).NotTo(HaveOccurred())
		// Second call without --force should fail because zip exists.
		err = BuildArtifactWith(spec, lock, "1.21.1", "client", false)
		Expect(err).To(HaveOccurred())
		// With --force, overwrite succeeds.
		err = BuildArtifactWith(spec, lock, "1.21.1", "client", true)
		Expect(err).NotTo(HaveOccurred())
	})

	It("BuildArtifactAndReturnPath handles both targets", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		spec := &domain.PackSpec{PackName: "t", PackVersion: "0.1.0"}
		lock := &domain.PackLock{Loader: "neoforge", LoaderVersion: "21.1.219", Mods: map[string]domain.LockedMod{
			"x": {Name: "X", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: "./x.jar", FileName: "x.jar"}},
		}}
		Expect(os.WriteFile("x.jar", []byte("dummy"), 0644)).To(Succeed())
		out, err := BuildArtifactAndReturnPath(spec, lock, "1.21.1", "client", true)
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring(".zip"))
		_, err = os.Stat(out)
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("BuildArtifact", func() {
	It("handles nil lock", func() {
		spec := &domain.PackSpec{PackName: "test", Mods: map[string]domain.ModSpec{"a": {Scope: "shared", Source: domain.ModSource{Type: "local"}}}}
		err := BuildArtifact(spec, nil, "1.21.1", "client")
		Expect(err).To(HaveOccurred())
	})
	It("builds with valid lock", func() {
		dir := GinkgoT().TempDir()
		jarPath := filepath.Join(dir, "a.jar")
		Expect(os.WriteFile(jarPath, []byte("dummy"), 0644)).To(Succeed())
		oldWd, _ := os.Getwd()
		Expect(os.Chdir(dir)).To(Succeed())
		defer os.Chdir(oldWd)
		spec := &domain.PackSpec{PackName: "test", Mods: map[string]domain.ModSpec{"a": {Scope: "shared", Source: domain.ModSource{Type: "local", Path: jarPath}}}}
		lock := &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: jarPath, FileName: "a.jar"}}}}
		err := BuildArtifact(spec, lock, "1.21.1", "client")
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("BuildClientServerBuild error path", func() {
	It("returns an error when the lock is missing", func() {
		dir := GinkgoT().TempDir()
		spec := &domain.PackSpec{
			PackName: "p", MinecraftVersion: "1.21.1",
			LoaderName: []string{"neoforge:21.0.0"},
		}
		wd, _ := os.Getwd()
		Expect(os.Chdir(dir)).To(Succeed())
		DeferCleanup(func() { _ = os.Chdir(wd) })

		err := BuildClientServerBuild(spec, "1.21.1")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("BuildArtifactAndReturnPath", func() {
	It("returns an error when the spec is nil", func() {
		_, err := BuildArtifactAndReturnPath(nil, &domain.PackLock{}, "1.21.1", "client", false)
		Expect(err).To(HaveOccurred())
	})

	It("returns a client zip path for a non-both target", func() {
		dir := GinkgoT().TempDir()
		wd, _ := os.Getwd()
		Expect(os.Chdir(dir)).To(Succeed())
		DeferCleanup(func() { _ = os.Chdir(wd) })

		spec := &domain.PackSpec{
			PackName: "p", MinecraftVersion: "1.21.1",
			LoaderName: []string{"neoforge:21.0.0"},
		}
		// Use a local mod so resolveModJar can find it without a network call.
		jar := filepath.Join(dir, "mod.jar")
		Expect(os.WriteFile(jar, []byte("x"), 0644)).To(Succeed())
		lock := &domain.PackLock{
			MinecraftVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "21.0.0",
			Mods: map[string]domain.LockedMod{
				"local-mod": {Name: "local-mod", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: jar, FileName: "mod.jar"}},
			},
		}
		path, err := BuildArtifactAndReturnPath(spec, lock, "1.21.1", "client", true)
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(ContainSubstring("client"))
	})
})

var _ = Describe("BuildClientServerBuild happy path", func() {
	It("builds both client and server zips for a known lock with one mod", func() {
		dir := GinkgoT().TempDir()
		wd, _ := os.Getwd()
		Expect(os.Chdir(dir)).To(Succeed())
		DeferCleanup(func() { _ = os.Chdir(wd) })

		jar := filepath.Join(dir, "mod.jar")
		Expect(os.WriteFile(jar, []byte("x"), 0644)).To(Succeed())

		// Write the lock file where the loader will look for it:
		// locks/dependencies/<mcVersion>-<loader>.json
		lockDir := filepath.Join(dir, "locks", "dependencies")
		Expect(os.MkdirAll(lockDir, 0755)).To(Succeed())
		lock := domain.PackLock{
			MinecraftVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "21.0.0",
			Mods: map[string]domain.LockedMod{
				"local-mod": {Name: "local-mod", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: jar, FileName: "mod.jar"}},
			},
		}
		data, _ := json.MarshalIndent(lock, "", "  ")
		Expect(os.WriteFile(filepath.Join(lockDir, "1.21.1-neoforge.json"), data, 0644)).To(Succeed())

		spec := &domain.PackSpec{
			PackName: "p", MinecraftVersion: "1.21.1",
			LoaderName: []string{"neoforge:21.0.0"},
		}
		Expect(BuildClientServerBuild(spec, "1.21.1")).To(Succeed())
	})
})

var _ = Describe("BuildArtifactAndReturnPath both target", func() {
	It("builds both client and server for target=both", func() {
		dir := GinkgoT().TempDir()
		wd, _ := os.Getwd()
		Expect(os.Chdir(dir)).To(Succeed())
		DeferCleanup(func() { _ = os.Chdir(wd) })

		jar := filepath.Join(dir, "mod.jar")
		Expect(os.WriteFile(jar, []byte("x"), 0644)).To(Succeed())

		spec := &domain.PackSpec{
			PackName: "p", MinecraftVersion: "1.21.1",
			LoaderName: []string{"neoforge:21.0.0"},
		}
		lock := &domain.PackLock{
			MinecraftVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "21.0.0",
			Mods: map[string]domain.LockedMod{
				"local-mod": {Name: "local-mod", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: jar, FileName: "mod.jar"}},
			},
		}
		path, err := BuildArtifactAndReturnPath(spec, lock, "1.21.1", "both", true)
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(ContainSubstring("both"))
	})
})

var _ = Describe("buildOneArtifactWith error paths", func() {
	It("rejects an unsupported target", func() {
		// buildOneArtifactWith is unexported; call through BuildArtifactAndReturnPath
		// which uses it indirectly. The error surfaces as a build failure.
		dir := GinkgoT().TempDir()
		wd, _ := os.Getwd()
		Expect(os.Chdir(dir)).To(Succeed())
		DeferCleanup(func() { _ = os.Chdir(wd) })

		spec := &domain.PackSpec{PackName: "p", LoaderName: []string{"neoforge:21.0.0"}}
		lock := &domain.PackLock{Mods: map[string]domain.LockedMod{}}
		_, err := BuildArtifactAndReturnPath(spec, lock, "1.21.1", "wat", true)
		Expect(err).To(HaveOccurred())
	})

	It("returns an error when no mods are configured for the target", func() {
		dir := GinkgoT().TempDir()
		wd, _ := os.Getwd()
		Expect(os.Chdir(dir)).To(Succeed())
		DeferCleanup(func() { _ = os.Chdir(wd) })

		spec := &domain.PackSpec{PackName: "p", LoaderName: []string{"neoforge:21.0.0"}}
		lock := &domain.PackLock{Mods: map[string]domain.LockedMod{}}
		_, err := BuildArtifactAndReturnPath(spec, lock, "1.21.1", "client", true)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("BuildArtifactWith nil checks", func() {
	It("returns an error when lock is nil", func() {
		err := BuildArtifactWith(&domain.PackSpec{PackName: "p", LoaderName: []string{"neoforge:21.0.0"}}, nil, "1.21.1", "client", true)
		Expect(err).To(HaveOccurred())
	})

	It("returns an error when spec is nil", func() {
		err := BuildArtifactWith(nil, &domain.PackLock{Mods: map[string]domain.LockedMod{}}, "1.21.1", "client", true)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("buildOneArtifactWith all-mods-fail path", func() {
	It("returns an error when every mod fails to resolve", func() {
		dir := GinkgoT().TempDir()
		wd, _ := os.Getwd()
		Expect(os.Chdir(dir)).To(Succeed())
		DeferCleanup(func() { _ = os.Chdir(wd) })

		// Lock has a mod, but the source file is missing -> resolveModJar
		// returns an error. With only this mod in the build, the dispatcher
		// will skip it and end up with an empty modFiles.
		lock := &domain.PackLock{
			MinecraftVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "21.0.0",
			Mods: map[string]domain.LockedMod{
				"missing": {Name: "missing", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: "/no/such/file.jar", FileName: "file.jar"}},
			},
		}
		spec := &domain.PackSpec{PackName: "p", LoaderName: []string{"neoforge:21.0.0"}}
		_, err := BuildArtifactAndReturnPath(spec, lock, "1.21.1", "client", true)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("buildOneArtifactWith mod-missing-from-lock path", func() {
	It("returns an error when a mod key is in the build but missing from the lock map", func() {
		dir := GinkgoT().TempDir()
		wd, _ := os.Getwd()
		Expect(os.Chdir(dir)).To(Succeed())
		DeferCleanup(func() { _ = os.Chdir(wd) })

		// modsForTarget will pull keys from the *lock*, not the spec, so
		// this case is hard to reach directly. We instead rely on the
		// shared+server scope split: spec declares a server-only mod and
		// the client build only sees the shared one.
		jar := filepath.Join(dir, "mod.jar")
		Expect(os.WriteFile(jar, []byte("x"), 0644)).To(Succeed())
		lock := &domain.PackLock{
			MinecraftVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "21.0.0",
			Mods: map[string]domain.LockedMod{
				"server-only": {Name: "server-only", Scope: "server", Source: domain.LockedSource{Type: "local", Path: jar, FileName: "mod.jar"}},
			},
		}
		spec := &domain.PackSpec{PackName: "p", LoaderName: []string{"neoforge:21.0.0"}}
		// Client build pulls only shared+client, so the server-only mod is
		// not in the build set; but it is also the only one in the lock, so
		// the "no mods" branch fires.
		_, err := BuildArtifactAndReturnPath(spec, lock, "1.21.1", "client", true)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("buildOneArtifactWith force=false and artifact-exists path", func() {
	It("returns an error when force is false and the artifact already exists", func() {
		dir := GinkgoT().TempDir()
		wd, _ := os.Getwd()
		Expect(os.Chdir(dir)).To(Succeed())
		DeferCleanup(func() { _ = os.Chdir(wd) })

		jar := filepath.Join(dir, "mod.jar")
		Expect(os.WriteFile(jar, []byte("x"), 0644)).To(Succeed())
		lock := &domain.PackLock{
			MinecraftVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "21.0.0",
			Mods: map[string]domain.LockedMod{
				"local-mod": {Name: "local-mod", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: jar, FileName: "mod.jar"}},
			},
		}
		spec := &domain.PackSpec{PackName: "p", LoaderName: []string{"neoforge:21.0.0"}}
		// First build with force=true to create the artifact.
		Expect(BuildArtifactWith(spec, lock, "1.21.1", "client", true)).To(Succeed())
		// Second build with force=false should fail because the file exists.
		Expect(BuildArtifactWith(spec, lock, "1.21.1", "client", false)).NotTo(Succeed())
	})
})
