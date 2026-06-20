// File: internal/metadata/boost_test.go
// Created: 2026-06-20
// Description: Boost coverage for metadata ReadJarMetadata.

package metadata

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metadata boost", func() {
	It("ReadJarMetadata on nonexistent file returns error", func() {
		_, err := ReadJarMetadata("/nonexistent/mod.jar")
		Expect(err).To(HaveOccurred())
	})

	It("DepInfoFromIdentity with colon", func() {
		info := DepInfoFromIdentity("curseforge:create")
		Expect(info.ModID).To(Equal("create"))
		Expect(info.Required).To(BeTrue())
	})

	It("DepInfoFromIdentity without colon", func() {
		info := DepInfoFromIdentity("simplemod")
		Expect(info.ModID).To(Equal("simplemod"))
	})

	It("SourceIdentity basic", func() {
		info := SourceIdentity("curseforge", "12345")
		Expect(info).To(Equal("curseforge:12345"))
	})

	It("SourceIdentity github returns github:", func() {
		info := SourceIdentity("github-release", "create")
		Expect(info).To(Equal("github:create"))
	})

	It("SourceIdentity default", func() {
		info := SourceIdentity("other", "modid")
		Expect(info).To(Equal("modid"))
	})

	It("InternalIdentity basic", func() {
		info := InternalIdentity("neoforge", "create")
		Expect(info).To(Equal("neoforge:create"))
	})
})
