# API v1

Contract sources: [JSON Schema](../api/import.schema.json), [OpenAPI](../api/openapi.json), and [Synthetic Import Request](../api/import-example.json). All paths are relative to `/aicalendar/`.

## Authentication

The web API is under `api/v1/` and protected by the ServerPortal device cookie; write requests additionally require one exact configured HTTPS Origin, including its port. See [Operations](OPERATIONS.md#startup-and-diagnostics) for the primary and additional Origin configuration. Programmatic calls use `ingest/v1/` and `Authorization: Bearer <token>` independently of that browser Origin check. Do not extract the site-wide cookie from a browser for agent use.

| Method | Web path | Import-credential path | Purpose |
| --- | --- | --- | --- |
| GET | api/v1/info | — | Name, default time zone, today's date, and demo status |
| GET | api/v1/schema | ingest/v1/schema | JSON Schema for the currently running version |
| POST | api/v1/imports/preview | ingest/v1/imports/preview | Validate and return per-record insert/update/unchanged/conflict results |
| POST | api/v1/imports | ingest/v1/imports | Commit an entire batch atomically |
| GET | api/v1/imports | ingest/v1/imports | Import list and coverage declarations, without per-record results |
| GET | api/v1/imports/{id} | ingest/v1/imports/{id} | Persisted batch receipt |
| GET | api/v1/activities | ingest/v1/activities | Query normalized records and annotations |
| PATCH | api/v1/activities/{id}/annotation | — | Replace a user annotation under optimistic version protection |
| GET | api/v1/export | — | Export records, annotations, and a batch overview |

The health check is `GET /healthz`, used by loopback release checks. It still requires portal authentication through the public endpoint.

GET JSON responses under both prefixes are compressed when the request includes `Accept-Encoding: gzip` and return `Vary: Accept-Encoding`. Archive downloads retain their original bytes and are not recompressed.

## Queries

`activities` parameters: `from`, `to` (YYYY-MM-DD, inclusive), `source`, `q`, `tag`, `include_hidden=true`, `offset`, and `limit` (1–5000, default 500). The response is `{items,total,offset,limit}`. Each item contains `id`, source `version`, `annotation_version`, `record`, `annotation`, and `updated_at`.

`q` searches the final displayed title, summary, topic tags, and source label; user annotations take precedence over source values. `tag` exactly matches a topic tag or `source_label`. The returned `record` preserves the original normalized source for reimport verification. Hidden items are omitted by default.

## Import

A request requires `schema_version:1`, `idempotency_key`, `mode`, and `records`; it may include `coverage`. Each batch permits at most 500 records, 50 coverage declarations, and 1 MiB. `records` may be empty only when `coverage` is nonempty.

The default mode is `insert_only`. An existing unique key with identical content is `unchanged`; different content is a `conflict`. An upsert that updates an existing record must provide `expected_version`. Counts are complete snapshots and must not be added incrementally. The service rejects reducing/clearing a known count, downgrading an `observed` count, changing `final` to `partial`, deleting known times/spans, or changing the conversation, date, or time zone.

A record must provide a meaningful quality note. `time` supports `observed_interval`, `observed_timestamps`, `estimated_interaction_span`, `date_only`, and `unknown`; `spans` is empty when no interval evidence exists. Date and time zone are part of record identity. Source `source` allows letters, digits, and `_.-`; the organizer generates a stable `external_id`.

`record_state` is the organization progress of one record: use `partial` when organization remains, and `final` when the selected material has been fully organized. It is independent of source-range `coverage[].status` and conversation-archive `manifest.coverage`; partial source coverage does not require the record itself to be partial. A final record can still be supplemented with new material under version checking. See [Record Organization State](DESIGN.md#record-organization-state).

Optional `source_label` is a user-specified display source, for example the exact labels “ChatGPT · 个人账号” or “Trae · 工作电脑”. After trimming leading and trailing whitespace, it is at most 80 characters. The calendar displays this label and allows filtering by it; when omitted, it displays `source`. `source` remains the stable source/account namespace and must not be renamed when the label changes. Update a label through an ordinary upsert with `expected_version`, without changing `external_id` or increasing counts. Topic tags in manual annotations remain independent of source labels.

Preview returns 200 with the number of conflicts in `conflicts`. A write with a conflict returns 409 and `committed=false`, with no change to the batch. A successful receipt contains `id`, time, `inserted`/`updated`/`unchanged`, and per-item IDs/versions. Retrying the same request with the same idempotency key returns `replay=true`; a different request under the same key returns 409.

Parse failures, unknown fields, and validation failures return 400. A network failure may retry the original request; do not generate a new identifier to bypass retry semantics. All externally returned 5xx details are simplified, with full causes written to the journal.

## Annotations

A PATCH request is `{expected_version: <annotation_version>, annotation: {...}}` and fully replaces the annotation object. Supported fields are `title`, `summary`, `tags`, and `hidden`. An omitted or null text/tag value falls back to the source value; an empty tag array means the user cleared all tags. Setting `{hidden:false}` restores the complete source presentation. A version mismatch returns 409.

## Export and Recovery

The export shape is `{schema_version,exported_at,activities,imports}`, where `activities` contains `record` and `annotation`. An export package is not directly an import request; reimport requires extracting `record` and explicitly planning annotation recovery. Complete database recovery uses a consistent native SQLite snapshot including conversation archives, original chunks, message indexes, and unfinished uploads; see [OPERATIONS.md](OPERATIONS.md). A JSON calendar export cannot replace it.

## Full Conversations

Calendar import v1 remains compatible. Full conversations use the `/archives` endpoint family in the same authenticated namespace to preserve normalized messages and original exports as immutable conversation snapshots. See [ARCHIVES.md](ARCHIVES.md) for chunk format, validation, title search (`q`), paginated reading (up to 200 items and approximately 2 MiB per page), and downloads. The schema is `api/archive.schema.json`.
