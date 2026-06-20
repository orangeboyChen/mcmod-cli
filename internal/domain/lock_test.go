// File: internal/domain/lock_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/domain/lock.go (PackLock, LockedMod, LockedSource, Identity, DepRef).

package domain

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Lock model", func() {
	It("Identity has the expected zero value", func() {
		var id Identity
		Expect(id.Source).To(BeEmpty())
		Expect(id.Internal).To(BeEmpty())
		Expect(id.Confidence).To(BeEmpty())
	})

	It("Identity JSON round-trip preserves fields", func() {
		original := Identity{Source: "curseforge", Internal: "mod|1|2", Confidence: "exact"}
		data, err := json.Marshal(original)
		Expect(err).NotTo(HaveOccurred())
		var got Identity
		Expect(json.Unmarshal(data, &got)).To(Succeed())
		Expect(got).To(Equal(original))
	})

	It("DepRef has the expected zero value", func() {
		var d DepRef
		Expect(d.ID).To(BeEmpty())
		Expect(d.VersionRange).To(BeEmpty())
		Expect(d.Required).To(BeFalse())
	})

	It("LockedSource JSON parses with URL and mod/file ids", func() {
		raw := []byte(`{"type":"curseforge","url":"https://x","modId":1,"fileId":2,"fileName":"m.jar"}`)
		var ls LockedSource
		Expect(json.Unmarshal(raw, &ls)).To(Succeed())
		Expect(ls.Type).To(Equal("curseforge"))
		Expect(ls.URL).To(Equal("https://x"))
		Expect(ls.ModID).To(Equal(1))
		Expect(ls.FileID).To(Equal(2))
		Expect(ls.FileName).To(Equal("m.jar"))
	})

	It("LockedMod JSON parses scope and sources", func() {
		raw := []byte(`{"name":"a","scope":"client","source":{"type":"local","path":"./a.jar"}}`)
		var lm LockedMod
		Expect(json.Unmarshal(raw, &lm)).To(Succeed())
		Expect(lm.Name).To(Equal("a"))
		Expect(lm.Scope).To(Equal("client"))
		Expect(lm.Source.Type).To(Equal("local"))
		Expect(lm.Source.Path).To(Equal("./a.jar"))
	})

	It("PackLock JSON round-trip preserves versions map", func() {
		original := PackLock{
			MinecraftVersion: "1.21.1",
			Loader:           "neoforge",
			LoaderVersion:    "21.0.0",
			Mods:             map[string]LockedMod{"a": {Name: "a", Scope: "shared"}},
		}
		data, err := json.Marshal(original)
		Expect(err).NotTo(HaveOccurred())
		var got PackLock
		Expect(json.Unmarshal(data, &got)).To(Succeed())
		Expect(got.MinecraftVersion).To(Equal("1.21.1"))
		Expect(got.Loader).To(Equal("neoforge"))
		Expect(got.Mods).To(HaveKey("a"))
	})
})
