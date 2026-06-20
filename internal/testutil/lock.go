// File: internal/testutil/lock.go
// Created: 2026-06-20
// Description: Helpers for constructing PackLock / LockedMod values in tests.

package testutil

import "github.com/orangeboyChen/mcmod-cli/internal/domain"

// MinimalLock returns an empty lock for the given (mcVersion, loader) pair.
func MinimalLock(mcVersion, loader string) *domain.PackLock {
	return &domain.PackLock{
		Loader:           loader,
		LoaderVersion:    "0",
		MinecraftVersion: mcVersion,
		Mods:             make(map[string]domain.LockedMod),
	}
}

// LockWithMod returns a MinimalLock with one LockedMod entry. The caller
// supplies the LockedSource to avoid coupling to any specific source type.
func LockWithMod(mcVersion, loader, key string, locked domain.LockedMod) *domain.PackLock {
	l := MinimalLock(mcVersion, loader)
	l.Mods[key] = locked
	return l
}
