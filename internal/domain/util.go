// File: internal/domain/util.go
// Created: 2026-06-20
// Description: Spec helpers (loader parsing, mod collection, key normalization, URL expansion).

package domain

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ParseLoaderName parses "neoforge:21.1.219" into name and version.
func ParseLoaderName(s string) (name, version string) {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}

// splitLoaderSpec is an alias for ParseLoaderName.
func splitLoaderSpec(s string) (name, version string) {
	return ParseLoaderName(s)
}

// LoaderEntries returns parsed loader entries from packspec.json.
func LoaderEntries(spec PackSpec) []LoaderEntry {
	var entries []LoaderEntry
	for _, ln := range spec.LoaderName {
		name, version := ParseLoaderName(ln)
		entries = append(entries, LoaderEntry{Name: name, Version: version})
	}
	return entries
}

// PrimaryLoaderName returns the first loader name.
func PrimaryLoaderName(spec PackSpec) string {
	if len(spec.LoaderName) == 0 {
		return ""
	}
	name, _ := ParseLoaderName(spec.LoaderName[0])
	return name
}

// PrimaryLoaderVersion returns the first loader version.
func PrimaryLoaderVersion(spec PackSpec) string {
	if len(spec.LoaderName) == 0 {
		return ""
	}
	_, version := ParseLoaderName(spec.LoaderName[0])
	return version
}

// LoaderVersionFor returns the version for a specific loader.
func LoaderVersionFor(spec PackSpec, loader string) string {
	for _, ln := range spec.LoaderName {
		name, version := ParseLoaderName(ln)
		if name == loader {
			return version
		}
	}
	return ""
}

// LoaderMatches checks if a loader is supported by the spec.
func LoaderMatches(spec PackSpec, loader string) bool {
	for _, ln := range spec.LoaderName {
		name, _ := ParseLoaderName(ln)
		if name == loader {
			return true
		}
	}
	return false
}

// VariantKey creates a variant key from loader name and version.
func VariantKey(loader, version string) string {
	if version != "" {
		return loader + ":" + version
	}
	return loader
}

// DefaultVariantKey returns the first variant key.
func DefaultVariantKey(spec PackSpec) string {
	if len(spec.LoaderName) == 0 {
		return ""
	}
	return spec.LoaderName[0]
}

// ArtifactBaseName builds the base name for a build artifact.
func ArtifactBaseName(spec PackSpec, mcVersion, loader, loaderVersion, target string) string {
	packName := spec.PackName
	if target == TargetServer && spec.ServerPackName != "" {
		packName = spec.ServerPackName
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s", packName, mcVersion, loader, loaderVersion, target)
}

// BaseName builds artifact base name using primary loader info.
func BaseName(spec PackSpec, mcVersion, packVersion, target string) string {
	loader := PrimaryLoaderName(spec)
	version := PrimaryLoaderVersion(spec)
	return ArtifactBaseName(spec, mcVersion, loader, version, target)
}

// normalizeLoaderList normalizes and deduplicates a loader list.
func normalizeLoaderList(loaders []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, l := range loaders {
		l = strings.TrimSpace(strings.ToLower(l))
		if l != "" && !seen[l] {
			seen[l] = true
			result = append(result, l)
		}
	}
	return result
}

// modScope returns the actual scope for a mod.
func modScope(mod ModSpec) string {
	if mod.Scope == "" {
		return ScopeShared
	}
	return mod.Scope
}

// ModsForScope returns mods matching a scope from PackSpec.Mods.
func ModsForScope(spec PackSpec, scope string) []ModSpec {
	var result []ModSpec
	for _, mod := range spec.Mods {
		actualScope := mod.Scope
		if actualScope == "" {
			actualScope = ScopeShared
		}
		if actualScope == scope {
			result = append(result, mod)
		}
	}
	return result
}

// Mods returns all mods from the unified map.
func Mods(spec PackSpec) []ModSpec {
	var result []ModSpec
	for _, mod := range spec.Mods {
		result = append(result, mod)
	}
	return result
}

// Dependencies returns the dependency mods.
func Dependencies(spec PackSpec) []ModSpec {
	return spec.Dependencies
}

// SetDependencies sets the dependencies in packspec.
func SetDependencies(spec PackSpec, deps []ModSpec) PackSpec {
	spec.Dependencies = deps
	return spec
}

// SetModsForScope sets all mods with a given scope.
func SetModsForScope(spec PackSpec, scope string, mods []ModSpec) PackSpec {
	spec.Mods = make(map[string]ModSpec)
	for _, m := range mods {
		m.Scope = scope
		key := NormalizeModKey(m.Name)
		if key == "" {
			key = m.Name
		}
		spec.Mods[key] = m
	}
	return spec
}

// SetMods replaces all mods in a PackSpec.
func SetMods(spec PackSpec, mods []ModSpec) PackSpec {
	spec.Mods = make(map[string]ModSpec)
	for _, m := range mods {
		key := NormalizeModKey(m.Name)
		if key == "" {
			key = m.Name
		}
		spec.Mods[key] = m
	}
	return spec
}

// EntryIndex returns an EntrySpec for a mod.
func EntryIndex(spec PackSpec, modKey string) (EntrySpec, bool) {
	// First check entriesByMod for the exact entry name
	if spec.entriesByMod != nil {
		for _, entries := range spec.entriesByMod {
			for _, e := range entries {
				if e.Name == modKey {
					return e, true
				}
			}
		}
	}
	if spec.Variants == nil {
		return EntrySpec{}, false
	}
	for _, vs := range spec.Variants {
		for _, m := range vs.Mods {
			if m.Name == modKey {
				return EntrySpec{Name: m.Name}, true
			}
		}
	}
	// Search in mods directly
	for _, m := range spec.Mods {
		if m.Name == modKey {
			return EntrySpec{Name: m.Name}, true
		}
	}
	return EntrySpec{}, false
}

// SetEntriesForMod sets entries for a mod.
func SetEntriesForMod(spec PackSpec, modKey string, entries []EntrySpec) PackSpec {
	if spec.entriesByMod == nil {
		spec.entriesByMod = make(map[string][]EntrySpec)
	}
	spec.entriesByMod[modKey] = append(spec.entriesByMod[modKey], entries...)
	return spec
}

// EntriesForMod returns entries for a mod.
func EntriesForMod(spec PackSpec, modKey string) []EntrySpec {
	if spec.entriesByMod == nil {
		return nil
	}
	entries, ok := spec.entriesByMod[modKey]
	if !ok {
		return nil
	}
	// Deep copy
	result := make([]EntrySpec, len(entries))
	copy(result, entries)
	return result
}

// AllMods returns all mods including variants.
func AllMods(spec PackSpec) []ModSpec {
	result := make([]ModSpec, 0, len(spec.Mods))
	for _, m := range spec.Mods {
		result = append(result, m)
	}
	// Only include variant mods if Mods is empty (legacy behavior)
	if len(spec.Mods) == 0 && spec.Variants != nil {
		for _, vs := range spec.Variants {
			result = append(result, vs.Mods...)
		}
	}
	return result
}

// AllEntries returns all entries from variants.
func AllEntries(spec PackSpec) []EntrySpec {
	var result []EntrySpec
	// First check entriesByMod
	if spec.entriesByMod != nil {
		for _, entries := range spec.entriesByMod {
			result = append(result, entries...)
		}
	}
	// Only collect from Variants if entriesByMod is empty and Mods is empty
	if len(spec.entriesByMod) == 0 && len(spec.Mods) == 0 && spec.Variants != nil {
		for _, vs := range spec.Variants {
			for _, m := range vs.Mods {
				result = append(result, EntrySpec{Name: m.Name})
			}
		}
	}
	return result
}

// AllModsForVariant returns all mods for a variant.
func AllModsForVariant(spec PackSpec, variantKey string) []ModSpec {
	var all []ModSpec
	// Include main mods
	for _, m := range spec.Mods {
		all = append(all, m)
	}
	// Include dependencies as mods
	for _, d := range spec.Dependencies {
		key := NormalizeModKey(d.Name)
		if key == "" {
			key = d.Name
		}
		// Avoid duplicates
		found := false
		for _, a := range all {
			if NormalizeModKey(a.Name) == key {
				found = true
				break
			}
		}
		if !found {
			all = append(all, d)
		}
	}
	// Sort by normalized name for deterministic lowercase order
	sort.Slice(all, func(i, j int) bool {
		return NormalizeModKey(all[i].Name) < NormalizeModKey(all[j].Name)
	})
	return all
}

// AllEntriesForVariant returns all entries for a variant.
func AllEntriesForVariant(spec PackSpec, variantKey string) []EntrySpec {
	var result []EntrySpec
	if spec.entriesByMod == nil {
		return result
	}
	// Collect all entries for variants that match the variantKey
	for _, entries := range spec.entriesByMod {
		result = append(result, entries...)
	}
	return result
}

// FileNameForURL extracts filename from URL or returns fallback.
func FileNameForURL(url, fallback string) string {
	if url == "" {
		return fallback
	}
	basename := filepath.Base(url)
	// Strip query string
	if idx := strings.Index(basename, "?"); idx >= 0 {
		basename = basename[:idx]
	}
	if basename == "" || basename == "." {
		return fallback
	}
	return basename
}

// NormalizeModKey converts a display name to a stable ID key per spec 3.2.
func NormalizeModKey(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, "\u2019", "")
	specialCharRe := regexp.MustCompile(`[^\w\s-]`)
	dashRunRe := regexp.MustCompile(`-{2,}`)
	s = specialCharRe.ReplaceAllString(s, "-")
	s = dashRunRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	// Replace remaining spaces with dash
	s = strings.ReplaceAll(s, " ", "-")
	s = dashRunRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// cloneModMap creates a shallow clone of a mod map.
func cloneModMap(m map[string]ModSpec) map[string]ModSpec {
	clone := make(map[string]ModSpec, len(m))
	for k, v := range m {
		clone[k] = v
	}
	return clone
}

// modMapFromSlice creates a mod map from a slice of ModSpec.
func modMapFromSlice(slice []ModSpec) map[string]ModSpec {
	m := make(map[string]ModSpec)
	for _, mod := range slice {
		key := NormalizeModKey(mod.Name)
		if key == "" {
			key = mod.Name
		}
		m[key] = mod
	}
	return m
}

// modsForScopeFromMap returns mods matching a scope from a mod map.
func modsForScopeFromMap(mods map[string]ModSpec, scope string) []ModSpec {
	var result []ModSpec
	for _, mod := range mods {
		actualScope := mod.Scope
		if actualScope == "" {
			actualScope = ScopeShared
		}
		if actualScope == scope {
			result = append(result, mod)
		}
	}
	return result
}

// setModsForScopeInMap sets mods for a scope in a mod map.
func setModsForScopeInMap(mods map[string]ModSpec, scope string, newMods []ModSpec) {
	// First remove all existing mods with the given scope
	for k, m := range mods {
		if m.Scope == scope {
			delete(mods, k)
		}
	}
	// Then add new mods
	for _, m := range newMods {
		key := NormalizeModKey(m.Name)
		if key == "" {
			key = m.Name
		}
		mod := m
		mod.Scope = scope
		mods[key] = mod
	}
}

// RenderURL returns the final CDN download URL for a curseforge mod, using
// URLPattern from the spec if present (placeholders are expanded) and
// falling back to the bare URL field. Returns "" when neither is set.
// The version is appended from the resolved LockedMod so urlPattern can use
// {modVersion} / {mcVersion} placeholders to construct fileNames that
// embed the loader version.
func (s ModSource) RenderURL(modID, fileID int, fileName, version, mcVersion string) string {
	if s.URL != "" {
		return s.URL
	}
	if s.URLPattern == "" {
		return ""
	}
	return ExpandURLPattern(s.URLPattern, modID, fileID, fileName, version, mcVersion)
}

// ExpandURLPattern substitutes placeholders in a URL template. Supported
// placeholders:
//
//	{modId}        CurseForge project id, e.g. 676721
//	{fileId}       full CurseForge file id, e.g. 8240058
//	{fileId4}      first 4 digits of fileId, a slash, then the rest without
//	               leading zero (e.g. 8240/58). This matches the standard
//	               edge.forgecdn.net / mediafilez.forgecdn.net path layout.
//	{fileName}     the resolved file name, e.g. create-aeronautics-bundled-1.21.1-1.3.0.jar
//	{fileNameUrl}  fileName with URL-escaped characters
//	{modVersion}   the mod's resolved version, e.g. 1.3.0
//	{mcVersion}    the lock's minecraft version, e.g. 1.21.1
func ExpandURLPattern(pattern string, modID, fileID int, fileName, version, mcVersion string) string {
	idStr := fmt.Sprintf("%d", fileID)
	var fileID4 string
	if len(idStr) >= 4 {
		// ForgeCDN path layout is "<first4>/<rest>" where the rest is the
		// digits after position 4 *without* the leading zero (so fileId
		// 8240058 renders as 8240/58, not 8240/058). The leading-zero form
		// returns 403 on mediafilez.forgecdn.net.
		rest := strings.TrimLeft(idStr[4:], "0")
		if rest == "" {
			rest = "0"
		}
		fileID4 = idStr[:4] + "/" + rest
	} else {
		fileID4 = idStr
	}
	out := pattern
	out = strings.ReplaceAll(out, "{modId}", fmt.Sprintf("%d", modID))
	out = strings.ReplaceAll(out, "{fileId}", idStr)
	out = strings.ReplaceAll(out, "{fileId4}", fileID4)
	out = strings.ReplaceAll(out, "{fileName}", fileName)
	out = strings.ReplaceAll(out, "{fileNameUrl}", url.PathEscape(fileName))
	out = strings.ReplaceAll(out, "{modVersion}", version)
	out = strings.ReplaceAll(out, "{mcVersion}", mcVersion)
	return out
}
