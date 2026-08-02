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

Refresh the runtime data used by the sibling `hackertracker-info` checkout:

```sh
go run ./cmd/hackertracker info --conference DCSG2026 DEFCON34 DEFCON33 DEFCONBAHRAIN2025 DCME2026 DCTSG202610
rm -rf ../hackertracker-info/public/ht
cp -r out/ht ../hackertracker-info/public/
```

Replacing the directory prevents removed schema files from surviving a merge.
The web application fetches these files at runtime; regenerating them does not
require rebuilding the application bundle.

When multiple conferences are exported with `--out`, each conference is written below that root using the lower-case conference code.

## Output Structure

The `info` command writes:

```text
out/ht/<lowercase-conference>/
  conference.json
  manifest.json

  exports/
    schedule.json
    schedule.csv

  views/
    announcementsList.json
    contentCards.json
    contentFilterIndex.json
    documentsList.json
    locationCards.json
    organizationsBrowse.json
    peopleCards.json
    scheduleBrowse.json
    scheduleFilterIndex.json
    searchData.json
    tagTypesBrowse.json

  details/
    content/00.json ... 07.json
    documents/00.json
    organizations/00.json ... 03.json
    people/00.json ... 07.json
```

The website export is runtime-only. It intentionally does not publish
`raw/**/*.json`, `entities/*.json`, `indexes/*.json`, or unused session,
location, and tag detail views. Content, people, organization, and document
details are emitted as deterministic ID-keyed shards. Content and people use
eight shards, organizations use four, and documents use one. These paths and
counts are the fixed schema-4 runtime contract. The manifest contains only the
schema version and build timestamp used for cache invalidation. The fixed shard
layout keeps every conference export at 36 files while avoiding a
conference-wide detail download for a single route.

`views/scheduleBrowse.json` is the compact runtime schedule contract shared by
schedule and bookmark pages. It stores display-ready sessions once under
`days`, omits redundant nested content/session copies, and provides
`sessionPositionsById` for bookmark lookups.

The focused content and schedule filter indexes contain only stable item IDs by
tag. They let filter routes calculate interactive combinations without loading
or scanning the full card or schedule payload. `organizationsBrowse.json`
contains the prejoined all-organizations list, category lists, and label lookup
used by organization routes.

Each `info` run recreates the generated subdirectories so stale JSON and CSV
files are removed.

`views/searchData.json` is the compact runtime search index for
`info.defcon.org`. It includes direct records for content titles, people names,
organization names, and content tags. Tag records use stable numeric tag IDs and
include `contentIds` plus `contentCount` so the web app can show matching tags
as first-class results and link to all associated content without duplicating
large tag objects inside every content record. Search text is normalized with
the same case-insensitive, accent-folding, punctuation-to-space behavior used by
the web client. This is part of the fixed schema-4 runtime contract.

## Public Schedule Exports

The `exports/` directory contains exactly two stable public schedule exports:

- `schedule.json`
- `schedule.csv`

Both files contain one record or row per scheduled session occurrence. Content
with multiple sessions appears once for each occurrence, and content without a
scheduled session is not included. Organizations, tags, locations, and people
are resolved during generation, so consumers do not need to download or join
against files under `details/`. Schedule exports are scoped to the conference
page or conference directory they are downloaded from, so they do not repeat the
conference name in each session record or CSV row.

`exports/schedule.json` uses `schemaVersion: 4` and is the rich,
self-contained format for LLMs, agents, applications, search and retrieval,
integrations, and archival use. The top-level object contains compact
`metadata` followed by the `sessions` array. Metadata includes conference code,
conference name, conference timezone, session count, timestamp timezone and
format, and the maximum text snippet length. Session fields include
`sessionId`, `contentId`, `title`, `descriptionSnippet`, RFC 3339 `start` and
`end` timestamps in UTC, location name, nested `speakers`, nested
`organizations`, and `tags`. Long text fields are whitespace-normalized and
capped at 1200 characters, including the trailing `...` marker when truncated.

`exports/schedule.csv` is the flat dataframe-oriented format for Polars,
Pandas, R, the Tidyverse, DuckDB, spreadsheet applications, and similar tools.
Column names are unique lower-case `snake_case`, one scheduled session
occurrence appears per row, UTC timestamp columns are explicitly suffixed with
`_utc`, and multi-value CSV fields use `;`. The CSV intentionally avoids sparse
speaker affiliation/title columns, full free-text biographies, descriptions,
URLs, presentation asset fields, and related-content pointers; JSON remains the
richer representation for LLMs and nested speaker, organization, and tag data.

CSV columns:

| Column | Description |
| --- | --- |
| `session_id` | Scheduled occurrence ID. |
| `content_id` | Parent content ID. |
| `start_utc` | UTC RFC 3339 start timestamp. |
| `end_utc` | UTC RFC 3339 end timestamp. |
| `location_name` | Resolved location name. |
| `title` | Resolved session title. |
| `speaker_names` | `;`-delimited speaker names. |
| `organization_names` | `;`-delimited associated event or community organization names. |
| `tag_names` | `;`-delimited tag names. |
| `description_snippet` | Whitespace-normalized public description preview, capped at 1200 characters including `...` when truncated. |

`views/scheduleBrowse.json` is the application view model for the website UI.
It is not the stable public export contract.

## Raw Data vs Generated Artifacts

Raw fetch output follows HackerTracker Firestore collection names. Generated web artifacts use the domain names expected by `info.defcon.org`.

| Raw source                       | Generated artifacts                                                     |
| -------------------------------- | ----------------------------------------------------------------------- |
| `content`                        | schedule views, content cards, content details                          |
| `speakers`                       | `people`, people cards, people details                                  |
| `tagtypes` and embedded tag data | tag browse views                                                        |
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
