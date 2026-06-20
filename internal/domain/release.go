// File: internal/domain/release.go
// Created: 2026-06-20
// Description: Release index models (ReleaseIndex, ReleaseRecord, ReleaseGitHub, ReleaseArtifactSet).

package domain

// ReleaseIndex is the build artifact index.
type ReleaseIndex struct {
	Type             string          `json:"type"`
	PackName         string          `json:"packName"`
	MinecraftVersion string          `json:"minecraftVersion"`
	Releases         []ReleaseRecord `json:"releases"`
}

// Normalize sets default fields on a ReleaseIndex.
func (ri *ReleaseIndex) Normalize() {
	if ri.Type == "" {
		ri.Type = "package"
	}
}

// EnsureRelease finds or creates a release record for a version.
func (ri *ReleaseIndex) EnsureRelease(version, releaseType string) *ReleaseRecord {
	for i := range ri.Releases {
		if ri.Releases[i].Version == version {
			return &ri.Releases[i]
		}
	}
	rec := ReleaseRecord{Version: version, Type: releaseType, Artifact: make(map[string]ReleaseArtifactSet)}
	ri.Releases = append(ri.Releases, rec)
	return &ri.Releases[len(ri.Releases)-1]
}

// FindRelease returns a pointer to the release record for the given version, or nil.
func (ri *ReleaseIndex) FindRelease(version string) *ReleaseRecord {
	for i := range ri.Releases {
		if ri.Releases[i].Version == version {
			return &ri.Releases[i]
		}
	}
	return nil
}

// DeleteRelease removes a release record by version.
func (ri *ReleaseIndex) DeleteRelease(version string) bool {
	for i, r := range ri.Releases {
		if r.Version == version {
			ri.Releases = append(ri.Releases[:i], ri.Releases[i+1:]...)
			return true
		}
	}
	return false
}

// ReleaseRecord represents a single release version.
type ReleaseRecord struct {
	Version  string                        `json:"version"`
	Type     string                        `json:"type"`
	GitHub   ReleaseGitHub                 `json:"github,omitempty"`
	Artifact map[string]ReleaseArtifactSet `json:"artifact"`
}

// ReleaseGitHub contains GitHub release metadata.
type ReleaseGitHub struct {
	Repo  string `json:"repo"`
	Tag   string `json:"tag"`
	Name  string `json:"name,omitempty"`
	Body  string `json:"body,omitempty"`
	Draft bool   `json:"draft,omitempty"`
	Pre   bool   `json:"prerelease,omitempty"`
}

// ReleaseArtifactSet contains client/server artifact paths for one loader.
type ReleaseArtifactSet struct {
	Client string `json:"client,omitempty"`
	Server string `json:"server,omitempty"`
}

// SetArtifact sets an artifact path for a loader and target.
func (r *ReleaseRecord) SetArtifact(loader string, target BuildTarget, path string) {
	if r.Artifact == nil {
		r.Artifact = make(map[string]ReleaseArtifactSet)
	}
	set := r.Artifact[loader]
	switch target {
	case TargetClient:
		set.Client = path
	case TargetServer:
		set.Server = path
	case TargetBoth:
		set.Client = path
		set.Server = path
	}
	r.Artifact[loader] = set
}

// ArtifactFor returns the artifact path for a loader and target.
func (r *ReleaseRecord) ArtifactFor(loader string, target BuildTarget) string {
	set, ok := r.Artifact[loader]
	if !ok {
		return ""
	}
	switch target {
	case TargetClient:
		return set.Client
	case TargetServer:
		return set.Server
	case TargetBoth:
		return set.Client
	default:
		return set.Client
	}
}

// RemoveArtifact removes an artifact for a loader and target.
func (r *ReleaseRecord) RemoveArtifact(loader string, target BuildTarget) {
	set, ok := r.Artifact[loader]
	if !ok {
		return
	}
	switch target {
	case TargetClient:
		set.Client = ""
	case TargetServer:
		set.Server = ""
	case TargetBoth:
		set.Client = ""
		set.Server = ""
	}
	r.Artifact[loader] = set
}
