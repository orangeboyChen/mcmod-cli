// File: internal/domain/release_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/domain/release.go (ReleaseIndex, ReleaseRecord, ReleaseGitHub, ReleaseArtifactSet).

package domain

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// --- from spec_test.go consolidated (Domain FindRelease) ---
var _ = Describe("Domain FindRelease", func() {
	It("returns the matching release", func() {
		ri := &ReleaseIndex{Releases: []ReleaseRecord{
			{Version: "0.1.0"}, {Version: "0.2.0"},
		}}
		r := ri.FindRelease("0.2.0")
		Expect(r).NotTo(BeNil())
		Expect(r.Version).To(Equal("0.2.0"))
	})
	It("returns nil when not found", func() {
		ri := &ReleaseIndex{Releases: []ReleaseRecord{{Version: "0.1.0"}}}
		r := ri.FindRelease("0.3.0")
		Expect(r).To(BeNil())
	})
	It("handles empty index", func() {
		ri := &ReleaseIndex{}
		Expect(ri.FindRelease("0.1.0")).To(BeNil())
	})
})

var _ = Describe("Release artifact routing", func() {
	rec := func() *ReleaseRecord {
		return &ReleaseRecord{Version: "1.0.0", Type: "release"}
	}

	It("SetArtifact stores the path and ArtifactFor reads it back", func() {
		r := rec()
		r.SetArtifact("neoforge", "client", "releases/v1.0.0/client.jar")
		Expect(r.ArtifactFor("neoforge", "client")).To(Equal("releases/v1.0.0/client.jar"))
	})

	It("ArtifactFor returns empty when the artifact is missing", func() {
		r := rec()
		Expect(r.ArtifactFor("neoforge", "client")).To(BeEmpty())
	})

	It("RemoveArtifact clears the entry", func() {
		r := rec()
		r.SetArtifact("neoforge", "client", "x.jar")
		r.RemoveArtifact("neoforge", "client")
		Expect(r.ArtifactFor("neoforge", "client")).To(BeEmpty())
	})
})

var _ = Describe("ReleaseIndex mutation", func() {
	It("EnsureRelease creates a new entry when version is new", func() {
		ri := &ReleaseIndex{}
		rec := ri.EnsureRelease("1.0.0", "release")
		Expect(rec).NotTo(BeNil())
		Expect(rec.Version).To(Equal("1.0.0"))
		Expect(rec.Type).To(Equal("release"))
	})

	It("EnsureRelease returns the existing entry when version is known", func() {
		ri := &ReleaseIndex{Releases: []ReleaseRecord{{Version: "1.0.0", Type: "draft"}}}
		rec := ri.EnsureRelease("1.0.0", "release")
		Expect(rec.Type).To(Equal("draft")) // existing wins
	})

	It("DeleteRelease removes an existing version", func() {
		ri := &ReleaseIndex{Releases: []ReleaseRecord{{Version: "1.0.0"}}}
		Expect(ri.DeleteRelease("1.0.0")).To(BeTrue())
		Expect(ri.Releases).To(BeEmpty())
	})

	It("DeleteRelease returns false for an unknown version", func() {
		ri := &ReleaseIndex{Releases: []ReleaseRecord{{Version: "1.0.0"}}}
		Expect(ri.DeleteRelease("9.9.9")).To(BeFalse())
	})
})

var _ = Describe("Release artifact routing extended", func() {
	It("SetArtifact for server target", func() {
		r := &ReleaseRecord{Version: "1.0.0", Type: "release"}
		r.SetArtifact("neoforge", "server", "server.jar")
		Expect(r.ArtifactFor("neoforge", "server")).To(Equal("server.jar"))
	})

	It("SetArtifact for both target sets both fields", func() {
		r := &ReleaseRecord{Version: "1.0.0", Type: "release"}
		r.SetArtifact("neoforge", "both", "both.jar")
		Expect(r.ArtifactFor("neoforge", "client")).To(Equal("both.jar"))
		Expect(r.ArtifactFor("neoforge", "server")).To(Equal("both.jar"))
	})

	It("ArtifactFor default branch returns client", func() {
		r := &ReleaseRecord{Version: "1.0.0", Type: "release"}
		r.SetArtifact("neoforge", "client", "c.jar")
		Expect(r.ArtifactFor("neoforge", "bogus")).To(Equal("c.jar"))
	})

	It("RemoveArtifact for server target", func() {
		r := &ReleaseRecord{Version: "1.0.0", Type: "release"}
		r.SetArtifact("neoforge", "server", "s.jar")
		r.RemoveArtifact("neoforge", "server")
		Expect(r.ArtifactFor("neoforge", "server")).To(BeEmpty())
	})

	It("RemoveArtifact for both clears both", func() {
		r := &ReleaseRecord{Version: "1.0.0", Type: "release"}
		r.SetArtifact("neoforge", "both", "b.jar")
		r.RemoveArtifact("neoforge", "both")
		Expect(r.ArtifactFor("neoforge", "client")).To(BeEmpty())
		Expect(r.ArtifactFor("neoforge", "server")).To(BeEmpty())
	})

	It("RemoveArtifact on missing loader is a no-op", func() {
		r := &ReleaseRecord{Version: "1.0.0", Type: "release"}
		r.RemoveArtifact("unknown", "client")
		Expect(r.Artifact).To(BeEmpty())
	})
})

var _ = Describe("ReleaseIndex normalize", func() {
	It("Normalize sets default type", func() {
		ri := &ReleaseIndex{}
		ri.Normalize()
		Expect(ri.Type).To(Equal("package"))
	})

	It("Normalize preserves existing type", func() {
		ri := &ReleaseIndex{Type: "custom"}
		ri.Normalize()
		Expect(ri.Type).To(Equal("custom"))
	})
})
