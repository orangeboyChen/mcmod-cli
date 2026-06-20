// File: internal/resolver/curseforge_id_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/resolver/curseforge_id.go (ResolveCurseForgeByQuery, ResolveCurseForgeByID, ResolveCurseForgeBySlug).

package resolver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// withRedirectedHTTP replaces http.DefaultTransport with a redirect to the
// given test server and registers a DeferCleanup to restore it. It also
// sets CURSEFORGE_API_KEY so resolver functions do not bail out early.
func withRedirectedHTTP(srvURL string) {
	prev := http.DefaultTransport
	http.DefaultTransport = redirectTransport{target: srvURL, base: prev}
	DeferCleanup(func() { http.DefaultTransport = prev })

	prevKey, hadKey := os.LookupEnv("CURSEFORGE_API_KEY")
	os.Setenv("CURSEFORGE_API_KEY", "fake-key")
	DeferCleanup(func() {
		if hadKey {
			os.Setenv("CURSEFORGE_API_KEY", prevKey)
		} else {
			os.Unsetenv("CURSEFORGE_API_KEY")
		}
	})
}

var _ = Describe("ResolveCurseForgeByQuery via httptest", func() {
	It("returns an error when no API key is set", func() {
		prevKey, hadKey := os.LookupEnv("CURSEFORGE_API_KEY")
		os.Unsetenv("CURSEFORGE_API_KEY")
		DeferCleanup(func() {
			if hadKey {
				os.Setenv("CURSEFORGE_API_KEY", prevKey)
			}
		})

		_, err := ResolveCurseForgeByQuery("jei", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})

	It("returns a LockedSource when the search hits a single match", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.Contains(r.URL.Path, "/search"):
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": []map[string]any{
						{"id": 42, "name": "JEI", "slug": "jei", "summary": ""},
					},
				})
			case strings.Contains(r.URL.Path, "/files"):
				// File lookup needs a 1.21.1/neoforge-matching entry. The
				// production code will accept the first match with the
				// right game version + loader type.
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": []map[string]any{
						{
							"id":           100,
							"fileName":     "jei-1.21.1.jar",
							"downloadUrl":  "https://example/jei.jar",
							"gameVersions": []string{"1.21.1"},
						},
					},
				})
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		})
		srv := httptest.NewServer(mux)
		DeferCleanup(srv.Close)
		withRedirectedHTTP(srv.URL)

		src, err := ResolveCurseForgeByQuery("jei", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(src).NotTo(BeNil())
		Expect(src.ModID).To(Equal(42))
		Expect(src.Type).To(Equal("curseforge"))
	})

	It("returns an error when the search response is empty", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		})
		srv := httptest.NewServer(mux)
		DeferCleanup(srv.Close)
		withRedirectedHTTP(srv.URL)

		_, err := ResolveCurseForgeByQuery("missing", "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("ResolveCurseForgeByID error path", func() {
	It("returns an error when no API key is set", func() {
		prevKey, hadKey := os.LookupEnv("CURSEFORGE_API_KEY")
		os.Unsetenv("CURSEFORGE_API_KEY")
		DeferCleanup(func() {
			if hadKey {
				os.Setenv("CURSEFORGE_API_KEY", prevKey)
			}
		})

		_, err := ResolveCurseForgeByID(42, 123, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("ResolveCurseForgeByID via httptest", func() {
	It("returns a LockedSource when the file endpoint has a matching entry", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// /mods/{id}/files/{fileId} returns a single-object envelope.
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id":           100,
					"fileName":     "mod-1.21.1.jar",
					"downloadUrl":  "https://example/mod.jar",
					"gameVersions": []string{"1.21.1"},
				},
			})
		})
		srv := httptest.NewServer(mux)
		DeferCleanup(srv.Close)
		withRedirectedHTTP(srv.URL)

		src, err := ResolveCurseForgeByID(42, 100, "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(src).NotTo(BeNil())
		Expect(src.ModID).To(Equal(42))
		Expect(src.FileID).To(Equal(100))
		Expect(src.FileName).To(Equal("mod-1.21.1.jar"))
	})

	It("returns an error when the file endpoint has no matching version", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		})
		srv := httptest.NewServer(mux)
		DeferCleanup(srv.Close)
		withRedirectedHTTP(srv.URL)

		_, err := ResolveCurseForgeByID(42, 100, "1.21.1", "neoforge")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("ResolveCurseForgeBySlug via httptest", func() {
	It("returns a LockedSource when the slug lookup succeeds", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "/search") {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": []map[string]any{
						{"id": 42, "name": "JEI", "slug": "jei"},
					},
				})
				return
			}
			if strings.Contains(r.URL.Path, "/files") {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": []map[string]any{
						{
							"id":           100,
							"fileName":     "jei.jar",
							"downloadUrl":  "https://example/jei.jar",
							"gameVersions": []string{"1.21.1"},
						},
					},
				})
				return
			}
			w.WriteHeader(http.StatusNotFound)
		})
		srv := httptest.NewServer(mux)
		DeferCleanup(srv.Close)
		withRedirectedHTTP(srv.URL)

		src, err := ResolveCurseForgeBySlug("jei", "1.21.1", "neoforge")
		Expect(err).NotTo(HaveOccurred())
		Expect(src).NotTo(BeNil())
		Expect(src.ModID).To(Equal(42))
		Expect(src.FileID).To(Equal(100))
	})
})
