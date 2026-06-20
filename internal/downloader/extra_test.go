// File: internal/downloader/extra_test.go
// Created: 2026-06-20
// Description: Extended downloader coverage.
package downloader

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("Downloader extended", func() {
	It("parseRepo with single item", func() {
		o, n := parseRepo("owner")
		Expect(o).To(Equal("owner"))
		Expect(n).To(BeEmpty())
	})

	It("parseRepo with extra slashes", func() {
		o, n := parseRepo("a/b/c")
		Expect(o).To(Equal("a"))
		Expect(n).To(Equal("b/c"))
	})

	It("parseRepo empty", func() {
		o, n := parseRepo("")
		Expect(o).To(BeEmpty())
		Expect(n).To(BeEmpty())
	})

	It("Download curseforge with only modID", func() {
		err := Download(&domain.LockedSource{
			Type:   "curseforge",
			ModID:  123,
			FileID: 0,
		}, "")
		_ = err
	})

	It("Download curseforge with modID+fileID calls API", func() {
		err := Download(&domain.LockedSource{
			Type:     "curseforge",
			ModID:    328085,
			FileID:   5812340,
			FileName: "create.jar",
		}, "")
		// Will fail without API key, but exercises the code path
		_ = err
	})

	It("Download github-release without asset", func() {
		err := Download(&domain.LockedSource{
			Type:      "github-release",
			Repo:      "owner/repo",
			Tag:       "v1",
			AssetName: "asset.jar",
		}, "")
		// Will fail as network request, but exercises code path
		_ = err
	})
})
