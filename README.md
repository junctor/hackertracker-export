## Output Contract

`info` generates production artifacts only. On each run, it recreates the
generated directories below the target output directory so stale generated JSON
is removed.

The exporter uses the Hacker Tracker v2 model internally:

- `content` is the canonical talk, workshop, demo, or activity record.
- `session` is the scheduled occurrence of content at a time and location.

For compatibility with existing `info.defcon.org` consumers, some generated
artifact filenames still use the legacy `event` naming. These files contain
session data and should be treated as compatibility aliases until the web app is
fully migrated.

Required artifacts include:

- `manifest.json`
- `derived/tagIdsByLabel.json`
- `entities/articles.json`
- `entities/content.json`
- `entities/documents.json`
- `entities/events.json`
- `entities/sessions.json`
- `entities/locations.json`
- `entities/organizations.json`
- `entities/people.json`
- `entities/tags.json`
- `entities/tagTypes.json`
- `indexes/eventsByDay.json`
- `indexes/eventsByTag.json`
- `indexes/sessionsByDay.json`
- `indexes/sessionsByTag.json`
- `views/announcementsList.json`
- `views/bookmarkEventsById.json`
- `views/bookmarkSessionsById.json`
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

Compatibility aliases:

| Legacy artifact                 | New artifact                      | Notes                                                                |
| ------------------------------- | --------------------------------- | -------------------------------------------------------------------- |
| `entities/events.json`          | `entities/sessions.json`          | Same session records, legacy filename retained for existing clients. |
| `indexes/eventsByDay.json`      | `indexes/sessionsByDay.json`      | Day index over sessions.                                             |
| `indexes/eventsByTag.json`      | `indexes/sessionsByTag.json`      | Tag index over sessions.                                             |
| `views/bookmarkEventsById.json` | `views/bookmarkSessionsById.json` | Bookmark-ready session view models.                                  |

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
- Content records describe what something is.
- Session records describe when and where content happens.
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
