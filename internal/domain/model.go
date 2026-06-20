// File: internal/domain/model.go
// Created: 2026-06-20
// Description: Data models for packspec, lock, and release index.

package domain

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Source type constants.
const (
	SourceCurseForge    = "curseforge"
	SourceGitHubRelease = "github-release"
	SourceGit           = "git"
	SourceLocal         = "local"
	SourceURL           = "url"
)

// Scope constants.
const (
	ScopeShared = "shared"
	ScopeClient = "client"
	ScopeServer = "server"
)

// BuildTarget constants.
const (
	TargetClient = "client"
	TargetServer = "server"
	TargetBoth   = "both"
)

// PackSpec is the top-level packspec.json structure.
type PackSpec struct {
	PackName          string                     `json:"packName"`
	PackVersion       string                     `json:"packVersion"`
	ServerPackName    string                     `json:"serverPackName,omitempty"`
	MinecraftVersion  string                     `json:"minecraftVersion"`
	LoaderName        []string                   `json:"loaderName"`
	Author            string                     `json:"author,omitempty"`
	Mods              map[string]ModSpec         `json:"mods,omitempty"`
	SharedMods        []ModSpec                  `json:"sharedMods,omitempty"`
	ClientMods        []ModSpec                  `json:"clientMods,omitempty"`
	ServerMods        []ModSpec                  `json:"serverMods,omitempty"`
	Dependencies      []ModSpec                  `json:"dependencies,omitempty"`
	ExternalFiles     []ModSpec                  `json:"externalFiles,omitempty"`
	Variants          map[string]PackVariantSpec `json:"variants,omitempty"`
	LoaderNameIsArray bool                       `json:"-"`
	// entriesByMod stores entry specs indexed by mod key for programmatic access.
	entriesByMod map[string][]EntrySpec `json:"-"`
}

// UnmarshalJSON implements json.Unmarshaler for PackSpec.
// Detects whether loaderName is a JSON array (new format) or single string (legacy).
func (s *PackSpec) UnmarshalJSON(data []byte) error {
	type packSpecAlias struct {
		PackName         string                     `json:"packName"`
		PackVersion      string                     `json:"packVersion"`
		ServerPackName   string                     `json:"serverPackName,omitempty"`
		MinecraftVersion string                     `json:"minecraftVersion"`
		LoaderName       []string                   `json:"loaderName"`
		Author           string                     `json:"author,omitempty"`
		Mods             map[string]ModSpec         `json:"mods,omitempty"`
		SharedMods       []ModSpec                  `json:"sharedMods,omitempty"`
		ClientMods       []ModSpec                  `json:"clientMods,omitempty"`
		ServerMods       []ModSpec                  `json:"serverMods,omitempty"`
		Dependencies     []ModSpec                  `json:"dependencies,omitempty"`
		ExternalFiles    []ModSpec                  `json:"externalFiles,omitempty"`
		Variants         map[string]PackVariantSpec `json:"variants,omitempty"`
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	lnRaw := raw["loaderName"]
	delete(raw, "loaderName")

	remaining, _ := json.Marshal(raw)
	var alias packSpecAlias
	if err := json.Unmarshal(remaining, &alias); err != nil {
		return err
	}
	*s = PackSpec{
		PackName:         alias.PackName,
		PackVersion:      alias.PackVersion,
		ServerPackName:   alias.ServerPackName,
		MinecraftVersion: alias.MinecraftVersion,
		Author:           alias.Author,
		Mods:             alias.Mods,
		SharedMods:       alias.SharedMods,
		ClientMods:       alias.ClientMods,
		ServerMods:       alias.ServerMods,
		Dependencies:     alias.Dependencies,
		ExternalFiles:    alias.ExternalFiles,
		Variants:         alias.Variants,
	}

	if lnRaw != nil {
		var arr []string
		if err := json.Unmarshal(lnRaw, &arr); err == nil {
			s.LoaderName = arr
			s.LoaderNameIsArray = true
		} else {
			var single string
			if err := json.Unmarshal(lnRaw, &single); err == nil {
				s.LoaderName = []string{single}
				s.LoaderNameIsArray = false
			}
		}
	}
	return nil
}

// MarshalJSON implements json.Marshaler for PackSpec.
// Merges legacy SharedMods/ClientMods/ServerMods into the unified mods map
// and omits the legacy array fields from output.
func (s PackSpec) MarshalJSON() ([]byte, error) {
	type packSpecAlias struct {
		PackName         string                     `json:"packName"`
		PackVersion      string                     `json:"packVersion"`
		ServerPackName   string                     `json:"serverPackName,omitempty"`
		MinecraftVersion string                     `json:"minecraftVersion"`
		LoaderName       []string                   `json:"loaderName"`
		Author           string                     `json:"author,omitempty"`
		Mods             map[string]ModSpec         `json:"mods,omitempty"`
		Dependencies     []ModSpec                  `json:"dependencies,omitempty"`
		ExternalFiles    []ModSpec                  `json:"externalFiles,omitempty"`
		Variants         map[string]PackVariantSpec `json:"variants,omitempty"`
	}
	alias := packSpecAlias{
		PackName:         s.PackName,
		PackVersion:      s.PackVersion,
		ServerPackName:   s.ServerPackName,
		MinecraftVersion: s.MinecraftVersion,
		LoaderName:       s.LoaderName,
		Author:           s.Author,
		Mods:             s.Mods,
		Dependencies:     s.Dependencies,
		ExternalFiles:    s.ExternalFiles,
		Variants:         s.Variants,
	}

	// If Mods is empty but legacy arrays have content, build Mods from legacy
	if len(alias.Mods) == 0 {
		mods := make(map[string]ModSpec)
		for _, m := range s.SharedMods {
			key := NormalizeModKey(m.Name)
			if key == "" {
				key = m.Name
			}
			m.Scope = ScopeShared
			mods[key] = m
		}
		for _, m := range s.ClientMods {
			key := NormalizeModKey(m.Name)
			if key == "" {
				key = m.Name
			}
			m.Scope = ScopeClient
			mods[key] = m
		}
		for _, m := range s.ServerMods {
			key := NormalizeModKey(m.Name)
			if key == "" {
				key = m.Name
			}
			m.Scope = ScopeServer
			mods[key] = m
		}
		if len(mods) > 0 {
			alias.Mods = mods
		}
	}

	return json.Marshal(alias)
}

// ModSpec represents a mod definition in packspec.json.
type ModSpec struct {
	Name   string    `json:"name,omitempty"`
	Scope  string    `json:"scope,omitempty"`
	Loader []string  `json:"loader,omitempty"`
	Source ModSource `json:"source"`
}

// ModSource defines a single mod source type.
type ModSource struct {
	Type                 string            `json:"type"`
	Query                string            `json:"query,omitempty"`
	Slug                 string            `json:"slug,omitempty"`
	ModID                int               `json:"modId,omitempty"`
	FileID               int               `json:"fileId,omitempty"`
	Repo                 string            `json:"repo,omitempty"`
	Tag                  string            `json:"tag,omitempty"`
	AssetPattern         string            `json:"assetPattern,omitempty"`
	AssetPatternByLoader map[string]string `json:"assetPatternByLoader,omitempty"`
	URL                  string            `json:"url,omitempty"`
	// URLPattern is a template for the CDN download URL. The resolver renders
	// placeholders at lock time so the build step can skip the rate-limited
	// /download-url endpoint and download straight from the CDN. Supported
	// placeholders: {modId}, {fileId}, {fileId4} (first 4 digits of fileId
	// used in the standard edge.forgecdn.net path), {fileName}, {fileNameUrl}
	// (fileName with URL-escaped characters). If both URL and URLPattern are
	// set, URL wins.
	URLPattern string `json:"urlPattern,omitempty"`
	FileName   string `json:"fileName,omitempty"`
	Path       string `json:"path,omitempty"`
	Latest     bool   `json:"latest,omitempty"`
}

// UnmarshalJSON accepts the assetPattern field as either a JSON string (treated
// as AssetPattern) or a JSON object (treated as AssetPatternByLoader) so that
// packspec.json can use the convenient per-loader form without breaking the
// internal string representation.
func (s *ModSource) UnmarshalJSON(data []byte) error {
	type alias ModSource
	var probe alias
	if err := json.Unmarshal(data, &probe); err == nil {
		*s = ModSource(probe)
		return nil
	}

	// Fallback: pull the raw bytes for `assetPattern` and coerce based on shape.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	ap, ok := raw["assetPattern"]
	if !ok {
		return json.Unmarshal(data, s)
	}
	var asString string
	if err := json.Unmarshal(ap, &asString); err == nil {
		delete(raw, "assetPattern")
		cleaned, _ := json.Marshal(raw)
		if err := json.Unmarshal(cleaned, s); err != nil {
			return err
		}
		s.AssetPattern = asString
		return nil
	}
	var asMap map[string]string
	if err := json.Unmarshal(ap, &asMap); err == nil {
		delete(raw, "assetPattern")
		cleaned, _ := json.Marshal(raw)
		if err := json.Unmarshal(cleaned, s); err != nil {
			return err
		}
		s.AssetPatternByLoader = asMap
		return nil
	}
	return json.Unmarshal(data, s)
}

// SelectedAssetPattern returns the asset pattern for a given loader.
func (s ModSource) SelectedAssetPattern(loader string) string {
	if s.AssetPatternByLoader != nil {
		if p, ok := s.AssetPatternByLoader[loader]; ok {
			return p
		}
		// Fallback to first entry in AssetPatternByLoader
		for _, v := range s.AssetPatternByLoader {
			return v
		}
	}
	return s.AssetPattern
}

// PackVariantSpec represents a loader variant in packspec.json.
type PackVariantSpec struct {
	LoaderName []string  `json:"loaderName"`
	Mods       []ModSpec `json:"mods"`
}

// PackLock represents a dependency lock file.
type PackLock struct {
	Loader           string               `json:"loader"`
	LoaderVersion    string               `json:"loaderVersion,omitempty"`
	MinecraftVersion string               `json:"minecraftVersion"`
	Mods             map[string]LockedMod `json:"mods"`
}

// LockedMod represents a resolved mod in a lock file.
type LockedMod struct {
	Name         string       `json:"name,omitempty"`
	Version      string       `json:"version,omitempty"`
	Scope        string       `json:"scope"`
	Identity     *Identity    `json:"identity,omitempty"`
	Dependencies []DepRef     `json:"dependencies,omitempty"`
	Source       LockedSource `json:"source"`
	// Hash is a fingerprint of the spec source that produced this lock entry
	// (canonical-JSON SHA256, first 16 bytes hex). When a subsequent lock
	// run sees the spec source for a key produce a different hash, it knows
	// the mod needs to be re-resolved even if the previous lock entry was
	// kept around. Stored alongside the lock entry so the diff is local to
	// the lock file.
	Hash string `json:"hash,omitempty"`
}

// LockedSource is the resolved source in a lock file.
type LockedSource struct {
	Type                 string            `json:"type"`
	ModID                int               `json:"modId,omitempty"`
	FileID               int               `json:"fileId,omitempty"`
	FileName             string            `json:"fileName"`
	Repo                 string            `json:"repo,omitempty"`
	Tag                  string            `json:"tag,omitempty"`
	AssetName            string            `json:"assetName,omitempty"`
	LockAssetName        string            `json:"lockAssetName,omitempty"`
	Path                 string            `json:"path,omitempty"`
	URL                  string            `json:"url,omitempty"`
	AssetPatternByLoader map[string]string `json:"assetPatternByLoader,omitempty"`
}

// Identity contains identity information for a resolved mod.
type Identity struct {
	Source     string `json:"source"`
	Internal   string `json:"internal,omitempty"`
	Confidence string `json:"confidence"`
}

// DepRef represents a dependency reference from jar metadata.
type DepRef struct {
	ID           string `json:"id"`
	VersionRange string `json:"versionRange,omitempty"`
	Required     bool   `json:"required"`
}

// ReleaseIndex is the build artifact index.
type ReleaseIndex struct {
	Type             string          `json:"type"`
	PackName         string          `json:"packName"`
	MinecraftVersion string          `json:"minecraftVersion"`
	Releases         []ReleaseRecord `json:"releases"`
}

// Normalize sets default fields on a ReleaseIndex.
func (ri *ReleaseIndex) Normalize() {
	if ri.Type == "" {
		ri.Type = "package"
	}
}

// EnsureRelease finds or creates a release record for a version.
func (ri *ReleaseIndex) EnsureRelease(version, releaseType string) *ReleaseRecord {
	for i := range ri.Releases {
		if ri.Releases[i].Version == version {
			return &ri.Releases[i]
		}
	}
	rec := ReleaseRecord{Version: version, Type: releaseType, Artifact: make(map[string]ReleaseArtifactSet)}
	ri.Releases = append(ri.Releases, rec)
	return &ri.Releases[len(ri.Releases)-1]
}

// FindRelease returns a pointer to the release record for the given version, or nil.
func (ri *ReleaseIndex) FindRelease(version string) *ReleaseRecord {
	for i := range ri.Releases {
		if ri.Releases[i].Version == version {
			return &ri.Releases[i]
		}
	}
	return nil
}

// DeleteRelease removes a release record by version.
func (ri *ReleaseIndex) DeleteRelease(version string) bool {
	for i, r := range ri.Releases {
		if r.Version == version {
			ri.Releases = append(ri.Releases[:i], ri.Releases[i+1:]...)
			return true
		}
	}
	return false
}

// ReleaseRecord represents a single release version.
type ReleaseRecord struct {
	Version  string                        `json:"version"`
	Type     string                        `json:"type"`
	GitHub   ReleaseGitHub                 `json:"github,omitempty"`
	Artifact map[string]ReleaseArtifactSet `json:"artifact"`
}

// ReleaseGitHub contains GitHub release metadata.
type ReleaseGitHub struct {
	Repo  string `json:"repo"`
	Tag   string `json:"tag"`
	Name  string `json:"name,omitempty"`
	Body  string `json:"body,omitempty"`
	Draft bool   `json:"draft,omitempty"`
	Pre   bool   `json:"prerelease,omitempty"`
}

// ReleaseArtifactSet contains client/server artifact paths for one loader.
type ReleaseArtifactSet struct {
	Client string `json:"client,omitempty"`
	Server string `json:"server,omitempty"`
}

// SetArtifact sets an artifact path for a loader and target.
func (r *ReleaseRecord) SetArtifact(loader string, target BuildTarget, path string) {
	if r.Artifact == nil {
		r.Artifact = make(map[string]ReleaseArtifactSet)
	}
	set := r.Artifact[loader]
	switch target {
	case TargetClient:
		set.Client = path
	case TargetServer:
		set.Server = path
	case TargetBoth:
		set.Client = path
		set.Server = path
	}
	r.Artifact[loader] = set
}

// ArtifactFor returns the artifact path for a loader and target.
func (r *ReleaseRecord) ArtifactFor(loader string, target BuildTarget) string {
	set, ok := r.Artifact[loader]
	if !ok {
		return ""
	}
	switch target {
	case TargetClient:
		return set.Client
	case TargetServer:
		return set.Server
	case TargetBoth:
		return set.Client
	default:
		return set.Client
	}
}

// RemoveArtifact removes an artifact for a loader and target.
func (r *ReleaseRecord) RemoveArtifact(loader string, target BuildTarget) {
	set, ok := r.Artifact[loader]
	if !ok {
		return
	}
	switch target {
	case TargetClient:
		set.Client = ""
	case TargetServer:
		set.Server = ""
	case TargetBoth:
		set.Client = ""
		set.Server = ""
	}
	r.Artifact[loader] = set
}

// BuildTarget represents a build target type.
type BuildTarget string

// EntrySpec represents an entry configuration.
type EntrySpec struct {
	Name         string      `json:"name"`
	ArtifactName string      `json:"artifactName,omitempty"`
	Target       BuildTarget `json:"target,omitempty"`
}

// LoaderEntry holds parsed loader name and version.
type LoaderEntry struct {
	Name    string
	Version string
}

// ReadPackSpec reads and parses packspec.json from a directory.
func ReadPackSpec(dir string) (*PackSpec, error) {
	data, err := os.ReadFile(filepath.Join(dir, "packspec.json"))
	if err != nil {
		return nil, err
	}
	var spec PackSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, err
	}
	return &spec, nil
}

// WritePackSpec writes packspec.json to a directory.
func WritePackSpec(dir string, spec *PackSpec) error {
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "packspec.json"), data, 0644)
}

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
