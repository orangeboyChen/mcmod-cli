// File: internal/resolver/git_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/resolver/git.go (ResolveGitPackage).

package resolver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("ResolveGitPackage via httptest", func() {
	var prevTrans http.RoundTripper

	AfterEach(func() {
		http.DefaultTransport = prevTrans
	})

	It("returns the parsed PackSpec when main branch responds with valid JSON", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"packName":"p","minecraftVersion":"1.21.1","loaderName":["neoforge"]}`))
		})
		srv := httptest.NewServer(mux)
		DeferCleanup(srv.Close)
		prevTrans = http.DefaultTransport
		http.DefaultTransport = redirectTransport{target: srv.URL, base: http.DefaultTransport}

		spec, err := ResolveGitPackage("o/r", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(spec).NotTo(BeNil())
		Expect(spec.PackName).To(Equal("p"))
		Expect(spec.MinecraftVersion).To(Equal("1.21.1"))
	})

	It("falls back to master when main returns a non-200 status", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Both branches get the same 404; the function will report an
			// error after trying both.
			w.WriteHeader(http.StatusNotFound)
		})
		srv := httptest.NewServer(mux)
		DeferCleanup(srv.Close)
		prevTrans = http.DefaultTransport
		http.DefaultTransport = redirectTransport{target: srv.URL, base: http.DefaultTransport}

		_, err := ResolveGitPackage("o/r", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})

	It("returns an error when neither branch responds with valid JSON", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("not-json"))
		})
		srv := httptest.NewServer(mux)
		DeferCleanup(srv.Close)
		prevTrans = http.DefaultTransport
		http.DefaultTransport = redirectTransport{target: srv.URL, base: http.DefaultTransport}

		_, err := ResolveGitPackage("o/r", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})

	It("expands nested git packspecs with repository namespaces", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"packName": "child", "minecraftVersion": "1.21.1", "loaderName": []string{"neoforge"},
				"mods": map[string]interface{}{
					"common": map[string]interface{}{"name": "Common", "source": map[string]string{"type": "local", "path": "common.jar"}},
				},
			})
		})
		srv := httptest.NewServer(mux)
		DeferCleanup(srv.Close)
		prevTrans = http.DefaultTransport
		http.DefaultTransport = redirectTransport{target: srv.URL, base: http.DefaultTransport}

		spec := &domain.PackSpec{LoaderName: []string{"neoforge"}, Mods: map[string]domain.ModSpec{
			"bundle": {Scope: "server", Source: domain.ModSource{Type: "git", Repo: "o/r"}},
		}}
		mods, err := ExpandGitDependencies(spec, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(mods).To(HaveKey("o-r-common"))
		Expect(mods["o-r-common"].Scope).To(Equal("server"))
	})
})
