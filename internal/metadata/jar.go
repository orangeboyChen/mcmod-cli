// File: internal/metadata/jar.go
// Created: 2026-06-20
// Description: Generic jar metadata reader that dispatches to NeoForge or Fabric readers.

package metadata

import (
	"fmt"
	"strings"
)

// ReadJarMetadata reads mod metadata from a jar file, auto-detecting the mod format.
func ReadJarMetadata(jarPath string) (*ModInfo, error) {
	info, err := ReadNeoForgeMetadata(jarPath)
	if err == nil && info != nil && info.ModID != "" {
		return info, nil
	}

	info, err = ReadFabricMetadata(jarPath)
	if err == nil && info != nil && info.ModID != "" {
		return info, nil
	}

	return nil, fmt.Errorf("unable to read mod metadata from %s", jarPath)
}

// DepInfoFromIdentity extracts dependency info from an identity source string.
func DepInfoFromIdentity(identityStr string) DepInfo {
	parts := strings.SplitN(identityStr, ":", 2)
	modID := parts[0]
	if len(parts) > 1 {
		modID = parts[1]
	}
	return DepInfo{ModID: modID, Required: true}
}
