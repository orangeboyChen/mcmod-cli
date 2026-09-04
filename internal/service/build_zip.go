// File: internal/service/build_zip.go
// Created: 2026-06-20
// Description: Build-time zip output (path naming, content layout, file/dir writers).

package service

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// zipBaseName returns the zip basename prefix for a target build.
// Per spec 7.5.4/7.5.5, client uses packName; server uses serverPackName (or packName fallback).
func (bc *buildContext) zipBaseName(target string) string {
	if target == "server" {
		name := bc.Spec.ServerPackName
		if name == "" {
			name = bc.Spec.PackName
		}
		return name
	}
	return bc.Spec.PackName
}

// zipPath returns the full output zip path for a target build.
func (bc *buildContext) zipPath(target string) string {
	base := bc.zipBaseName(target)
	loaderVer := bc.LoaderVersion
	if loaderVer == "" {
		loaderVer = "0"
	}
	fname := fmt.Sprintf("%s-%s-%s-%s-%s.zip", base, bc.McVersion, bc.Loader, loaderVer, target)
	return filepath.Join(bc.RootDir, "releases", "v"+bc.Spec.PackVersion, fname)
}

func (bc *buildContext) buildZip(target, path string, modFiles map[string]string) error {
	return bc.buildZipWith(target, path, modFiles, false)
}

func (bc *buildContext) buildZipWith(target, path string, modFiles map[string]string, force bool) error {
	if err := validateModFiles(bc, modFiles); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("artifact already exists at %s (use --force to overwrite)", path)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := zip.NewWriter(f)
	defer w.Close()

	// mods/<fileName> in deterministic key order
	keys := make([]string, 0, len(modFiles))
	for k := range modFiles {
		keys = append(keys, k)
	}
	// stable sort by key
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	for _, key := range keys {
		jarPath := modFiles[key]
		baseName := filepath.Base(jarPath)
		if err := addFileToZip(w, jarPath, "mods/"+baseName); err != nil {
			return fmt.Errorf("add mod %s: %w", key, err)
		}
	}

	// Optional config/ always
	if err := addDirToZip(w, filepath.Join(bc.RootDir, "config"), "config"); err != nil {
		return err
	}
	// defaultconfigs/ for server, optional for client
	if target == "server" {
		if err := addDirToZip(w, filepath.Join(bc.RootDir, "defaultconfigs"), "defaultconfigs"); err != nil {
			return err
		}
	}
	// resourcepacks/ for client only
	if target == "client" {
		if err := addDirToZip(w, filepath.Join(bc.RootDir, "resourcepacks"), "resourcepacks"); err != nil {
			return err
		}
	}
	// server-only files
	if target == "server" {
		for _, fname := range []string{"server.properties", "whitelist.json", "ops.json"} {
			_ = addFileToZip(w, filepath.Join(bc.RootDir, fname), fname)
		}
	}
	return nil
}

// addFileToZip adds a single file to the zip under the given entry name.
// If the source file is missing it returns nil (optional files).
func addFileToZip(w *zip.Writer, src, entry string) error {
	if _, err := os.Stat(src); err != nil {
		return nil
	}
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = entry
	header.Method = zip.Deflate
	wr, err := w.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(wr, f)
	return err
}

// addDirToZip walks dir and writes every file under prefix in the archive.
// Returns nil when dir does not exist so optional folders (config, resourcepacks, ...)
// can be passed unconditionally.
func addDirToZip(w *zip.Writer, dir, prefix string) error {
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		entry := filepath.ToSlash(filepath.Join(prefix, rel))
		return addFileToZip(w, path, entry)
	})
}
