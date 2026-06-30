# The symprint Markdown contract

A `symprint` document is **Markdown with a leading YAML frontmatter block**:

```markdown
---
profile: report
title: "My document"
---
# Body in Markdown
```

The frontmatter carries **semantic metadata only** — never layout. All
presentation lives in the selected [profile](profiles.md). The contract is
**strict**: an unknown key is an error (so typos fail loudly), and the parser
reports the exact line.

## Precedence

When the same setting can come from several places, higher wins:

```
built-in defaults  <  config (~/.config/symprint)  <  profile  <  frontmatter  <  CLI / MCP flags
```

## Fields

| Key | Type | Profiles | Notes |
|---|---|---|---|
| `profile` | string | all | Selects the profile. May be overridden by `--profile`. |
| `lang` | string | all | Document language (e.g. `de`). **Required** for PDF/UA profiles. |
| `title` | string | all | Document title → PDF metadata. **Required** for `report`, `behoerde`. |
| `subtitle` | string | report | Shown under the title on the cover. |
| `author` | string \| list | report | Scalar or list: `author: Jane` or `author: [Jane, John]`. |
| `date` | string | all | `TT.MM.JJJJ` or `JJJJ-MM-TT` (DIN 5008; leading zeros validated). |
| `keywords` | list | report | PDF metadata keywords. |
| `form` | `A` \| `B` | brief, behoerde | DIN 5008 form. Defaults from the profile. |
| `sender` | `{name, address[]}` | brief, behoerde, rechnung | Return address (Rücksendeangabe). |
| `recipient` | `{name, address[]}` | brief, behoerde, rechnung | Anschriftfeld. **Required** for letters/invoice. |
| `infoblock` | map | brief, behoerde | DIN 5008 Infoblock; key/value lines, ends with Datum. |
| `betreff` | string | brief, behoerde | Subject line (bold). |
| `titlepage` | bool | report | Toggle the cover page (default true). |
| `toc` | bool | report | Toggle the table of contents (default true). |
| `toc-depth` | int | report | TOC heading depth (default 3). |
| `header-left` / `header-right` | string | report | Running-header overrides. |
| `footer-left` / `footer-right` | string | report | Running-footer overrides. |
| `data` | map | rechnung | Structured payload (invoice number, items, currency). |
| `pdf.standard` | list | all | typst `--pdf-standard` ids, e.g. `[a-2a, ua-1]`. |
| `pdf.reproducible` | bool | all | Force byte-stable output for this document. |

Fields not relevant to the selected profile are accepted but ignored — only
*unknown* keys (typos) are rejected.

## Examples

A minimal report:

```yaml
---
profile: report
lang: de
title: "Quartalsbericht Q2 2026"
author: ["Daniel Justus"]
toc: true
---
```

An authority letter (accessible + archivable):

```yaml
---
profile: behoerde
lang: de
title: "Anhörung nach § 28 VwVfG"
date: 2026-06-30
recipient: { name: "Frau Erika Mustermann", address: ["Musterweg 12", "54321 Beispielstadt"] }
infoblock: { "Unser Zeichen": "BAU-2026-04711" }
betreff: "Anhörung im Verfahren BAU-2026-04711"
pdf: { standard: [a-2a, ua-1] }
---
```

See [`examples/`](../examples/) for one complete document per profile. Validate
any document without rendering:

```bash
symprint validate mydoc.md
```
