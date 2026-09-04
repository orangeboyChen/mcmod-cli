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

			parsed := parseSimpleTOML(data)
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
	}

	if info.ModID == "" {
		return nil, fmt.Errorf("neoforge: no modid found in jar metadata")
	}
	return &info, nil
}

func parseNeoForgeDependencies(data []byte) []DepInfo {
	lines := strings.Split(string(data), "\n")
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
		if dep.ModID != "" {
			filtered = append(filtered, dep)
		}
	}
	return filtered
}

func parseSimpleTOML(data []byte) map[string]interface{} {
	result := make(map[string]interface{})
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "=") && !strings.HasPrefix(line, "[") && !strings.HasPrefix(line, "#") {
			parts := strings.SplitN(line, "=", 2)
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
