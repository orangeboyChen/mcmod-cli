// File: internal/service/build_validation.go
// Created: 2026-06-20
// Description: Build-time validation (class conflict scan, missing required deps).

package service

import (
	"archive/zip"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/orangeboyChen/mcmod-cli/internal/metadata"
)

type modJarScan struct {
	key      string
	path     string
	metadata *metadata.ModInfo
}

// validateModFiles checks all selected mod jars before an artifact is written.
// It reports every class conflict, unreadable jar, and missing required
// dependency found in the same pass.
func validateModFiles(bc *buildContext, modFiles map[string]string) error {
	keys := make([]string, 0, len(modFiles))
	for key := range modFiles {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	owners := make(map[string][]modJarScan)
	scans := make([]modJarScan, 0, len(keys))
	var unreadable []string
	for _, key := range keys {
		jarPath := modFiles[key]
		r, err := zip.OpenReader(jarPath)
		if err != nil {
			unreadable = append(unreadable, fmt.Sprintf("  - %s (%s): %v", key, filepath.Base(jarPath), err))
			continue
		}
		for _, entry := range r.File {
			if strings.HasSuffix(entry.Name, ".class") && !strings.HasSuffix(entry.Name, "/") {
				owners[entry.Name] = append(owners[entry.Name], modJarScan{key: key, path: jarPath})
			}
		}
		_ = r.Close()
		info, metadataErr := metadata.ReadJarMetadata(jarPath)
		if metadataErr != nil || info == nil || info.ModID == "" {
			info = nil
		}
		scans = append(scans, modJarScan{key: key, path: jarPath, metadata: info})
	}

	var sections []string
	var conflicts []string
	classPaths := make([]string, 0, len(owners))
	for classPath := range owners {
		classPaths = append(classPaths, classPath)
	}
	sort.Strings(classPaths)
	for _, classPath := range classPaths {
		entries := owners[classPath]
		if len(entries) < 2 {
			continue
		}
		parts := make([]string, 0, len(entries))
		for _, entry := range entries {
			parts = append(parts, fmt.Sprintf("%s (%s)", entry.key, filepath.Base(entry.path)))
		}
		conflicts = append(conflicts, fmt.Sprintf("  - %s: %s", classPath, strings.Join(parts, ", ")))
	}
	if len(conflicts) > 0 {
		sections = append(sections, "class conflicts:\n"+strings.Join(conflicts, "\n"))
	}
	if len(unreadable) > 0 {
		sections = append(sections, "unreadable jars:\n"+strings.Join(unreadable, "\n"))
	}
	missing := collectMissingRequiredDeps(bc, scans)
	if len(missing) > 0 {
		lines := make([]string, 0, len(missing))
		for _, item := range missing {
			lines = append(lines, "  - "+item)
		}
		sections = append(sections, "missing required dependencies:\n"+strings.Join(lines, "\n"))
	}
	if len(sections) == 0 {
		return nil
	}
	return fmt.Errorf("build: mod validation failed\n%s\nhint: resolve every listed conflict or dependency before rebuilding", strings.Join(sections, "\n"))
}

func detectMissingRequiredDeps(bc *buildContext, modFiles map[string]string) error {
	scans := make([]modJarScan, 0, len(modFiles))
	for key, jarPath := range modFiles {
		info, err := metadata.ReadJarMetadata(jarPath)
		if err == nil && info != nil && info.ModID != "" {
			scans = append(scans, modJarScan{key: key, path: jarPath, metadata: info})
		}
	}
	missing := collectMissingRequiredDeps(bc, scans)
	if len(missing) == 0 {
		return nil
	}
	lines := make([]string, 0, len(missing))
	for _, item := range missing {
		lines = append(lines, "  - "+item)
	}
	return fmt.Errorf("build: mod validation failed\nmissing required dependencies:\n%s\nhint: resolve every listed conflict or dependency before rebuilding", strings.Join(lines, "\n"))
}

type missingDependency struct {
	modID   string
	version string
	owners  []string
}

func collectMissingRequiredDeps(bc *buildContext, scans []modJarScan) []string {
	provided := make(map[string]bool)
	for _, scan := range scans {
		if scan.metadata != nil {
			provided[strings.ToLower(scan.metadata.ModID)] = true
		}
	}
	whitelist := builtInModDeps[loaderFamily(bc.Loader)]
	missing := make(map[string]*missingDependency)
	for _, scan := range scans {
		if scan.metadata == nil {
			continue
		}
		for _, dep := range scan.metadata.Dependencies {
			if !dep.Required {
				continue
			}
			id := strings.ToLower(dep.ModID)
			if whitelist[id] || provided[id] {
				continue
			}
			version := dep.Ref
			if version == "" {
				version = "any"
			}
			item, ok := missing[id]
			if !ok {
				item = &missingDependency{modID: id, version: version}
				missing[id] = item
			}
			if item.version == "any" && version != "any" {
				item.version = version
			}
			item.owners = append(item.owners, fmt.Sprintf("%s requires %s", scan.key, version))
		}
	}
	ids := make([]string, 0, len(missing))
	for id := range missing {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		item := missing[id]
		sort.Strings(item.owners)
		result = append(result, fmt.Sprintf("%s %s; required by %s", item.modID, item.version, strings.Join(item.owners, ", ")))
	}
	return result
}

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
