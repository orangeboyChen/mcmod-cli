// File: internal/resolver/curseforge_search.go
// Created: 2026-06-20
// Description: CurseForge internal search and file-lookup helpers.

package resolver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/orangeboyChen/mcmod-cli/internal/config"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
	"github.com/orangeboyChen/mcmod-cli/internal/netutil"
)

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
