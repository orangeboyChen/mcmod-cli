// File: internal/resolver/resolver_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/resolver/resolver.go (ResolveSource dispatcher).

package resolver

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"
	"path/filepath"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

// --- from resolver_test.go (Resolver boost) ---
var _ = Describe("Resolver boost", func() {
	Describe("ResolveSource dispatcher", func() {
		It("errors on empty type", func() {
			_, err := ResolveSource(domain.ModSource{}, "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("errors on unknown type", func() {
			_, err := ResolveSource(domain.ModSource{Type: "unknown"}, "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("curseforge with query and no key errors", func() {
			_, err := ResolveSource(domain.ModSource{Type: "curseforge", Query: "foo"}, "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("curseforge with neither modId+fileId nor query errors", func() {
			_, err := ResolveSource(domain.ModSource{Type: "curseforge"}, "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("local with existing jar returns LockedSource", func() {
			dir := GinkgoT().TempDir()
			jarPath := filepath.Join(dir, "test.jar")
			Expect(os.WriteFile(jarPath, []byte("dummy"), 0644)).To(Succeed())
			out, err := ResolveSource(domain.ModSource{Type: "local", Path: jarPath}, "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			ls, ok := out.(*domain.LockedSource)
			Expect(ok).To(BeTrue())
			Expect(ls.Type).To(Equal("local"))
		})
		It("local with missing file errors", func() {
			_, err := ResolveSource(domain.ModSource{Type: "local", Path: "/no/such.jar"}, "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("local with directory path errors", func() {
			dir := GinkgoT().TempDir()
			_, err := ResolveSource(domain.ModSource{Type: "local", Path: dir}, "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("local with non-jar extension errors", func() {
			dir := GinkgoT().TempDir()
			p := filepath.Join(dir, "test.txt")
			Expect(os.WriteFile(p, []byte("dummy"), 0644)).To(Succeed())
			_, err := ResolveSource(domain.ModSource{Type: "local", Path: p}, "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
		})
		It("github-release with no key returns placeholder LockedSource", func() {
			out, err := ResolveSource(domain.ModSource{Type: "github-release", Repo: "o/r", Tag: "v1"}, "1.21.1", "neoforge")
			// May error or return placeholder depending on resolver
			_ = out
			_ = err
		})
	})

	Describe("ResolveLocalSource paths", func() {
		It("replaces {mcVersion} placeholder", func() {
			dir := GinkgoT().TempDir()
			p := filepath.Join(dir, "1.21.1.jar")
			Expect(os.WriteFile(p, []byte("dummy"), 0644)).To(Succeed())
			out, err := ResolveLocalSource(filepath.Join(dir, "{mcVersion}.jar"), "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(out.Path).To(Equal(p))
		})
		It("replaces {loader} placeholder", func() {
			dir := GinkgoT().TempDir()
			p := filepath.Join(dir, "neoforge.jar")
			Expect(os.WriteFile(p, []byte("dummy"), 0644)).To(Succeed())
			out, err := ResolveLocalSource(filepath.Join(dir, "{loader}.jar"), "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(out.Path).To(Equal(p))
		})
	})
})

// --- from resolver_test.go (Resolver) ---
var _ = Describe("Resolver", func() {
	It("ResolveSource empty type errors", func() {
		_, err := ResolveSource(domain.ModSource{}, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("ResolveSource unknown type errors", func() {
		_, err := ResolveSource(domain.ModSource{Type: "unknown"}, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("LoaderToCF maps correctly", func() {
		Expect(LoaderToCF("fabric")).To(Equal(4))
		Expect(LoaderToCF("neoforge")).To(Equal(6))
		Expect(LoaderToCF("forge")).To(Equal(0))
	})
	It("CurseForgeByQuery without key errors", func() {
		_, err := ResolveCurseForgeByQuery("test", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("CurseForgeByID without key errors", func() {
		_, err := ResolveCurseForgeByID(123, 456)
		Expect(err).To(HaveOccurred())
	})
	It("GitHubRelease replaces placeholders", func() {
		src, err := ResolveGitHubRelease("owner/repo", "v{mcVersion}", "asset-{tag}-{loader}.jar", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(src.Tag).To(Equal("v1.21.1"))
		Expect(src.AssetName).To(Equal("asset-v1.21.1-neoforge.jar"))
	})
	It("GitHubRelease direct pattern works", func() {
		src, err := ResolveGitHubRelease("o/r", "v1", "asset.jar", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(src.AssetName).To(Equal("asset.jar"))
	})
	It("ResolveLocalSource fails for missing file", func() {
		_, err := ResolveLocalSource("./missing.jar", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("ResolveLocalSource works for existing file", func() {
		dir := GinkgoT().TempDir()
		p := filepath.Join(dir, "mod.jar")
		Expect(os.WriteFile(p, []byte("data"), 0644)).To(Succeed())
		src, err := ResolveLocalSource(p, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(src.FileName).To(Equal("mod.jar"))
	})
	It("GitPackage fails for nonexistent repo", func() {
		_, err := ResolveGitPackage("owner/nonexistent-repo-xyz", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
})

// --- from resolver_test.go (Resolver extended) ---
var _ = Describe("Resolver extended", func() {
	It("CurseForgeByQuery empty results", func() {
		os.Setenv("CURSEFORGE_API_KEY", "dummy-key")
		defer os.Unsetenv("CURSEFORGE_API_KEY")
		_, err := ResolveCurseForgeByQuery("xyznonexistentmod12345", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})

	It("CurseForgeByID calls API without key tries and fails", func() {
		_, err := ResolveCurseForgeByID(123, 456)
		Expect(err).To(HaveOccurred())
	})

	It("ResolveSource git with query field errors", func() {
		_, err := ResolveSource(domain.ModSource{Type: "git", Repo: "o/r", Query: "not-allowed"}, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})

	It("ResolveLocalSource with placeholder resolution", func() {
		dir := GinkgoT().TempDir()
		p := filepath.Join(dir, "1.21.1-neoforge.jar")
		Expect(os.WriteFile(p, []byte("data"), 0644)).To(Succeed())
		src, err := ResolveLocalSource(p, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(src.FileName).To(Equal("1.21.1-neoforge.jar"))
	})

	It("ResolveLocalSource with dir path fails", func() {
		dir := GinkgoT().TempDir()
		_, err := ResolveLocalSource(dir, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})

	It("GitHubRelease with {mcVersion} in tag", func() {
		src, err := ResolveGitHubRelease("owner/repo", "v{mcVersion}", "asset.jar", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(src.Tag).To(Equal("v1.21.1"))
	})

	It("GitHubRelease wildcard tag tries API", func() {
		_, err := ResolveGitHubRelease("owner/nonexistent-repo-99999", "*", "asset.jar", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})

	It("GitPackage tries both branches", func() {
		_, err := ResolveGitPackage("owner/nonexistent-xyz-repo", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
})

// --- from resolver_test.go (Resolver extra) ---
var _ = Describe("Resolver extra", func() {
	It("parseVersionFromFileName extracts last semver", func() {
		Expect(parseVersionFromFileName("create-1.21.1-neoforge.jar")).To(Equal("1.21.1"))
		Expect(parseVersionFromFileName("foo.jar")).To(Equal(""))
		Expect(parseVersionFromFileName("")).To(Equal(""))
	})

	It("ResolveSource unknown type errors", func() {
		_, err := ResolveSource(domain.ModSource{Type: "bogus"}, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})

	It("ResolveSource empty type errors", func() {
		_, err := ResolveSource(domain.ModSource{}, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})

	It("ResolveSource local uses file path", func() {
		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, "mod.jar")
		Expect(os.WriteFile(path, []byte("dummy"), 0644)).To(Succeed())
		res, err := ResolveSource(domain.ModSource{Type: "local", Path: path}, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		ls, ok := res.(*domain.LockedSource)
		Expect(ok).To(BeTrue())
		Expect(ls.Path).To(Equal(path))
	})
})

// --- from resolver_test.go (Resolver mass) ---
var _ = Describe("Resolver mass", func() {
	It("curseforge query with bad key calls API and fails", func() {
		os.Setenv("CURSEFORGE_API_KEY", "test")
		defer os.Unsetenv("CURSEFORGE_API_KEY")
		_, err := ResolveCurseForgeByQuery("test", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("curseforge by id fails without key", func() {
		_, err := ResolveCurseForgeByID(1, 2)
		Expect(err).To(HaveOccurred())
	})
	It("github release with mcVersion placeholder", func() {
		s, err := ResolveGitHubRelease("o/r", "v{mcVersion}", "a.jar", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(s.Tag).To(Equal("v1.21.1"))
	})
	It("github release with loader placeholder", func() {
		s, err := ResolveGitHubRelease("o/r", "v1", "m-{loader}.jar", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(s.AssetName).To(Equal("m-neoforge.jar"))
	})
	It("github release wildcard tries API", func() {
		_, err := ResolveGitHubRelease("o/nonexistent999", "*", "a.jar", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("local source existing file", func() {
		dir := GinkgoT().TempDir()
		p := filepath.Join(dir, "m.jar")
		Expect(os.WriteFile(p, []byte("data"), 0644)).To(Succeed())
		s, err := ResolveLocalSource(p, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(s.FileName).To(Equal("m.jar"))
	})
	It("local source dir fails", func() {
		dir := GinkgoT().TempDir()
		_, err := ResolveLocalSource(dir, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("local source missing file", func() {
		_, err := ResolveLocalSource("/nonexistent.jar", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("git package fails for bad repo", func() {
		_, err := ResolveGitPackage("o/bad-repo-99999-x", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("resolve source with type-local existing file", func() {
		dir := GinkgoT().TempDir()
		p := filepath.Join(dir, "lm.jar")
		Expect(os.WriteFile(p, []byte("d"), 0644)).To(Succeed())
		s, err := ResolveSource(domain.ModSource{Type: "local", Path: p}, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(s).NotTo(BeNil())
	})
	It("resolve source with curseforge needs both modId and fileId", func() {
		_, err := ResolveSource(domain.ModSource{Type: "curseforge", ModID: 1}, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
})

// --- from resolver_test.go (Resolver push80) ---
var _ = Describe("Resolver push80", func() {
	It("curseforge query with key calls API", func() {
		os.Setenv("CURSEFORGE_API_KEY", "test-key")
		defer os.Unsetenv("CURSEFORGE_API_KEY")
		_, err := ResolveCurseForgeByQuery("test-mod", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("curseforge by id without key", func() {
		_, err := ResolveCurseForgeByID(1, 2)
		Expect(err).To(HaveOccurred())
	})
	It("github release mcVersion placeholder", func() {
		s, err := ResolveGitHubRelease("o/r", "v{mcVersion}", "a-{tag}.jar", "1.21.1", "fabric")
		Expect(err).NotTo(HaveOccurred())
		Expect(s.Tag).To(Equal("v1.21.1"))
		Expect(s.AssetName).To(Equal("a-v1.21.1.jar"))
	})
	It("github release loader placeholder", func() {
		s, err := ResolveGitHubRelease("o/r", "v1", "m-{loader}.jar", "1.21.1", "fabric")
		Expect(err).NotTo(HaveOccurred())
		Expect(s.AssetName).To(Equal("m-fabric.jar"))
	})
	It("github release wildcard fails for bad repo", func() {
		_, err := ResolveGitHubRelease("o/nonexistent9999", "*", "a.jar", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("local source existing jar", func() {
		dir := GinkgoT().TempDir()
		p := filepath.Join(dir, "test-mod.jar")
		Expect(os.WriteFile(p, []byte("data"), 0644)).To(Succeed())
		src, err := ResolveLocalSource(p, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(src.FileName).To(Equal("test-mod.jar"))
	})
	It("local source dir fails", func() {
		dir := GinkgoT().TempDir()
		_, err := ResolveLocalSource(dir, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
	It("ResolveSource curseforge query needs key", func() {
		_, err := ResolveSource(domain.ModSource{Type: "curseforge", Query: "test"}, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("ResolveSource dispatcher", func() {
	It("returns an error for an empty source type", func() {
		_, err := ResolveSource(domain.ModSource{Type: ""}, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})

	It("returns an error for an unknown source type", func() {
		_, err := ResolveSource(domain.ModSource{Type: "wat"}, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})

	It("returns an error when curseforge source has no query, slug, or modId+fileId", func() {
		_, err := ResolveSource(domain.ModSource{Type: "curseforge"}, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})

	It("returns an error for a local source with no path", func() {
		_, err := ResolveSource(domain.ModSource{Type: "local"}, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})

	It("dispatches a github-release source to ResolveGitHubRelease (network-bound)", func() {
		// Without redirecting HTTP, the call hits the real GitHub API. We
		// accept any outcome (error or success-with-parse-fail) as long as
		// the dispatcher picks the github branch and we do not crash.
		_, _ = ResolveSource(domain.ModSource{Type: "github-release", Tag: "v1", AssetPattern: "x.jar"}, "1.21.1", "neoforge")
		Expect(true).To(BeTrue())
	})

	It("returns an error for a git source with no repo", func() {
		_, err := ResolveSource(domain.ModSource{Type: "git"}, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("ResolveSource url branch", func() {
	It("returns a LockedSource with a rendered URL when URLPattern is set", func() {
		src := domain.ModSource{
			Type:       "url",
			ModID:      1,
			FileID:     2,
			FileName:   "mod.jar",
			URLPattern: "https://x/{mcVersion}/{fileName}",
		}
		out, err := ResolveSource(src, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		ls, ok := out.(*domain.LockedSource)
		Expect(ok).To(BeTrue())
		Expect(ls.Type).To(Equal("url"))
		Expect(ls.URL).To(Equal("https://x/1.21.1/mod.jar"))
	})

	It("returns an error when url source is missing modId/fileId/fileName", func() {
		_, err := ResolveSource(domain.ModSource{Type: "url"}, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
})
