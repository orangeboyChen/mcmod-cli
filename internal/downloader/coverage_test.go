// File: internal/downloader/coverage_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for downloader package.
package downloader

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("Downloader", func() {
	It("local type returns nil", func() {
		Expect(Download(&domain.LockedSource{Type: "local"}, "")).To(Succeed())
	})
	It("unknown type errors", func() {
		Expect(Download(&domain.LockedSource{Type: "unknown"}, "")).NotTo(Succeed())
	})
	It("curseforge without modID fails gracefully", func() {
		err := Download(&domain.LockedSource{Type: "curseforge", ModID: 0, FileID: 0}, "")
		Expect(err).To(HaveOccurred())
	})
	It("parseRepo splits correctly", func() {
		o, n := parseRepo("owner/repo")
		Expect(o).To(Equal("owner"))
		Expect(n).To(Equal("repo"))
	})
	It("parseRepo handles edge cases", func() {
		o, n := parseRepo("norepo")
		Expect(o).To(Equal("norepo"))
		Expect(n).To(BeEmpty())
	})
})
