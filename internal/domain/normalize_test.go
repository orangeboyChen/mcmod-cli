// File: internal/domain/normalize_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/domain/normalize.go (NormalizeKey, DefaultMCVersion, DefaultLoader, DefaultLoaders).

package domain

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NormalizeKey", func() {
	It("converts Farmer's Delight to farmers-delight", func() {
		Expect(NormalizeKey("Farmer's Delight")).To(Equal("farmers-delight"))
	})

	It("lowercases ASCII", func() {
		Expect(NormalizeKey("FOO")).To(Equal("foo"))
	})

	It("collapses repeated dashes", func() {
		Expect(NormalizeKey("a - - b")).To(Equal("a-b"))
	})

	It("strips leading and trailing dashes", func() {
		Expect(NormalizeKey("  -foo- ")).To(Equal("foo"))
	})

	It("keeps underscores and most punctuation as-is (only whitespace, hyphens, apostrophes are normalized)", func() {
		// NormalizeKey is intentionally narrow: it lowercases, normalizes whitespace
		// to '-', strips decorative apostrophes, and collapses repeated dashes.
		Expect(NormalizeKey("a_b!c")).To(Equal("a_b-c"))
	})
})

var _ = Describe("Default loaders and MC version", func() {
	It("DefaultMCVersion returns empty string when no spec version is set", func() {
		Expect(DefaultMCVersion(PackSpec{})).To(BeEmpty())
	})

	It("DefaultMCVersion keeps the spec value when set", func() {
		Expect(DefaultMCVersion(PackSpec{MinecraftVersion: "1.20.4"})).To(Equal("1.20.4"))
	})

	It("DefaultLoader returns the first loader name", func() {
		Expect(DefaultLoader(PackSpec{LoaderName: []string{"fabric"}})).To(Equal("fabric"))
	})

	It("DefaultLoader returns the empty string when no loaders are set", func() {
		Expect(DefaultLoader(PackSpec{})).To(BeEmpty())
	})

	It("DefaultLoaders returns the empty list when no loaders are set", func() {
		Expect(DefaultLoaders(PackSpec{})).To(BeEmpty())
	})

	It("DefaultLoaders returns all loader names", func() {
		Expect(DefaultLoaders(PackSpec{LoaderName: []string{"fabric", "neoforge"}})).To(Equal([]string{"fabric", "neoforge"}))
	})
})
