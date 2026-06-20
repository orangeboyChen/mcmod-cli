// File: internal/cli/commands_build_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for `mcmod build` subcommand.

package cli

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("build", func() {
	It("build without spec errors", func() {
		chdirTemp("")
		_, _, err := runCLI("build")
		Expect(err).To(HaveOccurred())
	})
	It("build with missing lock prints hint", func() {
		chdirTemp(`{"packName":"b","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		_, stderr, err := runCLI("build", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
		Expect(stderr).To(ContainSubstring("hint"))
	})
	It("build with invalid target errors", func() {
		dir := chdirTemp(`{"packName":"b","packVersion":"1","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`)
		ensureLocksDir(dir)
		writeLockJSON(dir, "1.21.1", "neoforge", &domain.PackLock{
			Loader: "neoforge", MinecraftVersion: "1.21.1", Mods: map[string]domain.LockedMod{},
		})
		_, stderr, err := runCLI("build", "1.21.1", "neoforge", "--target", "wrong")
		Expect(err).To(HaveOccurred())
		Expect(stderr).To(ContainSubstring("invalid target"))
	})
})
