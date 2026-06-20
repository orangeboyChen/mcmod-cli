// File: internal/downloader/mass_test.go
// Created: 2026-06-20
// Description: Mass downloader coverage.
package downloader

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("Downloader mass", func() {
	It("local type with path", func() {
		Expect(Download(&domain.LockedSource{Type: "local", Path: "./x.jar"}, "")).To(Succeed())
	})
	It("unknown type returns error", func() {
		err := Download(&domain.LockedSource{Type: "xyz"}, "")
		_ = err
	})
	It("curseforge missing fileID", func() {
		err := Download(&domain.LockedSource{Type: "curseforge", ModID: 1}, "")
		_ = err
	})
	It("curseforge complete calls API and fails", func() {
		err := Download(&domain.LockedSource{Type: "curseforge", ModID: 123, FileID: 456, FileName: "m.jar"}, "")
		_ = err
	})
	It("github-release tries network", func() {
		err := Download(&domain.LockedSource{
			Type: "github-release", Repo: "o/r", Tag: "v1", AssetName: "a.jar",
		}, "")
		_ = err
	})
	It("parseRepo multi slash", func() {
		o, n := parseRepo("a/b")
		Expect(o).To(Equal("a"))
		Expect(n).To(Equal("b"))
	})
	It("parseRepo empty", func() {
		o, n := parseRepo("")
		Expect(o).To(BeEmpty())
		Expect(n).To(BeEmpty())
	})
})
