// File: internal/resolver/curseforge.go
// Created: 2026-06-20
// Description: CurseForge resolver common types and constants.

package resolver

// CFBaseURL is the base URL for the CurseForge API.
const CFBaseURL = "https://api.curseforge.com/v1"

// cfCandidate is an internal scoring wrapper for CurseForge search results.
type cfCandidate struct {
	ID   int
	Name string
	Slug string
	// 0 = exact match on slug or name, 1 = slug starts with query, 2 = contains
	Score int
}

// LoaderToCF converts a Minecraft loader name to a CurseForge game version string.
func LoaderToCF(loader string) int {
	switch loader {
	case "fabric":
		return 4
	case "neoforge":
		return 6
	}
	return 0
}
