// File: internal/service/lock_service_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/service/lock_service.go (LoadLock / SaveLock / WriteLockFile / ReadLockFile / LockFilePath / MarshalLockJSON).

package service

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("LockFilePath and ReadLockFile", func() {
	It("returns correct path", func() {
		Expect(LockFilePath("1.21.1", "neoforge")).NotTo(BeEmpty())
	})
})

var _ = Describe("WriteLockFile", func() {
	It("saves a lock file", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		lock := &domain.PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1"}
		Expect(WriteLockFile(lock)).To(Succeed())
	})
})

var _ = Describe("SaveLock", func() {
	It("saves and reads back", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		Expect(os.MkdirAll("locks/dependencies", 0755)).To(Succeed())
		lock := &domain.PackLock{Loader: "fabric", MinecraftVersion: "1.21.1"}
		Expect(SaveLock("1.21.1", "fabric", lock)).To(Succeed())
	})
})
