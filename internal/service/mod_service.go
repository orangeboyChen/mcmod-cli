// File: internal/service/mod_service.go
// Created: 2026-06-20
// Description: Mod and lock service logic.

package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/orangeboyChen/mcmod-cli/internal/cache"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
	"github.com/orangeboyChen/mcmod-cli/internal/resolver"
)

// SpecFingerprint computes a stable SHA256 fingerprint of a spec source
// definition. The fingerprint is used to detect when a mod's spec entry
// has changed between lock runs so the resolver can be re-invoked even
// when the previous lock entry is otherwise still valid. Encoding rule:
// marshal to JSON, sort object keys recursively, then SHA256 and return
// the first 16 bytes hex.
func SpecFingerprint(src domain.ModSource) string {
	enc, err := canonicalJSON(src)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(enc)
	return hex.EncodeToString(sum[:16])
}

func canonicalJSON(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return canonicalJSONValue(m)
}

func canonicalJSONValue(v interface{}) ([]byte, error) {
	switch t := v.(type) {
	case map[string]interface{}:
		var keys []string
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var buf []byte
		buf = append(buf, '{')
		for i, k := range keys {
			if i > 0 {
				buf = append(buf, ',')
			}
			kb, _ := json.Marshal(k)
			buf = append(buf, kb...)
			buf = append(buf, ':')
			vb, err := canonicalJSONValue(t[k])
			if err != nil {
				return nil, err
			}
			buf = append(buf, vb...)
		}
		buf = append(buf, '}')
		return buf, nil
	case []interface{}:
		var buf []byte
		buf = append(buf, '[')
		for i, x := range t {
			if i > 0 {
				buf = append(buf, ',')
			}
			vb, err := canonicalJSONValue(x)
			if err != nil {
				return nil, err
			}
			buf = append(buf, vb...)
		}
		buf = append(buf, ']')
		return buf, nil
	default:
		return json.Marshal(t)
	}
}

// parseVersionFromFileName extracts a semver-like version (e.g. "1.3.0") from
// a CurseForge file name. It is best-effort: when the file name does not
// contain a recognisable version, "" is returned and the spec's urlPattern
// will simply leave {modVersion} unsubstituted.
func parseVersionFromFileName(name string) string {
	if name == "" {
		return ""
	}
	base := name
	if i := strings.LastIndex(base, "."); i > 0 {
		base = base[:i]
	}
	// Find every x.y[.z][-suffix] token in the basename. The mod's own
	// version is almost always the LAST such token; the loader / mc
	// version sits earlier. For names like "1.21.1-1.3.0" the regex
	// matches the whole token, so we split on the last "-" when the
	// right-hand side still looks like a version.
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

// resolvedCache maps mod key -> known good modId/fileId/fileName so lock can
// skip the CurseForge search step on subsequent runs. The cache lives at
// .mcmod/cache/resolved/<mc>-<loader>.json. It is best-effort: a stale entry
// (file removed upstream) is fine because BuildLock re-validates the file
// via findCurseForgeFile before trusting it.
type resolvedCache map[string]resolvedEntry

type resolvedEntry struct {
	Type      string `json:"type"`
	ModID     int    `json:"modId,omitempty"`
	FileID    int    `json:"fileId,omitempty"`
	FileName  string `json:"fileName,omitempty"`
	Repo      string `json:"repo,omitempty"`
	Tag       string `json:"tag,omitempty"`
	AssetName string `json:"assetName,omitempty"`
}

func resolvedCachePath(mcVersion, loader string) string {
	return cache.ResolvedIDPath(mcVersion, loader)
}

func loadResolvedCache(mcVersion, loader string) resolvedCache {
	path := resolvedCachePath(mcVersion, loader)
	data, err := os.ReadFile(path)
	if err != nil {
		return resolvedCache{}
	}
	var c resolvedCache
	if err := json.Unmarshal(data, &c); err != nil {
		return resolvedCache{}
	}
	return c
}

func saveResolvedCache(mcVersion, loader string, c resolvedCache) error {
	if err := cache.EnsureCacheDir(); err != nil {
		return err
	}
	path := resolvedCachePath(mcVersion, loader)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// ListMods returns a formatted listing of mods grouped by scope.
func ListMods(spec *domain.PackSpec) (string, error) {
	blocks := []string{
		fmt.Sprintf("pack %s (%s)", spec.PackName, spec.PackVersion),
		"loader:",
	}
	for _, ln := range spec.LoaderName {
		blocks = append(blocks, fmt.Sprintf("  - %s", ln))
	}
	blocks = append(blocks, "")

	scopes := []struct{ name, order string }{
		{"Server", "server"},
		{"Client", "client"},
		{"Shared", "shared"},
	}

	for _, s := range scopes {
		blocks = append(blocks, fmt.Sprintf("[%s]", s.name))
		var items []string
		for key, mod := range spec.Mods {
			actualScope := mod.Scope
			if actualScope == "" {
				actualScope = "shared"
			}
			if actualScope != s.order {
				continue
			}
			name := mod.Name
			if name == "" {
				name = key
			}
			src := mod.Source.Type
			if src == "" {
				src = "unknown"
			}
			items = append(items, fmt.Sprintf("  %s [%s]", name, src))
		}
		sort.Strings(items)
		if len(items) == 0 {
			blocks = append(blocks, "  (empty)")
		} else {
			blocks = append(blocks, items...)
		}
		blocks = append(blocks, "")
	}

	return strings.Join(blocks, "\n"), nil
}

// BuildLock generates a dependency lock from packspec.
// BuildLock resolves every mod in spec.Mods to a LockedMod and returns a
// PackLock. It runs resolver calls concurrently with a small worker pool so
// the whole batch finishes in roughly one mod's worth of wall time even when
// some endpoints are slow or rate-limited.
//
// Behaviour:
//   - Mods that already exist in `existing` (when non-nil) are carried over
//     verbatim; the resolver is not called for them again. This is the
//     incremental lock path: re-run mcmod lock and only the failed / new
//     entries are retried.
//   - Mods that fail to resolve are recorded in lf.Failed. The lock file is
//     still returned (and the caller still writes it) so the operator can see
//     which mods succeeded alongside the failure list.
//   - BuildLock returns a non-nil error only when the resolver pipeline is
//     structurally broken (e.g. unknown source type) so the lock file is
//     always persisted when at least one mod resolved.
func BuildLock(spec *domain.PackSpec, mcVersion, loader string) (*domain.PackLock, int, error) {
	return BuildLockWithExisting(spec, mcVersion, loader, nil)
}

// BuildLockWithExisting is BuildLock plus an optional previously-written lock
// file. Mods already present in `existing` are skipped (cached lock entries),
// so the resolver is only invoked for new or previously-failed entries.
//
// The second return value is the number of mods in spec.Mods that the
// resolver failed to resolve (zero or more). It is reported separately
// from `error` so the caller can decide whether a partial lock is fatal:
// the lock file is still written so the operator can see which mods
// succeeded alongside the failure list, but callers (like the CLI) can
// still surface the failure to stderr and exit non-zero. The function
// only returns a non-nil error when the resolver pipeline is structurally
// broken (e.g. unknown source type), mirroring BuildLock's contract.
func BuildLockWithExisting(spec *domain.PackSpec, mcVersion, loader string, existing *domain.PackLock) (*domain.PackLock, int, error) {
	expandedMods, err := resolver.ExpandGitDependencies(spec, mcVersion, loader)
	if err != nil {
		return nil, 0, err
	}
	effectiveSpec := *spec
	effectiveSpec.Mods = expandedMods
	loaderVer := ""
	for _, ln := range effectiveSpec.LoaderName {
		name, ver := domain.ParseLoaderName(ln)
		if name == loader {
			loaderVer = ver
			break
		}
	}
	lf := &domain.PackLock{
		MinecraftVersion: mcVersion,
		Loader:           loader,
		LoaderVersion:    loaderVer,
		Mods:             make(map[string]domain.LockedMod),
	}

	// Load the resolved-id cache (.mcmod/cache/resolved/<mc>-<loader>.json) so we
	// can skip the expensive CurseForge search step on subsequent runs. The
	// cache is keyed by spec mod key; a hit lets the worker construct the
	// LockedSource directly from the cached modId/fileId/fileName.
	cache := sync.Map{}
	loadedCache := loadResolvedCache(mcVersion, loader)
	for k, v := range loadedCache {
		cache.Store(k, v)
	}
	cacheDirty := false

	type pending struct {
		key    string
		modDef domain.ModSpec
	}
	kept := 0
	added := 0
	queue := make([]pending, 0, len(effectiveSpec.Mods))
	for k, m := range effectiveSpec.Mods {
		if existing != nil {
			if prev, ok := existing.Mods[k]; ok {
				// Detect spec changes via a stable fingerprint. If the spec
				// source has changed since the previous lock, drop the cached
				// entry and let the resolver run again.
				wantHash := SpecFingerprint(m.Source)
				if prev.Hash != "" && prev.Hash != wantHash {
					fmt.Fprintf(os.Stderr, "lock: spec changed for mod %s (hash %s -> %s); will re-resolve\n", k, prev.Hash, wantHash)
				} else {
					lf.Mods[k] = prev
					kept++
					fmt.Fprintf(os.Stderr, "lock: keeping existing mod %s\n", k)
					continue
				}
			}
		}
		queue = append(queue, pending{key: k, modDef: m})
		added++
	}

	// Detect mods that were in the previous lock but are no longer in the
	// spec. The diff is reported in lf.Removed and the next build will simply
	// not see these mods. The previous lock entries are dropped from lf.Mods
	// (they were never copied in since spec no longer lists them).
	removed := []string{}
	if existing != nil {
		for k := range existing.Mods {
			if _, stillInSpec := effectiveSpec.Mods[k]; !stillInSpec {
				removed = append(removed, k)
			}
		}
		sort.Strings(removed)
		for _, k := range removed {
			fmt.Fprintf(os.Stderr, "lock: mod %s removed (no longer in spec)\n", k)
		}
	}

	type result struct {
		key    string
		modDef domain.ModSpec
		source domain.LockedSource
		err    error
	}
	const workers = 4
	jobs := make(chan pending)
	out := make(chan result)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				// Cache fast-path: if we already have a resolved modId/fileId
				// for this key, validate it exists with a single file-detail
				// call and skip the search step entirely. A 404 from CF means
				// the file was withdrawn/replaced upstream; in that case we
				// fall back to the full search so the operator gets a fresh
				// fileId.
				if entryIface, ok := cache.Load(p.key); ok && entryIface.(resolvedEntry).Type == p.modDef.Source.Type {
					entry := entryIface.(resolvedEntry)
					switch entry.Type {
					case "curseforge":
						if entry.ModID != 0 && entry.FileID != 0 {
							ls, err := resolver.ResolveCurseForgeByID(entry.ModID, entry.FileID, p.key)
							if err == nil {
								if p.modDef.Source.URLPattern != "" {
									ver := parseVersionFromFileName(ls.FileName)
									rendered := p.modDef.Source.RenderURL(ls.ModID, ls.FileID, ls.FileName, ver, mcVersion)
									if rendered != "" {
										ls.URL = rendered
									}
								}
								fmt.Fprintf(os.Stderr, "resolving mod %s (cache hit, file verified curseforge/%d/%d/%s)\n", p.key, ls.ModID, ls.FileID, ls.FileName)
								out <- result{key: p.key, modDef: p.modDef, source: *ls}
								continue
							}
							fmt.Fprintf(os.Stderr, "resolving mod %s (cache stale: %v; will re-search)\n", p.key, err)
						}
					case "github-release":
						if entry.Repo != "" && entry.Tag != "" && entry.AssetName != "" {
							fmt.Fprintf(os.Stderr, "resolving mod %s (cache hit github-release/%s/%s)\n", p.key, entry.Repo, entry.AssetName)
							out <- result{key: p.key, modDef: p.modDef, source: domain.LockedSource{Type: "github-release", Repo: entry.Repo, Tag: entry.Tag, AssetName: entry.AssetName}}
							continue
						}
					}
				}
				fmt.Fprintf(os.Stderr, "resolving mod %s (type=%s) for %s/%s\n", p.key, p.modDef.Source.Type, mcVersion, loader)
				ls, err := resolver.ResolveSource(p.modDef.Source, mcVersion, loader, p.key)
				if err != nil {
					out <- result{key: p.key, modDef: p.modDef, err: err}
					continue
				}
				var source domain.LockedSource
				switch v := ls.(type) {
				case domain.LockedSource:
					source = v
				case *domain.LockedSource:
					if v != nil {
						source = *v
					}
				}
				// Persist the resolved ids to the cache so the next run can skip
				// the search step.
				if source.Type != "" {
					cache.Store(p.key, resolvedEntry{
						Type:      source.Type,
						ModID:     source.ModID,
						FileID:    source.FileID,
						FileName:  source.FileName,
						Repo:      source.Repo,
						Tag:       source.Tag,
						AssetName: source.AssetName,
					})
					cacheDirty = true
				}
				out <- result{key: p.key, modDef: p.modDef, source: source}
			}
		}()
	}

	go func() {
		for _, p := range queue {
			jobs <- p
		}
		close(jobs)
		wg.Wait()
		close(out)
	}()

	type lockFailure struct{ key, errMsg string }
	failures := make([]lockFailure, 0)
	for r := range out {
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "lock: mod %s failed: %v\n", r.key, r.err)
			failures = append(failures, lockFailure{key: r.key, errMsg: r.err.Error()})
			continue
		}
		scope := r.modDef.Scope
		if scope == "" {
			scope = "shared"
		}
		lf.Mods[r.key] = domain.LockedMod{
			Name:  r.modDef.Name,
			Scope: scope,
			Identity: &domain.Identity{
				Source:     fmt.Sprintf("%s:%s", r.source.Type, r.source.FileName),
				Confidence: "source-only",
			},
			Source: r.source,
			Hash:   SpecFingerprint(r.modDef.Source),
		}
		fmt.Fprintf(os.Stderr, "locked mod %s -> %s/%d/%d/%s\n", r.key, r.source.Type, r.source.ModID, r.source.FileID, r.source.FileName)
	}

	// Final summary: counts for each category plus per-failure detail so the
	// operator can see exactly what happened without inspecting the lock file.
	fmt.Fprintf(os.Stderr, "lock summary: added=%d kept=%d removed=%d failed=%d (total in spec=%d)\n",
		added, kept, len(removed), len(failures), len(effectiveSpec.Mods))
	if cacheDirty {
		// Drain sync.Map into a plain map for saveResolvedCache which is
		// not concurrency-aware. The single-writer rule applies only to
		// background workers; at this point all workers are joined and
		// it is safe to read.
		cacheMap := make(map[string]resolvedEntry, len(loadedCache))
		cache.Range(func(k, v interface{}) bool {
			cacheMap[k.(string)] = v.(resolvedEntry)
			return true
		})
		if err := saveResolvedCache(mcVersion, loader, cacheMap); err != nil {
			fmt.Fprintf(os.Stderr, "lock: warning: failed to persist resolved cache: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "lock: persisted %d resolved mod id(s) to %s\n", len(cacheMap), resolvedCachePath(mcVersion, loader))
		}
	}
	if len(failures) > 0 {
		fmt.Fprintf(os.Stderr, "lock: the following mods failed to resolve and need attention:\n")
		for _, f := range failures {
			fmt.Fprintf(os.Stderr, "  - %s: %s\n", f.key, f.errMsg)
		}
		fmt.Fprintf(os.Stderr, "lock: failed mods are NOT in the lock file; fix the underlying problem and re-run `mcmod lock`\n")
	}
	return lf, len(failures), nil
}
