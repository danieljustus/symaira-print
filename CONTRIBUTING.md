# Contributing to symaira-print

Thanks for helping improve `symprint`. It is part of the Symaira ecosystem and
follows the shared conventions in the root `AGENTS.md`.

## Build & test

```bash
make build        # → ./symprint  (CGO_ENABLED=0)
make test         # go test ./...
make test-race    # with race detector
make lint         # go fmt + go vet
```

To exercise the real rendering path you need Typst on `PATH`:

```bash
brew install typst         # or see `symprint doctor`
make examples              # renders examples/*.md into dist/
```

## Conventions

- **CGO-free.** The build must stay `CGO_ENABLED=0`. The typesetting engine is
  reached over `PATH` (`exec.LookPath`), never linked.
- **Engine errors are mapped, not dumped.** Return typed `*press.RenderError` /
  `*press.ContractError`; the CLI/MCP layer turns them into exit codes and
  plain-language hints.
- **Contract is strict.** Unknown frontmatter keys are an error. Add new fields
  to `internal/press/frontmatter.go` *and* document them in
  `docs/markdown-contract.md`.
- **Templates are layout-only.** Presentation lives in `internal/assets/templates/*.typ`;
  documents never carry layout knobs. After changing a template, render every
  `examples/*.md` with `typst` and eyeball the PDFs.
- **Profiles declare their guarantees.** If a profile emits PDF/A or PDF/UA, set
  `PDFStandard` so `Capability()` and validation stay truthful.

## Commit messages

Conventional-commit prefixes (`feat:`, `fix:`, `docs:`, `test:`, `chore:`) keep
the changelog clean.
