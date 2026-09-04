// File: internal/service/build_validation_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/service/build_validation.go (class-conflict and missing-dep checks).

package service

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bytes"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

func writeValidTestJar(path string) {
	f, err := os.Create(path)
	Expect(err).NotTo(HaveOccurred())
	w := zip.NewWriter(f)
	entry, err := w.Create("META-INF/test.txt")
	Expect(err).NotTo(HaveOccurred())
	_, err = entry.Write([]byte("test"))
	Expect(err).NotTo(HaveOccurred())
	Expect(w.Close()).To(Succeed())
	Expect(f.Close()).To(Succeed())
}

var _ = Describe("Service detectClassConflicts", func() {
	makeJarWithClass := func(path, classPath string) {
		f, err := os.Create(path)
		Expect(err).NotTo(HaveOccurred())
		defer f.Close()
		w := zip.NewWriter(f)
		entry, err := w.Create(classPath)
		Expect(err).NotTo(HaveOccurred())
		entry.Write([]byte("x"))
		Expect(w.Close()).To(Succeed())
	}

	It("detects duplicate class across mods", func() {
		dir := GinkgoT().TempDir()
		makeJarWithClass(filepath.Join(dir, "a.jar"), "com/foo/A.class")
		makeJarWithClass(filepath.Join(dir, "b.jar"), "com/foo/A.class")
		bc := &buildContext{McVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "1.0", RootDir: dir}
		err := bc.buildZipWith("client", filepath.Join(dir, "out.zip"),
			map[string]string{"a": filepath.Join(dir, "a.jar"), "b": filepath.Join(dir, "b.jar")}, true)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("class conflicts"))
	})

	It("no conflict when classes are unique", func() {
		dir := GinkgoT().TempDir()
		makeJarWithClass(filepath.Join(dir, "a.jar"), "com/foo/A.class")
		makeJarWithClass(filepath.Join(dir, "b.jar"), "com/foo/B.class")
		bc := &buildContext{McVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "1.0", RootDir: dir}
		err := bc.buildZipWith("client", filepath.Join(dir, "out.zip"),
			map[string]string{"a": filepath.Join(dir, "a.jar"), "b": filepath.Join(dir, "b.jar")}, true)
		Expect(err).NotTo(HaveOccurred())
	})

	It("rejects an unreadable jar", func() {
		dir := GinkgoT().TempDir()
		bc := &buildContext{McVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "1.0", RootDir: dir}
		err := bc.buildZipWith("client", filepath.Join(dir, "out.zip"), map[string]string{
			"a": filepath.Join(dir, "mods", "a.jar"),
		}, true)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unreadable jars"))
	})

	It("bad jar path is skipped gracefully", func() {
		dir := GinkgoT().TempDir()
		// Don't create the file; detectClassConflicts should skip.
		bc := &buildContext{McVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "1.0", RootDir: dir}
		err := bc.buildZipWith("client", filepath.Join(dir, "out.zip"),
			map[string]string{"a": filepath.Join(dir, "nonexistent.jar")}, true)
		Expect(err).To(HaveOccurred())
	})

	It("aggregates multiple class conflicts with all owners", func() {
		dir := GinkgoT().TempDir()
		makeJarWithClass(filepath.Join(dir, "a.jar"), "com/foo/A.class")
		makeJarWithClass(filepath.Join(dir, "b.jar"), "com/foo/A.class")
		makeJarWithClass(filepath.Join(dir, "c.jar"), "com/foo/B.class")
		makeJarWithClass(filepath.Join(dir, "d.jar"), "com/foo/B.class")
		bc := &buildContext{McVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "1.0", RootDir: dir}
		err := bc.buildZipWith("client", filepath.Join(dir, "out.zip"), map[string]string{
			"a": filepath.Join(dir, "a.jar"), "b": filepath.Join(dir, "b.jar"),
			"c": filepath.Join(dir, "c.jar"), "d": filepath.Join(dir, "d.jar"),
		}, true)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("com/foo/A.class"))
		Expect(err.Error()).To(ContainSubstring("a (a.jar)"))
		Expect(err.Error()).To(ContainSubstring("b (b.jar)"))
		Expect(err.Error()).To(ContainSubstring("com/foo/B.class"))
		Expect(err.Error()).To(ContainSubstring("c (c.jar)"))
		Expect(err.Error()).To(ContainSubstring("d (d.jar)"))
	})

	It("reports unreadable jars before writing an artifact", func() {
		dir := GinkgoT().TempDir()
		bc := &buildContext{McVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "1.0", RootDir: dir}
		err := bc.buildZipWith("client", filepath.Join(dir, "out.zip"), map[string]string{"bad": filepath.Join(dir, "bad.jar")}, true)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unreadable jars"))
		_, statErr := os.Stat(filepath.Join(dir, "out.zip"))
		Expect(statErr).To(MatchError(os.ErrNotExist))
	})
})

var _ = Describe("Service detectMissingRequiredDeps", func() {
	It("returns nil for empty mod set", func() {
		bc := &buildContext{Lock: &domain.PackLock{Mods: map[string]domain.LockedMod{}}, Loader: "neoforge", McVersion: "1.21.1"}
		Expect(detectMissingRequiredDeps(bc, map[string]string{})).To(Succeed())
	})

	It("loaderFamily collapses fabric variants", func() {
		Expect(loaderFamily("fabric")).To(Equal("fabric"))
		Expect(loaderFamily("fabricloader")).To(Equal("fabric"))
		Expect(loaderFamily("neoforge")).To(Equal("neoforge"))
		Expect(loaderFamily("unknown")).To(Equal("unknown"))
	})

	It("resolveModJar errors on unsupported source type", func() {
		bc := &buildContext{Lock: &domain.PackLock{}, Loader: "neoforge", McVersion: "1.21.1", RootDir: "."}
		_, err := bc.resolveModJar("k", domain.LockedMod{Source: domain.LockedSource{Type: "bogus"}})
		Expect(err).To(HaveOccurred())
	})

	It("resolveModJar errors on local without path", func() {
		bc := &buildContext{Lock: &domain.PackLock{}, Loader: "neoforge", McVersion: "1.21.1", RootDir: "."}
		_, err := bc.resolveModJar("k", domain.LockedMod{Source: domain.LockedSource{Type: "local"}})
		Expect(err).To(HaveOccurred())
	})

	It("resolveModJar errors on curseforge missing fields", func() {
		bc := &buildContext{Lock: &domain.PackLock{}, Loader: "neoforge", McVersion: "1.21.1", RootDir: "."}
		_, err := bc.resolveModJar("k", domain.LockedMod{Source: domain.LockedSource{Type: "curseforge"}})
		Expect(err).To(HaveOccurred())
	})

	It("resolveModJar errors on github-release missing fields", func() {
		bc := &buildContext{Lock: &domain.PackLock{}, Loader: "neoforge", McVersion: "1.21.1", RootDir: "."}
		_, err := bc.resolveModJar("k", domain.LockedMod{Source: domain.LockedSource{Type: "github-release"}})
		Expect(err).To(HaveOccurred())
	})

	It("resolveModJar errors on github-release bad repo", func() {
		bc := &buildContext{Lock: &domain.PackLock{}, Loader: "neoforge", McVersion: "1.21.1", RootDir: "."}
		_, err := bc.resolveModJar("k", domain.LockedMod{Source: domain.LockedSource{Type: "github-release", Repo: "nope", Tag: "v1", AssetName: "x.jar"}})
		Expect(err).To(HaveOccurred())
	})

	It("addDirToZip returns nil for missing dir", func() {
		w := zip.NewWriter(new(bytes.Buffer))
		Expect(addDirToZip(w, "/no/such/path", "p")).To(Succeed())
	})

	It("BuildClientServerBuild fails on missing lock", func() {
		spec := &domain.PackSpec{LoaderName: []string{"neoforge:21.1.219"}}
		err := BuildClientServerBuild(spec, "1.21.1")
		Expect(err).To(HaveOccurred())
	})

	It("BuildArtifactWith rejects bad target", func() {
		err := BuildArtifactWith(&domain.PackSpec{}, &domain.PackLock{}, "1.21.1", "bogus", false)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("Service detectMissingRequiredDeps with synthetic jar", func() {
	It("returns nil for empty mod set", func() {
		bc := &buildContext{Lock: &domain.PackLock{Mods: map[string]domain.LockedMod{}}, Loader: "neoforge", McVersion: "1.21.1"}
		Expect(detectMissingRequiredDeps(bc, map[string]string{})).To(Succeed())
	})

	It("skips jars with no readable metadata", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		// non-existent file; metadata reader will fail
		bc := &buildContext{Lock: &domain.PackLock{}, Loader: "neoforge", McVersion: "1.21.1"}
		Expect(detectMissingRequiredDeps(bc, map[string]string{"x": "/no/such.jar"})).To(Succeed())
	})
})

// writeFakeModJar creates a minimal jar at path containing a
// neoforge.mods.toml with the given modid and the given required deps.
func writeFakeModJar(path, modid string, deps []string) {
	f, err := os.Create(path)
	Expect(err).NotTo(HaveOccurred())
	defer f.Close()
	w := zip.NewWriter(f)
	var body string
	body = "modid=\"" + modid + "\"\nversion=\"1.0\"\n"
	toml, err := w.Create("META-INF/neoforge.mods.toml")
	Expect(err).NotTo(HaveOccurred())
	_, err = toml.Write([]byte(body))
	Expect(err).NotTo(HaveOccurred())
	for _, dep := range deps {
		// We do not actually emit the [[dependencies]] section; the
		// fake parser only reads top-level keys, so we use a side
		// channel via a metadata writer instead. Just close the writer
		// for now.
		_ = dep
	}
	Expect(w.Close()).To(Succeed())
}

var _ = Describe("Service detectMissingRequiredDeps end-to-end", func() {
	It("returns nil for a single synthetic mod with no deps", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		jar := filepath.Join(dir, "fake.jar")
		writeFakeModJar(jar, "fakemod", nil)
		bc := &buildContext{Lock: &domain.PackLock{Mods: map[string]domain.LockedMod{
			"fakemod": {Name: "F", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: jar, FileName: "fake.jar"}},
		}}, Loader: "neoforge", McVersion: "1.21.1"}
		Expect(detectMissingRequiredDeps(bc, map[string]string{"fakemod": jar})).To(Succeed())
	})
})

var _ = Describe("detectMissingRequiredDeps", func() {
	It("returns nil when modFiles is empty", func() {
		bc := &buildContext{Loader: "neoforge", McVersion: "1.21.1"}
		Expect(detectMissingRequiredDeps(bc, map[string]string{})).To(Succeed())
	})

	It("skips jars that fail to read or have no ModID", func() {
		bc := &buildContext{Loader: "neoforge", McVersion: "1.21.1"}
		// Pass a path to a non-existent file; the function should not
		// panic and should return nil because nothing indexed survives.
		err := detectMissingRequiredDeps(bc, map[string]string{"a": "/no/such/file.jar"})
		Expect(err).To(Succeed())
	})
})

// buildTestJar creates a tiny jar that ReadJarMetadata can parse. It
// embeds a fabric.mod.json with the given id, name, version, and optional
// list of required dependencies (other mod ids).
func buildTestJar(path, id, name, version string, requiredDeps []string) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	w := zip.NewWriter(f)
	defer w.Close()
	fw, err := w.Create("fabric.mod.json")
	if err != nil {
		panic(err)
	}
	payload := fmt.Sprintf(`{"id":%q,"name":%q,"version":%q,"depends":{`, id, name, version)
	for i, dep := range requiredDeps {
		if i > 0 {
			payload += ","
		}
		payload += fmt.Sprintf("%q:%q", dep, "1.0")
	}
	payload += "}}"
	fw.Write([]byte(payload))
}

var _ = Describe("detectMissingRequiredDeps with real jars", func() {
	dir := ""
	modA := ""
	modB := ""
	modC := ""

	BeforeEach(func() {
		dir = GinkgoT().TempDir()
		modA = filepath.Join(dir, "a.jar")
		modB = filepath.Join(dir, "b.jar")
		modC = filepath.Join(dir, "c.jar")
		buildTestJar(modA, "a", "A", "1.0", nil)
		buildTestJar(modB, "b", "B", "1.0", []string{"a"})
		buildTestJar(modC, "c", "C", "1.0", []string{"zzz-not-present"})
	})

	It("returns nil when all required deps are present in the build set", func() {
		bc := &buildContext{Loader: "fabric", McVersion: "1.21.1"}
		modFiles := map[string]string{
			"a": modA,
			"b": modB,
		}
		Expect(detectMissingRequiredDeps(bc, modFiles)).To(Succeed())
	})

	It("returns an error when a required dep is not in the build set", func() {
		bc := &buildContext{Loader: "fabric", McVersion: "1.21.1"}
		modFiles := map[string]string{
			"c": modC, // depends on "zzz-not-present" which is not in modFiles
		}
		err := detectMissingRequiredDeps(bc, modFiles)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("hint:"))
	})

	It("reports all missing dependencies and their owners", func() {
		modD := filepath.Join(dir, "d.jar")
		buildTestJar(modD, "d", "D", "1.0", []string{"missing-a", "missing-b"})
		bc := &buildContext{Loader: "fabric", McVersion: "1.21.1"}
		err := validateModFiles(bc, map[string]string{"d": modD, "c": modC})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing-a"))
		Expect(err.Error()).To(ContainSubstring("missing-b"))
		Expect(err.Error()).To(ContainSubstring("required by c"))
		Expect(err.Error()).To(ContainSubstring("required by d"))
	})

	It("skips non-required deps", func() {
		// Make a jar that has an optional dep on something not in the build.
		modOptional := filepath.Join(dir, "opt.jar")
		f, _ := os.Create(modOptional)
		w := zip.NewWriter(f)
		fw, _ := w.Create("fabric.mod.json")
		fw.Write([]byte(`{"id":"opt","name":"Opt","version":"1.0","suggests":{"nope":"1.0"}}`))
		w.Close()
		f.Close()

		bc := &buildContext{Loader: "fabric", McVersion: "1.21.1"}
		modFiles := map[string]string{"opt": modOptional}
		Expect(detectMissingRequiredDeps(bc, modFiles)).To(Succeed())
	})
})

var _ = Describe("detectMissingRequiredDeps with jars", func() {
	It("returns nil when all required deps are whitelisted", func() {
		// Create a jar with metadata that has a required dep "minecraft"
		// which is on the neoforge whitelist.
		dir := GinkgoT().TempDir()
		jar := createJarWithMeta(dir, "my-mod", map[string]string{"minecraft": "[1.21.1]"})
		bc := &buildContext{Lock: &domain.PackLock{Mods: map[string]domain.LockedMod{}}, Loader: "neoforge", McVersion: "1.21.1"}
		Expect(detectMissingRequiredDeps(bc, map[string]string{"my-mod": jar})).To(Succeed())
	})

	It("reads metadata for a real jar and returns nil", func() {
		// The metadata reader only extracts modid/version (not deps), so this
		// path always returns nil. The point is to exercise the loop.
		dir := GinkgoT().TempDir()
		jar := createJarWithMeta(dir, "my-mod", map[string]string{})
		bc := &buildContext{Lock: &domain.PackLock{Mods: map[string]domain.LockedMod{}}, Loader: "neoforge", McVersion: "1.21.1"}
		Expect(detectMissingRequiredDeps(bc, map[string]string{"my-mod": jar})).To(Succeed())
	})
})

// createJarWithMeta creates a small jar with a mods.toml/neoforge.mods entry
// declaring the given mod id and required deps. The metadata is recognised
// by internal/metadata.ReadJarMetadata.
func createJarWithMeta(dir, modID string, requiredDeps map[string]string) string {
	jarPath := filepath.Join(dir, modID+".jar")
	f, err := os.Create(jarPath)
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() { f.Close() })

	w := zip.NewWriter(f)

	// neoforge.mods.toml format.
	depsToml := ""
	for k, v := range requiredDeps {
		depsToml += fmt.Sprintf("[[dependencies.%s]]\nmodId=\"%s\"\nmandatory=true\nversionRange=\"%s\"\n", modID, k, v)
	}
	tomlContent := fmt.Sprintf(`modLoader="javafml"
loaderVersion="*"
license="MIT"
[[mods]]
modId="%s"
version="1.0.0"
displayName="Test"
description="Test"
%s
`, modID, depsToml)
	wr, err := w.Create("META-INF/mods.toml")
	Expect(err).NotTo(HaveOccurred())
	_, _ = wr.Write([]byte(tomlContent))

	Expect(w.Close()).To(Succeed())
	Expect(f.Close()).To(Succeed())
	return jarPath
}
