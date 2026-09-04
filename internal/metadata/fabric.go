// File: internal/metadata/fabric.go
// Created: 2026-06-20
// Description: Fabric jar metadata reader (fabric.mod.json).

package metadata

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"strings"
)

// FabricMod represents fabric.mod.json content.
type FabricMod struct {
	ID           string
	Version      string
	Dependencies json.RawMessage
	Raw          map[string]json.RawMessage
}

// ParseFabricDepends parses Fabric depends into DepInfo list.
func ParseFabricDepends(raw json.RawMessage) []DepInfo {
	var result []DepInfo
	var deps map[string]interface{}
	if err := json.Unmarshal(raw, &deps); err != nil {
		return result
	}
	for key, val := range deps {
		required := true
		s := fmt.Sprintf("%v", val)
		if strings.Contains(s, "suggests") || strings.Contains(s, "recommends") {
			required = false
		}
		result = append(result, DepInfo{
			ModID:    key,
			Ref:      s,
			Required: required,
		})
	}
	return result
}

// ReadFabricMetadata reads mod info from fabric.mod.json in a jar.
func ReadFabricMetadata(jarPath string) (*ModInfo, error) {
	r, err := zip.OpenReader(jarPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var info ModInfo
	for _, f := range r.File {
		if f.Name != "fabric.mod.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		data := make([]byte, f.UncompressedSize64)
		_, _ = rc.Read(data)
		rc.Close()

		var mod FabricMod
		mod.Raw = make(map[string]json.RawMessage)
		if err := json.Unmarshal(data, &mod.Raw); err != nil {
			continue
		}
		if id, ok := mod.Raw["id"]; ok {
			json.Unmarshal(id, &mod.ID)
		}
		if ver, ok := mod.Raw["version"]; ok {
			json.Unmarshal(ver, &mod.Version)
		}
		if deps, ok := mod.Raw["depends"]; ok {
			mod.Dependencies = deps
			info.Dependencies = ParseFabricDepends(deps)
		}
		info.ModID = mod.ID
		info.Version = mod.Version
		break
	}

	return &info, nil
}
