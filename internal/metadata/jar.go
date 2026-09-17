// File: internal/metadata/jar.go
// Created: 2026-06-20
// Description: Generic jar metadata reader that dispatches to NeoForge or Fabric readers.

package metadata

import (
	"archive/zip"
	"fmt"
	"io"
	"strings"
)

// emptyEntrySize is the uncompressed size of a zero-length zip entry.
const emptyEntrySize = 0

// readZipEntry returns the full uncompressed contents of a zip entry.
//
// A single io.Reader.Read on a zip entry is not guaranteed to fill the
// buffer: the standard library returns at most 32KiB per call for deflated
// entries. Reading a large neoforge.mods.toml or fabric.mod.json with one
// Read therefore yields a truncated buffer — the dependency tables, which
// sit at the end of the file, are silently dropped, and a truncated
// fabric.mod.json is not even valid JSON, so the jar looks like it has no
// metadata at all. Use io.ReadFull so the whole entry is always read.
func readZipEntry(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	if f.UncompressedSize64 == emptyEntrySize {
		return nil, nil
	}
	data := make([]byte, f.UncompressedSize64)
	if _, err := io.ReadFull(rc, data); err != nil {
		return nil, fmt.Errorf("read %s: %w", f.Name, err)
	}
	return data, nil
}

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
