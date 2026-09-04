// File: internal/service/cfkey_service_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/service/cfkey_service.go (GetCFKey / ConfigureCFKey / ConfigureUserCFKey).

package service

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	It("GetCFKey reads from project config", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(ConfigureCFKey("proj-key")).To(Succeed())
		Expect(GetCFKey()).To(Equal("proj-key"))
	})
	It("ConfigureCFKey writes project config", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(ConfigureCFKey("test-key")).To(Succeed())
	})
})
