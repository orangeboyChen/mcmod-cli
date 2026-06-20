// File: internal/service/build_service.go
// Created: 2026-06-20
// Description: Build artifact generation service.

package service

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/orangeboyChen/mcmod-cli/internal/cache"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
	"github.com/orangeboyChen/mcmod-cli/internal/downloader"
	"github.com/orangeboyChen/mcmod-cli/internal/metadata"
)

// buildContext carries resolved information for one (mc, loader) build run.
type buildContext struct {
	Spec          *domain.PackSpec
	Lock          *domain.PackLock
	McVersion     string
	Loader        string
	LoaderVersion string
	RootDir       string
}

// resolveModJar returns the on-disk path to the mod jar for a given lock entry.
// local sources use the spec's path directly; remote sources look in .cache.
func (bc *buildContext) resolveModJar(key string, locked domain.LockedMod) (string, error) {
	src := locked.Source
	switch src.Type {
	case "local":
		var path string
		if bc.Spec != nil {
			if mod, ok := bc.Spec.Mods[key]; ok {
				path = mod.Source.Path
			}
		}
		if path == "" {
			path = src.Path
		}
		// Fallback: if the path is empty but FileName is set, try to find it in
		// .cache/local/ and the project root.
		if path == "" && src.FileName != "" {
			candidates := []string{
				filepath.Join(bc.RootDir, ".cache", "local", src.FileName),
				filepath.Join(bc.RootDir, src.FileName),
			}
			for _, c := range candidates {
				if _, err := os.Stat(c); err == nil {
					path = c
					break
				}
			}
		}
		if path == "" {
			return "", fmt.Errorf("mod %s: local source missing path", key)
		}
		// Replace template variables in path
		path = strings.ReplaceAll(path, "{mcVersion}", bc.McVersion)
		path = strings.ReplaceAll(path, "{loader}", bc.Loader)
		if !filepath.IsAbs(path) {
			path = filepath.Join(bc.RootDir, path)
		}
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("mod %s: local file %q not found: %w", key, path, err)
		}
		return path, nil
	case "curseforge":
		if src.ModID == 0 || src.FileID == 0 || src.FileName == "" {
			return "", fmt.Errorf("mod %s: curseforge source missing modId/fileId/fileName in lock", key)
		}
		p := filepath.Join(bc.RootDir, ".cache", "curseforge", fmt.Sprint(src.ModID), fmt.Sprint(src.FileID), src.FileName)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		if err := bc.populateCache(key, &src); err != nil {
			return "", fmt.Errorf("mod %s: %w", key, err)
		}
		return p, nil
	case "github-release":
		if src.Repo == "" || src.Tag == "" || src.AssetName == "" {
			return "", fmt.Errorf("mod %s: github-release source missing repo/tag/assetName in lock", key)
		}
		parts := strings.SplitN(src.Repo, "/", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("mod %s: invalid github repo %q", key, src.Repo)
		}
		p := filepath.Join(bc.RootDir, ".cache", "github-release", parts[0], parts[1], src.Tag, src.AssetName)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		if err := bc.populateCache(key, &src); err != nil {
			return "", fmt.Errorf("mod %s: %w", key, err)
		}
		return p, nil
	default:
		return "", fmt.Errorf("mod %s: unsupported source type %q in lock", key, src.Type)
	}
}

// populateCache downloads a remote mod jar into the local .cache/ tree on
// demand. It is called when resolveModJar cannot find a cached file so the
// build step stays self-contained (lock is purely a resolve step).
func (bc *buildContext) populateCache(key string, src *domain.LockedSource) error {
	if err := cache.EnsureCacheDir(); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "build: caching mod %s from %s\n", key, src.Type)
	if err := downloader.Download(src, key); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "build: cached mod %s\n", key)
	return nil
}

// modsForTarget returns mod keys that should be packaged for the given target.
// client: shared + client; server: shared + server.
func (bc *buildContext) modsForTarget(target string) []string {
	keys := make([]string, 0)
	for key, locked := range bc.Lock.Mods {
		scope := locked.Scope
		if scope == "" {
			scope = "shared"
		}
		if target == "client" {
			if scope == "shared" || scope == "client" {
				keys = append(keys, key)
			}
		} else if target == "server" {
			if scope == "shared" || scope == "server" {
				keys = append(keys, key)
			}
		}
	}
	return keys
}

// zipBaseName returns the zip basename prefix for a target build.
// Per spec 7.5.4/7.5.5, client uses packName; server uses serverPackName (or packName fallback).
func (bc *buildContext) zipBaseName(target string) string {
	if target == "server" {
		name := bc.Spec.ServerPackName
		if name == "" {
			name = bc.Spec.PackName
		}
		return name
	}
	return bc.Spec.PackName
}

// zipPath returns the full output zip path for a target build.
func (bc *buildContext) zipPath(target string) string {
	base := bc.zipBaseName(target)
	loaderVer := bc.LoaderVersion
	if loaderVer == "" {
		loaderVer = "0"
	}
	fname := fmt.Sprintf("%s-%s-%s-%s-%s.zip", base, bc.McVersion, bc.Loader, loaderVer, target)
	return filepath.Join(bc.RootDir, "releases", "v"+bc.Spec.PackVersion, fname)
}

// buildZip writes the zip at the given path. It zips the resolved jar files
// under mods/<fileName>, plus any optional config/, defaultconfigs/ (server),
// resourcepacks/ (client), server.properties/whitelist.json/ops.json (server).
func (bc *buildContext) buildZip(target, path string, modFiles map[string]string) error {
	return bc.buildZipWith(target, path, modFiles, false)
}

// buildZipWith writes the zip at the given path. When force is false and the
// target file already exists, it returns an error instead of overwriting.
func (bc *buildContext) buildZipWith(target, path string, modFiles map[string]string, force bool) error {
	if err := detectClassConflicts(modFiles); err != nil {
		return err
	}
	if err := detectMissingRequiredDeps(bc, modFiles); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("artifact already exists at %s (use --force to overwrite)", path)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := zip.NewWriter(f)
	defer w.Close()

	// mods/<fileName> in deterministic key order
	keys := make([]string, 0, len(modFiles))
	for k := range modFiles {
		keys = append(keys, k)
	}
	// stable sort by key
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	for _, key := range keys {
		jarPath := modFiles[key]
		baseName := filepath.Base(jarPath)
		if err := addFileToZip(w, jarPath, "mods/"+baseName); err != nil {
			return fmt.Errorf("add mod %s: %w", key, err)
		}
	}

	// Optional config/ always
	if err := addDirToZip(w, filepath.Join(bc.RootDir, "config"), "config"); err != nil {
		return err
	}
	// defaultconfigs/ for server, optional for client
	if target == "server" {
		if err := addDirToZip(w, filepath.Join(bc.RootDir, "defaultconfigs"), "defaultconfigs"); err != nil {
			return err
		}
	}
	// resourcepacks/ for client only
	if target == "client" {
		if err := addDirToZip(w, filepath.Join(bc.RootDir, "resourcepacks"), "resourcepacks"); err != nil {
			return err
		}
	}
	// server-only files
	if target == "server" {
		for _, fname := range []string{"server.properties", "whitelist.json", "ops.json"} {
			_ = addFileToZip(w, filepath.Join(bc.RootDir, fname), fname)
		}
	}
	return nil
}

// addFileToZip adds a single file to the zip under the given entry name.
// If the source file is missing it returns nil (optional files).
func addFileToZip(w *zip.Writer, src, entry string) error {
	if _, err := os.Stat(src); err != nil {
		return nil
	}
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	hdr.Name = entry
	hdr.Method = zip.Deflate
	wr, err := w.CreateHeader(hdr)
	if err != nil {
		return err
	}
	_, err = io.Copy(wr, f)
	return err
}

// addDirToZip walks dir and adds every file under prefix/. Missing dir is OK.
func addDirToZip(w *zip.Writer, dir, prefix string) error {
	if _, err := os.Stat(dir); err != nil {
		return nil
	}
	return filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		entry := filepath.ToSlash(filepath.Join(prefix, rel))
		return addFileToZip(w, path, entry)
	})
}

// BuildArtifact builds a modpack zip for the specified target.
// client: includes shared + client mods.
// server: includes shared + server mods.
func BuildArtifact(spec *domain.PackSpec, lock *domain.PackLock, mcVersion, target string) error {
	return BuildArtifactWith(spec, lock, mcVersion, target, false)
}

// BuildArtifactAndReturnPath behaves like BuildArtifactWith but returns the
// on-disk zip path that was produced for the given target. It is used by the
// CLI to print a "artifact <target>: <path>" line per spec 7.5.
func BuildArtifactAndReturnPath(spec *domain.PackSpec, lock *domain.PackLock, mcVersion, target string, force bool) (string, error) {
	loader, loaderVer := loaderFromLock(lock)
	bc := &buildContext{
		Spec:          spec,
		Lock:          lock,
		McVersion:     mcVersion,
		Loader:        loader,
		LoaderVersion: loaderVer,
		RootDir:       ".",
	}
	if target == "both" || target == "" {
		if err := buildOneArtifactWith(spec, lock, mcVersion, "client", force); err != nil {
			return "", err
		}
		if err := buildOneArtifactWith(spec, lock, mcVersion, "server", force); err != nil {
			return "", err
		}
		return bc.zipPath("both"), nil
	}
	if err := buildOneArtifactWith(spec, lock, mcVersion, target, force); err != nil {
		return "", err
	}
	return bc.zipPath(target), nil
}

// BuildArtifactWith behaves like BuildArtifact but optionally overwrites
// existing artifacts when force is true.
func BuildArtifactWith(spec *domain.PackSpec, lock *domain.PackLock, mcVersion, target string, force bool) error {
	if spec == nil {
		return fmt.Errorf("spec is nil")
	}
	if lock == nil {
		return fmt.Errorf("lock is nil")
	}
	if target == "both" || target == "" {
		if err := buildOneArtifactWith(spec, lock, mcVersion, "client", force); err != nil {
			return err
		}
		return buildOneArtifactWith(spec, lock, mcVersion, "server", force)
	}
	return buildOneArtifactWith(spec, lock, mcVersion, target, force)
}

// buildOneArtifact creates a single zip for the given target.
func buildOneArtifact(spec *domain.PackSpec, lock *domain.PackLock, mcVersion, target string) error {
	return buildOneArtifactWith(spec, lock, mcVersion, target, false)
}

// buildOneArtifactWith creates a single zip for the given target, optionally
// allowing overwrite of an existing artifact when force is true.
func buildOneArtifactWith(spec *domain.PackSpec, lock *domain.PackLock, mcVersion, target string, force bool) error {
	if target != "client" && target != "server" {
		return fmt.Errorf("unsupported build target %q (want client, server, or both)", target)
	}
	loader, loaderVer := loaderFromLock(lock)
	bc := &buildContext{
		Spec:          spec,
		Lock:          lock,
		McVersion:     mcVersion,
		Loader:        loader,
		LoaderVersion: loaderVer,
		RootDir:       ".",
	}
	keys := bc.modsForTarget(target)
	if len(keys) == 0 {
		return fmt.Errorf("no mods for target %s", target)
	}
	modFiles := make(map[string]string)
	skipped := []string{}
	for _, key := range keys {
		locked, ok := lock.Mods[key]
		if !ok {
			return fmt.Errorf("mod %s missing from lock", key)
		}
		jarPath, err := bc.resolveModJar(key, locked)
		if err != nil {
			// Record the failure and skip the mod so other mods can still be
			// packaged. A partial build is more useful than no build when a
			// single remote source is rate-limited; the next run will retry
			// the failed mod once the rate limit clears.
			fmt.Fprintf(os.Stderr, "build: skipping mod %s: %v\n", key, err)
			skipped = append(skipped, key)
			continue
		}
		modFiles[key] = jarPath
	}
	if len(modFiles) == 0 {
		return fmt.Errorf("no mod jars resolved for target %s (all %d mods failed)", target, len(skipped))
	}
	out := bc.zipPath(target)
	if err := bc.buildZipWith(target, out, modFiles, force); err != nil {
		return err
	}
	if len(skipped) > 0 {
		fmt.Fprintf(os.Stderr, "build %s %s: %d mod(s) packaged, %d skipped due to resolution failures\n", mcVersion, target, len(modFiles), len(skipped))
	}
	return nil
}

// BuildClientServerBuild processes the full build pipeline for all loaders.
func BuildClientServerBuild(spec *domain.PackSpec, mcVersion string) error {
	for _, ln := range spec.LoaderName {
		loader, _ := domain.ParseLoaderName(ln)
		lock, err := LoadLock(mcVersion, loader)
		if err != nil {
			return fmt.Errorf("load lock for %s: %w", loader, err)
		}
		if err := BuildArtifact(spec, lock, mcVersion, "client"); err != nil {
			return fmt.Errorf("build client %s: %w", loader, err)
		}
		if err := BuildArtifact(spec, lock, mcVersion, "server"); err != nil {
			return fmt.Errorf("build server %s: %w", loader, err)
		}
	}
	return nil
}

// detectClassConflicts walks every jar in modFiles and reports any pair of
// jars that share a class entry. The returned error mentions the duplicated
// class path and the two jar names so the operator can remove the conflict.
func detectClassConflicts(modFiles map[string]string) error {
	type seen struct {
		key  string
		path string
	}
	owners := make(map[string]seen)
	for key, jarPath := range modFiles {
		r, err := zip.OpenReader(jarPath)
		if err != nil {
			continue
		}
		for _, f := range r.File {
			if !strings.HasSuffix(f.Name, ".class") {
				continue
			}
			if prev, ok := owners[f.Name]; ok {
				r.Close()
				return fmt.Errorf("build: duplicate class path %q in mods %q (%s) and %q (%s)\nhint: remove one of the conflicting jars", f.Name, prev.key, filepath.Base(prev.path), key, filepath.Base(jarPath))
			}
			owners[f.Name] = seen{key: key, path: jarPath}
		}
		r.Close()
	}
	return nil
}

// loaderFromLock returns the loader name and version for a PackLock.
func loaderFromLock(lock *domain.PackLock) (string, string) {
	if lock == nil {
		return "", ""
	}
	return lock.Loader, lock.LoaderVersion
}

// builtInModDeps is the set of internal mod ids that we never require as
// user-provided mods. These are loader / runtime helpers that ship with
// the loader itself; spec 7.5.11 says they must not be required. The
// whitelist is keyed by loader family so e.g. `fabricloader` is OK on
// fabric but unknown on neoforge.
var builtInModDeps = map[string]map[string]bool{
	"neoforge": {
		"minecraft":  true,
		"java":       true,
		"neoforge":   true,
		"forge":      true,
		"minecraftc": true,
	},
	"fabric": {
		"minecraft":                           true,
		"java":                                true,
		"fabricloader":                        true,
		"fabric-api-base":                     true,
		"fabric-api":                          true,
		"fabric-rendering-data-attachment-v1": true,
	},
}

// detectMissingRequiredDeps walks every resolved jar in modFiles, reads
// its loader-specific metadata, and reports any required dependency that
// is not provided by another jar in the build set. Per spec 7.5.32-34
// this check happens at build time and never short-circuits on --force.
// Errors follow the spec 7.8 format (error: ... hint: ...).
func detectMissingRequiredDeps(bc *buildContext, modFiles map[string]string) error {
	// index every jar's internal mod id and the key it came from.
	type slot struct {
		key     string
		jarPath string
	}
	indexed := make(map[string]slot)
	for key, jarPath := range modFiles {
		info, err := metadata.ReadJarMetadata(jarPath)
		if err != nil || info == nil || info.ModID == "" {
			continue
		}
		loaderFam := loaderFamily(bc.Loader)
		ident := metadata.InternalIdentity(loaderFam, info.ModID)
		indexed[ident] = slot{key: key, jarPath: jarPath}
	}
	// whitelist of built-in deps for the current loader family.
	whitelist := builtInModDeps[loaderFamily(bc.Loader)]
	for ident, owner := range indexed {
		_ = ident // identifier is used in cross-loader fallback below
		info, err := metadata.ReadJarMetadata(owner.jarPath)
		if err != nil || info == nil {
			continue
		}
		for _, dep := range info.Dependencies {
			if !dep.Required {
				continue
			}
			id := strings.ToLower(dep.ModID)
			if whitelist[id] {
				continue
			}
			loaderFam := loaderFamily(bc.Loader)
			// Match against indexed jars in the current loader family first.
			want := metadata.InternalIdentity(loaderFam, id)
			if _, ok := indexed[want]; ok {
				continue
			}
			// Fall back to a cross-loader match using the mod's own identity
			// (the key the jar registered under, regardless of loader family).
			crossMatched := false
			for ident := range indexed {
				if strings.HasSuffix(ident, ":"+id) {
					crossMatched = true
					break
				}
			}
			if crossMatched {
				continue
			}
			versionRange := dep.Ref
			if versionRange == "" {
				versionRange = "any"
			}
			return fmt.Errorf(
				"build: missing required mod dependency: %s\n"+
					"required by:\n"+
					"  - %s\n"+
					"requires:\n"+
					"  - %s %s\n"+
					"loader: %s\n"+
					"hint: add a lock entry that provides %s for %s %s",
				id, owner.key, id, versionRange, bc.Loader, id, bc.McVersion, bc.Loader)
		}
	}
	return nil
}

// loaderFamily returns the loader family used by metadata identifiers per
// spec 9.15. Both `fabric` and the legacy `fabricloader` collapse to
// `fabric`; the rest of the supported loaders are 1-to-1.
func loaderFamily(loader string) string {
	switch loader {
	case "fabric", "fabricloader":
		return "fabric"
	default:
		return loader
	}
}
