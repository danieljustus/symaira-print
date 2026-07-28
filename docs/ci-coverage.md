# CI coverage: what the render and standards gates actually check

This note documents which document features and accessibility constructs are
verified in CI (`.github/workflows/ci.yml`, `render-smoke` job) and which are
not. It exists so a green CI run is not read as broader than it is.

## Render smoke + hash gate (all examples)

Every `examples/*.md` file is rendered with the pinned Typst version and
`--reproducible` (which exports `SOURCE_DATE_EPOCH=0` for byte-stable output).
The SHA-256 of each rendered PDF is then compared against the committed
baseline `testdata/example-hashes.sha256`.

- A **silent rendering regression** (template change, profile change, Typst or
  cmarker upgrade) changes a hash and fails CI.
- A **deliberate change** requires regenerating the baseline:

  ```bash
  typst --version   # must match the typst-version pinned in ci.yml
  make hashes       # renders all examples --reproducible, rewrites the file
  ```

  Review the resulting diff of `testdata/example-hashes.sha256` like code.
- Hashes are only comparable across machines because the engine ignores
  system fonts by default; they are **not** stable across Typst versions.

## veraPDF standards gate (PDF/A-2a + PDF/UA-1)

Validated documents and the constructs they exercise:

| Document | Profile | Constructs covered |
|---|---|---|
| `examples/behoerde.md` | behoerde | prose, lists, level-1 headings, `lang`/`title` metadata |
| `examples/meeting.md` | meeting | prose, lists, task lists, level-1 headings, `lang`/`title` |
| `examples/a11y-fixture.md` | behoerde | multi-level non-skipping heading hierarchy (H1→H2→H3), table with header row |

The `a11y-fixture` was added because the gate previously passed **vacuously**:
the validated fixtures contained no tables, no images and no multi-level
heading structure, so large parts of the PDF/UA-1 rule set were never
exercised. A veraPDF failure on this fixture is a real finding and must be
tracked as an issue — never silenced.

## Explicitly NOT covered

- **Images with alt text.** `symprint` renders in a temporary engine root and
  does not copy assets from the source document's directory, so a relative
  reference like `![alt](logo.png)` cannot resolve and the render fails. This
  is a known product limitation (asset-path handling), not a CI gap that can
  be closed by adding an image to a fixture today.
- **Math.** No example or fixture contains math markup; PDF/UA-1 behavior for
  equations is unverified.
- **Cross-platform font fallback.** CI always ignores system fonts, so results
  say nothing about rendering on machines where users opt into system fonts.
