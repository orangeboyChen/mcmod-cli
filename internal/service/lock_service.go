// File: internal/service/lock_service.go
// Created: 2026-06-20
// Description: Lock-related service operations.

package service

import (
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

// LoadLock loads a lock file from the store.
func LoadLock(mcVersion, loader string) (*domain.PackLock, error) {
	return domain.ReadLockFile(domain.LockFilePath(mcVersion, loader))
}

// SaveLock saves a lock file via the store.
func SaveLock(mcVersion, loader string, lock *domain.PackLock) error {
	path := domain.LockFilePath(mcVersion, loader)
	return domain.WriteLockFile(path, lock)
}
