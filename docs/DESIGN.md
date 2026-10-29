# Design

## Boundaries

One calendar record corresponds to an activity summary for one source conversation on one calendar day. Full conversations are stored separately as linked conversation snapshots; see [Full Conversation Archives](ARCHIVES.md) for formats and upload procedures. A source is a freely named account namespace, and an external agent transforms material from each platform. The service does not hold platform credentials, invoke summarization models, or collect continuously.

Dates, sources, conversation identifiers, send counts, and quality notes come from the organized material. Titles, summaries, and tags support review. The user can edit display content on the web, while source records remain preserved.

## Storage

SQLite core schema 1; full conversations use the additional `archive_schema` version 1:

- `activities`: unique `source`/`external_id`, date, normalized record JSON, content digest, source version, annotation JSON, and independent annotation version.
- `import_batches`: unique idempotency key, normalized request digest, commit result, and coverage range.
- `import_changes`: old Activity snapshot and new source record for every actual change, available for maintenance audits; the first version has no automatic undo API.

A single process uses a single database connection. Process mutexes and database transactions serialize writes. An import first validates the entire batch; if any conflict exists, nothing is written. Repeated requests return a persisted receipt. Concurrent updates from different requests must satisfy `expected_version`.

The record content digest ignores `expected_version` and `provenance.produced_at`, so changing only the reorganization time does not create a new source version. The batch request digest retains the complete normalized request; reusing an idempotency key for different content returns a conflict.

## Counts and Time

- User-send counts include messages appended while an execution is running and exclude subagents, tools, compaction, and automatic continuation.
- An `unknown` count is `null` and is not displayed as a known zero. `observed` means directly identifiable in the selected material and does not guarantee that the original platform retained complete history.
- `observed_timestamps` represents known send times but provides no duration basis; it does not generate spans or calculate a first-to-last difference.
- When spans exist, the daily view shows the approximate minutes in the union of intervals, avoiding duplicate counting for parallel conversations on the same day. This is not the user's focused time.
- Time values accept RFC3339 with an offset and normalize to UTC. The calendar validates against the record's declared IANA time zone. An end time may fall exactly at midnight on the following day.
- Monthly conversation counts are deduplicated by source plus conversation identifier. Active days are based only on records with interaction or an unknown interaction count. Missing conversation IDs are reported separately.

## Import Coverage

Coverage status is `complete`, `partial`, or `unknown` and is stored by source and date range. No record does not mean no usage; an unimported date is not shown as a confirmed zero. The client places coverage batches after data batches so a preceding failure does not submit a complete-coverage declaration.

## Record Organization State

`record_state` represents the organization progress of an individual calendar record, separate from source coverage and the full-conversation snapshot's `manifest.coverage`:

- `partial`: the selected material's daily snapshot still awaits extraction, review, or completion; the web page displays the exact UI labels “待补全” or “记录待补全”.
- `final`: the selected material's daily snapshot has been fully organized, with known omissions and unknown values recorded accurately; the web page displays the exact UI label “已整理”. This does not guarantee that all platform history is obtainable or that the original export contains every image or body.

Therefore, a reviewed export record can have `record_state=final` while source coverage is `partial`. Do not mark every record as incomplete merely because complete platform history cannot be proven. If new material is obtained later, a `final` record can still be supplemented through an upsert with version checking; existing downgrade-prevention rules remain in effect.

## Web Interface

Vue 3 + Vite, with assets embedded in Go. At startup, the browser fetches all records page by page and aggregates them locally; move aggregation into the service only when data grows significantly.

- Time range: month, quarter, year, custom start/end, and all time, with presets and previous/next shifting. Ranges of 62 days or less use a weekly grid; longer ranges use compact monthly heatmaps grouped by year. Range, view, selected day, search, source, tags, and hidden toggle are synchronized to the URL for direct sharing and back navigation.
- Filters: search plus source and tags (multi-select, match any) apply to all records. Statistics, calendar, list, and conversations show only the current range and report matching records outside it. An ongoing range is compared with a preceding period of equal length.
- The page has no maximum width. The calendar occupies the full row, and records for the selected day tile in multiple columns below it according to available width. If details are outside the viewport when a date is selected, scrolling reveals only the first detail row while keeping part of the calendar visible.
- Heat levels use the 40th/70th/90th percentiles of send counts across all active days. Source colors are fixed; unknown sources are hashed by name.
- Records are initially read-only. Editing toggles within details; after save, the item is replaced in place without refetching all records. Separation between source data and user annotations remains unchanged.
- Chat reading: 200 items per page, merging segments and separating dates. Markdown is rendered only for AI replies; HTML is disabled, and images are represented only as text. Platform JSON messages from TraeX, ChatGPT, Claude, and Cursor have their body text extracted, while reasoning and tool calls go into collapsible groups. When a date is provided, the reader loads and navigates to it automatically.
- The import page shows a coverage timeline and actual record span by source, with batches collapsed by date.

`main.ts` intercepts unmodified Tab / Shift+Tab during the capture phase while preserving modifier combinations and IME. CSS removes focus outlines and shadows, while business selection styles remain. The `favicon.svg` source and a standard-library PNG generation script are both committed.

## Authentication

The service listens only on a loopback IP. Browser routes rely on ServerPortal device authentication in shared Nginx, and write requests additionally require one exact HTTPS Origin from the explicit configured set. This lets the `ali`, `vps:8443`, and `nas` LAN entrances reach the same database without changing browser cookie scopes. The import prefix `/ingest/v1/` uses the service's own random Bearer token; the server stores only its SHA-256. A missing or invalid token always returns 401. The prefix provides chunked archive upload, message reads, and full-archive download. It does not provide web-annotation editing, deletion, calendar-summary export, or other application operations.

Nginx disables the portal subrequest only for the import prefix and delegates to this application to enforce authentication. All other pages and APIs remain protected by the portal. The only public resources are GET/HEAD for `favicon.svg` and `apple-touch-icon.png`. The application adds no `add_header`, preserving inherited shared-cookie renewal.
