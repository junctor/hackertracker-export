# hackertracker-export

Go CLI for turning HackerTracker Firestore data into the static JSON artifacts used by [info.defcon.org](https://info.defcon.org) and [`junctor/hackertracker-info`](https://github.com/junctor/hackertracker-info).

The exporter does two things:

1. **Raw export**: fetches HackerTracker conference data from Firestore and writes raw Firestore-shaped JSON for inspection, fixtures, schema audits, and debugging.
2. **Artifact export**: transforms that raw data into the normalized static JSON tree consumed by `info.defcon.org`.

In practice, `fetch` is for raw data and `info` is for the generated web artifacts.

## What This Does

`hackertracker-export` has two main jobs:

| Command | Purpose                                                                                  |
| ------- | ---------------------------------------------------------------------------------------- |
| `fetch` | Writes raw Firestore-shaped JSON for inspection, fixtures, debugging, and schema checks. |
| `info`  | Writes the normalized static artifact tree consumed by `info.defcon.org`.                |

The raw data is useful for auditing what HackerTracker currently provides. The normalized export is what the web app actually serves.

## Run Locally

Run the CLI from this checkout:

```sh
go run ./cmd/hackertracker --help
```

Install a local binary:

```sh
go install ./cmd/hackertracker
```

Then run:

```sh
hackertracker --help
```

Firestore-backed commands require network access and permission to read the HackerTracker Firestore project.

## CLI

```sh
hackertracker conferences
hackertracker fetch <target> --conference <code> [--stdout] [--out <dir>]
hackertracker info --conference <code> [--conference <code>] [--out <dir>]
```

The same commands can be run without installing:

```sh
go run ./cmd/hackertracker conferences
go run ./cmd/hackertracker fetch content --conference DEFCON34 --stdout
go run ./cmd/hackertracker info --conference DEFCON34
```

## Commands

### List Conferences

Print available HackerTracker conferences as JSON:

```sh
go run ./cmd/hackertracker conferences
```

### Fetch Raw Firestore Data

Raw fetch targets mirror known Firestore collections and raw conference metadata.

```text
conference
articles
content
documents
locations
maps
menus
organizations
speakers
tags
tagTypes
all
```

Examples:

```sh
go run ./cmd/hackertracker fetch conference --conference DEFCON34 --stdout
go run ./cmd/hackertracker fetch content --conference DEFCON34 --stdout
go run ./cmd/hackertracker fetch speakers --conference DEFCON34 --stdout
go run ./cmd/hackertracker fetch maps --conference DEFCON34 --stdout
go run ./cmd/hackertracker fetch menus --conference DEFCON34 --stdout
go run ./cmd/hackertracker fetch all --conference DEFCON34
```

By default, raw files are written to:

```text
out/ht/<conference>/raw/
  conference.json
  articles.json
  content.json
  documents.json
  locations.json
  maps.json
  menus.json
  organizations.json
  speakers.json
  tags.json
  tagTypes.json
```

Use `--stdout` when you want a single raw document or collection printed to standard output:

```sh
go run ./cmd/hackertracker fetch content --conference DEFCON34 --stdout
```

Use `--out` to write raw files somewhere else:

```sh
go run ./cmd/hackertracker fetch all --conference DEFCON34 --out ./tmp/defcon34/raw
```

### Export Static Artifacts

Generate the normalized static artifact tree for one conference:

```sh
go run ./cmd/hackertracker info --conference DEFCON34
```

Generate multiple conferences into one root directory:

```sh
go run ./cmd/hackertracker info --out ./out/ht --conference DCSG2026 --conference DEFCON34
```

Without `--out`, each conference is exported to:

```text
out/ht/<conference>/
```

When multiple conferences are exported with one `--out` directory, each conference is written below that directory by lower-case conference code.

## Output Structure

The `info` command writes the generated JSON tree used by `info.defcon.org`.

Generated output includes files like:

```text
out/ht/<conference>/
  manifest.json

  derived/
    tagIdsByLabel.json

  entities/
    articles.json
    content.json
    documents.json
    locations.json
    organizations.json
    people.json
    sessions.json
    tags.json
    tagTypes.json

  indexes/
    sessionsByDay.json
    sessionsByTag.json

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
    locations/<id>.json
    organizations/<id>.json
    people/<id>.json
    sessions/<id>.json
    tags/<id>.json
```

Entity files use an IndexedDB-friendly shape:

```json
{
  "allIds": [123, 456],
  "byId": {
    "123": { "id": 123 },
    "456": { "id": 456 }
  }
}
```

`info` recreates generated output directories on each run so stale generated JSON is removed.

## Raw Collections vs Exported Artifacts

Raw Firestore collections do not always match exported artifact names.

| Raw Firestore                    | Exported artifact/domain concept      |
| -------------------------------- | ------------------------------------- |
| `content`                        | `content`, `sessions`, schedule views |
| `speakers`                       | `people`                              |
| `tagTypes` and embedded tag data | `tags`, `tagTypes`, tag indexes       |
| `menus`                          | navigation and app menu metadata      |
| `maps`                           | map-related raw data                  |
| `documents`                      | document entities and details         |

There are no raw Firestore `people` or `sessions` collections.

`people` artifacts are derived from raw `speakers`.

`sessions` are embedded inside raw `content` and are transformed into session entities, schedule views, and session indexes.

The raw fetch CLI mirrors Firestore collection names, not every exported artifact name.

## Naming

Use current HackerTracker terminology:

| Term       | Meaning                                                 |
| ---------- | ------------------------------------------------------- |
| `content`  | Top-level HackerTracker content records.                |
| `sessions` | Scheduled instances or occurrences embedded in content. |
| `speakers` | Raw Firestore speaker collection.                       |
| `people`   | Exported/domain artifacts derived from speakers.        |

Avoid legacy `event` terminology in internal code unless it is required for raw upstream compatibility or a public output contract.

## Development

Run local checks:

```sh
gofmt -w .
go test ./...
go run ./cmd/hackertracker --help
go run ./cmd/hackertracker fetch --help
go run ./cmd/hackertracker fetch content -h
go run ./cmd/hackertracker fetch speakers -h
go run ./cmd/hackertracker info -h
```

Run Firestore-backed checks when credentials and permissions are available:

```sh
go run ./cmd/hackertracker conferences
go run ./cmd/hackertracker fetch content --conference DEFCON34 --stdout
go run ./cmd/hackertracker fetch speakers --conference DEFCON34 --stdout
go run ./cmd/hackertracker fetch all --conference DEFCON34
go run ./cmd/hackertracker info --conference DEFCON34
```

If the default Go build cache is not writable in a sandboxed environment, set a writable cache path:

```sh
GOCACHE=/tmp/hackertracker-go-build go test ./...
```

## Useful Validation Flow

A typical verification pass looks like this:

```sh
gofmt -w .
go test ./...
go run ./cmd/hackertracker fetch all --conference DEFCON34
go run ./cmd/hackertracker info --conference DEFCON34
```

Then compare the generated tree under:

```text
out/ht/defcon34/
```

with the structure expected by [`hackertracker-info`](https://github.com/junctor/hackertracker-info).

## Notes

- `fetch all` is for raw data inspection and fixtures.
- `info` is the command used to generate the static artifacts for `info.defcon.org`.
- Raw collection names should stay explicit and hardcoded.
- Dynamic Firestore collection discovery should be used only for audits, not normal export behavior.
- Public output paths are part of the `info.defcon.org` contract and should be changed carefully.
