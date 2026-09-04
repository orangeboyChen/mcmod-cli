// File: internal/service/spec_io_service_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/service/spec_io_service.go (ReadPackSpec / WriteReleaseIndex / ReadReleaseIndex).

package service

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ReadPackSpec", func() {
	It("fails in empty dir", func() {
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(GinkgoT().TempDir())
		_, err := ReadPackSpec(".")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("ReadReleaseIndex", func() {
	It("fails for missing index", func() {
		_, err := ReadReleaseIndex("99.99")
		Expect(err).To(HaveOccurred())
	})
})
