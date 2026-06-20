// File: internal/service/lock_service.go
// Created: 2026-06-20
// Description: Lock-related service operations: read, write, path, marshal.

package service

import (
	"encoding/json"

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

// WriteLockFile writes a lock file.
func WriteLockFile(lf *domain.PackLock) error {
	path := domain.LockFilePath(lf.MinecraftVersion, lf.Loader)
	return domain.WriteLockFile(path, lf)
}

// LockFilePath returns the lock file path.
func LockFilePath(mcVersion, loader string) string {
	return domain.LockFilePath(mcVersion, loader)
}

// MarshalLockJSON marshals lock file to JSON bytes.
func MarshalLockJSON(lf *domain.PackLock) ([]byte, error) {
	return json.MarshalIndent(lf, "", "  ")
}

// ReadLockFile reads a lock file.
func ReadLockFile(dir string) (*domain.PackLock, error) {
	return domain.ReadLockFile(dir)
}
