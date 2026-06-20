// File: internal/service/release_lock_service_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/service/release_lock_service.go (CreateReleaseRecord).

package service

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("CreateReleaseRecord", func() {
	It("creates a release record", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.MkdirAll("locks/releases", 0755)).To(Succeed())
		index, err := CreateReleaseRecord("1.21.1", "0.1.0", "github-release", &domain.ReleaseGitHub{Repo: "o/r", Tag: "v0.1.0"})
		Expect(err).NotTo(HaveOccurred())
		Expect(index.Releases).To(HaveLen(1))
	})
})
