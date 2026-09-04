// File: internal/downloader/downloader.go
// Created: 2026-06-20
// Description: Unified downloader entry point.

package downloader

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/orangeboyChen/mcmod-cli/internal/cache"
	"github.com/orangeboyChen/mcmod-cli/internal/config"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
	"github.com/orangeboyChen/mcmod-cli/internal/netutil"
)

// Download fetches a file from a URL into the cache directory.
func Download(src *domain.LockedSource, label string) error {
	switch src.Type {
	case "curseforge":
		return dlCurseForge(src, label)
	case "github-release":
		return dlGitHub(src, label)
	case "local":
		return nil
	case "url":
		// Operator-supplied CDN URL. Just stream from src.URL into the cache.
		return downloadCFToCache(src, src.URL, label)
	default:
		return fmt.Errorf("unsupported download type: %s", src.Type)
	}
}

func dlCurseForge(src *domain.LockedSource, label string) error {
	// The standard ForgeCDN URL avoids the heavily rate-limited API endpoint.
	// Set MCMOD_CURSEFORGE_USE_DOWNLOAD_URL to opt into the API fallback.
	if !useCurseForgeDownloadURL() {
		downloadURL := src.URL
		if downloadURL == "" && src.ModID != 0 && src.FileID != 0 && src.FileName != "" {
			downloadURL = domain.DefaultCurseForgeURL(src.FileID, src.FileName)
		}
		if downloadURL != "" {
			return downloadCFToCache(src, downloadURL, label)
		}
	}
	// Explicitly enabled fallback: obtain the URL from CurseForge.
	if src.ModID == 0 || src.FileID == 0 {
		if src.URL != "" {
			return downloadCFToCache(src, src.URL, label)
		}
		return fmt.Errorf("curseforge source missing modId/fileId")
	}
	key := config.GetCFKey()
	url := fmt.Sprintf("https://api.curseforge.com/v1/mods/%d/files/%d/download-url", src.ModID, src.FileID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Api-Key", key)
	if label != "" {
		req.Header.Set("X-Netutil-Label", label)
	}
	resp, _, err := netutil.DoWithRetry(http.DefaultClient, req, netutil.DefaultRetry)
	if err != nil {
		return fmt.Errorf("curseforge download-url failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("curseforge download-url returned %d body=%q", resp.StatusCode, readBodyPreview(resp.Body, 512))
	}
	type DLResp struct {
		Data json.RawMessage `json:"data"`
	}
	var dl DLResp
	if err := json.NewDecoder(resp.Body).Decode(&dl); err != nil {
		return fmt.Errorf("decode failed: %w", err)
	}
	downloadURL, err := extractDownloadURL(dl.Data)
	if err != nil {
		return err
	}
	return downloadCFToCache(src, downloadURL, label)
}

func useCurseForgeDownloadURL() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MCMOD_CURSEFORGE_USE_DOWNLOAD_URL"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// downloadCFToCache streams a file from the given URL into the .mcmod/cache
// directory for the given CF modId/fileId/fileName. It honours the label
// for netutil retry logging and surfaces server error bodies in the failure
// message.
func downloadCFToCache(src *domain.LockedSource, downloadURL, label string) error {
	req, _ := http.NewRequest("GET", downloadURL, nil)
	if label != "" {
		req.Header.Set("X-Netutil-Label", label)
	}
	resp, _, err := netutil.DoWithRetry(http.DefaultClient, req, netutil.DefaultRetry)
	if err != nil {
		return fmt.Errorf("curseforge download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("curseforge download returned %d body=%q", resp.StatusCode, readBodyPreview(resp.Body, 512))
	}
	dlDir := filepath.Dir(cache.CurseForgePath(fmt.Sprint(src.ModID), fmt.Sprint(src.FileID), src.FileName))
	if err := os.MkdirAll(dlDir, 0755); err != nil {
		return fmt.Errorf("curseforge cache dir: %w", err)
	}
	path := cache.CurseForgePath(fmt.Sprint(src.ModID), fmt.Sprint(src.FileID), src.FileName)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("curseforge cache file: %w", err)
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func dlGitHub(src *domain.LockedSource, label string) error {
	owner, name := parseRepo(src.Repo)
	assetURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s", owner, name, src.Tag, src.AssetName)
	resp, _, err := netutil.GetWithRetry(http.DefaultClient, assetURL, netutil.DefaultRetry)
	if err != nil {
		return fmt.Errorf("github download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("github download returned %d for %s body=%q", resp.StatusCode, assetURL, readBodyPreview(resp.Body, 512))
	}

	cacheDir := filepath.Dir(cache.GitHubReleasePath(owner, name, src.Tag, src.AssetName))
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("github cache dir: %w", err)
	}
	path := cache.GitHubReleasePath(owner, name, src.Tag, src.AssetName)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("github cache file: %w", err)
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// readBodyPreview reads up to n bytes of body and returns it trimmed. The body
// is then drained/closed by the caller. Used to surface the server-side
// reason in error messages.
func readBodyPreview(body io.Reader, n int) string {
	if body == nil {
		return ""
	}
	limited := io.LimitReader(body, int64(n))
	buf, err := io.ReadAll(limited)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(buf))
}

func parseRepo(repo string) (owner, name string) {
	for i := 0; i < len(repo); i++ {
		if repo[i] == '/' {
			return repo[:i], repo[i+1:]
		}
	}
	return repo, ""
}

// extractDownloadURL pulls the first non-empty "url" string out of the
// raw data payload returned by the CurseForge /download-url endpoint. The
// endpoint has shipped in two shapes: a bare string and a {url: "..."}
// object. Both are accepted.
func extractDownloadURL(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", fmt.Errorf("curseforge download-url: empty data")
	}
	trimmed := string(raw)
	if len(trimmed) >= 2 && trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return "", fmt.Errorf("curseforge download-url: %w", err)
		}
		return s, nil
	}
	var obj struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", fmt.Errorf("curseforge download-url: %w", err)
	}
	if obj.URL == "" {
		return "", fmt.Errorf("curseforge download-url: url field is empty")
	}
	return obj.URL, nil
}
