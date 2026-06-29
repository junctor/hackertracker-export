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
    content/<id>.json
    documents/<id>.json
    organizations/<id>.json
    people/<id>.json
    tags/<id>.json
```

The website export is runtime-only. It intentionally does not publish
`raw/**/*.json`, `entities/*.json`, `indexes/*.json`,
`details/sessions/<id>.json`, or `details/locations/<id>.json`. Session detail
pages and location detail pages are not part of the current `info.defcon.org`
runtime contract; content, people, tag, organization, and document details
remain available under `details/`.

Each `info` run recreates the generated subdirectories so stale JSON is removed.

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
