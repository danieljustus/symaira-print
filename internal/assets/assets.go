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
