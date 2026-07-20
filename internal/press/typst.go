package press

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/danieljustus/symaira-print/internal/assets"
)

// cmarkerVersion pins the @preview/cmarker package that parses CommonMark inside
// the Typst binary. Typst fetches & caches it on first compile (needs network
// once); Phase 2 vendors it into the package cache for full offline operation.
const cmarkerVersion = "0.1.9"

var (
	assetsCacheMu  sync.Mutex
	assetsCacheDir string
	assetsCacheErr error
)

// ResetAssetsCache cleans up the cached directory and resets cache state.
// It is intended for testing.
func ResetAssetsCache() {
	assetsCacheMu.Lock()
	defer assetsCacheMu.Unlock()
	if assetsCacheDir != "" {
		os.RemoveAll(assetsCacheDir)
		assetsCacheDir = ""
	}
	assetsCacheErr = nil
}

func getOrInitializeAssetsCache() (string, error) {
	assetsCacheMu.Lock()
	defer assetsCacheMu.Unlock()
	if assetsCacheDir != "" {
		return assetsCacheDir, nil
	}
	if assetsCacheErr != nil {
		return "", assetsCacheErr
	}

	dir, err := os.MkdirTemp("", "symprint-assets-cache-*")
	if err != nil {
		assetsCacheErr = err
		return "", err
	}

	tplDir := filepath.Join(dir, "templates")
	if err := assets.Materialize(tplDir); err != nil {
		os.RemoveAll(dir)
		assetsCacheErr = err
		return "", err
	}

	fontDir := filepath.Join(dir, "fonts")
	if _, err := assets.MaterializeFonts(fontDir); err != nil {
		os.RemoveAll(dir)
		assetsCacheErr = err
		return "", err
	}

	assetsCacheDir = dir
	return assetsCacheDir, nil
}

type typstJob struct {
	profile      Profile
	front        Frontmatter
	body         []byte
	outputPath   string
	pdfStandard  []string
	reproducible bool
	fontPaths    []string
	ignoreFonts  bool
	timeout      time.Duration
}

// renderTypst materializes the work dir (templates + fonts + meta.json + body.md
// + main.typ) and runs `typst compile`, capturing stderr separately so a failure
// yields the engine's real diagnostic instead of a swallowed exit code.
func renderTypst(ctx context.Context, eng EngineInfo, job typstJob) (*Result, error) {
	start := time.Now()

	work, err := os.MkdirTemp("", "symprint-*")
	if err != nil {
		return nil, &RenderError{Stage: "write", Message: "could not create work dir", Err: err}
	}
	defer os.RemoveAll(work)

	cacheDir, err := getOrInitializeAssetsCache()
	if err != nil {
		return nil, &RenderError{Stage: "write", Message: "could not initialize assets cache", Err: err}
	}

	tplDir := filepath.Join(work, "templates")
	if err := os.Symlink(filepath.Join(cacheDir, "templates"), tplDir); err != nil {
		if err := assets.Materialize(tplDir); err != nil {
			return nil, &RenderError{Stage: "write", Message: "could not materialize templates", Err: err}
		}
	}

	fontDir := filepath.Join(work, "fonts")
	if err := os.Symlink(filepath.Join(cacheDir, "fonts"), fontDir); err != nil {
		if _, err := assets.MaterializeFonts(fontDir); err != nil {
			return nil, &RenderError{Stage: "write", Message: "could not materialize fonts", Err: err}
		}
	}

	metaJSON, err := json.MarshalIndent(job.front, "", "  ")
	if err != nil {
		return nil, &RenderError{Stage: "write", Message: "could not encode metadata", Err: err}
	}
	if err := os.WriteFile(filepath.Join(work, "meta.json"), metaJSON, 0o644); err != nil {
		return nil, &RenderError{Stage: "write", Message: "could not write metadata", Err: err}
	}
	if err := os.WriteFile(filepath.Join(work, "body.md"), job.body, 0o644); err != nil {
		return nil, &RenderError{Stage: "write", Message: "could not write body", Err: err}
	}
	if err := os.WriteFile(filepath.Join(work, "main.typ"), []byte(mainTyp(job.profile.Template)), 0o644); err != nil {
		return nil, &RenderError{Stage: "write", Message: "could not write entry", Err: err}
	}

	out, err := filepath.Abs(job.outputPath)
	if err != nil {
		return nil, &RenderError{Stage: "write", Message: "invalid output path", Err: err}
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return nil, &RenderError{Stage: "write", Message: "could not create output dir", Err: err}
	}

	args := []string{"compile", "--root", work}

	// Always include the embedded font directory for machine-independent output.
	args = append(args, "--font-path", fontDir)

	// Add user-specified font paths after the embedded dir (user paths take
	// priority when font names collide).
	for _, fp := range job.fontPaths {
		args = append(args, "--font-path", fp)
	}

	// When embedded fonts are available, ignore system fonts by default for
	// deterministic output. Users can override via --ignore-system-fonts=false.
	if job.ignoreFonts || len(job.fontPaths) == 0 {
		args = append(args, "--ignore-system-fonts")
	}
	if len(job.pdfStandard) > 0 {
		args = append(args, "--pdf-standard", strings.Join(job.pdfStandard, ","))
	}
	args = append(args, filepath.Join(work, "main.typ"), out)

	timeout := job.timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, eng.Path, args...)
	cmd.Dir = work
	cmd.WaitDelay = 5 * time.Second
	if job.reproducible {
		cmd.Env = append(os.Environ(), "SOURCE_DATE_EPOCH=0")
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if cctx.Err() == context.DeadlineExceeded {
			return nil, &RenderError{
				Stage:   "compile",
				Message: fmt.Sprintf("typst compile timed out after %s", timeout),
				Hint:    "raise engine.timeout_seconds in the config",
				Err:     err,
			}
		}
		return nil, &RenderError{
			Stage:   "compile",
			Message: "typst could not render the document",
			Detail:  cleanTypstError(stderr.String()),
			Hint:    "fix the source/template, then re-run; full engine output is above",
			Err:     err,
		}
	}

	res := &Result{
		OutputPath:    out,
		Profile:       job.profile.Name,
		Engine:        "typst",
		EngineVersion: eng.Version,
		PDFStandard:   job.pdfStandard,
		Reproducible:  job.reproducible,
		DurationMS:    time.Since(start).Milliseconds(),
	}
	if fi, err := os.Stat(out); err == nil {
		res.Bytes = fi.Size()
	}
	return res, nil
}

// mainTyp is the generated entrypoint. It renders the Markdown body with cmarker
// (CommonMark parsed in-process — no pandoc) and applies the selected profile's
// `apply(meta, doc)` show rule.
func mainTyp(template string) string {
	return fmt.Sprintf(`// Generated by symprint — do not edit.
#import "@preview/cmarker:%s"
#import "/templates/%s" as profile
#let meta = json("/meta.json")
#show: profile.apply.with(meta)
#cmarker.render(read("/body.md"), scope: (
  image: (source, ..args) => image(source, ..args),
))
`, cmarkerVersion, template)
}

// cleanTypstError keeps typst's helpful, location-pointing error lines and drops
// noise, so the CLI/MCP surface a readable diagnostic rather than a raw dump.
func cleanTypstError(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) > 40 {
		lines = append(lines[:40], "  … (truncated)")
	}
	return strings.Join(lines, "\n")
}
