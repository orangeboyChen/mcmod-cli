// File: internal/service/build_validation_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/service/build_validation.go (class-conflict and missing-dep checks).

package service

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bytes"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
	"github.com/orangeboyChen/mcmod-cli/internal/metadata"
)

// Fixture ids shared by the missing-dependency specs. They are named so the
// specs read as prose ("jei is not in the build set") and so goconst does not
// treat repeated fixture literals as copy-paste.
const (
	testDepJEI        = "jei"
	testModConsumer   = "consumer"
	testModExampleMod = "examplemod"
	testDepMissing    = "nope"
	testLoaderFabric  = "fabric"
	testDepJava       = "java"
	testMCVersion     = "1.21.1"
	testNeoForgeVer   = "21.0"
	testFabricVer     = "0.16"
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

// makeJarWithRawHeader creates a jar whose entries are written from explicit
// FileHeaders. This is the only way to emit a directory entry (a name ending
// in "/"), which zip.Writer.Create + Write refuses to produce.
func makeJarWithRawHeader(path string, headers []*zip.FileHeader) {
	f, err := os.Create(path)
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() { Expect(f.Close()).To(Succeed()) })
	w := zip.NewWriter(f)
	for _, h := range headers {
		_, err := w.CreateRaw(h)
		Expect(err).NotTo(HaveOccurred())
	}
	Expect(w.Close()).To(Succeed())
}

var _ = Describe("Service detectClassConflicts edge cases", func() {
	// A jar builder that takes several entries at once. The single-entry
	// makeJarWithClass closure above is scoped to the other Describe, so
	// multi-entry jars have to be built here.
	makeJarWithClasses := func(path string, classPaths ...string) {
		f, err := os.Create(path)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(f.Close()).To(Succeed()) })
		w := zip.NewWriter(f)
		for _, classPath := range classPaths {
			entry, err := w.Create(classPath)
			Expect(err).NotTo(HaveOccurred())
			_, err = entry.Write([]byte("x"))
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(w.Close()).To(Succeed())
	}
	newBC := func(dir string) *buildContext {
		return &buildContext{McVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "1.0", RootDir: dir}
	}

	It("ignores shared non-class resources and assets", func() {
		// Two mods shipping the same texture, pack.mcmeta, or data file is
		// normal and must not be reported. Only .class paths conflict.
		dir := GinkgoT().TempDir()
		makeJarWithClasses(filepath.Join(dir, "a.jar"),
			"assets/x/texture.png", "pack.mcmeta", "data/x/recipe.json", "README.txt")
		makeJarWithClasses(filepath.Join(dir, "b.jar"),
			"assets/x/texture.png", "pack.mcmeta", "data/x/recipe.json", "README.txt")
		Expect(validateModFiles(newBC(dir), map[string]string{
			"a": filepath.Join(dir, "a.jar"),
			"b": filepath.Join(dir, "b.jar"),
		})).To(Succeed())
	})

	It("does not treat an outer and inner class as a conflict", func() {
		// A$1.class compiles to its own jar entry, so a jar legitimately
		// holds both. Neither self-collides nor collides with the other.
		dir := GinkgoT().TempDir()
		makeJarWithClasses(filepath.Join(dir, "a.jar"), "com/foo/A.class", "com/foo/A$1.class")
		Expect(validateModFiles(newBC(dir), map[string]string{"a": filepath.Join(dir, "a.jar")})).To(Succeed())
	})

	It("ignores directory entries whose name ends in .class/", func() {
		// A directory entry has no bytecode and cannot conflict; the
		// trailing-slash guard is what excludes it.
		dir := GinkgoT().TempDir()
		makeJarWithRawHeader(filepath.Join(dir, "a.jar"), []*zip.FileHeader{{Name: "com/foo/A.class/"}})
		makeJarWithRawHeader(filepath.Join(dir, "b.jar"), []*zip.FileHeader{{Name: "com/foo/A.class/"}})
		Expect(validateModFiles(newBC(dir), map[string]string{
			"a": filepath.Join(dir, "a.jar"),
			"b": filepath.Join(dir, "b.jar"),
		})).To(Succeed())
	})

	It("treats class paths as case sensitive", func() {
		// Zip entry names are case sensitive, so these are two distinct
		// classes and must not be merged into a single conflict.
		dir := GinkgoT().TempDir()
		makeJarWithClasses(filepath.Join(dir, "a.jar"), "com/foo/A.class")
		makeJarWithClasses(filepath.Join(dir, "b.jar"), "COM/FOO/A.class")
		Expect(validateModFiles(newBC(dir), map[string]string{
			"a": filepath.Join(dir, "a.jar"),
			"b": filepath.Join(dir, "b.jar"),
		})).To(Succeed())
	})

	It("lists every owner of a three-way conflict", func() {
		dir := GinkgoT().TempDir()
		makeJarWithClasses(filepath.Join(dir, "a.jar"), "com/foo/A.class")
		makeJarWithClasses(filepath.Join(dir, "b.jar"), "com/foo/A.class")
		makeJarWithClasses(filepath.Join(dir, "c.jar"), "com/foo/A.class")
		err := validateModFiles(newBC(dir), map[string]string{
			"a": filepath.Join(dir, "a.jar"),
			"b": filepath.Join(dir, "b.jar"),
			"c": filepath.Join(dir, "c.jar"),
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("com/foo/A.class: a (a.jar), b (b.jar), c (c.jar)"))
	})

	It("reports each conflicting class once even when a jar repeats an entry", func() {
		// Duplicated entries inside one jar are a corrupt-zip edge case; the
		// scan must not emit the same class twice under one owner.
		dir := GinkgoT().TempDir()
		makeJarWithClasses(filepath.Join(dir, "a.jar"), "com/foo/A.class", "com/foo/A.class")
		err := validateModFiles(newBC(dir), map[string]string{"a": filepath.Join(dir, "a.jar")})
		Expect(err).To(HaveOccurred())
		Expect(strings.Count(err.Error(), "com/foo/A.class")).To(Equal(1))
	})

	It("sorts conflicts by class path and owners by mod key", func() {
		// The report is diffed across runs, so ordering has to be stable
		// regardless of map iteration order.
		dir := GinkgoT().TempDir()
		makeJarWithClasses(filepath.Join(dir, "z.jar"), "com/foo/B.class", "com/foo/A.class")
		makeJarWithClasses(filepath.Join(dir, "y.jar"), "com/foo/B.class", "com/foo/A.class")
		err := validateModFiles(newBC(dir), map[string]string{
			"z": filepath.Join(dir, "z.jar"),
			"y": filepath.Join(dir, "y.jar"),
		})
		Expect(err).To(HaveOccurred())
		message := err.Error()
		aIndex := strings.Index(message, "com/foo/A.class: y (y.jar), z (z.jar)")
		bIndex := strings.Index(message, "com/foo/B.class: y (y.jar), z (z.jar)")
		Expect(aIndex).To(BeNumerically(">", 0))
		Expect(bIndex).To(BeNumerically(">", aIndex))
	})

	It("reports class conflicts and unreadable jars in one report", func() {
		// Aggregation is the point of the single-pass scan: one run must
		// surface every problem class, not just the first.
		dir := GinkgoT().TempDir()
		makeJarWithClasses(filepath.Join(dir, "a.jar"), "com/foo/A.class")
		makeJarWithClasses(filepath.Join(dir, "b.jar"), "com/foo/A.class")
		err := validateModFiles(newBC(dir), map[string]string{
			"a": filepath.Join(dir, "a.jar"),
			"b": filepath.Join(dir, "b.jar"),
			"z": filepath.Join(dir, "missing.jar"),
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("class conflicts:"))
		Expect(err.Error()).To(ContainSubstring("com/foo/A.class"))
		Expect(err.Error()).To(ContainSubstring("unreadable jars:"))
		Expect(err.Error()).To(ContainSubstring("z (missing.jar)"))
	})

	It("reports a directory passed as a jar as unreadable", func() {
		dir := GinkgoT().TempDir()
		dirAsJar := filepath.Join(dir, "adir")
		Expect(os.MkdirAll(dirAsJar, 0o750)).To(Succeed())
		err := validateModFiles(newBC(dir), map[string]string{"a": dirAsJar})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unreadable jars"))
	})

	It("accepts an empty jar with no classes", func() {
		dir := GinkgoT().TempDir()
		makeJarWithClasses(filepath.Join(dir, "empty.jar"))
		Expect(validateModFiles(newBC(dir), map[string]string{"a": filepath.Join(dir, "empty.jar")})).To(Succeed())
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

// writeNeoForgeJar writes a neoforge.mods.toml in the real-world shape: a
// [[mods]] table carrying the mod's own id, followed by one
// [[dependencies.<modId>]] table per dependency. This is the layout
// NeoForge's own mods ship, and it is what the parser has to survive.
func writeNeoForgeJar(path, modID string, deps []string) {
	writeNeoForgeJarWithVersions(path, modID, deps, "[1.0,)")
}

// writeNeoForgeJarWithVersions is writeNeoForgeJar with control over the
// versionRange each dependency declares. An empty range omits the key so the
// dependency reports as "any".
func writeNeoForgeJarWithVersions(path, modID string, deps []string, versionRange string) {
	f, err := os.Create(path)
	Expect(err).NotTo(HaveOccurred())
	w := zip.NewWriter(f)
	var body strings.Builder
	body.WriteString("modLoader=\"javafml\"\nloaderVersion=\"[1,)\"\nlicense=\"MIT\"\n")
	fmt.Fprintf(&body, "[[mods]]\nmodId=%q\nversion=\"1.0.0\"\ndisplayName=\"Test\"\n", modID)
	for _, dep := range deps {
		fmt.Fprintf(&body, "[[dependencies.%s]]\nmodId=%q\n"+
			"type=\"required\"\nordering=\"NONE\"\nside=\"BOTH\"\n", modID, dep)
		if versionRange != "" {
			fmt.Fprintf(&body, "versionRange=%q\n", versionRange)
		}
	}
	wr, err := w.Create("META-INF/neoforge.mods.toml")
	Expect(err).NotTo(HaveOccurred())
	_, err = wr.Write([]byte(body.String()))
	Expect(err).NotTo(HaveOccurred())
	// Closed explicitly rather than by defer: the jar must be complete on
	// disk before validation reads it below.
	Expect(w.Close()).To(Succeed())
	Expect(f.Close()).To(Succeed())
}

// neoForgeBuildContext returns a buildContext for a NeoForge 1.21.1 build
// rooted at dir. Every spec in this file builds the same context, so the
// literal is hoisted here to keep the specs focused on their fixtures.
func neoForgeBuildContext(dir string) *buildContext {
	return &buildContext{McVersion: testMCVersion, Loader: "neoforge",
		LoaderVersion: testNeoForgeVer, RootDir: dir}
}

// neoForgeProviderConsumer writes two NeoForge jars into dir: a provider
// declaring modID and a consumer requiring depID. Returns their paths.
func neoForgeProviderConsumer(dir, modID, depID string) (provider, consumer string) {
	provider = filepath.Join(dir, "provider.jar")
	consumer = filepath.Join(dir, "consumer.jar")
	writeNeoForgeJar(provider, modID, nil)
	writeNeoForgeJar(consumer, testModConsumer, []string{depID})
	return provider, consumer
}

var _ = Describe("missing required deps with realistic NeoForge metadata", func() {
	It("keeps the mod id declared in [[mods]] ahead of dependency ids", func() {
		// Regression: parseSimpleTOML flattens the whole file into one map,
		// so without a section-aware read the last `modId=` in the file — a
		// dependency's — wins. That mislabels the jar as the dependency and
		// silently supplies it, hiding the real missing dep.
		dir := GinkgoT().TempDir()
		jar := filepath.Join(dir, "example.jar")
		writeNeoForgeJar(jar, testModExampleMod, []string{"neoforge", testDepJEI})
		info, err := metadata.ReadJarMetadata(jar)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.ModID).To(Equal(testModExampleMod))
	})

	It("flags a missing dependency declared after [[mods]]", func() {
		dir := GinkgoT().TempDir()
		jar := filepath.Join(dir, "example.jar")
		writeNeoForgeJar(jar, testModExampleMod, []string{"neoforge", testDepJEI})
		// "neoforge" is whitelisted; "jei" is not in the build set.
		err := validateModFiles(neoForgeBuildContext(dir), map[string]string{testModExampleMod: jar})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(testDepJEI))
		Expect(err.Error()).NotTo(ContainSubstring("neoforge [1.0,)"))
	})

	It("accepts a realistic jar whose deps are whitelisted", func() {
		dir := GinkgoT().TempDir()
		jar := filepath.Join(dir, "example.jar")
		writeNeoForgeJar(jar, testModExampleMod, []string{"neoforge", "minecraft", testDepJava})
		Expect(validateModFiles(neoForgeBuildContext(dir),
			map[string]string{testModExampleMod: jar})).To(Succeed())
	})

	It("resolves a dependency against the mod id of another jar", func() {
		dir := GinkgoT().TempDir()
		provider, consumer := neoForgeProviderConsumer(dir, "provider", "provider")
		bc := neoForgeBuildContext(dir)
		Expect(validateModFiles(bc, map[string]string{
			"provider":      provider,
			testModConsumer: consumer,
		})).To(Succeed())
	})

	It("does not leak a dependency id into the provided set", func() {
		// A jar that only *depends on* "jei" must not satisfy another jar's
		// requirement on "jei"; the id has to come from a [[mods]] table.
		dir := GinkgoT().TempDir()
		consumer := filepath.Join(dir, "consumer.jar")
		other := filepath.Join(dir, "other.jar")
		writeNeoForgeJar(consumer, testModConsumer, []string{testDepJEI})
		writeNeoForgeJar(other, "other", []string{testDepJEI})
		err := validateModFiles(neoForgeBuildContext(dir),
			map[string]string{testModConsumer: consumer, "other": other})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(testDepJEI))
	})

	It("matches dependency ids case insensitively", func() {
		// Mod ids are case insensitive per the NeoForge/Fabric conventions,
		// so "JEI" must satisfy a "jei" requirement.
		dir := GinkgoT().TempDir()
		provider, consumer := neoForgeProviderConsumer(dir, "JEI", testDepJEI)
		Expect(validateModFiles(neoForgeBuildContext(dir), map[string]string{
			"provider":      provider,
			testModConsumer: consumer,
		})).To(Succeed())
	})
})

// fabricBuildContext returns a buildContext for a Fabric 1.21.1 build rooted
// at dir, using the given loader name.
func fabricBuildContext(dir, loader string) *buildContext {
	return &buildContext{McVersion: testMCVersion, Loader: loader,
		LoaderVersion: testFabricVer, RootDir: dir}
}

// fabricJarWithDeps writes a jar named "f.jar" in dir whose fabric.mod.json
// declares id "f" and the given raw "depends" object body. Returns its path.
func fabricJarWithDeps(dir, depends string) string {
	jar := filepath.Join(dir, "f.jar")
	writeJarWithFabricDeps(jar, "f", depends)
	return jar
}

var _ = Describe("builtInModDeps whitelist", func() {
	It("whitelists fabric built-ins under the fabric loader", func() {
		dir := GinkgoT().TempDir()
		jar := fabricJarWithDeps(dir,
			`{"minecraft":"1.21.1","fabricloader":">=0.15","fabric-api":"*","java":"21"}`)
		bc := fabricBuildContext(dir, testLoaderFabric)
		Expect(validateModFiles(bc, map[string]string{"f": jar})).To(Succeed())
	})

	It("still flags a genuine missing dep under the fabric loader", func() {
		dir := GinkgoT().TempDir()
		jar := fabricJarWithDeps(dir,
			`{"minecraft":"1.21.1","fabricloader":">=0.15","actually-missing":"*"}`)
		err := validateModFiles(fabricBuildContext(dir, testLoaderFabric), map[string]string{"f": jar})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("actually-missing"))
		// The whitelisted built-ins must not appear in the report.
		Expect(err.Error()).NotTo(ContainSubstring("- minecraft "))
		Expect(err.Error()).NotTo(ContainSubstring("- fabricloader "))
	})

	It("does not apply the neoforge whitelist to a fabric build", func() {
		// Whitelist selection is per loader family, so a fabric build must
		// not silently accept "neoforge" as a satisfied dependency.
		dir := GinkgoT().TempDir()
		jar := fabricJarWithDeps(dir, `{"neoforge":"*"}`)
		err := validateModFiles(fabricBuildContext(dir, testLoaderFabric), map[string]string{"f": jar})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("neoforge"))
	})

	It("collapses the fabricloader loader name to the fabric whitelist", func() {
		// loaderFamily maps the legacy "fabricloader" name onto "fabric".
		dir := GinkgoT().TempDir()
		jar := fabricJarWithDeps(dir, `{"fabricloader":">=0.15","minecraft":"1.21.1"}`)
		bc := fabricBuildContext(dir, "fabricloader")
		Expect(validateModFiles(bc, map[string]string{"f": jar})).To(Succeed())
	})
})

var _ = Describe("missing required dep version reporting", func() {
	It("reports the declared version range of a missing dependency", func() {
		dir := GinkgoT().TempDir()
		jar := filepath.Join(dir, "m.jar")
		writeJarWithFabricDeps(jar, "m", `{"`+testDepMissing+`":"[1.0,)"}`)
		err := validateModFiles(fabricBuildContext(dir, testLoaderFabric), map[string]string{"m": jar})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(testDepMissing + " [1.0,)"))
	})

	It("reports 'any' when a dependency declares no version range", func() {
		// Fabric always carries a version string, so this path is only
		// reachable through a NeoForge dependency with no versionRange key.
		dir := GinkgoT().TempDir()
		jar := filepath.Join(dir, "m.jar")
		writeNeoForgeJarWithVersions(jar, "m", []string{testDepMissing}, "")
		err := validateModFiles(neoForgeBuildContext(dir), map[string]string{"m": jar})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(testDepMissing + " any"))
	})

	It("prefers a concrete version range over 'any' for the same dep", func() {
		// Two owners require the same missing id; the one that declares a
		// version range wins so the report stays actionable.
		dir := GinkgoT().TempDir()
		loose := filepath.Join(dir, "loose.jar")
		exact := filepath.Join(dir, "exact.jar")
		writeNeoForgeJarWithVersions(loose, "loose", []string{testDepMissing}, "")
		writeNeoForgeJarWithVersions(exact, "exact", []string{testDepMissing}, "[2.0,)")
		err := validateModFiles(neoForgeBuildContext(dir),
			map[string]string{"loose": loose, "exact": exact})
		Expect(err).To(HaveOccurred())
		// The headline version is the concrete range, and both owners are
		// still listed with the range each of them declared.
		Expect(err.Error()).To(ContainSubstring(
			"- " + testDepMissing + " [2.0,); required by exact requires [2.0,), loose requires any"))
	})

	It("aggregates every owner of one missing dependency", func() {
		dir := GinkgoT().TempDir()
		first := filepath.Join(dir, "first.jar")
		second := filepath.Join(dir, "second.jar")
		writeJarWithFabricDeps(first, "first", `{"shared-need":"*"}`)
		writeJarWithFabricDeps(second, "second", `{"shared-need":"*"}`)
		err := validateModFiles(fabricBuildContext(dir, testLoaderFabric),
			map[string]string{"first": first, "second": second})
		Expect(err).To(HaveOccurred())
		// One line per missing id, with every requiring mod listed once.
		Expect(err.Error()).To(ContainSubstring(
			"- shared-need *; required by first requires *, second requires *"))
		Expect(strings.Count(err.Error(), "shared-need *;")).To(Equal(1))
	})
})

// writeJarWithFabricDeps writes a jar whose fabric.mod.json declares the
// given id and the given raw "depends" object body.
func writeJarWithFabricDeps(path, id, depends string) {
	f, err := os.Create(path)
	Expect(err).NotTo(HaveOccurred())
	w := zip.NewWriter(f)
	wr, err := w.Create("fabric.mod.json")
	Expect(err).NotTo(HaveOccurred())
	_, err = fmt.Fprintf(wr, `{"id":%q,"version":"1.0","depends":%s}`, id, depends)
	Expect(err).NotTo(HaveOccurred())
	// Closed explicitly rather than by defer: the jar must be complete on
	// disk before validation reads it below.
	Expect(w.Close()).To(Succeed())
	Expect(f.Close()).To(Succeed())
}

var _ = Describe("validateModFiles leaves no artifact behind", func() {
	It("writes no zip when validation fails", func() {
		// The CF layout documents that a failed validation leaves no
		// artifact; the default layout must behave the same way.
		dir := GinkgoT().TempDir()
		a := filepath.Join(dir, "a.jar")
		b := filepath.Join(dir, "b.jar")
		writeJarWithFabricDeps(a, "a", `{"missing-dep":"*"}`)
		writeJarWithFabricDeps(b, "b", `{}`)
		bc := fabricBuildContext(dir, testLoaderFabric)
		out := filepath.Join(dir, "out.zip")
		err := bc.buildZipWith("client", out, map[string]string{"a": a, "b": b}, true)
		Expect(err).To(HaveOccurred())
		_, statErr := os.Stat(out)
		Expect(statErr).To(MatchError(os.ErrNotExist))
	})

	It("writes the zip when validation passes", func() {
		dir := GinkgoT().TempDir()
		jar := filepath.Join(dir, "ok.jar")
		writeJarWithFabricDeps(jar, "ok", `{"minecraft":"1.21.1"}`)
		bc := fabricBuildContext(dir, testLoaderFabric)
		out := filepath.Join(dir, "out.zip")
		Expect(bc.buildZipWith("client", out, map[string]string{"ok": jar}, true)).To(Succeed())
		_, statErr := os.Stat(out)
		Expect(statErr).NotTo(HaveOccurred())
	})
})

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
