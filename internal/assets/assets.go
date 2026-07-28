// Package assets embeds the Typst profile templates and brand fonts into the
// binary and materializes them to a real directory so the external typst process
// can read them. Keeping templates and fonts embedded keeps symprint a single
// self-contained binary (standalone-first).
package assets

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed templates/*.typ
var fsys embed.FS

//go:embed fonts/*.ttf fonts/*.txt
var fontFS embed.FS

// packagesFS embeds vendored Typst packages so every render works offline.
// Currently holds @preview/cmarker:0.1.9 (MIT license; see NOTICE). The
// directory layout mirrors the Typst package layout
// (packages/{namespace}/{name}/{version}/) so the materialized tree can be
// passed to typst via --package-path.
//
//go:embed packages/preview/cmarker/0.1.9/*
var packagesFS embed.FS

// PackagesFS exposes the embedded vendored-package tree for regression tests
// and callers that want to read package files directly.
func PackagesFS() embed.FS { return packagesFS }

// Materialize writes every embedded template into dir as real files so typst
// can import them (typst resolves imports from the filesystem, not embed.FS).
func Materialize(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return fs.WalkDir(fsys, "templates", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := fsys.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, d.Name()), b, 0o644)
	})
}

// MaterializeFonts writes every embedded font into dir as real files so typst
// can discover them via --font-path. Returns the path to the font directory.
func MaterializeFonts(dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	err := fs.WalkDir(fontFS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := fontFS.ReadFile(p)
		if err != nil {
			return err
		}
		// Write to the root of the font dir (flatten the structure).
		return os.WriteFile(filepath.Join(dir, d.Name()), b, 0o644)
	})
	if err != nil {
		return "", err
	}
	return dir, nil
}

// MaterializePackages writes the vendored Typst packages into dir, preserving
// the packages/{namespace}/{name}/{version}/ layout, so typst can resolve
// imports like `@preview/cmarker:0.1.9` from the local tree without network
// access. Pass filepath.Join(dir, "packages") to typst via --package-path.
func MaterializePackages(dir string) error {
	return fs.WalkDir(packagesFS, "packages", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dir, p), 0o755)
		}
		b, err := packagesFS.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, p), b, 0o644)
	})
}
