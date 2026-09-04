// File: internal/cli/commands_tree_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for `mcmod tree` and `mcmod lock tree` (runTree).

package cli

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("tree alias", func() {
	It("tree without lock errors", func() {
		chdirTemp(`{"packName":"t","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		_, _, err := runCLI("tree", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("tree with lock works", func() {
		dir := chdirTemp(`{"packName":"t","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{
			Loader: "neoforge", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Version: "1", Scope: "shared", Source: domain.LockedSource{Type: "local", FileName: "a.jar"}},
			},
		})
		stdout, _, err := runCLI("tree", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("dependency tree"))
	})
})

var _ = Describe("lock tree", func() {
	It("lock tree without lock errors", func() {
		chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		_, _, err := runCLI("lock", "tree", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("lock tree with lock works", func() {
		dir := chdirTemp(`{"packName":"l","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{
			Loader: "neoforge", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Version: "1", Scope: "shared", Source: domain.LockedSource{Type: "local", FileName: "a.jar"}},
			},
		})
		_, _, err := runCLI("lock", "tree", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
	})
})
