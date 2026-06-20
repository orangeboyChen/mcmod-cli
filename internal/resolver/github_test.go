// File: internal/resolver/github_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/resolver/github.go (ResolveGitHubRelease, matchAssetName, httptest).

package resolver

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// --- from resolver_test.go (matchAssetName) ---
var _ = Describe("matchAssetName", func() {
	It("returns the first matching asset", func() {
		got, err := matchAssetName([]string{"a-1.0.0.jar", "a-1.0.1.jar"}, "a-*.jar")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("a-1.0.0.jar"))
	})
	It("errors when nothing matches", func() {
		_, err := matchAssetName([]string{"other-1.0.jar"}, "a-*.jar")
		Expect(err).To(HaveOccurred())
	})
	It("errors on empty list", func() {
		_, err := matchAssetName(nil, "*.jar")
		Expect(err).To(HaveOccurred())
	})
	It("handles literal-only pattern", func() {
		got, err := matchAssetName([]string{"exact.jar", "other.jar"}, "exact.jar")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("exact.jar"))
	})
})

// --- from resolver_test.go (Resolver GitHub (httptest)) ---
var _ = Describe("Resolver GitHub (httptest)", func() {
	var srv *httptest.Server
	var prevTransport http.RoundTripper

	BeforeEach(func() {
		// Stand up a tiny GitHub-like server. Handlers return canned data so
		// we never touch the public GitHub API from tests. The trick is that
		// we install a custom DefaultTransport whose RoundTrip redirects
		// every request to the test server, regardless of the host header.
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/releases", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[{"tag_name":"v1.0.0"},{"tag_name":"v1.0.1"}]`))
		})
		mux.HandleFunc("/repos/owner/repo/releases/tags/v1.0.0", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"assets":[{"name":"mod-1.0.0.jar"},{"name":"other.txt"}]}`))
		})
		mux.HandleFunc("/repos/owner/repo/releases/tags/missing", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		srv = httptest.NewServer(mux)
		prevTransport = http.DefaultTransport
		http.DefaultTransport = redirectTransport{target: srv.URL, base: http.DefaultTransport}
	})

	AfterEach(func() {
		http.DefaultTransport = prevTransport
		if srv != nil {
			srv.Close()
		}
	})

	It("listReleaseAssets returns asset names", func() {
		names, err := listReleaseAssets("owner/repo", "v1.0.0")
		Expect(err).NotTo(HaveOccurred())
		Expect(names).To(ConsistOf("mod-1.0.0.jar", "other.txt"))
	})

	It("listReleaseAssets returns error on 404", func() {
		_, err := listReleaseAssets("owner/repo", "missing")
		Expect(err).To(HaveOccurred())
	})

	It("ResolveGitHubRelease with wildcard tag picks first matching release", func() {
		src, err := ResolveGitHubRelease("owner/repo", "v*", "mod-{tag}.jar", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(src.Tag).To(Equal("v1.0.0"))
		Expect(src.AssetName).To(Equal("mod-v1.0.0.jar"))
	})

	It("ResolveGitHubRelease with wildcard asset picks first matching asset", func() {
		src, err := ResolveGitHubRelease("owner/repo", "v1.0.0", "mod-*.jar", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(src.AssetName).To(Equal("mod-1.0.0.jar"))
	})

	It("ResolveGitHubRelease with wildcard asset but no matches errors", func() {
		_, err := ResolveGitHubRelease("owner/repo", "v1.0.0", "nope-*.jar", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("Resolver GitHub (httptest)", func() {
	var srv *httptest.Server
	var prevTransport http.RoundTripper

	BeforeEach(func() {
		// Stand up a tiny GitHub-like server. Handlers return canned data so
		// we never touch the public GitHub API from tests. The trick is that
		// we install a custom DefaultTransport whose RoundTrip redirects
		// every request to the test server, regardless of the host header.
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/releases", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[{"tag_name":"v1.0.0"},{"tag_name":"v1.0.1"}]`))
		})
		mux.HandleFunc("/repos/owner/repo/releases/tags/v1.0.0", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"assets":[{"name":"mod-1.0.0.jar"},{"name":"other.txt"}]}`))
		})
		mux.HandleFunc("/repos/owner/repo/releases/tags/missing", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		srv = httptest.NewServer(mux)
		prevTransport = http.DefaultTransport
		http.DefaultTransport = redirectTransport{target: srv.URL, base: http.DefaultTransport}
	})

	AfterEach(func() {
		http.DefaultTransport = prevTransport
		if srv != nil {
			srv.Close()
		}
	})

	It("listReleaseAssets returns asset names", func() {
		names, err := listReleaseAssets("owner/repo", "v1.0.0")
		Expect(err).NotTo(HaveOccurred())
		Expect(names).To(ConsistOf("mod-1.0.0.jar", "other.txt"))
	})

	It("listReleaseAssets returns error on 404", func() {
		_, err := listReleaseAssets("owner/repo", "missing")
		Expect(err).To(HaveOccurred())
	})

	It("ResolveGitHubRelease with wildcard tag picks first matching release", func() {
		src, err := ResolveGitHubRelease("owner/repo", "v*", "mod-{tag}.jar", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(src.Tag).To(Equal("v1.0.0"))
		Expect(src.AssetName).To(Equal("mod-v1.0.0.jar"))
	})

	It("ResolveGitHubRelease with wildcard asset picks first matching asset", func() {
		src, err := ResolveGitHubRelease("owner/repo", "v1.0.0", "mod-*.jar", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(src.AssetName).To(Equal("mod-1.0.0.jar"))
	})

	It("ResolveGitHubRelease with wildcard asset but no matches errors", func() {
		_, err := ResolveGitHubRelease("owner/repo", "v1.0.0", "nope-*.jar", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
})

// redirectTransport rewrites the request URL to the test server, preserving
// the path and query. This is the only way to redirect http.DefaultClient
// (used by the resolver) without refactoring every call site to accept a
// custom client.
type redirectTransport struct {
	target string
	base   http.RoundTripper
}

func (t redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Strip scheme+host and prepend the test server URL.
	rewritten := *req
	rewritten.URL.Scheme = "http"
	rewritten.URL.Host = t.target[len("http://"):]
	return t.base.RoundTrip(&rewritten)
}
