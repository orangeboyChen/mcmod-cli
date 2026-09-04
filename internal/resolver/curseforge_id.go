// File: internal/resolver/curseforge_id.go
// Created: 2026-06-20
// Description: CurseForge public resolver entry points (query, id, slug).

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

// ResolveCurseForgeByQuery searches CurseForge for mods whose name matches
// the given query and returns the best match as a LockedSource. The
// resolver scores candidates by exact-slug, then starts-with, then
// substring, and errors when more than one top-scoring candidate
// exists.
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

// ResolveCurseForgeByID locks an already-known (modID, fileID) pair
// without further searching. It validates the file metadata via the
// CurseForge file-detail endpoint and returns the resolved LockedSource.
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

	return &domain.LockedSource{
		Type:     "curseforge",
		ModID:    modID,
		FileID:   fileID,
		FileName: r.Data.FileName,
	}, nil
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
