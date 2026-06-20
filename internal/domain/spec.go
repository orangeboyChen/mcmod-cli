// File: internal/domain/spec.go
// Created: 2026-06-20
// Description: PackSpec, ModSpec, ModSource, and PackVariantSpec types and JSON methods.

package domain

import (
	"encoding/json"
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
