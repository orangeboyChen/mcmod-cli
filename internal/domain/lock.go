// File: internal/domain/lock.go
// Created: 2026-06-20
// Description: Lock file models (PackLock, LockedMod, LockedSource, Identity, DepRef).

package domain

// PackLock represents a dependency lock file.
type PackLock struct {
	Loader           string               `json:"loader"`
	LoaderVersion    string               `json:"loaderVersion,omitempty"`
	MinecraftVersion string               `json:"minecraftVersion"`
	Mods             map[string]LockedMod `json:"mods"`
}

// LockedMod represents a resolved mod in a lock file.
type LockedMod struct {
	Name         string       `json:"name,omitempty"`
	Version      string       `json:"version,omitempty"`
	Scope        string       `json:"scope"`
	Identity     *Identity    `json:"identity,omitempty"`
	Dependencies []DepRef     `json:"dependencies,omitempty"`
	Source       LockedSource `json:"source"`
	// Hash is a fingerprint of the spec source that produced this lock entry
	// (canonical-JSON SHA256, first 16 bytes hex). When a subsequent lock
	// run sees the spec source for a key produce a different hash, it knows
	// the mod needs to be re-resolved even if the previous lock entry was
	// kept around. Stored alongside the lock entry so the diff is local to
	// the lock file.
	Hash string `json:"hash,omitempty"`
}

// LockedSource is the resolved source in a lock file.
type LockedSource struct {
	Type                 string            `json:"type"`
	ModID                int               `json:"modId,omitempty"`
	FileID               int               `json:"fileId,omitempty"`
	FileName             string            `json:"fileName"`
	Repo                 string            `json:"repo,omitempty"`
	Tag                  string            `json:"tag,omitempty"`
	AssetName            string            `json:"assetName,omitempty"`
	LockAssetName        string            `json:"lockAssetName,omitempty"`
	Path                 string            `json:"path,omitempty"`
	URL                  string            `json:"url,omitempty"`
	AssetPatternByLoader map[string]string `json:"assetPatternByLoader,omitempty"`
}

// Identity contains identity information for a resolved mod.
type Identity struct {
	Source     string `json:"source"`
	Internal   string `json:"internal,omitempty"`
	Confidence string `json:"confidence"`
}

// DepRef represents a dependency reference from jar metadata.
type DepRef struct {
	ID           string `json:"id"`
	VersionRange string `json:"versionRange,omitempty"`
	Required     bool   `json:"required"`
}
