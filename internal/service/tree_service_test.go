// File: internal/service/tree_service_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/service/tree_service.go (BuildTree / FormatTree / treeSourceIdent).

package service

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("Service treeSourceIdent", func() {
	It("curseforge returns id", func() {
		id := treeSourceIdent(domain.LockedSource{Type: "curseforge", ModID: 123})
		Expect(id).To(Equal("curseforge:123"))
	})
	It("github returns repo", func() {
		id := treeSourceIdent(domain.LockedSource{Type: "github-release", Repo: "o/r"})
		Expect(id).To(Equal("github:o/r"))
	})
	It("local returns local", func() {
		id := treeSourceIdent(domain.LockedSource{Type: "local"})
		Expect(id).To(Equal("local"))
	})
	It("unknown returns type", func() {
		id := treeSourceIdent(domain.LockedSource{Type: "wat"})
		Expect(id).To(Equal("wat"))
	})
})

var _ = Describe("BuildTree FormatTree", func() {
	It("formats tree entries", func() {
		lock := &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{"a": {Name: "A", Version: "1.0", Scope: "shared", Source: domain.LockedSource{Type: "curseforge", ModID: 12345}}}}
		tree := BuildTree(lock)
		Expect(tree).To(HaveLen(1))
		output := FormatTree(tree)
		Expect(output).To(ContainSubstring("A curseforge:12345 1.0"))
	})
})

var _ = Describe("FormatTree", func() {
	It("renders roots with their child entries", func() {
		roots := []TreeEntry{
			{
				Name: "mod-a", Source: "curseforge", Version: "1.0",
				Children: []TreeEntry{
					{Name: "mod-b", Source: "github", Version: "2.0", SourceIdent: "owner/repo@v1"},
				},
			},
		}
		out := FormatTree(roots)
		Expect(out).To(ContainSubstring("mod-a"))
		Expect(out).To(ContainSubstring("mod-b"))
		Expect(out).To(ContainSubstring("owner/repo@v1"))
	})
})
