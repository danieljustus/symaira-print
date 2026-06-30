<!-- Keep PRs focused. CI runs vet, gitleaks, staticcheck, govulncheck, build,
     test, and a render smoke test (typst). Keep them green. -->

## What & why

<!-- Short description of the change and the motivation. -->

## Checklist

- [ ] `make build && make test` pass locally
- [ ] If templates changed: rendered `examples/*.md` with `typst` and checked the PDFs
- [ ] If the frontmatter contract changed: updated `docs/markdown-contract.md`
- [ ] No secrets, credentials, or personal data in code/tests/fixtures
