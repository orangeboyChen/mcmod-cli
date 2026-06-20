// File: internal/service/build_validation.go
// Created: 2026-06-20
// Description: Build-time validation (class conflict scan, missing required deps).

package service

import (
	"archive/zip"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/orangeboyChen/mcmod-cli/internal/metadata"
)

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
