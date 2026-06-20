// File: internal/testutil/spec.go
// Created: 2026-06-20
// Description: Helpers for constructing PackSpec values in tests.

package testutil

import "github.com/orangeboyChen/mcmod-cli/internal/domain"

// MinimalSpec returns a PackSpec with the given packName, version, and
// Minecraft version, plus a single "neoforge" loader entry. It is the
// minimum a spec needs to be considered valid for the lock and build
// services. Tests that need a richer spec can start from this and mutate.
func MinimalSpec(packName, packVersion, mcVersion string) *domain.PackSpec {
	return &domain.PackSpec{
		PackName:         packName,
		PackVersion:      packVersion,
		MinecraftVersion: mcVersion,
		LoaderName:       []string{"neoforge:21.1.219"},
	}
}

// SpecWithMod returns a MinimalSpec with one mod entry under the given key.
func SpecWithMod(packName, packVersion, mcVersion, key string, mod domain.ModSpec) *domain.PackSpec {
	s := MinimalSpec(packName, packVersion, mcVersion)
	if s.Mods == nil {
		s.Mods = make(map[string]domain.ModSpec)
	}
	s.Mods[key] = mod
	return s
}

// CurseForgeMod returns a ModSpec that resolves a CurseForge mod by name.
func CurseForgeMod(displayName string) domain.ModSpec {
	return domain.ModSpec{
		Name:  displayName,
		Scope: domain.ScopeShared,
		Source: domain.ModSource{
			Type:  domain.SourceCurseForge,
			Query: displayName,
		},
	}
}

// LocalMod returns a ModSpec that points at a local jar on disk.
func LocalMod(displayName, path string) domain.ModSpec {
	return domain.ModSpec{
		Name:  displayName,
		Scope: domain.ScopeShared,
		Source: domain.ModSource{
			Type: domain.SourceLocal,
			Path: path,
		},
	}
}
