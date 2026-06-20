// File: internal/domain/store.go
// Created: 2026-06-20
// Description: Lock file and release index read/write per spec section 5-6.

package domain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LocksDir is the directory containing lock files.
const LocksDir = "locks"

// DependenciesDir is the directory containing dependency locks.
const DependenciesDir = "locks/dependencies"

// ReleasesDir is the directory containing release indexes.
const ReleasesDir = "locks/releases"

// LockFilePath returns the path for a dependency lock file.
func LockFilePath(mcVersion, loader string) string {
	return filepath.Join(DependenciesDir, mcVersion+"-"+loader+".json")
}

// ReleaseIndexPath returns the path for a release index file.
func ReleaseIndexPath(mcVersion string) string {
	return filepath.Join(ReleasesDir, mcVersion+".json")
}

// ReadLockFile reads and parses a lock file by full path.
func ReadLockFile(path string) (*PackLock, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pf PackLock
	if err := json.Unmarshal(data, &pf); err != nil {
		return nil, err
	}
	return &pf, nil
}

// WriteLockFile writes a lock file to the given full path.
func WriteLockFile(path string, lf *PackLock) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ReadReleaseIndex reads and parses a release index file by full path.
func ReadReleaseIndex(path string) (*ReleaseIndex, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ri ReleaseIndex
	if err := json.Unmarshal(data, &ri); err != nil {
		return nil, err
	}
	return &ri, nil
}

// WriteReleaseIndex writes a release index file to the given full path.
func WriteReleaseIndex(path string, ri *ReleaseIndex) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(ri, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// FileStore provides file-based access to packspec, lock, and release index.
type FileStore struct {
	Root string
}

// DefaultFileStore returns a FileStore for the given root directory.
func DefaultFileStore(root string) *FileStore {
	return &FileStore{Root: filepath.Clean(root)}
}

// SaveSpec writes packspec.json.
func (s *FileStore) SaveSpec(spec PackSpec) error {
	return WritePackSpec(s.Root, &spec)
}

// LoadSpec reads packspec.json.
func (s *FileStore) LoadSpec() (PackSpec, error) {
	data, err := os.ReadFile(filepath.Join(s.Root, "packspec.json"))
	if err != nil {
		return PackSpec{}, err
	}
	var spec PackSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return PackSpec{}, err
	}
	return spec, nil
}

// SaveLock writes a dependency lock file. path is relative to Root.
func (s *FileStore) SaveLock(mcVersion, loader string, lock PackLock) error {
	path := filepath.Join(s.Root, LockFilePath(mcVersion, loader))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadLock reads a dependency lock file with legacy path fallback.
func (s *FileStore) LoadLock(mcVersion, loader string) (*PackLock, error) {
	// Primary: loader-specific path under Root
	primaryPath := filepath.Join(s.Root, LockFilePath(mcVersion, loader))
	if data, err := os.ReadFile(primaryPath); err == nil {
		var lock PackLock
		if err := json.Unmarshal(data, &lock); err == nil {
			return &lock, nil
		}
	}

	// Legacy: <mcVersion>.json in root
	legacyPath := filepath.Join(s.Root, mcVersion+".json")
	if data, err := os.ReadFile(legacyPath); err == nil {
		lock, err := unmarshalPackLock(data)
		if err == nil {
			return lock, nil
		}
	}

	// Single-candidate: loaderless directory
	loaderlessPath := filepath.Join(s.Root, "locks", "dependencies", mcVersion+"-"+loader+".json")
	if data, err := os.ReadFile(loaderlessPath); err == nil {
		var lock PackLock
		if err := json.Unmarshal(data, &lock); err == nil {
			return &lock, nil
		}
	}

	// Try loading with empty loader to find single candidate
	if loader == "" {
		depsDir := filepath.Join(s.Root, DependenciesDir)
		entries, err := os.ReadDir(depsDir)
		if err == nil {
			for _, e := range entries {
				if strings.HasPrefix(e.Name(), mcVersion+"-") {
					fpath := filepath.Join(depsDir, e.Name())
					if data, err := os.ReadFile(fpath); err == nil {
						lock, err := unmarshalPackLock(data)
						if err == nil && lock.Loader != "" {
							return lock, nil
						}
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("lock file not found for %s %s", mcVersion, loader)
}

// SaveReleaseIndex writes a release index file. path is relative to Root.
func (s *FileStore) SaveReleaseIndex(mcVersion string, ri ReleaseIndex) error {
	ri.Normalize()
	path := filepath.Join(s.Root, ReleaseIndexPath(mcVersion))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(ri, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadReleaseIndex reads a release index file.
func (s *FileStore) LoadReleaseIndex(mcVersion string) (ReleaseIndex, error) {
	path := filepath.Join(s.Root, ReleaseIndexPath(mcVersion))
	data, err := os.ReadFile(path)
	if err != nil {
		return ReleaseIndex{}, err
	}
	var ri ReleaseIndex
	if err := json.Unmarshal(data, &ri); err != nil {
		return ReleaseIndex{}, err
	}
	return ri, nil
}

// unmarshalPackLock is a helper that can unmarshal legacy lock formats.
func unmarshalPackLock(data []byte) (*PackLock, error) {
	var lock PackLock
	if err := json.Unmarshal(data, &lock); err == nil {
		if lock.Loader != "" && len(lock.Mods) > 0 {
			return &lock, nil
		}
	}

	var legacy struct {
		Versions map[string]struct {
			LoaderName    string    `json:"loaderName"`
			LoaderVersion string    `json:"loaderVersion,omitempty"`
			SharedMods    []ModSpec `json:"sharedMods"`
			ClientMods    []ModSpec `json:"clientMods"`
			ServerMods    []ModSpec `json:"serverMods"`
			ExternalFiles []ModSpec `json:"externalFiles"`
		} `json:"versions"`
	}
	if err := json.Unmarshal(data, &legacy); err == nil {
		for mcVer, v := range legacy.Versions {
			lock = PackLock{
				Loader:           v.LoaderName,
				LoaderVersion:    v.LoaderVersion,
				MinecraftVersion: mcVer,
				Mods:             make(map[string]LockedMod),
			}
			for _, m := range v.SharedMods {
				key := NormalizeModKey(m.Name)
				lock.Mods[key] = LockedMod{
					Name: m.Name, Scope: ScopeShared,
					Source: LockedSource{Type: m.Source.Type, Path: m.Source.Path, FileName: m.Source.FileName},
				}
			}
			for _, m := range v.ClientMods {
				key := NormalizeModKey(m.Name)
				lock.Mods[key] = LockedMod{
					Name: m.Name, Scope: ScopeClient,
					Source: LockedSource{Type: m.Source.Type, Path: m.Source.Path, FileName: m.Source.FileName},
				}
			}
			for _, m := range v.ServerMods {
				key := NormalizeModKey(m.Name)
				lock.Mods[key] = LockedMod{
					Name: m.Name, Scope: ScopeServer,
					Source: LockedSource{Type: m.Source.Type, Path: m.Source.Path, FileName: m.Source.FileName},
				}
			}
			for _, m := range v.ExternalFiles {
				key := NormalizeModKey(m.Name)
				lock.Mods[key] = LockedMod{
					Name: m.Name, Scope: ScopeShared,
					Source: LockedSource{Type: m.Source.Type, Path: m.Source.Path, FileName: m.Source.FileName},
				}
			}
			return &lock, nil
		}
	}

	return nil, fmt.Errorf("failed to unmarshal lock data")
}
