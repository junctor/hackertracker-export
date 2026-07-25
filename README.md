# hackertracker-export

Go CLI for exporting HackerTracker Firestore data into raw inspection files or the normalized static JSON artifacts consumed by [info.defcon.org](https://info.defcon.org) and [`junctor/hackertracker-info`](https://github.com/junctor/hackertracker-info).

The CLI has two primary workflows:

- `fetch` writes Firestore-shaped JSON for inspection, fixtures, audits, and debugging.
- `info` transforms HackerTracker data into the static artifact tree served by the web app.

## Install

Run from source:

```sh
go run ./cmd/hackertracker --help
```

Or install a local binary:

```sh
go install ./cmd/hackertracker
hackertracker --help
```

Firestore-backed commands require network access and permission to read the HackerTracker Firestore project.

## CLI Usage

```sh
hackertracker conferences
hackertracker fetch <target> --conference <code> [--stdout] [--out <dir>]
hackertracker info [--out <dir>] --conference <code> [--conference <code>]
```

The same commands can be run through `go run`:

```sh
go run ./cmd/hackertracker conferences
go run ./cmd/hackertracker fetch content --conference DEFCON34 --stdout
go run ./cmd/hackertracker info --conference DEFCON34
```

`info` also accepts additional conference codes as positional arguments after the flags.

## Fetch Raw Data

Fetch targets are:

```text
conference
articles
content
documents
locations
organizations
speakers
tagtypes
all
```

Examples:

```sh
go run ./cmd/hackertracker fetch conference --conference DEFCON34 --stdout
go run ./cmd/hackertracker fetch content --conference DEFCON34 --stdout
go run ./cmd/hackertracker fetch speakers --conference DEFCON34 --stdout
go run ./cmd/hackertracker fetch all --conference DEFCON34
```

By default, raw files are written to:

```text
out/ht/<lowercase-conference>/raw/
  conference.json
  articles.json
  content.json
  documents.json
  locations.json
  organizations.json
  speakers.json
  tagtypes.json
```

Use `--out` to choose the exact raw output directory. The command writes files directly into that directory:

```sh
go run ./cmd/hackertracker fetch all --conference DEFCON34 --out ./tmp/defcon34/raw
```

Use `--stdout` to print JSON instead of writing files. For `fetch all`, stdout contains the conference document and all supported raw collections.

## Generate Web Artifacts

Generate one conference into the default output directory:

```sh
go run ./cmd/hackertracker info --conference DEFCON34
```

Default output:

```text
out/ht/<lowercase-conference>/
```

Generate one conference into an exact output directory:

```sh
go run ./cmd/hackertracker info --conference DEFCON34 --out ./public/defcon34/data
```

Generate multiple conferences into one output root:

```sh
go run ./cmd/hackertracker info --out ./out/ht --conference DCSG2026 --conference DEFCON34
```

When multiple conferences are exported with `--out`, each conference is written below that root using the lower-case conference code.

## Output Structure

The `info` command writes:

```text
out/ht/<lowercase-conference>/
  conference.json
  manifest.json

  derived/
    tagIdsByLabel.json

  exports/
    schedule.json
    schedule.csv

  views/
    announcementsList.json
    bookmarkSessionsById.json
    contentCards.json
    documentsList.json
    locationCards.json
    organizationsCards.json
    peopleCards.json
    scheduleDays.json
    searchData.json
    tagTypesBrowse.json

  details/
    content.json
    documents.json
    organizations.json
    people.json
    tags.json
```

The website export is runtime-only. It intentionally does not publish
`raw/**/*.json`, `entities/*.json`, `indexes/*.json`,
`details/sessions/<id>.json`, or `details/locations/<id>.json`. Session detail
pages and location detail pages are not part of the current `info.defcon.org`
runtime contract; content, people, tag, organization, and document details
remain available under `details/` as aggregate lookup files keyed by string id.
For example, `details/content.json` is a `Record<string, ContentDetailView>`
replacement for the old `details/content/<id>.json` files, and the same lookup
shape applies to documents, organizations, people, and tags.

Each `info` run recreates the generated subdirectories so stale JSON and CSV
files are removed.

## Public Schedule Exports

The `exports/` directory contains exactly two stable public schedule exports:

- `schedule.json`
- `schedule.csv`

Both files contain one record or row per scheduled session occurrence. Content
with multiple sessions appears once for each occurrence, and content without a
scheduled session is not included. Speaker biographies, descriptions,
organizations, tags, locations, related content, and public URLs are resolved
during generation, so consumers do not need to download or join against files
under `details/`.

`exports/schedule.json` uses `schemaVersion: 1` and is the rich,
self-contained format for LLMs, agents, applications, search and retrieval,
integrations, and archival use. Important fields include conference metadata,
`sessionId`, `contentId`, `title`, `description`, RFC 3339 `start` and `end`
timestamps, IANA `timezone`, location ID/name, nested `speakers`, nested
`organizations`, `tags`, compact `relatedContent`, `logoUrl`, and the canonical
public `url`.

`exports/schedule.csv` is the flat dataframe-oriented format for Polars,
Pandas, R, the Tidyverse, DuckDB, spreadsheet applications, and similar tools.
Multi-value CSV fields use `;`. Corresponding multi-value columns are
positionally aligned; for example, the first value in `speaker_ids`,
`speaker_names`, `speaker_titles`, `speaker_organizations`, and `speaker_bios`
describes the same speaker. JSON remains the lossless representation for nested
speaker, organization, link, tag, and related-content data.

CSV columns:

| Column | Description |
| --- | --- |
| `conference_code` | Conference code. |
| `conference_name` | Conference display name. |
| `conference_timezone` | Conference IANA timezone. |
| `session_id` | Scheduled occurrence ID. |
| `content_id` | Parent content ID. |
| `title` | Resolved session title. |
| `description` | Resolved full public description. |
| `content_type` | Public content type when the source provides one. |
| `start` | RFC 3339 start timestamp with UTC offset. |
| `end` | RFC 3339 end timestamp with UTC offset. |
| `timezone` | Session IANA timezone. |
| `location_id` | Resolved location ID. |
| `location` | Resolved location name. |
| `speaker_ids` | `;`-delimited speaker IDs. |
| `speaker_names` | `;`-delimited speaker names aligned with `speaker_ids`. |
| `speaker_titles` | `;`-delimited speaker titles aligned with `speaker_ids`. |
| `speaker_organizations` | `;`-delimited speaker organization names aligned with `speaker_ids`. |
| `speaker_bios` | `;`-delimited speaker biographies aligned with `speaker_ids`. |
| `organization_ids` | `;`-delimited associated organization IDs. |
| `organization_names` | `;`-delimited organization names aligned with `organization_ids`. |
| `tag_ids` | `;`-delimited tag IDs. |
| `tags` | `;`-delimited tag names aligned with `tag_ids`. |
| `related_content_ids` | `;`-delimited related content IDs. |
| `related_content_titles` | `;`-delimited related titles aligned with `related_content_ids`. |
| `related_content_urls` | `;`-delimited related public URLs aligned with `related_content_ids`. |
| `logo_url` | Public content logo URL when available. |
| `url` | Canonical public content URL. |

`views/scheduleDays.json` remains an application view model for the website UI.
It is not the stable public export contract.

## Raw Data vs Generated Artifacts

Raw fetch output follows HackerTracker Firestore collection names. Generated web artifacts use the domain names expected by `info.defcon.org`.

| Raw source                       | Generated artifacts                                                     |
| -------------------------------- | ----------------------------------------------------------------------- |
| `content`                        | schedule views, content cards, content details                          |
| `speakers`                       | `people`, people cards, people details                                  |
| `tagtypes` and embedded tag data | tag browse views, tag details                                           |
| `documents`                      | document lists, document details                                        |
| `locations`                      | location cards                                                          |
| `organizations`                  | organization cards, organization details                                |
| `articles`                       | announcement views                                                      |

Use `content` for top-level HackerTracker content records, `sessions` for scheduled instances embedded in content records, `speakers` for the raw Firestore collection, and `people` for generated artifacts derived from speakers.

## Development and Validation

Run local checks:

```sh
gofmt -w .
go test ./...
go run ./cmd/hackertracker --help
go run ./cmd/hackertracker fetch --help
go run ./cmd/hackertracker info --help
```

Run Firestore-backed checks when network access and Firestore permissions are available:

```sh
go run ./cmd/hackertracker conferences
go run ./cmd/hackertracker fetch all --conference DEFCON34
go run ./cmd/hackertracker info --conference DEFCON34
```

If the default Go build cache is not writable in a sandboxed environment, set a writable cache path:

```sh
GOCACHE=/tmp/hackertracker-go-build go test ./...
```
