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

// TestVendoredMitexEmbedded guards the complete offline mitex package,
// including the WASM plugin and its Typst support files.
func TestVendoredMitexEmbedded(t *testing.T) {
	fsys := PackagesFS()
	base := "packages/preview/mitex/0.2.7"

	toml, err := fsys.ReadFile(base + "/typst.toml")
	if err != nil {
		t.Fatalf("embedded mitex typst.toml missing: %v", err)
	}
	if !strings.Contains(string(toml), `version = "0.2.7"`) {
		t.Errorf("typst.toml does not pin mitex 0.2.7, got:\n%s", toml)
	}
	if !strings.Contains(string(toml), `license = "Apache-2.0"`) {
		t.Errorf("typst.toml does not declare Apache-2.0, got:\n%s", toml)
	}

	for _, name := range []string{"LICENSE", "lib.typ", "mitex.typ", "mitex.wasm", "specs/mod.typ", "specs/prelude.typ", "specs/latex/standard.typ"} {
		data, err := fsys.ReadFile(base + "/" + name)
		if err != nil {
			t.Fatalf("embedded mitex %s missing: %v", name, err)
		}
		if len(data) == 0 {
			t.Fatalf("embedded mitex %s is empty", name)
		}
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

	for _, pkg := range []struct {
		name  string
		files []string
	}{
		{
			name:  "cmarker",
			files: []string{"typst.toml", "lib.typ", "plugin.wasm"},
		},
		{
			name:  "mitex",
			files: []string{"typst.toml", "lib.typ", "mitex.typ", "mitex.wasm", "LICENSE", "specs/mod.typ", "specs/latex/standard.typ"},
		},
	} {
		base := filepath.Join(dir, "packages", "preview", pkg.name, map[string]string{"cmarker": "0.1.9", "mitex": "0.2.7"}[pkg.name])
		for _, name := range pkg.files {
			fi, err := os.Stat(filepath.Join(base, name))
			if err != nil {
				t.Errorf("materialized %s file %s missing: %v", pkg.name, name, err)
				continue
			}
			if fi.Size() == 0 {
				t.Errorf("materialized %s file %s is empty", pkg.name, name)
			}
		}
	}
}
