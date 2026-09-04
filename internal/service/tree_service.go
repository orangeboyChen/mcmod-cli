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
	if lock == nil {
		return nil
	}
	children := make(map[string][]string)
	for key, mod := range lock.Mods {
		for _, dep := range mod.Dependencies {
			if _, ok := lock.Mods[dep.ID]; ok {
				children[key] = append(children[key], dep.ID)
			}
		}
	}
	dependent := make(map[string]bool)
	for _, deps := range children {
		for _, dep := range deps {
			dependent[dep] = true
		}
	}
	var roots []TreeEntry
	visited := make(map[string]bool)
	for key, mod := range lock.Mods {
		if dependent[key] {
			continue
		}
		entry := treeEntry(key, mod, children, lock, make(map[string]bool))
		roots = append(roots, entry)
		markTreeVisited(key, children, lock, visited, make(map[string]bool))
	}
	for key, mod := range lock.Mods {
		if visited[key] {
			continue
		}
		roots = append(roots, treeEntry(key, mod, children, lock, make(map[string]bool)))
		markTreeVisited(key, children, lock, visited, make(map[string]bool))
	}
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].Name < roots[j].Name
	})
	return roots
}

func markTreeVisited(key string, children map[string][]string, lock *domain.PackLock, visited, path map[string]bool) {
	if path[key] || visited[key] {
		return
	}
	path[key] = true
	visited[key] = true
	for _, childKey := range children[key] {
		if _, ok := lock.Mods[childKey]; ok {
			markTreeVisited(childKey, children, lock, visited, path)
		}
	}
	delete(path, key)
}

func treeEntry(key string, mod domain.LockedMod, children map[string][]string, lock *domain.PackLock, path map[string]bool) TreeEntry {
	entry := TreeEntry{Name: mod.Name, Version: mod.Version, Scope: mod.Scope, Source: mod.Source.Type, SourceIdent: treeSourceIdent(mod.Source)}
	if entry.Name == "" {
		entry.Name = key
	}
	if path[key] {
		return entry
	}
	path[key] = true
	defer delete(path, key)
	for _, childKey := range children[key] {
		if child, ok := lock.Mods[childKey]; ok {
			entry.Children = append(entry.Children, treeEntry(childKey, child, children, lock, path))
		}
	}
	sort.Slice(entry.Children, func(i, j int) bool { return entry.Children[i].Name < entry.Children[j].Name })
	return entry
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
		formatTreeNode(&b, r, 0)
	}
	return b.String()
}

func formatTreeNode(b *strings.Builder, entry TreeEntry, depth int) {
	b.WriteString(strings.Repeat("  ", depth))
	b.WriteString(formatTreeLine(entry))
	b.WriteString("\n")
	for _, child := range entry.Children {
		formatTreeNode(b, child, depth+1)
	}
}

// formatTreeLine renders a single tree node as "<name> <source:identifier> <version>".
func formatTreeLine(r TreeEntry) string {
	ident := r.Source
	if r.SourceIdent != "" {
		ident = r.SourceIdent
	}
	return fmt.Sprintf("%s %s %s", r.Name, ident, r.Version)
}
