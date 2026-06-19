# Hacker Tracker Export

Go CLI for exporting Hacker Tracker Firestore data. This repository replaces the
old JavaScript `info-export` pipeline and writes the static JSON artifacts used
by `info.defcon.org`.

The command provides three subcommands:

- `hackertracker conferences`: list available conferences from Firestore.
- `hackertracker fetch`: fetch raw Hacker Tracker collections for debugging or auditing.
- `hackertracker info-export`: export the normalized `info.defcon.org` JSON dataset.

## Install

From this repository, run the CLI directly:

```sh
go run ./cmd/hackertracker --help
```

Or install a `hackertracker` binary:

```sh
go install ./cmd/hackertracker
hackertracker --help
```

## Export for info.defcon.org

The `info.defcon.org` web app expects each conference dataset under:

```text
public/ht/<lowercase-conference-code>/
  manifest.json
  derived/
  details/
  entities/
  indexes/
  views/
```

When working from this repository next to `hackertracker-info`, write directly
to the app's static data directory:

```sh
hackertracker info-export --conference DEFCON34 --out ../hackertracker-info/public/ht/defcon34
```

For multiple conferences, pass all conference codes in one command and make
`--out` the root `public/ht` directory. Each conference is written to a
lowercased subdirectory:

```sh
hackertracker info-export --conference DCSG2026 DEFCON34 DEFCON33 DEFCONBAHRAIN2025 DCME2026 --out ../hackertracker-info/public/ht
```

That writes:

```text
../hackertracker-info/public/ht/dcsg2026/
../hackertracker-info/public/ht/defcon34/
../hackertracker-info/public/ht/defcon33/
../hackertracker-info/public/ht/defconbahrain2025/
../hackertracker-info/public/ht/dcme2026/
```

If `--out` is omitted, exports go to `./out/ht/<lowercase-conference-code>/`.
This is useful for local inspection or for staging artifacts before copying
them into another checkout:

```sh
hackertracker info-export --conference DCSG2026 DEFCON34 DEFCON33 DEFCONBAHRAIN2025 DCME2026
```

## Other Commands

List available conferences:

```sh
hackertracker conferences
```

Fetch raw Firestore collections for one conference:

```sh
hackertracker fetch --conference DEFCON34 --out ./raw/defcon34
```

Raw fetch output is separate from `info-export`. The old JavaScript exporter had
an `--emit-raw` flag; this Go replacement uses `hackertracker fetch` instead.

## Output Contract

`info-export` generates production artifacts only. On each run, it recreates the
generated directories below the target output directory so stale generated JSON
is removed.

Required artifacts include:

- `manifest.json`
- `derived/tagIdsByLabel.json`
- `entities/articles.json`
- `entities/content.json`
- `entities/documents.json`
- `entities/events.json`
- `entities/locations.json`
- `entities/organizations.json`
- `entities/people.json`
- `entities/tags.json`
- `entities/tagTypes.json`
- `indexes/eventsByDay.json`
- `indexes/eventsByTag.json`
- `views/announcementsList.json`
- `views/bookmarkEventsById.json`
- `views/contentCards.json`
- `views/documentsList.json`
- `views/locationCards.json`
- `views/organizationsCards.json`
- `views/peopleCards.json`
- `views/scheduleDays.json`
- `views/searchData.json`
- `views/tagTypesBrowse.json`
- `details/content/<id>.json`
- `details/documents/<id>.json`
- `details/locations/<id>.json`
- `details/organizations/<id>.json`
- `details/people/<id>.json`
- `details/tags/<id>.json`

Every entity file follows the same IndexedDB-friendly shape:

```json
{
  "allIds": [123, 456],
  "byId": {
    "123": { "id": 123 },
    "456": { "id": 456 }
  }
}
```

The app should use `allIds` for deterministic ordering and `byId` for constant
time record lookup.

## Data Model Notes

The exporter precomputes the data shapes the web app needs so the client does
not need to scan Firestore-shaped collections or perform expensive joins at
runtime.

- Entities are canonical normalized records.
- Views are minimal UI-ready projections.
- Indexes are query accelerators keyed by day or tag.
- Detail files are static per-record JSON documents for route-level loading.
- Schedule day bucketing uses the conference's IANA timezone from Firestore.
- Generated IDs, references, ordering, and paths are deterministic.

`manifest.json` contains:

- `code`: conference code
- `name`: display name
- `timezone`: IANA timezone string
- `schemaVersion`: generated artifact schema version
- `buildTimestamp`: UTC export timestamp used for client cache invalidation

Expected client behavior:

1. Fetch `manifest.json` on page load.
2. Compare `buildTimestamp` to the version stored in IndexedDB.
3. If changed or missing locally, download the required production artifacts.
4. Use IndexedDB for subsequent queries.

## Firebase Access

The exporter reads the public Hacker Tracker Firestore project with the Firebase
Go SDK using `option.WithoutAuthentication()`. No credential loading, REST
fallback, or custom auth flow is implemented.

If Firestore rejects unauthenticated access in an environment, the command fails
loudly.

## Development

Run tests:

```sh
go test ./...
```

Use a writable Go build cache if your environment restricts the default cache
location:

```sh
GOCACHE=/tmp/hackertracker-go-build go test ./...
```
