// File: internal/service/tree_service.go
// Created: 2026-06-20
// Description: Dependency tree display service.

package service

import (
	"fmt"
	"sort"
	"strings"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

// TreeEntry represents a node in the dependency tree.
type TreeEntry struct {
	Name        string
	Version     string
	Scope       string
	Source      string
	SourceIdent string
	Children    []TreeEntry
}

// BuildTree builds a dependency tree for display.
func BuildTree(lock *domain.PackLock) []TreeEntry {
	var roots []TreeEntry
	for key, mod := range lock.Mods {
		entry := TreeEntry{
			Name:        mod.Name,
			Version:     mod.Version,
			Scope:       mod.Scope,
			Source:      mod.Source.Type,
			SourceIdent: treeSourceIdent(mod.Source),
		}
		if entry.Name == "" {
			entry.Name = key
		}
		roots = append(roots, entry)
	}
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].Name < roots[j].Name
	})
	return roots
}

// treeSourceIdent returns a human-readable source identifier per spec 7.3
// examples: "curseforge:328085", "github:owner/repo", "local", "git:owner/repo".
func treeSourceIdent(src domain.LockedSource) string {
	switch src.Type {
	case "curseforge":
		return fmt.Sprintf("curseforge:%d", src.ModID)
	case "github-release":
		return "github:" + src.Repo
	case "git":
		return "git:" + src.Repo
	case "local":
		return "local"
	}
	return src.Type
}

// FormatTree formats a dependency tree as a string per spec 7.3:
//
//	<name> <source:identifier> <version>
//	  <child-name> <source:identifier> <version>
//	<name> <source:identifier> <version>
//
// followed by an optional "resolution:" block describing the selection reason.
func FormatTree(roots []TreeEntry) string {
	var b strings.Builder
	for _, r := range roots {
		b.WriteString(formatTreeLine(r))
		b.WriteString("\n")
		for _, c := range r.Children {
			b.WriteString("  ")
			b.WriteString(formatTreeLine(c))
			b.WriteString("\n")
		}
	}
	return b.String()
}

// formatTreeLine renders a single tree node as "<name> <source:identifier> <version>".
func formatTreeLine(r TreeEntry) string {
	ident := r.Source
	if r.SourceIdent != "" {
		ident = r.SourceIdent
	}
	return fmt.Sprintf("%s %s %s", r.Name, ident, r.Version)
}
