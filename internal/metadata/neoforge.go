// File: internal/metadata/neoforge.go
// Created: 2026-06-20
// Description: NeoForge jar metadata reader.

package metadata

import (
	"archive/zip"
	"fmt"
	"strings"
)

// DepInfo represents a mod dependency.
type DepInfo struct {
	ModID    string
	Ref      string
	Required bool
}

// ReadNeoForgeMetadata reads mod info from a jar's neoforge.mods.toml.
func ReadNeoForgeMetadata(jarPath string) (*ModInfo, error) {
	r, err := zip.OpenReader(jarPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var info ModInfo
	targetFiles := []string{"META-INF/neoforge.mods.toml", "META-INF/mods.toml"}
	for _, target := range targetFiles {
		for _, f := range r.File {
			if f.Name != target {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				continue
			}
			data := make([]byte, f.UncompressedSize64)
			_, _ = rc.Read(data)
			rc.Close()

			info.applyNeoForgeTOML(data)
		}
	}

	if info.ModID == tomlEmpty {
		return nil, fmt.Errorf("neoforge: no modid found in jar metadata")
	}
	return &info, nil
}

// applyNeoForgeTOML folds one neoforge.mods.toml payload into info, reading
// the mod id, version, and dependency tables.
func (info *ModInfo) applyNeoForgeTOML(data []byte) {
	parsed := parseSimpleTOML(data)
	// The mod id has to be read from the [[mods]] table only. A flattened
	// key=value scan would let the last `modId=` in the file win, which is a
	// [[dependencies.*]] entry: the jar would then be indexed under its
	// dependency's id and would also "provide" that id, hiding the missing
	// dependency.
	if _, ok := parsed["modid"].(string); !ok {
		if modsID, ok := firstTOMLTableValue(data, "modid", "modId"); ok {
			parsed["modid"] = modsID
		}
	}
	if modID, ok := parsed["modid"].(string); ok {
		info.ModID = modID
	} else if modID, ok := parsed["modId"].(string); ok {
		info.ModID = modID
	}
	if ver, ok := parsed["version"].(string); ok {
		info.Version = ver
	}
	info.Dependencies = append(info.Dependencies, parseNeoForgeDependencies(data)...)
}

func parseNeoForgeDependencies(data []byte) []DepInfo {
	lines := strings.Split(string(data), tomlNewline)
	var result []DepInfo
	current := -1
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[[dependencies.") {
			result = append(result, DepInfo{Required: true})
			current = len(result) - 1
			continue
		}
		if current < 0 || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		switch key {
		case "modId":
			result[current].ModID = value
		case "mandatory":
			result[current].Required = value != "false"
		case "versionRange":
			result[current].Ref = value
		}
	}
	filtered := result[:0]
	for _, dep := range result {
		if dep.ModID != tomlEmpty {
			filtered = append(filtered, dep)
		}
	}
	return filtered
}

// Constants shared by the TOML scanners in this file.
const (
	// modsTable is the TOML table carrying a NeoForge mod's own identity.
	modsTable = "mods"
	// tomlKeySep splits a TOML key from its value.
	tomlKeySep = "="
	// tomlKeyValue is the max number of parts a key=value line splits into.
	tomlKeyValue = 2
	// tomlNewline and tomlEmpty are the line separator and the empty-string
	// sentinel used by both TOML scanners.
	tomlNewline = "\n"
	tomlEmpty   = ""
)

// firstTOMLTableValue returns the value of the first matching key inside the
// first `[mods]` (or `[[mods]]`) section of a TOML document. Keys
// outside that section are ignored. It returns false when the table or the
// key is absent.
//
// NeoForge's neoforge.mods.toml declares the mod's own id under `[[mods]]`
// and every dependency's id under `[[dependencies.<modId>]]`, so both share
// the key name `modId`. Only the `[[mods]]` value identifies the jar.
func firstTOMLTableValue(data []byte, keys ...string) (string, bool) {
	lines := strings.Split(string(data), tomlNewline)
	inTable := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			// Any new header ends the table we were looking for; only an
			// inline table (`[a] b = 1`) could still carry a value, and
			// neither mods.toml flavor uses that form.
			name := strings.Trim(line, "[]")
			inTable = name == modsTable || strings.HasPrefix(name, modsTable+".")
			continue
		}
		if !inTable || !strings.Contains(line, tomlKeySep) || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, tomlKeySep, tomlKeyValue)
		key := strings.TrimSpace(parts[0])
		for _, want := range keys {
			if key == want {
				return strings.Trim(strings.TrimSpace(parts[1]), `"`), true
			}
		}
	}
	return tomlEmpty, false
}

func parseSimpleTOML(data []byte) map[string]interface{} {
	result := make(map[string]interface{})
	lines := strings.Split(string(data), tomlNewline)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, tomlKeySep) && !strings.HasPrefix(line, "[") && !strings.HasPrefix(line, "#") {
			parts := strings.SplitN(line, tomlKeySep, tomlKeyValue)
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"`)
			result[key] = val
		}
	}
	return result
}

// ModInfo is generic jar metadata.
type ModInfo struct {
	ModID        string
	Version      string
	Dependencies []DepInfo
}
