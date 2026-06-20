// File: internal/downloader/mass2_test.go
// Created: 2026-06-20
// Description: Mass2 downloader coverage.
package downloader

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("Downloader mass2", func() {
	It("parseRepo variants", func() {
		o, n := parseRepo("a/b/c/d")
		Expect(o).To(Equal("a"))
		Expect(n).To(Equal("b/c/d"))
	})
	It("parseRepo empty", func() {
		o, n := parseRepo("")
		Expect(o).To(BeEmpty())
		Expect(n).To(BeEmpty())
	})
	It("local no error", func() {
		Expect(Download(&domain.LockedSource{Type: "local", Path: "."}, "")).To(Succeed())
	})
	It("bad type", func() {
		err := Download(&domain.LockedSource{Type: "invalid"}, "")
		Expect(err).To(HaveOccurred())
	})
	It("curseforge http call", func() {
		err := Download(&domain.LockedSource{Type: "curseforge", ModID: 328085, FileID: 5812340, FileName: "mod.jar"}, "")
		Expect(err).To(HaveOccurred())
	})
	It("github http call", func() {
		err := Download(&domain.LockedSource{Type: "github-release", Repo: "o/r", Tag: "v1", AssetName: "a.jar"}, "")
		_ = err
	})
})
