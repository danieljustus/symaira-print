package press

import (
	"context"
	"os/exec"
	"strings"
)

// Capability describes what an engine/profile combination guarantees.
type Capability struct {
	Markdown     bool `json:"markdown"`
	PDFA         bool `json:"pdf_a"`
	PDFUA        bool `json:"pdf_ua"`
	DINWindow    bool `json:"din_window"`
	Reproducible bool `json:"reproducible"`
}

// EngineInfo is the result of probing for a typesetting engine on the host.
type EngineInfo struct {
	Name      string `json:"name"`
	Path      string `json:"path,omitempty"`
	Version   string `json:"version,omitempty"`
	Available bool   `json:"available"`
	Hint      string `json:"hint,omitempty"`
}

// TypstInstallHint is the actionable message shown when typst is not on PATH.
const TypstInstallHint = "typst not found on PATH. Install it with one of:\n" +
	"  • macOS:   brew install typst\n" +
	"  • Windows: winget install --id Typst.Typst\n" +
	"  • Cargo:   cargo install typst-cli\n" +
	"  • Binary:  https://github.com/typst/typst/releases (put it on PATH)"

// DetectTypst probes for the typst binary. binOverride may name a specific
// binary or path; empty means resolve "typst" from PATH. It never errors — an
// absent engine is reported via Available=false plus an install Hint, which is
// the standalone-first contract (graceful, not fatal).
func DetectTypst(ctx context.Context, binOverride string) EngineInfo {
	name := strings.TrimSpace(binOverride)
	if name == "" {
		name = "typst"
	}
	info := EngineInfo{Name: "typst"}
	path, err := exec.LookPath(name)
	if err != nil {
		info.Hint = TypstInstallHint
		return info
	}
	info.Path = path
	info.Available = true
	if v := typstVersion(ctx, path); v != "" {
		info.Version = v
	}
	return info
}

// typstVersion runs `typst --version` and extracts the semver. Best-effort.
func typstVersion(ctx context.Context, path string) string {
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		return ""
	}
	// Output looks like: "typst 0.15.0 (abcd1234)"
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) >= 2 {
		return fields[1]
	}
	return ""
}
