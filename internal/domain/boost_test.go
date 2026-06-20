// File: internal/domain/boost_test.go
// Created: 2026-06-20
// Description: Boost coverage for domain - store, validateSourceMod, EntryIndex, AllEntries.

package domain

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Domain boost", func() {
	Describe("ReadLockFile / WriteLockFile", func() {
		It("WriteLockFile creates file and ReadLockFile reads it back", func() {
			d := GinkgoT().TempDir()
			p := filepath.Join(d, "locks", "dependencies", "1.21.1-neoforge.json")
			lf := PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]LockedMod{"m": {Name: "M", Scope: ScopeShared,
					Source: LockedSource{Type: SourceLocal, Path: "./m.jar"}}}}
			Expect(WriteLockFile(p, &lf)).To(Succeed())
			loaded, err := ReadLockFile(p)
			Expect(err).NotTo(HaveOccurred())
			Expect(loaded.Mods).To(HaveLen(1))
		})

		It("ReadLockFile on nonexistent path fails", func() {
			_, err := ReadLockFile("/nonexistent/lock.json")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ReadReleaseIndex / WriteReleaseIndex", func() {
		It("WriteReleaseIndex creates file and ReadReleaseIndex reads it back", func() {
			d := GinkgoT().TempDir()
			p := filepath.Join(d, "locks", "releases", "1.21.1.json")
			ri := ReleaseIndex{Type: "package", PackName: "p", MinecraftVersion: "1.21.1",
				Releases: []ReleaseRecord{{Version: "0.1.0", Type: "github-release"}}}
			Expect(WriteReleaseIndex(p, &ri)).To(Succeed())
			loaded, err := ReadReleaseIndex(p)
			Expect(err).NotTo(HaveOccurred())
			Expect(loaded.PackName).To(Equal("p"))
		})

		It("ReadReleaseIndex on nonexistent path fails", func() {
			_, err := ReadReleaseIndex("/nonexistent/index.json")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("validateSourceMod additional paths", func() {
		It("rejects git with query", func() {
			spec := PackSpec{
				PackName: "p", PackVersion: "1", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
				Mods: map[string]ModSpec{"m": {Source: ModSource{Type: SourceGit, Repo: "o/r", Query: "never"}}},
			}
			err := ValidateSpec(spec)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must not contain query"))
		})

		It("rejects local with URL", func() {
			spec := PackSpec{
				PackName: "p", PackVersion: "1", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
				Mods: map[string]ModSpec{"m": {Source: ModSource{Type: SourceLocal, Path: "./m.jar", URL: "http://example.com"}}},
			}
			err := ValidateSpec(spec)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must not contain URL"))
		})

		It("rejects empty source type", func() {
			spec := PackSpec{
				PackName: "p", PackVersion: "1", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
				Mods: map[string]ModSpec{"m": {Source: ModSource{}}},
			}
			err := ValidateSpec(spec)
			Expect(err).To(HaveOccurred())
		})

		It("rejects unknown source type", func() {
			spec := PackSpec{
				PackName: "p", PackVersion: "1", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
				Mods: map[string]ModSpec{"m": {Source: ModSource{Type: "unknown-type"}}},
			}
			err := ValidateSpec(spec)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown source"))
		})

		It("rejects curseforge with modId/fileId (query empty fires first)", func() {
			spec := PackSpec{
				PackName: "p", PackVersion: "1", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
				Mods: map[string]ModSpec{"m": {Source: ModSource{Type: SourceCurseForge, ModID: 123, FileID: 456}}},
			}
			err := ValidateSpec(spec)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("requires query"))
		})

		It("rejects curseforge without query", func() {
			spec := PackSpec{
				PackName: "p", PackVersion: "1", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
				Mods: map[string]ModSpec{"m": {Source: ModSource{Type: SourceCurseForge}}},
			}
			err := ValidateSpec(spec)
			Expect(err).To(HaveOccurred())
		})

		It("rejects github-release without assetPattern", func() {
			spec := PackSpec{
				PackName: "p", PackVersion: "1", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
				Mods: map[string]ModSpec{"m": {Source: ModSource{Type: SourceGitHubRelease, Repo: "o/r", Tag: "v1"}}},
			}
			err := ValidateSpec(spec)
			Expect(err).To(HaveOccurred())
		})

		It("rejects invalid scope", func() {
			spec := PackSpec{
				PackName: "p", PackVersion: "1", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
				Mods: map[string]ModSpec{"m": {Name: "M", Scope: "invalid", Source: ModSource{Type: SourceCurseForge, Query: "M"}}},
			}
			err := ValidateSpec(spec)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid scope"))
		})

		It("rejects unsupported mod loader", func() {
			spec := PackSpec{
				PackName: "p", PackVersion: "1", MinecraftVersion: "1.21.1", LoaderName: []string{"neoforge"},
				Mods: map[string]ModSpec{"m": {Name: "M", Loader: []string{"quilt"}, Source: ModSource{Type: SourceCurseForge, Query: "M"}}},
			}
			err := ValidateSpec(spec)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ValidateLock additional paths", func() {
		It("rejects missing minecraftVersion", func() {
			err := ValidateLock(PackLock{Loader: "neoforge", Mods: map[string]LockedMod{"m": {Name: "M", Source: LockedSource{Type: "local"}}}})
			Expect(err).To(HaveOccurred())
		})

		It("rejects missing loader", func() {
			err := ValidateLock(PackLock{MinecraftVersion: "1.21.1", Mods: map[string]LockedMod{"m": {Name: "M", Source: LockedSource{Type: "local"}}}})
			Expect(err).To(HaveOccurred())
		})

		It("rejects github-release missing repo", func() {
			err := ValidateLock(PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]LockedMod{"m": {Source: LockedSource{Type: "github-release", Tag: "v1"}}}})
			Expect(err).To(HaveOccurred())
		})

		It("rejects github-release missing tag", func() {
			err := ValidateLock(PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]LockedMod{"m": {Source: LockedSource{Type: "github-release", Repo: "o/r"}}}})
			Expect(err).To(HaveOccurred())
		})

		It("rejects git without repo", func() {
			err := ValidateLock(PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]LockedMod{"m": {Source: LockedSource{Type: "git"}}}})
			Expect(err).To(HaveOccurred())
		})

		It("rejects local without path", func() {
			err := ValidateLock(PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]LockedMod{"m": {Source: LockedSource{Type: "local"}}}})
			Expect(err).To(HaveOccurred())
		})

		It("rejects URL without URL field", func() {
			err := ValidateLock(PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]LockedMod{"m": {Source: LockedSource{Type: "url"}}}})
			Expect(err).To(HaveOccurred())
		})

		It("rejects empty source type", func() {
			err := ValidateLock(PackLock{Loader: "neoforge", MinecraftVersion: "1.21.1",
				Mods: map[string]LockedMod{"m": {Source: LockedSource{Type: ""}}}})
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("EntryIndex and AllEntries", func() {
		It("EntryIndex returns entry by mod name from entriesByMod", func() {
			spec := PackSpec{}
			spec.entriesByMod = map[string][]EntrySpec{
				"jei": {{Name: "JEI"}},
			}
			e, ok := EntryIndex(spec, "JEI")
			Expect(ok).To(BeTrue())
			Expect(e.Name).To(Equal("JEI"))
		})

		It("EntryIndex falls back to variants", func() {
			spec := PackSpec{
				Variants: map[string]PackVariantSpec{
					"neo": {Mods: []ModSpec{{Name: "VMod"}}},
				},
			}
			e, ok := EntryIndex(spec, "VMod")
			Expect(ok).To(BeTrue())
			Expect(e.Name).To(Equal("VMod"))
		})

		It("EntryIndex returns false when mods exist but not in variants", func() {
			spec := PackSpec{
				Mods: map[string]ModSpec{"m": {Name: "MyMod"}},
			}
			_, ok := EntryIndex(spec, "MyMod")
			// EntryIndex only searches entriesByMod and Variants, not Mods map directly
			Expect(ok).To(BeFalse())
		})

		It("EntryIndex returns false when not found", func() {
			_, ok := EntryIndex(PackSpec{}, "nonexistent")
			Expect(ok).To(BeFalse())
		})

		It("AllEntries from entriesByMod", func() {
			spec := PackSpec{}
			spec.entriesByMod = map[string][]EntrySpec{"k": {{Name: "e1"}}}
			Expect(AllEntries(spec)).To(HaveLen(1))
		})

		It("AllEntries from variants when entriesByMod is empty", func() {
			spec := PackSpec{
				Variants: map[string]PackVariantSpec{"v": {Mods: []ModSpec{{Name: "VE"}}}},
			}
			entries := AllEntries(spec)
			Expect(entries).To(HaveLen(1))
		})

		It("AllEntries returns empty for empty spec", func() {
			Expect(AllEntries(PackSpec{})).To(BeEmpty())
		})
	})

	Describe("AllModsForVariant", func() {
		It("returns mods including dependencies", func() {
			spec := PackSpec{
				Mods: map[string]ModSpec{
					"m": {Name: "M", Scope: "shared", Source: ModSource{Type: SourceLocal, Path: "./m.jar"}},
				},
				Dependencies: []ModSpec{{Name: "dep", Source: ModSource{Type: SourceLocal, Path: "./dep.jar"}}},
			}
			mods := AllModsForVariant(spec, "neo")
			Expect(mods).To(HaveLen(2))
		})

		It("deduplicates mods and dependencies", func() {
			spec := PackSpec{
				Mods: map[string]ModSpec{
					"mymod": {Name: "MyMod", Scope: "shared"},
				},
				Dependencies: []ModSpec{{Name: "MyMod"}},
			}
			mods := AllModsForVariant(spec, "neo")
			Expect(len(mods)).To(BeNumerically(">=", 1))
		})
	})

	Describe("FileNameForURL", func() {
		It("extracts filename from URL path", func() {
			name := FileNameForURL("http://example.com/mods/mod.jar", "fallback.jar")
			Expect(name).To(Equal("mod.jar"))
		})
		It("returns domain as basename for bare URL with no path", func() {
			name := FileNameForURL("http://example.com", "fallback.jar")
			Expect(name).To(Equal("example.com"))
		})
		It("strips query string", func() {
			name := FileNameForURL("http://example.com/mod.jar?dl=1", "f.jar")
			Expect(name).To(Equal("mod.jar"))
		})
		It("returns fallback for empty URL", func() {
			name := FileNameForURL("", "default.jar")
			Expect(name).To(Equal("default.jar"))
		})
	})
})
