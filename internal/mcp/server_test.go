package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/danieljustus/symaira-print/internal/config"
)

func TestToJSON(t *testing.T) {
	s, err := toJSON(map[string]string{"a": "b"})
	if err != nil {
		t.Fatalf("toJSON() error = %v", err)
	}
	if !strings.Contains(s, `"a": "b"`) {
		t.Errorf("toJSON() = %q, want it to contain %q", s, `"a": "b"`)
	}

	// Functions are not JSON-marshalable; toJSON must surface the error.
	if _, err := toJSON(func() {}); err == nil {
		t.Error("toJSON(func) error = nil, want non-nil")
	}
}

func TestEngineFromConfig(t *testing.T) {
	cfg := &config.Config{
		Engine: config.Engine{
			Typst:             "/usr/bin/typst",
			FontPaths:         []string{"/fonts"},
			IgnoreSystemFonts: true,
			TimeoutSeconds:    30,
		},
	}
	ec := engineFromConfig(cfg)
	if ec.TypstBin != "/usr/bin/typst" {
		t.Errorf("TypstBin = %q, want %q", ec.TypstBin, "/usr/bin/typst")
	}
	if len(ec.FontPaths) != 1 || ec.FontPaths[0] != "/fonts" {
		t.Errorf("FontPaths = %v, want [/fonts]", ec.FontPaths)
	}
	if !ec.IgnoreSystemFonts {
		t.Error("IgnoreSystemFonts = false, want true")
	}
	if ec.Timeout != cfg.Engine.Timeout() {
		t.Errorf("Timeout = %v, want %v", ec.Timeout, cfg.Engine.Timeout())
	}
}

// rpcLine encodes a JSON-RPC request as a single newline-delimited line, the
// "line mode" transport ServeIO also accepts (alongside Content-Length framing).
func rpcLine(t *testing.T, id int, method string, params any) string {
	t.Helper()
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}
	if params != nil {
		req["params"] = params
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	return string(b) + "\n"
}

// readResponses reads newline-delimited JSON-RPC responses from out.
func readResponses(t *testing.T, out *bytes.Buffer) []map[string]any {
	t.Helper()
	var responses []map[string]any
	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var resp map[string]any
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			t.Fatalf("unmarshal response %q: %v", line, err)
		}
		responses = append(responses, resp)
	}
	return responses
}

func TestBuildServer_ToolRegistration(t *testing.T) {
	cfg := config.Default()
	s := buildServer(cfg)

	in := strings.NewReader(
		rpcLine(t, 1, "initialize", map[string]any{}) +
			rpcLine(t, 2, "tools/list", map[string]any{}),
	)
	var out bytes.Buffer

	if err := s.ServeIO(context.Background(), in, &out); err != nil {
		t.Fatalf("ServeIO() error = %v", err)
	}

	responses := readResponses(t, &out)
	if len(responses) != 2 {
		t.Fatalf("got %d responses, want 2", len(responses))
	}

	initResult, ok := responses[0]["result"].(map[string]any)
	if !ok {
		t.Fatalf("initialize response missing result: %v", responses[0])
	}
	serverInfo, ok := initResult["serverInfo"].(map[string]any)
	if !ok || serverInfo["name"] != "symprint" {
		t.Errorf("serverInfo = %v, want name=symprint", initResult["serverInfo"])
	}
	if initResult["instructions"] == "" || initResult["instructions"] == nil {
		t.Error("initialize result missing non-empty instructions")
	}

	listResult, ok := responses[1]["result"].(map[string]any)
	if !ok {
		t.Fatalf("tools/list response missing result: %v", responses[1])
	}
	tools, ok := listResult["tools"].([]any)
	if !ok {
		t.Fatalf("tools/list result.tools is not an array: %v", listResult)
	}

	wantTools := map[string]bool{
		"list_profiles":     false,
		"doctor":            false,
		"validate_document": false,
		"render_pdf":        false,
	}
	for _, raw := range tools {
		tool, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		name, _ := tool["name"].(string)
		if _, known := wantTools[name]; known {
			wantTools[name] = true
		}
	}
	for name, found := range wantTools {
		if !found {
			t.Errorf("tool %q was not registered", name)
		}
	}
}

func TestBuildServer_ListProfiles(t *testing.T) {
	cfg := config.Default()
	s := buildServer(cfg)

	in := strings.NewReader(rpcLine(t, 1, "tools/call", map[string]any{
		"name":      "list_profiles",
		"arguments": map[string]any{},
	}))
	var out bytes.Buffer

	if err := s.ServeIO(context.Background(), in, &out); err != nil {
		t.Fatalf("ServeIO() error = %v", err)
	}

	responses := readResponses(t, &out)
	if len(responses) != 1 {
		t.Fatalf("got %d responses, want 1", len(responses))
	}
	result, ok := responses[0]["result"].(map[string]any)
	if !ok {
		t.Fatalf("list_profiles response missing result: %v", responses[0])
	}
	if isErr, _ := result["isError"].(bool); isErr {
		t.Fatalf("list_profiles returned isError=true: %v", result)
	}
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("list_profiles result.content empty or missing: %v", result)
	}
}

func TestBuildServer_Doctor(t *testing.T) {
	cfg := config.Default()
	s := buildServer(cfg)

	in := strings.NewReader(rpcLine(t, 1, "tools/call", map[string]any{
		"name":      "doctor",
		"arguments": map[string]any{},
	}))
	var out bytes.Buffer

	if err := s.ServeIO(context.Background(), in, &out); err != nil {
		t.Fatalf("ServeIO() error = %v", err)
	}

	responses := readResponses(t, &out)
	if len(responses) != 1 {
		t.Fatalf("got %d responses, want 1", len(responses))
	}
	result, ok := responses[0]["result"].(map[string]any)
	if !ok {
		t.Fatalf("doctor response missing result: %v", responses[0])
	}
	if isErr, _ := result["isError"].(bool); isErr {
		t.Fatalf("doctor returned isError=true: %v", result)
	}
}

func TestBuildServer_ValidateDocument(t *testing.T) {
	cfg := config.Default()
	s := buildServer(cfg)

	markdown := "---\nprofile: report\ntitle: Test\n---\n\nHello world.\n"
	in := strings.NewReader(rpcLine(t, 1, "tools/call", map[string]any{
		"name": "validate_document",
		"arguments": map[string]any{
			"markdown": markdown,
		},
	}))
	var out bytes.Buffer

	if err := s.ServeIO(context.Background(), in, &out); err != nil {
		t.Fatalf("ServeIO() error = %v", err)
	}

	responses := readResponses(t, &out)
	if len(responses) != 1 {
		t.Fatalf("got %d responses, want 1", len(responses))
	}
	result, ok := responses[0]["result"].(map[string]any)
	if !ok {
		t.Fatalf("validate_document response missing result: %v", responses[0])
	}
	if isErr, _ := result["isError"].(bool); isErr {
		t.Fatalf("validate_document returned isError=true: %v", result)
	}
}

func TestBuildServer_ValidateDocument_InvalidArguments(t *testing.T) {
	cfg := config.Default()
	s := buildServer(cfg)

	// "arguments" is a JSON array, which cannot unmarshal into the handler's
	// struct{Markdown, Profile} args — exercises the unmarshal-error branch.
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"validate_document","arguments":["not","an","object"]}}` + "\n")
	var out bytes.Buffer

	if err := s.ServeIO(context.Background(), in, &out); err != nil {
		t.Fatalf("ServeIO() error = %v", err)
	}

	responses := readResponses(t, &out)
	if len(responses) != 1 {
		t.Fatalf("got %d responses, want 1", len(responses))
	}
	result, ok := responses[0]["result"].(map[string]any)
	if !ok {
		t.Fatalf("validate_document response missing result: %v", responses[0])
	}
	if isErr, _ := result["isError"].(bool); !isErr {
		t.Fatalf("validate_document with malformed arguments: isError = false, want true: %v", result)
	}
}

func TestBuildServer_ValidateDocument_ParseError(t *testing.T) {
	cfg := config.Default()
	s := buildServer(cfg)

	// Closed frontmatter block with an unknown key: KnownFields(true) rejects it.
	markdown := "---\nnot_a_real_field: oops\n---\n\nBody.\n"
	in := strings.NewReader(rpcLine(t, 1, "tools/call", map[string]any{
		"name": "validate_document",
		"arguments": map[string]any{
			"markdown": markdown,
		},
	}))
	var out bytes.Buffer

	if err := s.ServeIO(context.Background(), in, &out); err != nil {
		t.Fatalf("ServeIO() error = %v", err)
	}

	responses := readResponses(t, &out)
	if len(responses) != 1 {
		t.Fatalf("got %d responses, want 1", len(responses))
	}
	result, ok := responses[0]["result"].(map[string]any)
	if !ok {
		t.Fatalf("validate_document response missing result: %v", responses[0])
	}
	if isErr, _ := result["isError"].(bool); !isErr {
		t.Fatalf("validate_document with malformed frontmatter: isError = false, want true: %v", result)
	}
}

func TestBuildServer_ValidateDocument_FallsBackToConfigDefaultProfile(t *testing.T) {
	cfg := &config.Config{Defaults: config.Defaults{Profile: "report"}}
	s := buildServer(cfg)

	// No "profile" argument and no profile in frontmatter: falls back to cfg.Defaults.Profile.
	markdown := "---\ntitle: Test\n---\n\nHello.\n"
	in := strings.NewReader(rpcLine(t, 1, "tools/call", map[string]any{
		"name": "validate_document",
		"arguments": map[string]any{
			"markdown": markdown,
		},
	}))
	var out bytes.Buffer

	if err := s.ServeIO(context.Background(), in, &out); err != nil {
		t.Fatalf("ServeIO() error = %v", err)
	}

	responses := readResponses(t, &out)
	if len(responses) != 1 {
		t.Fatalf("got %d responses, want 1", len(responses))
	}
	result, ok := responses[0]["result"].(map[string]any)
	if !ok {
		t.Fatalf("validate_document response missing result: %v", responses[0])
	}
	if isErr, _ := result["isError"].(bool); isErr {
		t.Fatalf("validate_document returned isError=true: %v", result)
	}
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("validate_document result.content empty or missing: %v", result)
	}
	first, ok := content[0].(map[string]any)
	if !ok {
		t.Fatalf("content[0] is not an object: %v", content[0])
	}
	text, _ := first["text"].(string)
	if !strings.Contains(text, `"profile": "report"`) {
		t.Errorf("validate_document text = %q, want it to report profile %q (the config default)", text, "report")
	}
}

func TestBuildServer_ValidateDocument_RequiredFieldError(t *testing.T) {
	cfg := config.Default()
	s := buildServer(cfg)

	// "report" requires "title", which is omitted, so Validate() returns a
	// severity "error" issue and the handler must flip ok=false.
	markdown := "---\nprofile: report\n---\n\nHello.\n"
	in := strings.NewReader(rpcLine(t, 1, "tools/call", map[string]any{
		"name": "validate_document",
		"arguments": map[string]any{
			"markdown": markdown,
		},
	}))
	var out bytes.Buffer

	if err := s.ServeIO(context.Background(), in, &out); err != nil {
		t.Fatalf("ServeIO() error = %v", err)
	}

	responses := readResponses(t, &out)
	if len(responses) != 1 {
		t.Fatalf("got %d responses, want 1", len(responses))
	}
	result, ok := responses[0]["result"].(map[string]any)
	if !ok {
		t.Fatalf("validate_document response missing result: %v", responses[0])
	}
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("validate_document result.content empty or missing: %v", result)
	}
	first, ok := content[0].(map[string]any)
	if !ok {
		t.Fatalf("content[0] is not an object: %v", content[0])
	}
	text, _ := first["text"].(string)
	if !strings.Contains(text, `"ok": false`) {
		t.Errorf("validate_document text = %q, want it to report ok=false for a missing required field", text)
	}
}

func TestBuildServer_ValidateDocument_UnknownProfile(t *testing.T) {
	cfg := config.Default()
	s := buildServer(cfg)

	markdown := "---\nprofile: does-not-exist\ntitle: Test\n---\n\nHello world.\n"
	in := strings.NewReader(rpcLine(t, 1, "tools/call", map[string]any{
		"name": "validate_document",
		"arguments": map[string]any{
			"markdown": markdown,
		},
	}))
	var out bytes.Buffer

	if err := s.ServeIO(context.Background(), in, &out); err != nil {
		t.Fatalf("ServeIO() error = %v", err)
	}

	responses := readResponses(t, &out)
	if len(responses) != 1 {
		t.Fatalf("got %d responses, want 1", len(responses))
	}
	result, ok := responses[0]["result"].(map[string]any)
	if !ok {
		t.Fatalf("validate_document response missing result: %v", responses[0])
	}
	if isErr, _ := result["isError"].(bool); !isErr {
		t.Fatalf("validate_document with unknown profile: isError = false, want true: %v", result)
	}
}

func TestBuildServer_RenderPDF_MissingOutputPath(t *testing.T) {
	cfg := config.Default()
	s := buildServer(cfg)

	in := strings.NewReader(rpcLine(t, 1, "tools/call", map[string]any{
		"name": "render_pdf",
		"arguments": map[string]any{
			"markdown": "---\nprofile: report\ntitle: Test\n---\n\nHello.\n",
		},
	}))
	var out bytes.Buffer

	if err := s.ServeIO(context.Background(), in, &out); err != nil {
		t.Fatalf("ServeIO() error = %v", err)
	}

	responses := readResponses(t, &out)
	if len(responses) != 1 {
		t.Fatalf("got %d responses, want 1", len(responses))
	}
	result, ok := responses[0]["result"].(map[string]any)
	if !ok {
		t.Fatalf("render_pdf response missing result: %v", responses[0])
	}
	if isErr, _ := result["isError"].(bool); !isErr {
		t.Fatalf("render_pdf without output_path: isError = false, want true: %v", result)
	}
}

func TestBuildServer_RenderPDF_InvalidArguments(t *testing.T) {
	cfg := config.Default()
	s := buildServer(cfg)

	// "arguments" is a JSON array, which cannot unmarshal into the handler's
	// args struct — exercises the unmarshal-error branch.
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"render_pdf","arguments":["not","an","object"]}}` + "\n")
	var out bytes.Buffer

	if err := s.ServeIO(context.Background(), in, &out); err != nil {
		t.Fatalf("ServeIO() error = %v", err)
	}

	responses := readResponses(t, &out)
	if len(responses) != 1 {
		t.Fatalf("got %d responses, want 1", len(responses))
	}
	result, ok := responses[0]["result"].(map[string]any)
	if !ok {
		t.Fatalf("render_pdf response missing result: %v", responses[0])
	}
	if isErr, _ := result["isError"].(bool); !isErr {
		t.Fatalf("render_pdf with malformed arguments: isError = false, want true: %v", result)
	}
}

func TestBuildServer_RenderPDF_BuildsRequestAndInvokesRender(t *testing.T) {
	cfg := config.Default()
	s := buildServer(cfg)

	outputPath := t.TempDir() + "/out.pdf"
	in := strings.NewReader(rpcLine(t, 1, "tools/call", map[string]any{
		"name": "render_pdf",
		"arguments": map[string]any{
			"markdown":     "---\nprofile: report\ntitle: Test\n---\n\nHello.\n",
			"output_path":  outputPath,
			"profile":      "report",
			"pdf_standard": []string{"a-2a"},
			"reproducible": true,
		},
	}))
	var out bytes.Buffer

	if err := s.ServeIO(context.Background(), in, &out); err != nil {
		t.Fatalf("ServeIO() error = %v", err)
	}

	responses := readResponses(t, &out)
	if len(responses) != 1 {
		t.Fatalf("got %d responses, want 1", len(responses))
	}
	// The result depends on whether a typst binary is available in this
	// environment; either outcome exercises building the press.Request and
	// calling press.Render, which is all this test needs to cover.
	if _, ok := responses[0]["result"]; !ok {
		t.Fatalf("render_pdf response missing result: %v", responses[0])
	}
}

func TestStartServer_ReturnsOnImmediateEOF(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}

	cfg := config.Default()
	if err := StartServer(context.Background(), cfg); err != nil {
		t.Fatalf("StartServer() error = %v", err)
	}
}

func TestStartServer_WrapsServeError(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	// A header-shaped line (contains ":", doesn't start with "{") with no
	// Content-Length field is a framing protocol violation that ServeIO
	// surfaces as a non-EOF read error, which StartServer must wrap.
	if _, err := w.WriteString("X-Custom: 1\n\n"); err != nil {
		t.Fatalf("write malformed frame: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}

	err = StartServer(context.Background(), config.Default())
	if err == nil {
		t.Fatal("StartServer() error = nil, want a wrapped read error")
	}
	if !strings.Contains(err.Error(), "mcp server:") {
		t.Errorf("StartServer() error = %q, want it prefixed with %q", err.Error(), "mcp server:")
	}
}
