package press

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMainTyp(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     []string
	}{
		{
			name:     "brief template",
			template: "brief.typ",
			want: []string{
				"#import \"@preview/cmarker:0.1.9\"",
				"#import \"/templates/brief.typ\" as profile",
				"#let meta = json(\"/meta.json\")",
				"#show: profile.apply.with(meta)",
				"#cmarker.render(read(\"/body.md\")",
			},
		},
		{
			name:     "report template",
			template: "report.typ",
			want: []string{
				"#import \"@preview/cmarker:0.1.9\"",
				"#import \"/templates/report.typ\" as profile",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mainTyp(tt.template)
			for _, s := range tt.want {
				if !strings.Contains(got, s) {
					t.Errorf("mainTyp(%q) missing %q\nGot:\n%s", tt.template, s, got)
				}
			}
		})
	}
}

func TestCleanTypstError(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"whitespace only", "   \n  \n  ", ""},
		{"short error", "error: unknown variable", "error: unknown variable"},
		{
			"truncates at 40 lines",
			strings.Repeat("line\n", 50),
			strings.Repeat("line\n", 40) + "  … (truncated)",
		},
		{
			name:     "exactly 40 lines",
			in:       strings.Repeat("line\n", 40),
			want:     strings.Repeat("line\n", 39) + "line", // function strips trailing \n
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanTypstError(tt.in)
			if got != tt.want {
				t.Errorf("cleanTypstError() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderNoProfile(t *testing.T) {
	// Document with no profile in frontmatter and no override/default
	src := []byte("---\ntitle: Test\n---\nHello\n")
	req := Request{
		Source:    src,
		OutputPath: filepath.Join(t.TempDir(), "out.pdf"),
	}
	_, err := Render(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for no profile")
	}
	re, ok := err.(*RenderError)
	if !ok {
		t.Fatalf("want *RenderError, got %T: %v", err, err)
	}
	if re.Stage != "contract" || !strings.Contains(re.Message, "no profile") {
		t.Errorf("unexpected error: %+v", re)
	}
}

func TestRenderUnknownProfile(t *testing.T) {
	src := []byte("---\nprofile: nonexistent\n---\nHello\n")
	req := Request{
		Source:    src,
		OutputPath: filepath.Join(t.TempDir(), "out.pdf"),
	}
	_, err := Render(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for unknown profile")
	}
	re, ok := err.(*RenderError)
	if !ok {
		t.Fatalf("want *RenderError, got %T: %v", err, err)
	}
	if re.Stage != "contract" || !strings.Contains(re.Message, "unknown profile") {
		t.Errorf("unexpected error: %+v", re)
	}
}

func TestRenderProfileOverride(t *testing.T) {
	// Override to report (needs only title) while frontmatter has behoerde (needs recipient+title+lang)
	src := []byte("---\nprofile: behoerde\ntitle: Test\n---\nHello\n")
	req := Request{
		Source:         src,
		OutputPath:     filepath.Join(t.TempDir(), "out.pdf"),
		ProfileOverride: "report",
	}
	_, err := Render(context.Background(), req)
	if err == nil {
		t.Fatal("expected error (no typst)")
	}
	re, ok := err.(*RenderError)
	if !ok {
		t.Fatalf("want *RenderError, got %T: %v", err, err)
	}
	// Should fail at engine, not contract — override profile only needs title
	if re.Stage == "contract" {
		t.Errorf("should not fail at contract stage with valid profile override: %+v", re)
	}
}

func TestRenderDefaultProfile(t *testing.T) {
	// No profile in frontmatter, use DefaultProfile
	src := []byte("---\ntitle: Test\n---\nHello\n")
	req := Request{
		Source:         src,
		OutputPath:     filepath.Join(t.TempDir(), "out.pdf"),
		DefaultProfile: "report",
	}
	_, err := Render(context.Background(), req)
	if err == nil {
		t.Fatal("expected error (no typst)")
	}
	re, ok := err.(*RenderError)
	if !ok {
		t.Fatalf("want *RenderError, got %T: %v", err, err)
	}
	if re.Stage == "contract" {
		t.Errorf("should not fail at contract stage with default profile: %+v", re)
	}
}

func TestRenderValidationFails(t *testing.T) {
	// behoerde requires recipient, title, lang, date
	src := []byte("---\nprofile: behoerde\n---\nHello\n")
	req := Request{
		Source:    src,
		OutputPath: filepath.Join(t.TempDir(), "out.pdf"),
	}
	_, err := Render(context.Background(), req)
	if err == nil {
		t.Fatal("expected validation error")
	}
	ce, ok := err.(*ContractError)
	if !ok {
		t.Fatalf("want *ContractError, got %T: %v", err, err)
	}
	if len(ce.Issues) == 0 {
		t.Error("expected validation issues")
	}
}

// mockTypst creates a fake typst binary that either succeeds or fails
func mockTypst(t *testing.T, dir string, succeed bool) string {
	t.Helper()
	script := filepath.Join(dir, "typst")
	content := "#!/bin/sh\n"
	if succeed {
		content += "> \"$5\"\n" // create the output file (5th arg)
	} else {
		content += "echo 'error: test error' >&2\nexit 1\n"
	}
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestRenderTypstSuccess(t *testing.T) {
	// Create mock typst
	binDir := t.TempDir()
	mockTypst(t, binDir, true)
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", binDir+":"+origPath)
	defer os.Setenv("PATH", origPath)

	eng := EngineInfo{
		Available: true,
		Path:      filepath.Join(binDir, "typst"),
		Version:   "0.15.0",
	}
	job := typstJob{
		profile: Profile{
			Name:     "report",
			Template: "report.typ",
			Form:     "A",
		},
		front: Frontmatter{
			Profile: "report",
			Title:   "Test Document",
			Date:    "30.06.2026",
		},
		body:         []byte("# Hello\n"),
		outputPath:   filepath.Join(t.TempDir(), "out.pdf"),
		pdfStandard:  []string{"a-2a"},
		reproducible: true,
		timeout:      10 * time.Second,
	}

	result, err := renderTypst(context.Background(), eng, job)
	if err != nil {
		t.Fatalf("renderTypst failed: %v", err)
	}
	if result.OutputPath == "" {
		t.Error("expected output path")
	}
	if result.Profile != "report" {
		t.Errorf("profile = %q, want %q", result.Profile, "report")
	}
	if result.Engine != "typst" {
		t.Errorf("engine = %q, want %q", result.Engine, "typst")
	}
	if !result.Reproducible {
		t.Error("expected reproducible=true")
	}
	if len(result.PDFStandard) != 1 || result.PDFStandard[0] != "a-2a" {
		t.Errorf("pdfStandard = %v, want [a-2a]", result.PDFStandard)
	}
}

func TestRenderTypstFailure(t *testing.T) {
	// Create mock typst that fails
	binDir := t.TempDir()
	mockTypst(t, binDir, false)

	eng := EngineInfo{
		Available: true,
		Path:      filepath.Join(binDir, "typst"),
		Version:   "0.15.0",
	}
	job := typstJob{
		profile: Profile{
			Name:     "report",
			Template: "report.typ",
		},
		front:      Frontmatter{Profile: "report"},
		body:       []byte("# Hello\n"),
		outputPath: filepath.Join(t.TempDir(), "out.pdf"),
		timeout:    10 * time.Second,
	}

	_, err := renderTypst(context.Background(), eng, job)
	if err == nil {
		t.Fatal("expected error from failing typst")
	}
	re, ok := err.(*RenderError)
	if !ok {
		t.Fatalf("want *RenderError, got %T: %v", err, err)
	}
	if re.Stage != "compile" {
		t.Errorf("stage = %q, want %q", re.Stage, "compile")
	}
	if !strings.Contains(re.Detail, "test error") {
		t.Errorf("detail should contain engine error, got %q", re.Detail)
	}
}

func TestRenderTypstFontArgs(t *testing.T) {
	// Create mock typst that captures args
	binDir := t.TempDir()
	argCapture := filepath.Join(binDir, "args.txt")
	script := "#!/bin/sh\necho \"$@\" > " + argCapture + "\n> \"$5\"\n"
	if err := os.WriteFile(filepath.Join(binDir, "typst"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	eng := EngineInfo{
		Available: true,
		Path:      filepath.Join(binDir, "typst"),
		Version:   "0.15.0",
	}
	job := typstJob{
		profile: Profile{
			Name:     "report",
			Template: "report.typ",
		},
		front:        Frontmatter{Profile: "report"},
		body:         []byte("# Hello\n"),
		outputPath:   filepath.Join(t.TempDir(), "out.pdf"),
		fontPaths:    []string{"/custom/fonts", "/more/fonts"},
		ignoreFonts:  true,
		pdfStandard:  []string{"a-2a", "ua-1"},
		reproducible: true,
		timeout:      10 * time.Second,
	}

	_, err := renderTypst(context.Background(), eng, job)
	if err != nil {
		t.Fatalf("renderTypst failed: %v", err)
	}

	// Read captured args
	data, err := os.ReadFile(argCapture)
	if err != nil {
		t.Fatalf("failed to read captured args: %v", err)
	}
	args := string(data)

	if !strings.Contains(args, "--ignore-system-fonts") {
		t.Error("missing --ignore-system-fonts flag")
	}
	if !strings.Contains(args, "--font-path /custom/fonts") {
		t.Error("missing first --font-path")
	}
	if !strings.Contains(args, "--font-path /more/fonts") {
		t.Error("missing second --font-path")
	}
	if !strings.Contains(args, "--pdf-standard a-2a,ua-1") {
		t.Errorf("missing or wrong --pdf-standard, got: %s", args)
	}
}

func TestRenderTypstReproducibleEnv(t *testing.T) {
	// Create mock typst that captures environment
	binDir := t.TempDir()
	envCapture := filepath.Join(binDir, "env.txt")
	script := "#!/bin/sh\nenv > " + envCapture + "\ntouch \"$5\"\n"
	if err := os.WriteFile(filepath.Join(binDir, "typst"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	eng := EngineInfo{
		Available: true,
		Path:      filepath.Join(binDir, "typst"),
		Version:   "0.15.0",
	}
	job := typstJob{
		profile: Profile{
			Name:     "report",
			Template: "report.typ",
		},
		front:        Frontmatter{Profile: "report"},
		body:         []byte("# Hello\n"),
		outputPath:   filepath.Join(t.TempDir(), "out.pdf"),
		reproducible: true,
		timeout:      10 * time.Second,
	}

	_, err := renderTypst(context.Background(), eng, job)
	if err != nil {
		t.Fatalf("renderTypst failed: %v", err)
	}

	data, err := os.ReadFile(envCapture)
	if err != nil {
		t.Fatalf("failed to read env capture: %v", err)
	}
	if !strings.Contains(string(data), "SOURCE_DATE_EPOCH=0") {
		t.Error("missing SOURCE_DATE_EPOCH=0 in environment")
	}
}

func TestRenderTypstTimeout(t *testing.T) {
	// Create mock typst that sleeps forever
	binDir := t.TempDir()
	script := "#!/bin/sh\nsleep 100\n"
	if err := os.WriteFile(filepath.Join(binDir, "typst"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	eng := EngineInfo{
		Available: true,
		Path:      filepath.Join(binDir, "typst"),
		Version:   "0.15.0",
	}
	job := typstJob{
		profile: Profile{
			Name:     "report",
			Template: "report.typ",
		},
		front:      Frontmatter{Profile: "report"},
		body:       []byte("# Hello\n"),
		outputPath: filepath.Join(t.TempDir(), "out.pdf"),
		timeout:    100 * time.Millisecond, // very short timeout
	}

	ctx := context.Background()
	_, err := renderTypst(ctx, eng, job)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	re, ok := err.(*RenderError)
	if !ok {
		t.Fatalf("want *RenderError, got %T: %v", err, err)
	}
	if !strings.Contains(re.Message, "timed out") {
		t.Errorf("expected timeout message, got: %s", re.Message)
	}
}

func TestRenderIntegration(t *testing.T) {
	// Full integration test: Render() -> renderTypst() with mock typst
	binDir := t.TempDir()
	mockTypst(t, binDir, true)
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", binDir+":"+origPath)
	defer os.Setenv("PATH", origPath)

	src := []byte("---\nprofile: report\ntitle: Test\ndate: 30.06.2026\n---\n# Hello World\n")
	req := Request{
		Source:         src,
		OutputPath:     filepath.Join(t.TempDir(), "out.pdf"),
		Reproducible:   boolPtr(true),
		Engine: EngineConfig{
			TypstBin: filepath.Join(binDir, "typst"),
			Timeout:  10 * time.Second,
		},
	}

	result, err := Render(context.Background(), req)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Profile != "report" {
		t.Errorf("profile = %q, want %q", result.Profile, "report")
	}
	if result.Engine != "typst" {
		t.Errorf("engine = %q, want %q", result.Engine, "typst")
	}
}

func boolPtr(b bool) *bool { return &b }

// Ensure we import exec for potential future use
var _ = exec.Command
