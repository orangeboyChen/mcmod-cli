// File: internal/graph/graph.go
// Created: 2026-06-20
// Description: Dependency graph construction per spec section 9.

package graph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

// ModNode represents a mod in the dependency graph.
type ModNode struct {
	Key        string
	SpecMod    domain.ModSpec
	Scope      string
	SourceID   string
	InternalID string
	Version    string
	Deps       []string
}

// BuildGraph constructs the dependency graph from packspec for a loader.
func BuildGraph(spec *domain.PackSpec, mcVersion, loader string) ([]*ModNode, []string, error) {
	var nodes []*ModNode
	var edges []string

	mods := make(map[string]domain.ModSpec)
	for k, m := range spec.Mods {
		mods[k] = m
	}

	for key, mod := range mods {
		node := &ModNode{
			Key:      key,
			SpecMod:  mod,
			Scope:    mod.Scope,
			SourceID: domain.NormalizeKey(key),
		}
		nodes = append(nodes, node)
		edges = append(edges, fmt.Sprintf("%s->%s", "root", key))
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Key < nodes[j].Key
	})

	return nodes, edges, nil
}

// FilterModsByLoader filters mod definitions for a specific loader.
func FilterModsByLoader(mods map[string]domain.ModSpec, loader string) map[string]domain.ModSpec {
	result := make(map[string]domain.ModSpec)
	for k, m := range mods {
		if len(m.Loader) == 0 {
			result[k] = m
		} else {
			for _, l := range m.Loader {
				if l == loader {
					result[k] = m
					break
				}
			}
		}
	}
	return result
}

// DetectCycle detects cycles in the dependency graph.
func DetectCycle(nodes []*ModNode, edges []string) ([]string, error) {
	adj := make(map[string][]string)
	for _, e := range edges {
		parts := strings.Split(e, "->")
		if len(parts) == 2 {
			adj[parts[0]] = append(adj[parts[0]], parts[1])
		}
	}

	visited := make(map[string]bool)
	path := make([]string, 0)

	var dfs func(node string) ([]string, error)
	dfs = func(node string) ([]string, error) {
		if visited[node] {
			idx := -1
			for i, n := range path {
				if n == node {
					idx = i
					break
				}
			}
			if idx >= 0 {
				return append(path[idx:], node), nil
			}
			return nil, nil
		}
		visited[node] = true
		path = append(path, node)
		for _, dep := range adj[node] {
			if cycle, err := dfs(dep); err != nil {
				return cycle, err
			} else if len(cycle) > 0 {
				return cycle, nil
			}
		}
		path = path[:len(path)-1]
		return nil, nil
	}

	for _, node := range nodes {
		if cycle, err := dfs(node.Key); err != nil {
			return nil, err
		} else if len(cycle) > 0 {
			return cycle, nil
		}
	}

	return nil, nil
}
