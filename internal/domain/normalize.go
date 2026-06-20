// File: internal/domain/normalize.go
// Created: 2026-06-20
// Description: Mods key normalization per spec section 3.2.

package domain

import (
	"regexp"
	"strings"
)

var (
	specialCharRe = regexp.MustCompile(`[^\w\s-]`)
	dashRunRe     = regexp.MustCompile(`-{2,}`)
)

// NormalizeKey converts a display name to a stable ID key per spec 3.2 rules.
// 1. Lowercase
// 2. Spaces, underscores, most punctuation -> '-'
// 3. Decorative apostrophes deleted
// 4. Consecutive '-' collapsed
// 5. Strip leading/trailing '-'
func NormalizeKey(name string) string {
	s := strings.ToLower(name)

	// Remove decorative apostrophes (single quote, right single quote)
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, "\u2019", "")

	// Replace spaces, underscores, punctuation with dash
	// Note: we replace space with dash, and then collapse consecutive dashes
	s = specialCharRe.ReplaceAllString(s, "-")

	// Collapse consecutive dashes
	s = dashRunRe.ReplaceAllString(s, "-")

	// Strip leading/trailing dashes
	s = strings.Trim(s, "-")

	// If the result has no dashes (e.g. "farmers delight" became "farmers delight"),
	// we need to handle spaces that were not caught by the regex.
	// The regex [^\w\s-] does NOT match space, so spaces remain.
	// We explicitly replace any remaining space with dash.
	s = strings.ReplaceAll(s, " ", "-")

	// Collapse again after space replacement
	s = dashRunRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")

	return s
}

// DefaultMCVersion returns the default Minecraft version for a spec: the
// spec's minecraftVersion field, or "" when it is missing. Used by CLI
// commands that take an optional <minecraftVersion> positional argument
// and want a sensible fallback when the user does not pass one.
func DefaultMCVersion(spec PackSpec) string {
	return spec.MinecraftVersion
}

// DefaultLoader returns the default loader for a spec: the loader name
// parsed from the first loaderName entry, or "" when no loaders are
// declared. Used by CLI commands that take an optional <loader>
// positional argument and want a sensible fallback when the user does
// not pass one.
func DefaultLoader(spec PackSpec) string {
	return PrimaryLoaderName(spec)
}

// DefaultLoaders returns the full list of declared loader names for a
// spec. Used by CLI commands that want to iterate over every declared
// loader when the user does not pass a specific <loader>.
func DefaultLoaders(spec PackSpec) []string {
	var out []string
	for _, ln := range spec.LoaderName {
		name, _ := ParseLoaderName(ln)
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}
