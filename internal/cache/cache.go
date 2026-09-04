// File: internal/cache/cache.go
// Created: 2026-06-20
// Description: Cache management for downloaded files.

package cache

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

// CacheDir is the root directory under which all downloaded artifacts are cached.
const CacheDir = ".mcmod/cache"

// CurseForgePath returns the cache path for a CurseForge file.
func CurseForgePath(modID, fileID, fileName string) string {
	return filepath.Join(CacheDir, "curseforge", modID, fileID, fileName)
}

// GitHubReleasePath returns the cache path for a GitHub release asset.
func GitHubReleasePath(owner, repo, tag, assetName string) string {
	return filepath.Join(CacheDir, "github-release", owner, repo, tag, assetName)
}

// GitCachePath returns the cache path for git package metadata.
func GitCachePath(owner, repo string) string {
	return filepath.Join(CacheDir, "git", "github", owner, repo)
}

// ResolvedIDPath returns the cache path for the resolved-id cache written
// by `mcmod lock` after a successful resolver run. The cache maps mod keys
// to the (modId, fileId) pair CurseForge returned for them, so the next
// lock run can skip the search step and verify-by-id directly. Stale
// entries are tolerable because lock re-validates the file via CF before
// trusting it.
func ResolvedIDPath(mcVersion, loader string) string {
	return filepath.Join(CacheDir, "resolved", fmt.Sprintf("%s-%s.json", mcVersion, loader))
}

// CheckCurseForge checks if a CurseForge file is cached and valid.
func CheckCurseForge(modID, fileID, fileName string) (bool, int64, error) {
	path := CurseForgePath(modID, fileID, fileName)
	info, err := os.Stat(path)
	if err != nil {
		return false, 0, nil
	}
	return true, info.Size(), nil
}

// CheckGitHubRelease checks if a GitHub release asset is cached and valid.
func CheckGitHubRelease(owner, repo, tag, assetName string) (bool, int64, error) {
	path := GitHubReleasePath(owner, repo, tag, assetName)
	info, err := os.Stat(path)
	if err != nil {
		return false, 0, nil
	}
	return true, info.Size(), nil
}

// ComputeSHA256 computes SHA256 hex digest of a file.
func ComputeSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash), nil
}

// AtomicMove moves a temp file to the final cache path atomically.
func AtomicMove(src, dst string) error {
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}
	return os.Rename(src, dst)
}

// EnsureCacheDir creates the .mcmod/cache directory.
func EnsureCacheDir() error {
	return os.MkdirAll(CacheDir, 0755)
}
