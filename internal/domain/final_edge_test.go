// File: internal/domain/final_edge_test.go
// Created: 2026-06-20
// Description: Cover marshal edge cases and validations.
package domain

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FinalEdge", func() {
	It("MarshalJSON with ClientMods+ServerMods builds mods", func() {
		spec := PackSpec{PackName: "e", MinecraftVersion: "1.21.1", LoaderName: []string{"fabric"}, PackVersion: "1",
			ClientMods: []ModSpec{{Name: "Client", Scope: "client", Source: ModSource{Type: SourceCurseForge, Query: "C"}}},
			ServerMods: []ModSpec{{Name: "Server", Scope: "server", Source: ModSource{Type: SourceCurseForge, Query: "S"}}},
		}
		data, err := spec.MarshalJSON()
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(MatchRegexp(`"mods"`))
	})
	It("MarshalJSON with only SharedMods", func() {
		spec := PackSpec{PackName: "s", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"}, PackVersion: "1",
			SharedMods: []ModSpec{{Name: "Shared", Source: ModSource{Type: SourceCurseForge, Query: "S"}}}}
		data, err := spec.MarshalJSON()
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(MatchRegexp(`"mods"`))
	})
	It("ValidateLock missing loader", func() {
		Expect(ValidateLock(PackLock{MinecraftVersion: "1.21.1", Mods: map[string]LockedMod{}})).To(HaveOccurred())
	})
	It("ValidateReleaseIndex missing release type", func() {
		Expect(ValidateReleaseIndex(ReleaseIndex{Type: "package", PackName: "p", MinecraftVersion: "1.21.1",
			Releases: []ReleaseRecord{{Version: "0.1.0"}}})).To(HaveOccurred())
	})
	It("AllEntriesForVariant empty entriesByMod", func() {
		spec := PackSpec{}
		Expect(AllEntriesForVariant(spec, "k")).To(BeEmpty())
	})
	It("EntriesForMod nil entriesByMod", func() {
		spec := PackSpec{}
		Expect(EntriesForMod(spec, "k")).To(BeNil())
	})
	It("AllMods empty", func() {
		Expect(AllMods(PackSpec{})).To(BeEmpty())
	})
	It("AllEntries empty", func() {
		Expect(AllEntries(PackSpec{})).To(BeEmpty())
	})
})
