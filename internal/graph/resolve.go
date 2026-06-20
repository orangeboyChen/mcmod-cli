// File: internal/graph/resolve.go
// Created: 2026-06-20
// Description: Graph-based dependency resolution.

package graph

import (
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

// ResolveDependencies resolves transitive dependencies for a lock.
func ResolveDependencies(spec *domain.PackSpec, mcVersion, loader string) ([]string, error) {
	nodes, _, err := BuildGraph(spec, mcVersion, loader)
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, n := range nodes {
		keys = append(keys, n.Key)
	}
	return keys, nil
}
