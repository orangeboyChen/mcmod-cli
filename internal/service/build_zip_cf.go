// File: internal/service/build_zip_cf.go
// Created: 2026-06-21
// Description: CurseForge modpack layout writer (manifest.json + modlist.html + overrides/).

package service

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

// curseforgeManifest mirrors the on-disk JSON shape of a CurseForge modpack
// `manifest.json`. The fields are the documented subset that the official
// CurseForge app and third-party launchers consume. The struct uses
// snake_case JSON tags so the file matches the published schema byte for
// byte; do not rename fields without bumping the contract with downstream
// launchers.
type curseforgeManifest struct {
	Minecraft       curseforgeMinecraft     `json:"minecraft"`
	ManifestType    string                  `json:"manifestType"`
	ManifestVersion int                     `json:"manifestVersion"`
	Name            string                  `json:"name"`
	Version         string                  `json:"version"`
	Author          string                  `json:"author,omitempty"`
	Files           []curseforgeManifestMod `json:"files"`
	Overrides       string                  `json:"overrides"`
}

type curseforgeMinecraft struct {
	Version    string                  `json:"version"`
	ModLoaders []curseforgeModLoaderID `json:"modLoaders"`
}

type curseforgeModLoaderID struct {
	ID      string `json:"id"`
	Primary bool   `json:"primary"`
}

type curseforgeManifestMod struct {
	ProjectID int  `json:"projectID"`
	FileID    int  `json:"fileID"`
	Required  bool `json:"required"`
}

// BuildArtifactCF builds a CurseForge-compatible modpack zip for the given
// (mc, loader) pair. The zip contains three categories of content, all
// required for the launcher's import step to succeed:
//
//   - manifest.json             — CurseForge modpack manifest. One entry
//     per shared+client curseforge mod that has both
//     modId and fileId in the lock, sorted by
//     (projectID, fileID) for a stable on-disk artifact.
//   - modlist.html              — human-readable HTML summary listing
//     every shared+client mod in the lock (cf and
//     non-cf alike), with projectID links for
//     curseforge mods and a relative link into
//     overrides/mods/ for non-cf mods.
//   - overrides/config/         — copied from the project root when present.
//   - overrides/resourcepacks/  — copied from the project root when present.
//   - overrides/mods/<key>/<fileName> — every shared+client mod whose source
//     is not curseforge (github-release,
//     git, local, url). The launcher
//     picks jars up from overrides/mods/
//     when manifest.files[] cannot
//     resolve them, so non-cf mods
//     stay installed after import.
//     Each mod lands in its own
//     `<key>/` subdirectory so two
//     mods with the same file name
//     do not collide.
//
// Shared+client mods with source.type == "curseforge" AND a non-zero
// modId/fileId in the lock go into manifest.files[]. Server-scope mods
// are silently omitted (CurseForge publishes a single client pack; the
// server zip is an mcmod-only concept — see docs/002-cli-overview.md).
// Mods whose lock entry is missing modId/fileId, or whose source is
// anything other than curseforge, are bundled into overrides/mods/
// instead. A non-CurseForge mod whose jar cannot be resolved causes the
// build to fail so the import payload cannot silently drop it. The build
// still requires at least one CurseForge mod to be eligible so
// manifest.json stays a valid CF import payload.
//
// The zip is written to the same releases/v<packVersion>/ tree as the
// default layout, with a `-cf` target suffix so both flavors can coexist.
// Returns the on-disk zip path on success.
func BuildArtifactCF(spec *domain.PackSpec, lock *domain.PackLock, mcVersion string, force bool) (string, error) {
	if spec == nil {
		return "", fmt.Errorf("spec is nil")
	}
	if lock == nil {
		return "", fmt.Errorf("lock is nil")
	}
	if err := ValidateBuildLock(spec, lock, "client"); err != nil {
		return "", err
	}
	loader, loaderVer := loaderFromLock(lock)
	bc := &buildContext{
		Spec:          spec,
		Lock:          lock,
		McVersion:     mcVersion,
		Loader:        loader,
		LoaderVersion: loaderVer,
		RootDir:       ".",
	}
	out := bc.zipPath("cf")
	if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		return "", err
	}
	if !force {
		if _, err := os.Stat(out); err == nil {
			return "", fmt.Errorf("artifact already exists at %s (use --force to overwrite)", out)
		}
	}

	manifest, manifestMods, skipped := bc.buildCurseForgeManifest()
	if len(manifestMods) == 0 {
		return "", fmt.Errorf("build cf: no curseforge-sourced mods in lock (skipped %d non-cf or modid-less entries)", len(skipped))
	}
	modFiles := make(map[string]string)
	for _, key := range bc.modsForTarget("client") {
		source := bc.Lock.Mods[key].Source
		if source.Type == "curseforge" && (source.ModID == 0 || source.FileID == 0 || source.FileName == "") {
			continue
		}
		jarPath, err := bc.resolveModJar(key, bc.Lock.Mods[key])
		if err != nil {
			return "", fmt.Errorf("build cf: resolve mod %s: %w", key, err)
		}
		modFiles[key] = jarPath
	}
	if err := validateModFiles(bc, modFiles); err != nil {
		return "", err
	}

	f, err := os.Create(out)
	if err != nil {
		return "", err
	}
	defer f.Close()
	complete := false
	defer func() {
		if !complete {
			_ = os.Remove(out)
		}
	}()
	w := zip.NewWriter(f)
	defer w.Close()

	// manifest.json
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", fmt.Errorf("build cf: marshal manifest: %w", err)
	}
	if err := writeZipEntry(w, "manifest.json", manifestJSON); err != nil {
		return "", err
	}

	// Collect every shared+client mod for the modlist and the
	// overrides/mods/ bundling pass. manifestMods already covers
	// curseforge-sourced mods; we now add the non-cf ones so the HTML
	// reflects the full set, then bundle their jars below.
	allRefs := append([]manifestModRef(nil), manifestMods...)
	bundled := []string{}
	for _, ref := range bc.nonCFModsForBuild() {
		allRefs = append(allRefs, ref)
		jarPath := modFiles[ref.Key]
		base := filepath.Base(jarPath)
		entry := filepath.ToSlash(filepath.Join("overrides", "mods", ref.Key, base))
		if err := addFileToZip(w, jarPath, entry); err != nil {
			return "", fmt.Errorf("build cf: bundle %s into %s: %w", ref.Key, entry, err)
		}
		bundled = append(bundled, ref.Key)
	}
	// Re-sort so the HTML list is in stable key order regardless of
	// whether the mod came from manifest or overrides.
	sort.SliceStable(allRefs, func(i, j int) bool {
		return allRefs[i].Key < allRefs[j].Key
	})

	// modlist.html
	modlistHTML := renderCurseForgeModlistHTML(manifest, allRefs)
	if err := writeZipEntry(w, "modlist.html", []byte(modlistHTML)); err != nil {
		return "", err
	}

	// overrides/ — config and resourcepacks. Per the CurseForge
	// convention, anything that should land in the user's .minecraft/ root
	// goes under overrides/. We deliberately do NOT put mods/ at the top
	// of overrides/; non-cf mods land under overrides/mods/<key>/<jar>
	// above so each one is addressable.
	if err := addDirToZip(w, filepath.Join(bc.RootDir, "config"), "overrides/config"); err != nil {
		return "", err
	}
	if err := addDirToZip(w, filepath.Join(bc.RootDir, "resourcepacks"), "overrides/resourcepacks"); err != nil {
		return "", err
	}

	if len(skipped) > 0 {
		fmt.Fprintf(os.Stderr, "build cf: %d mod(s) omitted from manifest (will be bundled into overrides/mods/ if non-cf): %v\n", len(skipped), skipped)
	}
	if len(bundled) > 0 {
		fmt.Fprintf(os.Stderr, "build cf: bundled %d non-cf mod(s) into overrides/mods/: %v\n", len(bundled), bundled)
	}
	complete = true
	return out, nil
}

// buildCurseForgeManifest assembles the manifest payload and the list of
// curseforge mods eligible for the layout. Mods whose source.type is not
// "curseforge", or whose lock entry lacks modId/fileId, are returned in
// the skipped slice so the caller can report them. The return order of
// manifest.Files is sorted by projectID+fileID for a stable on-disk file.
func (bc *buildContext) buildCurseForgeManifest() (curseforgeManifest, []manifestModRef, []string) {
	loaderID := fmt.Sprintf("%s-%s", bc.Loader, bc.LoaderVersion)
	if bc.LoaderVersion == "" {
		loaderID = bc.Loader + "-0"
	}
	m := curseforgeManifest{
		Minecraft: curseforgeMinecraft{
			Version: bc.McVersion,
			ModLoaders: []curseforgeModLoaderID{
				{ID: loaderID, Primary: true},
			},
		},
		ManifestType:    "minecraftModpack",
		ManifestVersion: 1,
		Name:            bc.Spec.PackName,
		Version:         bc.Spec.PackVersion,
		Author:          bc.Spec.Author,
		Overrides:       "overrides",
	}
	var refs []manifestModRef
	var skipped []string
	for _, key := range sortedLockKeys(bc.Lock.Mods) {
		locked := bc.Lock.Mods[key]
		scope := locked.Scope
		if scope == "" {
			scope = domain.ScopeShared
		}
		if scope == domain.ScopeServer {
			continue
		}
		if locked.Source.Type != domain.SourceCurseForge {
			skipped = append(skipped, key)
			continue
		}
		if locked.Source.ModID == 0 || locked.Source.FileID == 0 {
			skipped = append(skipped, key)
			continue
		}
		m.Files = append(m.Files, curseforgeManifestMod{
			ProjectID: locked.Source.ModID,
			FileID:    locked.Source.FileID,
			Required:  true,
		})
		refs = append(refs, manifestModRef{
			Key:       key,
			Name:      locked.Name,
			Version:   locked.Version,
			ProjectID: locked.Source.ModID,
			FileID:    locked.Source.FileID,
		})
	}
	sort.SliceStable(m.Files, func(i, j int) bool {
		if m.Files[i].ProjectID != m.Files[j].ProjectID {
			return m.Files[i].ProjectID < m.Files[j].ProjectID
		}
		return m.Files[i].FileID < m.Files[j].FileID
	})
	return m, refs, skipped
}

// nonCFModsForBuild returns the manifestModRef list of every shared+client
// mod whose source is not curseforge, in stable key order. These are the
// mods the CF layout must bundle into overrides/mods/<key>/<jar> so the
// launcher can pick them up at import time (manifest.files[] is cf-only).
// Server-scope mods are excluded because CF publishes a single client
// pack; the server zip is built via the default layout.
func (bc *buildContext) nonCFModsForBuild() []manifestModRef {
	var refs []manifestModRef
	for _, key := range sortedLockKeys(bc.Lock.Mods) {
		locked := bc.Lock.Mods[key]
		scope := locked.Scope
		if scope == "" {
			scope = domain.ScopeShared
		}
		if scope == domain.ScopeServer {
			continue
		}
		if locked.Source.Type == domain.SourceCurseForge {
			continue
		}
		refs = append(refs, manifestModRef{
			Key:     key,
			Name:    locked.Name,
			Version: locked.Version,
		})
	}
	return refs
}

type manifestModRef struct {
	Key       string
	Name      string
	Version   string
	ProjectID int
	FileID    int
}

// renderCurseForgeModlistHTML builds a small HTML summary that mirrors the
// modlist.html page CurseForge publishes on the modpack detail page. The
// markup is intentionally minimal — no CSS, no JS — so the file
// round-trips through the standard library's html.EscapeString. Both
// curseforge and non-cf mods are listed. Curseforge mods link to the
// project page; non-cf mods link to a path inside overrides/mods/.
func renderCurseForgeModlistHTML(m curseforgeManifest, refs []manifestModRef) string {
	var buf bytes.Buffer
	buf.WriteString("<!DOCTYPE html>\n")
	buf.WriteString("<html lang=\"en\">\n<head>\n")
	fmt.Fprintf(&buf, "  <meta charset=\"utf-8\">\n")
	fmt.Fprintf(&buf, "  <title>%s - %s</title>\n", html.EscapeString(m.Name), html.EscapeString(m.Version))
	buf.WriteString("</head>\n<body>\n")
	fmt.Fprintf(&buf, "  <h1>%s</h1>\n", html.EscapeString(m.Name))
	fmt.Fprintf(&buf, "  <p>Version: %s</p>\n", html.EscapeString(m.Version))
	fmt.Fprintf(&buf, "  <p>Minecraft: %s</p>\n", html.EscapeString(m.Minecraft.Version))
	if m.Author != "" {
		fmt.Fprintf(&buf, "  <p>Author: %s</p>\n", html.EscapeString(m.Author))
	}
	fmt.Fprintf(&buf, "  <p>Generated by mcmod on %s</p>\n", html.EscapeString(time.Now().UTC().Format(time.RFC3339)))
	buf.WriteString("  <ul>\n")
	for _, r := range refs {
		label := r.Name
		if label == "" {
			label = r.Key
		}
		ver := r.Version
		if ver == "" {
			ver = "?"
		}
		switch {
		case r.ProjectID > 0:
			fmt.Fprintf(&buf, "    <li><a href=\"https://www.curseforge.com/minecraft/mc-mods/%d\">%s</a> (%s)</li>\n",
				r.ProjectID, html.EscapeString(label), html.EscapeString(ver))
		case r.Key != "":
			fmt.Fprintf(&buf, "    <li>%s (%s) [bundled: overrides/mods/%s/]</li>\n",
				html.EscapeString(label), html.EscapeString(ver), html.EscapeString(r.Key))
		default:
			fmt.Fprintf(&buf, "    <li>%s (%s)</li>\n",
				html.EscapeString(label), html.EscapeString(ver))
		}
	}
	buf.WriteString("  </ul>\n")
	buf.WriteString("</body>\n</html>\n")
	return buf.String()
}

// sortedLockKeys returns the lock's mod keys in stable, alphabetical order.
// Used by every layout writer so the produced zip is deterministic across
// runs (helps with diffing and with the "files sorted by projectID+fileID"
// contract in the CurseForge manifest).
func sortedLockKeys(mods map[string]domain.LockedMod) []string {
	keys := make([]string, 0, len(mods))
	for k := range mods {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// writeZipEntry writes a single in-memory byte slice into the zip under the
// given entry name. Centralised so the cf layout and the default layout
// share one path for "synthesised file" entries (manifest.json, modlist.html).
func writeZipEntry(w *zip.Writer, name string, data []byte) error {
	h := &zip.FileHeader{
		Name:   name,
		Method: zip.Deflate,
	}
	h.SetMode(0644)
	wr, err := w.CreateHeader(h)
	if err != nil {
		return err
	}
	_, err = wr.Write(data)
	return err
}
