# Hacker Tracker Export

Go tools for exporting Hacker Tracker Firestore data.

This repository contains two commands:

- `cmd/hackertracker`: generic raw Hacker Tracker fetcher.
- `cmd/info-export`: purpose-built exporter for the `info.defcon.org` JSON shape.

The implementation intentionally keeps dependencies small. The only Firebase SDK dependency is:

```sh
go get firebase.google.com/go/v4
```

## Usage

From this repository:

```sh
go run ./cmd/hackertracker --help
go run ./cmd/hackertracker conferences
go run ./cmd/hackertracker fetch --conference defcon34 --out ./raw
```

Generate the `info.defcon.org` artifacts:

```sh
go run ./cmd/info-export --conference defcon34 --out ./public/defcon34/data
```

The info exporter writes:

- `manifest.json`
- `entities/*.json`
- `indexes/*.json`
- `views/*.json`
- `derived/tagIdsByLabel.json`
- `details/<type>/<id>.json`

Generated JSON is valid minified Go JSON. The exporter preserves generated paths, filenames, IDs, relationships, and fields needed by web consumers without requiring byte-for-byte parity with the JavaScript exporter.

## Firebase Access

The JavaScript exporter uses the public Firebase web client without an auth flow. This Go rewrite initializes the Firebase Go SDK with `option.WithoutAuthentication()` and the Hacker Tracker project ID.

If Firestore rejects unauthenticated Admin SDK access in an environment, the fetch commands fail loudly. No credential loading, REST fallback, or custom auth flow is added here.

## Development

Use a writable Go build cache if your environment restricts the default cache location:

```sh
GOCACHE=/tmp/hackertracker-go-build go test ./...
```
