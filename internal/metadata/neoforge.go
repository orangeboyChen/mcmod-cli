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
			}
			if ver, ok := parsed["version"].(string); ok {
				info.Version = ver
			}
		}
	}

	if info.ModID == "" {
		return nil, fmt.Errorf("neoforge: no modid found in jar metadata")
	}
	return &info, nil
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
