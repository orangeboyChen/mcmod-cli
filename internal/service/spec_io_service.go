// File: internal/service/spec_io_service.go
// Created: 2026-06-20
// Description: packspec.json and release index IO helpers.

package service

import "github.com/orangeboyChen/mcmod-cli/internal/domain"

// ReadPackSpec reads packspec.json.
func ReadPackSpec(dir string) (*domain.PackSpec, error) {
	return domain.ReadPackSpec(dir)
}

// WriteReleaseIndex writes a release index file.
func WriteReleaseIndex(mcVersion string, ri *domain.ReleaseIndex) error {
	return domain.WriteReleaseIndex(domain.ReleaseIndexPath(mcVersion), ri)
}

// ReadReleaseIndex reads a release index file.
func ReadReleaseIndex(mcVersion string) (*domain.ReleaseIndex, error) {
	return domain.ReadReleaseIndex(domain.ReleaseIndexPath(mcVersion))
}
