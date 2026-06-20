// File: internal/domain/common_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/domain/common.go (constants, BuildTarget, EntrySpec, LoaderEntry).

package domain

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Common constants and value types", func() {
	It("Source type constants have the expected values", func() {
		Expect(SourceCurseForge).To(Equal("curseforge"))
		Expect(SourceGitHubRelease).To(Equal("github-release"))
		Expect(SourceGit).To(Equal("git"))
		Expect(SourceLocal).To(Equal("local"))
		Expect(SourceURL).To(Equal("url"))
	})

	It("Scope constants have the expected values", func() {
		Expect(ScopeShared).To(Equal("shared"))
		Expect(ScopeClient).To(Equal("client"))
		Expect(ScopeServer).To(Equal("server"))
	})

	It("Build target constants have the expected values", func() {
		Expect(TargetClient).To(Equal("client"))
		Expect(TargetServer).To(Equal("server"))
		Expect(TargetBoth).To(Equal("both"))
	})

	It("BuildTarget is a string-typed alias", func() {
		var t BuildTarget = "client"
		Expect(string(t)).To(Equal("client"))
	})

	It("EntrySpec JSON round-trip preserves target", func() {
		e := EntrySpec{Name: "main", ArtifactName: "mod.jar", Target: TargetClient}
		data, err := json.Marshal(e)
		Expect(err).NotTo(HaveOccurred())
		var got EntrySpec
		Expect(json.Unmarshal(data, &got)).To(Succeed())
		Expect(got).To(Equal(e))
	})
})
