// File: internal/downloader/downloader_suite_test.go
// Created: 2026-06-20
// Description: Ginkgo test suite for downloader package.
package downloader

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
	"github.com/orangeboyChen/mcmod-cli/internal/netutil"
)

func TestDownloader(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Downloader Suite")
}

// Tighten the global retry budget so the "tries the real API with a bad key"
// specs finish promptly when the remote returns 403/429. Default retry would
// back off for many seconds per call which is unacceptable in tests.
var _ = BeforeSuite(func() {
	netutil.SetDefaultRetry(netutil.RetryConfig{
		MaxAttempts: 1,
		BaseDelay:   time.Millisecond,
	})
})

var _ = Describe("Downloader", func() {
	It("Download local source is a no-op", func() {
		err := Download(&domain.LockedSource{Type: "local", Path: "./x.jar", FileName: "x.jar"}, "")
		Expect(err).NotTo(HaveOccurred())
	})

	It("Download unknown type errors", func() {
		err := Download(&domain.LockedSource{Type: "wat"}, "")
		Expect(err).To(HaveOccurred())
	})

	It("extractDownloadURL accepts bare string form", func() {
		raw := json.RawMessage(`"https://example.com/a.jar"`)
		u, err := extractDownloadURL(raw)
		Expect(err).NotTo(HaveOccurred())
		Expect(u).To(Equal("https://example.com/a.jar"))
	})

	It("extractDownloadURL accepts object form", func() {
		raw := json.RawMessage(`{"url":"https://example.com/b.jar"}`)
		u, err := extractDownloadURL(raw)
		Expect(err).NotTo(HaveOccurred())
		Expect(u).To(Equal("https://example.com/b.jar"))
	})

	It("extractDownloadURL rejects empty payload", func() {
		_, err := extractDownloadURL(json.RawMessage(``))
		Expect(err).To(HaveOccurred())
	})

	It("extractDownloadURL rejects malformed JSON", func() {
		_, err := extractDownloadURL(json.RawMessage(`not-json`))
		Expect(err).To(HaveOccurred())
	})

	It("extractDownloadURL rejects object with no url", func() {
		raw := json.RawMessage(`{"other":"x"}`)
		_, err := extractDownloadURL(raw)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("parseRepo", func() {
	It("splits owner/name on slash", func() {
		owner, name := parseRepo("o/r")
		Expect(owner).To(Equal("o"))
		Expect(name).To(Equal("r"))
	})
	It("returns whole string when no slash", func() {
		owner, name := parseRepo("onlyname")
		Expect(owner).To(Equal("onlyname"))
		Expect(name).To(Equal(""))
	})
	It("handles empty input", func() {
		owner, name := parseRepo("")
		Expect(owner).To(Equal(""))
		Expect(name).To(Equal(""))
	})
})

var _ = Describe("dlCurseForge error paths", func() {
	BeforeEach(func() {
		os.Setenv("CURSEFORGE_API_KEY", "")
	})
	AfterEach(func() {
		os.Unsetenv("CURSEFORGE_API_KEY")
	})
	It("dlCurseForge without key still attempts API (gets 401/403)", func() {
		// We use a known-nonexistent mod+file id to provoke a CF-side error.
		err := dlCurseForge(&domain.LockedSource{Type: "curseforge", ModID: 1, FileID: 1, FileName: "x.jar"}, "")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("dlGitHub error paths", func() {
	It("dlGitHub with invalid repo errors", func() {
		err := dlGitHub(&domain.LockedSource{
			Type:      "github-release",
			Repo:      "this-org-does-not-exist-1234567890/this-repo-too-0987654321",
			Tag:       "v0.0.1",
			AssetName: "asset.jar",
		}, "")
		Expect(err).To(HaveOccurred())
	})

	It("dlGitHub with empty repo errors", func() {
		err := dlGitHub(&domain.LockedSource{
			Type:      "github-release",
			Repo:      "",
			Tag:       "v0.0.1",
			AssetName: "asset.jar",
		}, "")
		Expect(err).To(HaveOccurred())
	})
})
