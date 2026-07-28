package press

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCollectAssetsCopiesLocalImages(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	png := []byte{0x89, 'P', 'N', 'G', 1, 2, 3}
	writeTestFile(t, filepath.Join(src, "logo.png"), png)
	writeTestFile(t, filepath.Join(src, "img", "photo.jpg"), png)

	body := []byte("# Doc\n\n![A logo](logo.png)\n\n![A photo](img/photo.jpg \"title\")\n")

	if err := collectAssets(src, dst, body); err != nil {
		t.Fatalf("collectAssets: %v", err)
	}

	for _, rel := range []string{"logo.png", filepath.Join("img", "photo.jpg")} {
		got, err := os.ReadFile(filepath.Join(dst, rel))
		if err != nil {
			t.Fatalf("expected %s copied into work dir: %v", rel, err)
		}
		if string(got) != string(png) {
			t.Fatalf("%s: content mismatch", rel)
		}
	}
}

func TestCollectAssetsRejectsTraversal(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	cases := []string{
		"![x](../escape.png)",
		"![x](../../etc/passwd)",
		"![x](/absolute/path.png)",
	}
	for _, c := range cases {
		err := collectAssets(src, dst, []byte(c))
		if err == nil {
			t.Fatalf("%s: expected traversal rejection, got nil error", c)
		}
		if !strings.Contains(err.Error(), "traversal") && !strings.Contains(err.Error(), "outside") && !strings.Contains(err.Error(), "absolute") {
			t.Fatalf("%s: error should name the problem, got: %v", c, err)
		}
	}
}

func TestCollectAssetsSkipsRemoteRefs(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	body := []byte("![remote](https://example.com/a.png)\n![data](data:image/png;base64,iVBOR)\n")
	if err := collectAssets(src, dst, body); err != nil {
		t.Fatalf("remote refs must not error: %v", err)
	}
	entries, err := os.ReadDir(dst)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("remote refs must not be copied, found %d entries", len(entries))
	}
}

func TestCollectAssetsMissingFileFailsClosed(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	err := collectAssets(src, dst, []byte("![missing](nope.png)"))
	if err == nil {
		t.Fatal("expected a clear error for a missing referenced file")
	}
	if !strings.Contains(err.Error(), "nope.png") {
		t.Fatalf("error should name the missing file, got: %v", err)
	}
}

func TestCollectAssetsWithoutSourceDir(t *testing.T) {
	dst := t.TempDir()

	err := collectAssets("", dst, []byte("![local](logo.png)"))
	if err == nil {
		t.Fatal("expected an error when a local ref has no source directory")
	}
	if !strings.Contains(err.Error(), "logo.png") {
		t.Fatalf("error should name the unresolvable ref, got: %v", err)
	}

	// No local refs → no error even without a source dir.
	if err := collectAssets("", dst, []byte("# plain doc\n")); err != nil {
		t.Fatalf("plain body must not error: %v", err)
	}
}

func TestCollectAssetsRejectsSymlinkEscape(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	outside := t.TempDir()

	secret := []byte("outside")
	writeTestFile(t, filepath.Join(outside, "secret.png"), secret)
	link := filepath.Join(src, "linked")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	err := collectAssets(src, dst, []byte("![x](linked/secret.png)"))
	if err == nil {
		t.Fatal("expected symlink escape outside the source dir to be rejected")
	}
}
