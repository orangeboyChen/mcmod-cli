// File: internal/service/cfkey_service.go
// Created: 2026-06-20
// Description: CurseForge API key accessors (project / user scope).

package service

import "github.com/orangeboyChen/mcmod-cli/internal/config"

// GetCFKey returns the effective CurseForge API key.
func GetCFKey() string {
	return config.GetCFKey()
}

// ConfigureCFKey sets the CurseForge API key at project level.
func ConfigureCFKey(key string) error {
	return config.WriteProjectConfig(key)
}

// ConfigureUserCFKey sets the CurseForge API key at user level.
func ConfigureUserCFKey(key string) error {
	return config.WriteUserConfig(key)
}
