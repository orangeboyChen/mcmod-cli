// File: internal/resolver/resolver.go
// Created: 2026-06-20
// Description: Unified resolver dispatcher per spec section 4.

package resolver

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

// parseVersionFromFileName extracts the mod's own version from a CF file
// name. It mirrors the service-layer helper so the resolver can re-render
// spec urlPattern placeholders right after constructing the LockedSource.
func parseVersionFromFileName(name string) string {
	if name == "" {
		return ""
	}
	base := name
	if i := strings.LastIndex(base, "."); i > 0 {
		base = base[:i]
	}
	re := regexp.MustCompile(`\b(\d+\.\d+(?:\.\d+)?(?:[+\.][0-9A-Za-z.-]+)*)\b`)
	matches := re.FindAllString(base, -1)
	if len(matches) == 0 {
		return ""
	}
	last := matches[len(matches)-1]
	if i := strings.LastIndex(last, "-"); i > 0 {
		right := last[i+1:]
		if regexp.MustCompile(`^\d`).MatchString(right) {
			return right
		}
	}
	return last
}

// ResolveSource looks up a mod spec against the configured source backend
// and returns a LockedSource describing the resolved file.
func ResolveSource(src domain.ModSource, mcVersion, loader string, modKey ...string) (interface{}, error) {
	if src.Type == "" {
		return nil, fmt.Errorf("source type is empty")
	}
	switch src.Type {
	case "curseforge":
		var ls *domain.LockedSource
		var err error
		if src.ModID != 0 && src.FileID != 0 {
			ls, err = ResolveCurseForgeByID(src.ModID, src.FileID, modKey...)
		} else if src.Slug != "" {
			ls, err = ResolveCurseForgeBySlug(src.Slug, mcVersion, loader, modKey...)
		} else if src.Query != "" {
			ls, err = ResolveCurseForgeByQuery(src.Query, mcVersion, loader, modKey...)
		} else {
			return nil, fmt.Errorf("curseforge needs query, slug, or modId+fileId")
		}
		if err != nil {
			return nil, err
		}
		// Apply the spec's urlPattern (or bare url) if present, so the build
		// step can download straight from the CDN without hitting the
		// rate-limited /download-url endpoint. Re-render here so the first
		// lock run after a spec edit picks up {modVersion} correctly.
		ver := parseVersionFromFileName(ls.FileName)
		if rendered := src.RenderURL(ls.ModID, ls.FileID, ls.FileName, ver, mcVersion); rendered != "" {
			ls.URL = rendered
		} else {
			ls.URL = domain.DefaultCurseForgeURL(ls.FileID, ls.FileName)
		}
		return ls, nil
	case "github-release":
		pattern := src.AssetPattern
		if pattern == "" {
			if v, ok := src.AssetPatternByLoader[loader]; ok {
				pattern = v
			}
		}
		return ResolveGitHubRelease(src.Repo, src.Tag, pattern, mcVersion, loader, modKey...)
	case "git":
		_, err := ResolveGitPackage(src.Repo, mcVersion, loader)
		if err != nil {
			return nil, err
		}
		return domain.LockedSource{Type: "git", Repo: src.Repo}, nil
	case "url":
		// Operator-supplied URL: no API calls. The spec already carries
		// modId, fileId, fileName and either a plain url or a urlPattern
		// template. Render and return a LockedSource.
		if src.ModID == 0 || src.FileID == 0 || src.FileName == "" {
			return nil, fmt.Errorf("url source requires modId, fileId, fileName")
		}
		ls := &domain.LockedSource{
			Type:     "url",
			ModID:    src.ModID,
			FileID:   src.FileID,
			FileName: src.FileName,
		}
		if rendered := src.RenderURL(src.ModID, src.FileID, src.FileName, "", mcVersion); rendered != "" {
			ls.URL = rendered
		}
		return ls, nil
	case "local":
		return ResolveLocalSource(src.Path, mcVersion, loader)
	default:
		return nil, fmt.Errorf("unknown source type: %s", src.Type)
	}
}
