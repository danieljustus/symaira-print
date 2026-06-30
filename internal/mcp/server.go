// Package mcp exposes symprint over the Model Context Protocol so AI agents can
// render PDFs, discover profiles, and validate documents. It is a thin wrapper
// around the press pipeline using the shared corekit MCP server.
//
// Zero stdio pollution: the stdio transport carries only JSON-RPC 2.0 messages;
// every log, warning, and the engine's own stderr stay off os.Stdout.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/danieljustus/symaira-corekit/mcpserver"
	"github.com/danieljustus/symaira-print/internal/config"
	"github.com/danieljustus/symaira-print/internal/press"
)

// ServerVersion is injected by main so the MCP handshake reports the binary version.
var ServerVersion = "dev"

func toJSON(v any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// StartServer runs the MCP stdio server backed by the given config.
func StartServer(ctx context.Context, cfg *config.Config) error {
	s := buildServer(cfg)
	if err := s.ServeStdio(ctx); err != nil {
		return fmt.Errorf("mcp server: %w", err)
	}
	return nil
}

func buildServer(cfg *config.Config) *mcpserver.Server {
	s := mcpserver.New("symprint", ServerVersion)
	s.SetInstructions("Render Markdown into beautiful PDFs via named profiles " +
		"(brief, behoerde, report, rechnung). Call list_profiles to choose a profile, " +
		"validate_document to check frontmatter before rendering, and render_pdf to " +
		"produce the file. The 'behoerde' profile emits accessible/archival PDF/A+UA. " +
		"Markdown carries YAML frontmatter; unknown keys are rejected.")

	s.RegisterTool(&mcpserver.Tool{
		Name:        "list_profiles",
		Description: "List the built-in use-case profiles and their output guarantees (PDF/A, PDF/UA, DIN window).",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		Handler: func(ctx context.Context, _ json.RawMessage) (any, error) {
			type entry struct {
				press.Profile
				Capability press.Capability `json:"capability"`
			}
			profs := press.All()
			out := make([]entry, 0, len(profs))
			for _, p := range profs {
				out = append(out, entry{p, p.Capability()})
			}
			return toJSON(out)
		},
	})

	s.RegisterTool(&mcpserver.Tool{
		Name:        "doctor",
		Description: "Report whether the Typst rendering engine is available, with an install hint if not.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		Handler: func(ctx context.Context, _ json.RawMessage) (any, error) {
			return toJSON(press.DetectTypst(ctx, cfg.Engine.Typst))
		},
	})

	s.RegisterTool(&mcpserver.Tool{
		Name: "validate_document",
		Description: "Parse and validate a Markdown document's frontmatter against a profile. " +
			"Returns issues without rendering. Provide 'markdown'; 'profile' is optional (falls back to frontmatter).",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"markdown":{"type":"string"},"profile":{"type":"string"}},"required":["markdown"]}`),
		Handler: func(ctx context.Context, in json.RawMessage) (any, error) {
			var args struct {
				Markdown string `json:"markdown"`
				Profile  string `json:"profile"`
			}
			if err := json.Unmarshal(in, &args); err != nil {
				return nil, err
			}
			doc, err := press.Parse([]byte(args.Markdown))
			if err != nil {
				return nil, err
			}
			name := args.Profile
			if name == "" {
				name = doc.Front.Profile
			}
			if name == "" {
				name = cfg.Defaults.Profile
			}
			p, ok := press.Lookup(name)
			if !ok {
				return nil, fmt.Errorf("unknown profile %q", name)
			}
			issues := doc.Validate(p)
			ok = true
			for _, is := range issues {
				if is.Severity == "error" {
					ok = false
				}
			}
			return toJSON(map[string]any{"profile": p.Name, "ok": ok, "issues": issues})
		},
	})

	s.RegisterTool(&mcpserver.Tool{
		Name: "render_pdf",
		Description: "Render a Markdown document (with YAML frontmatter) to a PDF file. " +
			"Provide 'markdown' and 'output_path'. Optional: 'profile', 'pdf_standard' (e.g. [\"a-2a\",\"ua-1\"]), " +
			"'reproducible'. Returns the output path and render metadata, or a structured error.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{` +
			`"markdown":{"type":"string","description":"Markdown source including YAML frontmatter"},` +
			`"output_path":{"type":"string","description":"Absolute path to write the PDF"},` +
			`"profile":{"type":"string","description":"Profile name (overrides frontmatter)"},` +
			`"pdf_standard":{"type":"array","items":{"type":"string"},"description":"typst --pdf-standard ids"},` +
			`"reproducible":{"type":"boolean"}},"required":["markdown","output_path"]}`),
		Handler: func(ctx context.Context, in json.RawMessage) (any, error) {
			var args struct {
				Markdown     string   `json:"markdown"`
				OutputPath   string   `json:"output_path"`
				Profile      string   `json:"profile"`
				PDFStandard  []string `json:"pdf_standard"`
				Reproducible *bool    `json:"reproducible"`
			}
			if err := json.Unmarshal(in, &args); err != nil {
				return nil, err
			}
			if args.OutputPath == "" {
				return nil, fmt.Errorf("output_path is required")
			}
			req := press.Request{
				Source:           []byte(args.Markdown),
				SourceName:       "mcp",
				OutputPath:       args.OutputPath,
				ProfileOverride:  args.Profile,
				DefaultProfile:   cfg.Defaults.Profile,
				StandardOverride: args.PDFStandard,
				Reproducible:     args.Reproducible,
				Engine:           engineFromConfig(cfg),
			}
			res, err := press.Render(ctx, req)
			if err != nil {
				return nil, err
			}
			return toJSON(res)
		},
	})

	return s
}

func engineFromConfig(cfg *config.Config) press.EngineConfig {
	return press.EngineConfig{
		TypstBin:          cfg.Engine.Typst,
		FontPaths:         cfg.Engine.FontPaths,
		IgnoreSystemFonts: cfg.Engine.IgnoreSystemFonts,
		Timeout:           cfg.Engine.Timeout(),
	}
}
