# Agent Instructions — symaira-print

Markdown → beautiful PDF via named use-case profiles. Binary: `symprint`.
Go 1.26.4, `CGO_ENABLED=0`, Apache-2.0. Part of the Symaira ecosystem.

## Build & test

```bash
make build        # → ./symprint  (CGO_ENABLED=0)
make test         # go test ./...
make test-race    # with race detector
make lint         # go fmt + go vet
make examples     # render examples/*.md (needs typst on PATH)
```

CI additionally runs gitleaks, staticcheck, govulncheck, and a **render smoke
test** that compiles `examples/*.md` with a real Typst. Keep them green.

## Layout

- `cmd/symprint/` — Cobra entrypoint. `version` injected via ldflags. One file
  per command (`render`, `profiles`, `validate`, `doctor`, `config`, `mcp`).
- `internal/config/` — corekit `configkit` loader (`SYMPRINT_*` env, TOML).
- `internal/press/` — the engine-agnostic core. **Self-contained**; no deps on
  cmd/ or mcp/. Owns the frontmatter contract, profile registry, validation,
  and the Typst shell-out.
- `internal/assets/` — `go:embed` Typst templates, materialized to a temp dir
  per render so the external engine can read them.
- `internal/mcp/` — corekit `mcpserver` stdio server.

## Conventions (match the rest of the ecosystem)

- **CGO-free.** The engine (Typst) is reached over `PATH` (`exec.LookPath`),
  never linked. A missing engine is a graceful error with an install hint, not
  a crash (standalone-first).
- **Errors are mapped, not dumped.** `internal/press` returns typed
  `*ContractError` / `*RenderError` (with a `Stage` and `Hint`); `cmd` maps them
  to corekit `exitcodes`. Never surface a raw Typst log unfiltered.
- **Contract is strict.** Frontmatter parsing uses `KnownFields(true)` — unknown
  keys are an error. Add a field to `internal/press/frontmatter.go` *and*
  `docs/markdown-contract.md` together.
- **Semantics vs presentation.** Frontmatter carries meaning only; all layout
  lives in `internal/assets/templates/*.typ`. Don't add layout knobs to docs.
- **Profiles declare guarantees.** Set `PDFStandard` on a profile so
  `Capability()` and accessibility validation stay truthful (e.g. a `ua-1`
  profile validates that `lang` and `title` are present).
- **Zero stdio pollution (MCP).** stdout carries only JSON-RPC; all logs and the
  engine's stderr stay off stdout.

## Engine notes

- Typst is invoked as `typst compile --root <work> [--pdf-standard …] main.typ out.pdf`.
- Markdown → Typst is done **in-engine** by the `@preview/cmarker` package (no
  pandoc). Typst fetches it into its package cache on first compile (network
  once); vendoring it is a Phase 2 task.
- PDF/A + PDF/UA come from `--pdf-standard a-2a,ua-1`. Typst validates PDF/UA at
  compile time and *fails closed* (e.g. headings must start at level 1) — this
  is intended; fix the document, don't loosen the profile.
- Reproducible output: `--reproducible` exports `SOURCE_DATE_EPOCH=0`.

## Don't

- Don't add Cloud/Pro/billing code here (public Apache-2.0 core).
- Don't bundle the Typst binary into the repo; depend on it via Homebrew /
  install hint.
- Don't commit secrets or real personal data in `examples/` or tests.
