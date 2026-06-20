// File: internal/domain/common.go
// Created: 2026-06-20
// Description: Shared constants and small value types for the domain package.

package domain

// Source type constants used by ModSource.Type and LockedSource.Type.
const (
	SourceCurseForge    = "curseforge"
	SourceGitHubRelease = "github-release"
	SourceGit           = "git"
	SourceLocal         = "local"
	SourceURL           = "url"
)

// Scope constants used by ModSpec.Scope and LockedMod.Scope.
const (
	ScopeShared = "shared"
	ScopeClient = "client"
	ScopeServer = "server"
)

// Build target constants used by EntrySpec.Target, BuildTarget, and release
// artifact routing.
const (
	TargetClient = "client"
	TargetServer = "server"
	TargetBoth   = "both"
)

// BuildTarget represents a build target type.
type BuildTarget string

// EntrySpec represents an entry configuration.
type EntrySpec struct {
	Name         string      `json:"name"`
	ArtifactName string      `json:"artifactName,omitempty"`
	Target       BuildTarget `json:"target,omitempty"`
}

// LoaderEntry holds a parsed loader name and version.
type LoaderEntry struct {
	Name    string
	Version string
}
