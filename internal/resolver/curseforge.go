// File: internal/resolver/curseforge.go
// Created: 2026-06-20
// Description: CurseForge resolver per spec section 4.1.

package resolver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/orangeboyChen/mcmod-cli/internal/config"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
	"github.com/orangeboyChen/mcmod-cli/internal/netutil"
)

// CFBaseURL is the base URL for the CurseForge API.
const CFBaseURL = "https://api.curseforge.com/v1"

// cfCandidate is an internal scoring wrapper for CurseForge search results.
type cfCandidate struct {
	ID   int
	Name string
	Slug string
	// 0 = exact match on slug or name, 1 = slug starts with query, 2 = contains
	Score int
}

// LoaderToCF converts a Minecraft loader name to a CurseForge game version string.
func LoaderToCF(loader string) int {
	switch loader {
	case "fabric":
		return 4
	case "neoforge":
		return 6
	}
	return 0
}

// ResolveCurseForgeByQuery looks up a mod by name on CurseForge.
func ResolveCurseForgeByQuery(query, mcVersion, loader string, modKey ...string) (*domain.LockedSource, error) {
	modLabel := strings.Join(modKey, " ")
	labelHTTP := func(req *http.Request) {
		if modLabel != "" {
			req.Header.Set("X-Netutil-Label", modLabel)
		}
	}
	cfKey := config.GetCFKey()
	if cfKey == "" {
		return nil, fmt.Errorf("CurseForge API key not set")
	}

	normQ := domain.NormalizeKey(query)

	req, _ := http.NewRequest("GET", CFBaseURL+"/mods/search", nil)
	labelHTTP(req)
	q := req.URL.Query()
	q.Set("gameId", "432")
	q.Set("classId", "6")
	q.Set("searchFilter", query)
	q.Set("pageSize", "50")
	if loaderType := LoaderToCF(loader); loaderType != 0 {
		q.Set("modLoaderType", fmt.Sprintf("%d", loaderType))
	}
	q.Set("sortField", "6") // 6 = TotalDownloads, descending
	req.URL.RawQuery = q.Encode()
	req.Header.Set("x-api-key", cfKey)

	resp, _, err := netutil.DoWithRetry(http.DefaultClient, req, netutil.DefaultRetry)
	if err != nil {
		return nil, fmt.Errorf("curseforge search failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("curseforge search returned %d", resp.StatusCode)
	}

	type Resp struct {
		Data []struct {
			ID   int    `json:"id"`
			Slug string `json:"slug"`
			Name string `json:"name"`
		} `json:"data"`
	}
	var r Resp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	var cands []cfCandidate
	for _, m := range r.Data {
		ns := domain.NormalizeKey(m.Slug)
		nn := domain.NormalizeKey(m.Name)
		switch {
		case ns == normQ || nn == normQ:
			cands = append(cands, cfCandidate{ID: m.ID, Name: m.Name, Slug: m.Slug, Score: 0})
		case strings.HasPrefix(ns, normQ+"-"):
			cands = append(cands, cfCandidate{ID: m.ID, Name: m.Name, Slug: m.Slug, Score: 1})
		case strings.Contains(ns, normQ) || strings.Contains(nn, normQ):
			cands = append(cands, cfCandidate{ID: m.ID, Name: m.Name, Slug: m.Slug, Score: 2})
		}
	}
	hasExact := false
	for _, c := range cands {
		if c.Score == 0 {
			hasExact = true
			break
		}
	}
	if !hasExact {
		// Fallback: search by exact slug, which CF indexes reliably even when
		// searchFilter cannot match queries that contain decorative characters
		// or returns the add-on family first.
		slugHits, slugErr := curseForgeSearchBySlug(cfKey, normQ, loader)
		if slugErr == nil && len(slugHits) > 0 {
			cands = slugHits
		} else if len(cands) == 0 {
			return nil, fmt.Errorf("curseforge: no high-confidence match for query %q", query)
		}
	}
	// Pick best score. If multiple candidates share the best score the API has
	// already sorted them by TotalDownloads desc, so the first one wins.
	best := cands[0].Score
	bestCands := []cfCandidate{cands[0]}
	for _, c := range cands[1:] {
		if c.Score < best {
			best = c.Score
			bestCands = []cfCandidate{c}
		} else if c.Score == best {
			bestCands = append(bestCands, c)
		}
	}
	if len(bestCands) > 1 {
		names := make([]string, len(bestCands))
		for i, c := range bestCands {
			names[i] = fmt.Sprintf("modId=%d name=%s slug=%s", c.ID, c.Name, c.Slug)
		}
		return nil, fmt.Errorf("curseforge: multiple matches for %q: %s", query, strings.Join(names, ", "))
	}

	modID := cands[0].ID
	// Resolve the matching file for (modID, mcVersion, loader).
	fileID, fileName, err := findCurseForgeFile(modID, mcVersion, loader)
	if err != nil {
		return nil, err
	}
	return &domain.LockedSource{Type: "curseforge", ModID: modID, FileID: fileID, FileName: fileName}, nil
}

// curseForgeSearchBySlug looks up a mod by its exact slug, which is the
// normalised display name. This is more reliable than the searchFilter
// endpoint for queries containing decorative characters like apostrophes.
func curseForgeSearchBySlug(cfKey, normQ, loader string) ([]cfCandidate, error) {
	req, _ := http.NewRequest("GET", CFBaseURL+"/mods/search", nil)
	q := req.URL.Query()
	q.Set("gameId", "432")
	q.Set("classId", "6")
	q.Set("slug", normQ)
	q.Set("pageSize", "5")
	if loaderType := LoaderToCF(loader); loaderType != 0 {
		q.Set("modLoaderType", fmt.Sprintf("%d", loaderType))
	}
	req.URL.RawQuery = q.Encode()
	req.Header.Set("x-api-key", cfKey)

	resp, _, err := netutil.DoWithRetry(http.DefaultClient, req, netutil.DefaultRetry)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("curseforge search returned %d", resp.StatusCode)
	}
	type Resp struct {
		Data []struct {
			ID   int    `json:"id"`
			Slug string `json:"slug"`
			Name string `json:"name"`
		} `json:"data"`
	}
	var r Resp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	var cands []cfCandidate
	for _, m := range r.Data {
		if domain.NormalizeKey(m.Slug) == normQ {
			cands = append(cands, cfCandidate{ID: m.ID, Name: m.Name, Slug: m.Slug, Score: 0})
		}
	}
	return cands, nil
}

// findCurseForgeFile lists files for a mod and picks the most recent release
// matching mcVersion and loader. The candidate set is sorted by fileDate
// descending and the first non-prerelease hit is returned. When every match
// is a prerelease, the newest one is returned so the caller can decide.
func findCurseForgeFile(modID int, mcVersion, loader string) (int, string, error) {
	cfKey := config.GetCFKey()
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/mods/%d/files", CFBaseURL, modID), nil)
	q := req.URL.Query()
	q.Set("pageSize", "50")
	if loaderType := LoaderToCF(loader); loaderType != 0 {
		q.Set("modLoaderType", fmt.Sprintf("%d", loaderType))
	}
	req.URL.RawQuery = q.Encode()
	req.Header.Set("x-api-key", cfKey)

	resp, _, err := netutil.DoWithRetry(http.DefaultClient, req, netutil.DefaultRetry)
	if err != nil {
		return 0, "", fmt.Errorf("curseforge files list failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return 0, "", fmt.Errorf("curseforge files list returned %d", resp.StatusCode)
	}

	type File struct {
		ID           int      `json:"id"`
		FileName     string   `json:"fileName"`
		FileDate     string   `json:"fileDate"`
		ReleaseType  int      `json:"releaseType"`
		GameVersions []string `json:"gameVersions"`
	}
	type Resp struct {
		Data []File `json:"data"`
	}
	var r Resp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return 0, "", fmt.Errorf("decode failed: %w", err)
	}

	// Pick newest release matching mcVersion + loader (loader is enforced by
	// the API filter; we still double-check gameVersions contains mcVersion).
	var bestFile *File
	for i := range r.Data {
		f := &r.Data[i]
		matched := false
		for _, gv := range f.GameVersions {
			if gv == mcVersion {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		// Prefer non-prerelease (releaseType==1). Otherwise take first (newest
		// by API default ordering = fileDate desc).
		if bestFile == nil {
			bestFile = f
			continue
		}
		if bestFile.ReleaseType != 1 && f.ReleaseType == 1 {
			bestFile = f
		}
	}
	if bestFile == nil {
		return 0, "", fmt.Errorf("curseforge: no file for mod %d matching %s/%s", modID, mcVersion, loader)
	}
	return bestFile.ID, bestFile.FileName, nil
}

// ResolveCurseForgeByID looks up a specific CurseForge file by mod and file ID.
func ResolveCurseForgeByID(modID, fileID int, modKey ...string) (*domain.LockedSource, error) {
	modLabel := strings.Join(modKey, " ")
	labelHTTP := func(req *http.Request) {
		if modLabel != "" {
			req.Header.Set("X-Netutil-Label", modLabel)
		}
	}
	cfKey := config.GetCFKey()
	url := fmt.Sprintf("%s/mods/%d/files/%d", CFBaseURL, modID, fileID)
	req, _ := http.NewRequest("GET", url, nil)
	labelHTTP(req)
	req.Header.Set("x-api-key", cfKey)
	resp, _, err := netutil.DoWithRetry(http.DefaultClient, req, netutil.DefaultRetry)
	if err != nil {
		return nil, fmt.Errorf("curseforge file detail failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("curseforge file detail returned %d", resp.StatusCode)
	}

	type Resp struct {
		Data struct {
			FileName    string `json:"fileName"`
			DownloadURL string `json:"downloadUrl"`
		} `json:"data"`
	}
	var r Resp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	// Prefer the CDN downloadUrl returned in the same response: it skips the
	// heavily rate-limited /download-url endpoint and goes straight to
	// edge.forgecdn.net which has no API key requirement and no hourly cap.
	ls := &domain.LockedSource{Type: "curseforge", ModID: modID, FileID: fileID, FileName: r.Data.FileName}
	if r.Data.DownloadURL != "" {
		ls.URL = r.Data.DownloadURL
	}
	return ls, nil
}

// ResolveCurseForgeBySlug looks up a mod by its exact CF slug, then picks the
// matching file. This is the most reliable path when the human-readable query
// in packspec.json does not match the actual CF slug.
func ResolveCurseForgeBySlug(slug, mcVersion, loader string, modKey ...string) (*domain.LockedSource, error) {
	modLabel := strings.Join(modKey, " ")
	labelHTTP := func(req *http.Request) {
		if modLabel != "" {
			req.Header.Set("X-Netutil-Label", modLabel)
		}
	}
	cfKey := config.GetCFKey()
	if cfKey == "" {
		return nil, fmt.Errorf("CurseForge API key not set")
	}
	req, _ := http.NewRequest("GET", CFBaseURL+"/mods/search", nil)
	labelHTTP(req)
	q := req.URL.Query()
	q.Set("gameId", "432")
	q.Set("classId", "6")
	q.Set("slug", slug)
	q.Set("pageSize", "5")
	if loaderType := LoaderToCF(loader); loaderType != 0 {
		q.Set("modLoaderType", fmt.Sprintf("%d", loaderType))
	}
	req.URL.RawQuery = q.Encode()
	req.Header.Set("x-api-key", cfKey)

	resp, _, err := netutil.DoWithRetry(http.DefaultClient, req, netutil.DefaultRetry)
	if err != nil {
		return nil, fmt.Errorf("curseforge slug search failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("curseforge slug search returned %d", resp.StatusCode)
	}
	type Resp struct {
		Data []struct {
			ID   int    `json:"id"`
			Slug string `json:"slug"`
			Name string `json:"name"`
		} `json:"data"`
	}
	var r Resp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}
	for _, m := range r.Data {
		if m.Slug == slug {
			fileID, fileName, err := findCurseForgeFile(m.ID, mcVersion, loader)
			if err != nil {
				return nil, err
			}
			return &domain.LockedSource{Type: "curseforge", ModID: m.ID, FileID: fileID, FileName: fileName}, nil
		}
	}
	return nil, fmt.Errorf("curseforge: no mod for slug %q", slug)
}
