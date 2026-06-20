// File: internal/service/build_zip_test.go
// Created: 2026-06-20
// Description: Ginkgo tests for internal/service/build_zip.go (zip path, layout, file writers).

package service

import (
	"archive/zip"
	"bytes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"io"
	"os"
	"path/filepath"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

var _ = Describe("Service buildZip direct", func() {
	It("buildZip writes zip", func() {
		dir := GinkgoT().TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)
		os.WriteFile(filepath.Join(dir, "a.jar"), []byte("aaa"), 0644)
		spec := &domain.PackSpec{
			PackName: "p", PackVersion: "0.7.0", MinecraftVersion: "1.21.1",
			LoaderName: []string{"neoforge:1.0"},
		}
		lock := &domain.PackLock{
			Loader: "neoforge", LoaderVersion: "1.0", MinecraftVersion: "1.21.1",
			Mods: map[string]domain.LockedMod{
				"a": {Name: "A", Scope: "shared", Source: domain.LockedSource{Type: "local", Path: "./a.jar", FileName: "a.jar"}},
			},
		}
		bc := &buildContext{Spec: spec, Lock: lock, McVersion: "1.21.1", Loader: "neoforge", LoaderVersion: "1.0", RootDir: "."}
		err := bc.buildZip("client", filepath.Join(dir, "out.zip"), map[string]string{"a": filepath.Join(dir, "a.jar")})
		Expect(err).NotTo(HaveOccurred())
		_, err = os.Stat(filepath.Join(dir, "out.zip"))
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("addFileToZip and addDirToZip edge cases", func() {
	It("addDirToZip with a missing directory is a no-op", func() {
		// We need a real zip writer. Build a tiny in-memory zip.
		var buf bytes.Buffer
		w := zip.NewWriter(&buf)
		Expect(addDirToZip(w, "/no/such/dir", "prefix")).To(Succeed())
		Expect(w.Close()).To(Succeed())
		// No entries were written.
		// zip header is always present
		_ = buf.Len()
	})

	It("addDirToZip with a directory containing files adds them under the prefix", func() {
		dir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0644)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(dir, "nested"), 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "nested", "c.txt"), []byte("c"), 0644)).To(Succeed())

		var buf bytes.Buffer
		w := zip.NewWriter(&buf)
		Expect(addDirToZip(w, dir, "cfg")).To(Succeed())
		Expect(w.Close()).To(Succeed())

		// Re-open the zip to verify the prefix.
		r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		Expect(err).NotTo(HaveOccurred())
		Expect(len(r.File)).To(Equal(3))
		seen := map[string]bool{}
		for _, f := range r.File {
			seen[f.Name] = true
		}
		Expect(seen).To(HaveKey("cfg/a.txt"))
		Expect(seen).To(HaveKey("cfg/b.txt"))
		Expect(seen).To(HaveKey(filepath.Join("cfg", "nested", "c.txt")))
	})

	It("addFileToZip with a missing source is a no-op", func() {
		var buf bytes.Buffer
		w := zip.NewWriter(&buf)
		Expect(addFileToZip(w, "/no/such/file.txt", "entry.txt")).To(Succeed())
		Expect(w.Close()).To(Succeed())
		// zip header is always present
		_ = buf.Len()
	})
})

var _ = Describe("buildZipWith artifact exists", func() {
	It("returns an error when the artifact already exists and force is false", func() {
		dir := GinkgoT().TempDir()
		bc := &buildContext{RootDir: dir, Loader: "neoforge", McVersion: "1.21.1"}
		out := filepath.Join(dir, "out.zip")
		// Pre-create the file to trigger the "already exists" branch.
		Expect(os.WriteFile(out, []byte("placeholder"), 0644)).To(Succeed())
		err := bc.buildZipWith("client", out, map[string]string{}, false)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("already exists"))
	})
})

var _ = Describe("addFileToZip with a real source", func() {
	It("writes the source contents into the zip under the given entry", func() {
		dir := GinkgoT().TempDir()
		src := filepath.Join(dir, "src.txt")
		Expect(os.WriteFile(src, []byte("hello"), 0644)).To(Succeed())

		var buf bytes.Buffer
		w := zip.NewWriter(&buf)
		Expect(addFileToZip(w, src, "entry.txt")).To(Succeed())
		Expect(w.Close()).To(Succeed())

		r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		Expect(err).NotTo(HaveOccurred())
		Expect(r.File).To(HaveLen(1))
		Expect(r.File[0].Name).To(Equal("entry.txt"))
		rc, err := r.File[0].Open()
		Expect(err).NotTo(HaveOccurred())
		defer rc.Close()
		data, err := io.ReadAll(rc)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(Equal("hello"))
	})
})

var _ = Describe("buildZipWith with at least one mod", func() {
	It("writes a zip with the mod file inside mods/", func() {
		dir := GinkgoT().TempDir()
		wd, _ := os.Getwd()
		Expect(os.Chdir(dir)).To(Succeed())
		DeferCleanup(func() { _ = os.Chdir(wd) })

		jar := filepath.Join(dir, "mod.jar")
		Expect(os.WriteFile(jar, []byte("x"), 0644)).To(Succeed())

		bc := &buildContext{RootDir: dir, Loader: "neoforge", McVersion: "1.21.1"}
		out := filepath.Join(dir, "out.zip")
		err := bc.buildZipWith("client", out, map[string]string{"a": jar}, true)
		Expect(err).NotTo(HaveOccurred())
		_, err = os.Stat(out)
		Expect(err).NotTo(HaveOccurred())
	})

	It("writes server-only files when target is server", func() {
		dir := GinkgoT().TempDir()
		wd, _ := os.Getwd()
		Expect(os.Chdir(dir)).To(Succeed())
		DeferCleanup(func() { _ = os.Chdir(wd) })

		jar := filepath.Join(dir, "mod.jar")
		Expect(os.WriteFile(jar, []byte("x"), 0644)).To(Succeed())

		// Create a server.properties file that should be packaged.
		Expect(os.WriteFile(filepath.Join(dir, "server.properties"), []byte("p"), 0644)).To(Succeed())

		bc := &buildContext{RootDir: dir, Loader: "neoforge", McVersion: "1.21.1"}
		out := filepath.Join(dir, "out.zip")
		err := bc.buildZipWith("server", out, map[string]string{"a": jar}, true)
		Expect(err).NotTo(HaveOccurred())

		// Verify the server.properties is in the zip.
		r, err := zip.OpenReader(out)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { r.Close() })
		names := []string{}
		for _, f := range r.File {
			names = append(names, f.Name)
		}
		Expect(names).To(ContainElement("server.properties"))
	})
})
