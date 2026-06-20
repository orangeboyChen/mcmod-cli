// File: internal/graph/coverage_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for graph package.
package graph

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("Graph", func() {
	It("BuildGraph returns nodes and edges", func() {
		spec := &domain.PackSpec{MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
			Mods: map[string]domain.ModSpec{
				"a": {Name: "A", Scope: "shared", Source: domain.ModSource{Type: "curseforge"}},
				"b": {Name: "B", Scope: "client", Source: domain.ModSource{Type: "curseforge"}},
			}}
		nodes, edges, err := BuildGraph(spec, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(nodes).To(HaveLen(2))
		Expect(edges).To(HaveLen(2))
	})

	It("FilterModsByLoader filters correctly", func() {
		mods := map[string]domain.ModSpec{
			"all":  {Loader: nil, Source: domain.ModSource{Type: "curseforge"}},
			"neo":  {Loader: []string{"neoforge"}, Source: domain.ModSource{Type: "curseforge"}},
			"fabc": {Loader: []string{"fabric"}, Source: domain.ModSource{Type: "curseforge"}},
		}
		filtered := FilterModsByLoader(mods, "neoforge")
		Expect(filtered).To(HaveLen(2))
	})

	It("DetectCycle finds cycles", func() {
		nodes := []*ModNode{{Key: "a"}, {Key: "b"}}
		edges := []string{"root->a", "a->b", "b->a"}
		cycle, err := DetectCycle(nodes, edges)
		Expect(err).NotTo(HaveOccurred())
		Expect(cycle).NotTo(BeEmpty())
	})

	It("DetectNoCycle returns empty", func() {
		nodes := []*ModNode{{Key: "a"}, {Key: "b"}}
		edges := []string{"root->a", "root->b"}
		cycle, err := DetectCycle(nodes, edges)
		Expect(err).NotTo(HaveOccurred())
		Expect(cycle).To(BeEmpty())
	})

	It("ResolveDependencies returns keys", func() {
		spec := &domain.PackSpec{MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
			Mods: map[string]domain.ModSpec{"a": {Name: "A", Source: domain.ModSource{Type: "curseforge"}}}}
		keys, err := ResolveDependencies(spec, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(keys).To(HaveLen(1))
	})
})
