# Security Policy

## Reporting a vulnerability

Please report security issues privately via GitHub Security Advisories
("Report a vulnerability" on the repository's Security tab) rather than opening a
public issue. You will receive an acknowledgement within a few days.

## Scope notes

- **External engine.** symprint shells out to the `typst` binary resolved from
  `PATH`. It passes a generated working directory as `--root`, so the engine can
  only read files inside that directory, not the wider filesystem.
- **Untrusted Markdown.** Frontmatter is parsed strictly (unknown keys are
  rejected). The Markdown body is rendered by Typst via the `cmarker` package;
  treat rendering of untrusted input with the same caution as any document
  pipeline and keep the engine version pinned.
- **No secrets.** symprint needs no credentials. Do not commit secrets in
  example documents or tests.
