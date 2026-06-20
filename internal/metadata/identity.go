// File: internal/metadata/identity.go
// Created: 2026-06-20
// Description: Identity resolution from jar metadata.

package metadata

// SourceIdentity returns a source-level identity string.
func SourceIdentity(srcType string, id string) string {
	switch srcType {
	case "curseforge":
		return "curseforge:" + id
	case "github-release", "git":
		return "github:" + id
	default:
		return id
	}
}

// InternalIdentity returns a loader-specific internal identity.
func InternalIdentity(loaderFamily, modID string) string {
	return loaderFamily + ":" + modID
}

// IdentityConfidence represents how confident we are in the identity.
type IdentityConfidence string

const (
	// ConfidenceMetadata describes the identity confidence level for metadata-derived sources.
	ConfidenceMetadata IdentityConfidence = "metadata"
	// ConfidenceSourceOnly describes the identity confidence level when only a source is known.
	ConfidenceSourceOnly IdentityConfidence = "source-only"
	// ConfidenceManual describes the identity confidence level for manually specified mods.
	ConfidenceManual IdentityConfidence = "manual"
)
