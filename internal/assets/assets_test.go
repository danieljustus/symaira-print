package assets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVendoredCmarkerEmbedded is the regression guard for the offline cmarker
// vendoring: the embedded package tree must never silently go empty again.
func TestVendoredCmarkerEmbedded(t *testing.T) {
	fsys := PackagesFS()

	toml, err := fsys.ReadFile("packages/preview/cmarker/0.1.9/typst.toml")
	if err != nil {
		t.Fatalf("embedded cmarker typst.toml missing: %v", err)
	}
	if len(toml) == 0 {
		t.Fatal("embedded cmarker typst.toml is empty")
	}
	if !strings.Contains(string(toml), `version = "0.1.9"`) {
		t.Errorf("typst.toml does not pin cmarker 0.1.9, got:\n%s", toml)
	}

	lib, err := fsys.ReadFile("packages/preview/cmarker/0.1.9/lib.typ")
	if err != nil {
		t.Fatalf("embedded cmarker lib.typ missing: %v", err)
	}
	if len(lib) == 0 {
		t.Fatal("embedded cmarker lib.typ is empty")
	}

	wasm, err := fsys.ReadFile("packages/preview/cmarker/0.1.9/plugin.wasm")
	if err != nil {
		t.Fatalf("embedded cmarker plugin.wasm missing: %v", err)
	}
	if len(wasm) == 0 {
		t.Fatal("embedded cmarker plugin.wasm is empty")
	}
}

// TestMaterializePackages verifies the on-disk layout mirrors the Typst
// package layout (packages/{namespace}/{name}/{version}/) so the tree can be
// passed to typst via --package-path.
func TestMaterializePackages(t *testing.T) {
	dir := t.TempDir()
	if err := MaterializePackages(dir); err != nil {
		t.Fatalf("MaterializePackages failed: %v", err)
	}

	base := filepath.Join(dir, "packages", "preview", "cmarker", "0.1.9")
	for _, name := range []string{"typst.toml", "lib.typ", "plugin.wasm"} {
		fi, err := os.Stat(filepath.Join(base, name))
		if err != nil {
			t.Errorf("materialized file %s missing: %v", name, err)
			continue
		}
		if fi.Size() == 0 {
			t.Errorf("materialized file %s is empty", name)
		}
	}
}
