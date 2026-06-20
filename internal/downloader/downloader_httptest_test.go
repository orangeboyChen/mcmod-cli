// File: internal/downloader/downloader_httptest_test.go
// Created: 2026-06-20
// Description: Downloader tests using httptest to exercise the happy paths
// of dlCurseForge and dlGitHub without touching the real CurseForge / GitHub
// APIs.

package downloader

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("Downloader httptest", func() {
	var (
		srv        *httptest.Server
		prevTrans  http.RoundTripper
		restoreEnv string
		oldWd      string
		tmpDir     string
	)

	handle := func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/download-url"):
			// The CurseForge /download-url endpoint returns a JSON envelope.
			w.Header().Set("Content-Type", "application/json")
			payload, _ := json.Marshal(map[string]any{
				"data": "http://placeholder/asset.jar",
			})
			w.Write(payload)
		case strings.HasSuffix(r.URL.Path, "/asset.jar") || strings.Contains(r.URL.Path, "/releases/download/"):
			// Both the redirected download URL and GitHub release asset
			// download paths land here.
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("fake jar payload"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}

	BeforeEach(func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", handle)
		srv = httptest.NewServer(mux)

		prevTrans = http.DefaultTransport
		http.DefaultTransport = redirectTransport{target: srv.URL, base: http.DefaultTransport}

		var err error
		tmpDir, err = os.MkdirTemp("", "dl-test-*")
		Expect(err).NotTo(HaveOccurred())
		oldWd, _ = os.Getwd()
		Expect(os.Chdir(tmpDir)).To(Succeed())

		restoreEnv = os.Getenv("CURSEFORGE_API_KEY")
		os.Setenv("CURSEFORGE_API_KEY", "fake-key")
	})

	AfterEach(func() {
		http.DefaultTransport = prevTrans
		if srv != nil {
			srv.Close()
		}
		os.Chdir(oldWd)
		if tmpDir != "" {
			os.RemoveAll(tmpDir)
		}
		os.Setenv("CURSEFORGE_API_KEY", restoreEnv)
	})

	It("dlCurseForge downloads a file to cache", func() {
		err := dlCurseForge(&domain.LockedSource{
			Type: "curseforge", ModID: 1, FileID: 2, FileName: "x.jar",
		}, "label")
		Expect(err).NotTo(HaveOccurred())
		Expect(filepath.Join(tmpDir, ".cache", "curseforge", "1", "2", "x.jar")).To(BeAnExistingFile())
	})

	It("dlGitHub downloads a file to cache", func() {
		err := dlGitHub(&domain.LockedSource{
			Type: "github-release", Repo: "owner/repo", Tag: "v1.0.0", AssetName: "asset.jar",
		}, "label")
		Expect(err).NotTo(HaveOccurred())
		Expect(filepath.Join(tmpDir, ".cache", "github-release", "owner", "repo", "v1.0.0", "asset.jar")).To(BeAnExistingFile())
	})
})

// redirectTransport rewrites the request URL to the test server, preserving
// the path and query. Used to redirect http.DefaultClient to a local httptest
// server for hermetic tests.
type redirectTransport struct {
	target string
	base   http.RoundTripper
}

func (t redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rewritten := *req
	rewritten.URL.Scheme = "http"
	rewritten.URL.Host = t.target[len("http://"):]
	return t.base.RoundTrip(&rewritten)
}
