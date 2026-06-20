// File: internal/service/release_lock_service.go
// Created: 2026-06-20
// Description: Release index lock and management service.

package service

import (
	"fmt"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

// CreateReleaseRecord creates or updates a release record in the index.
// The packName is sourced from the project packspec.json per spec.md section 6.2.
func CreateReleaseRecord(mcVersion, version, releaseType string, github *domain.ReleaseGitHub) (*domain.ReleaseIndex, error) {
	packName := ""
	if spec, err := ReadPackSpec("."); err == nil && spec != nil {
		packName = spec.PackName
	}

	index, err := ReadReleaseIndex(mcVersion)
	if err != nil {
		index = &domain.ReleaseIndex{
			Type:             "package",
			PackName:         packName,
			MinecraftVersion: mcVersion,
		}
	} else if index.PackName == "" {
		// Backfill missing packName from spec on existing indexes.
		index.PackName = packName
	}

	rec := index.EnsureRelease(version, releaseType)
	if github != nil {
		rec.GitHub = *github
	}

	if err := WriteReleaseIndex(mcVersion, index); err != nil {
		return nil, fmt.Errorf("write release index: %w", err)
	}
	return index, nil
}
