# Full Conversation Archives

## Preserved Content

Each source conversation is preserved as an immutable snapshot containing normalized messages and the original source files. It links to calendar records through `source + conversation_id`; the calendar groups by day, while a conversation snapshot retains cross-day context without cutting at day boundaries. Titles, summaries, and counts remain separate records. Hiding a calendar record does not delete the original text.

Optional `source_label` stores a user-supplied source label such as the exact label “ChatGPT · 个人账号”. After trimming leading and trailing whitespace, it is at most 80 characters and is displayed and searchable in both the calendar and archive. It is mutable display metadata, independent of stable `source` and the conversation identifier. When unspecified, `source` is displayed. The label is stored in the service manifest rather than immutable gzip content; the current label for a download can be retrieved from the manifest endpoint.

Normalized message fields are `id`, `role`, `at`, `content`, `part`/`parts`, and `attributes`. `role` supports `user`, `assistant`, `tool`, `system`, `developer`, and `unknown`; an unknown time is `null`. Message bodies are preserved verbatim. Long messages are segmented at UTF-8 boundaries and can be reassembled when read. Tool arguments, results, image structures, and similar structured content may be stored as a textual representation; the original files are preserved as well to avoid losing platform-specific fields during conversion. Archive message counts include AI and tool messages and must not be treated as user-usage counts.

Messages already missing from the source cannot be recovered by an archive. Attachment entities are preserved only when provided with the export and packaged as source files. When only a URL, path, or temporary ID exists, the reference is retained and the limitation is recorded in the completeness note. The service does not fetch links or read arbitrary local files mentioned in logs.

## File Format

One snapshot is a gzip-compressed UTF-8 JSONL file with every line terminated by LF. The maintained contract is `api/archive.schema.json`. In order, it contains:

1. One header with format=`aicalendar.conversation.v1`, source, conversation ID, title, and the original-source manifest. Each source records its original name, type, byte count, SHA-256, and chunk count.
2. Message lines in source order, with consecutive `part` values from 0 to `parts-1`. Each body segment is at most 65536 UTF-8 bytes, and `attributes` is at most 16384 bytes. Empty messages are retained. Different revisions use independent stable IDs.
3. `source_chunk` lines containing source ID, sequence number, and base64-encoded original bytes. Each chunk is at most 384 KiB. Export JSON, logs, images, or other binary attachments can be preserved. Files can be reconstructed and verified individually after download; base64 does not mean the original binary is discarded.

An archive is limited to 128 MiB compressed, 512 MiB expanded, 200000 messages, and 100 original files. An oversized archive is rejected explicitly rather than truncated. The organizer can split an oversized source into consecutive volumes, each with independent original files, an accurate range note, and completeness status.

## Upload Protocol

Browser `/api/v1/archives` uses portal device authentication. External-agent `/ingest/v1/archives` uses the existing import Bearer credential. Nginx and ordinary APIs retain the 1 MiB request limit; the gzip file is uploaded in fixed 384 KiB chunks of raw compressed bytes.

| Request | Behavior |
| --- | --- |
| GET /archives/schema | Return the archive contract |
| POST /archives/prepare | Submit a manifest and receive a stable archive ID plus already uploaded chunks; identical content is idempotent |
| POST /archives/{id}/parts | Upload `index`, `data` (base64), and `sha256`; the same number and content may be retried, but replacement is prohibited |
| POST /archives/{id}/commit | Verify chunks, gzip, normalized messages, and original-file order/size/hash; commit atomically only after every check passes |
| GET /archives/{id} | Query status, manifest, and `received_parts` for an incomplete upload |
| GET /archives | List only completed snapshots; filters are `source`, `conversation_id`, `label` (source-label keyword), `q` (title or source-label keyword), `offset`, and `limit`, with at most 100 items |
| GET /archives/{id}/messages | Read verbatim message segments; `after` defaults to -1 and `limit` defaults to 20 with a maximum of 200. A page ends early at approximately 2 MiB of accumulated data, after at least one segment. Use `has_more` to decide whether to continue. Returns `next_after` and `has_more` |
| GET /archives/{id}/download | Download the complete `.jsonl.gz`; supports HEAD and requires authentication |

The manifest contains `source`, optional `source_label`, `conversation_id`, `title`, `captured_at`, `coverage`, `note`, compressed and expanded bytes/SHA-256, `parts`, `message_count`, and `source_count`. The ID is determined by source, conversation, and compressed-content hash; a different `captured_at` packaging time does not create a duplicate of identical content. Repeating `prepare` for the same file can update its label through a nonempty `source_label`; omitting it or supplying an empty value preserves the current label, and the archive ID and count remain unchanged. Other manifest differences conflict. Added or revised content creates a new snapshot without overwriting the old one.

`prepare`/`parts` uploads do not appear in the conversation list. A failed `commit` transaction leaves no visible message index. Network interruption does not prove that commit failed: the service may continue validation for up to two minutes after accepting `commit`. Read status or reuse the same fixed plan before retrying. Uploaded chunks are retained for recovery and incomplete material is not cleaned automatically. After commit, the client downloads the actual bytes and compares hashes; HTTP 200 alone must not be presented as proof of information completeness.

The web interface reads messages page by page from a snapshot. Text is always escaped, HTML/scripts in conversations are not executed, and remote images are not loaded automatically. Full-package download preserves complete bodies and source files. Existing `/api/v1/export` still exports calendar summaries, annotations, and import receipts; complete archives are retrieved separately through the list and download endpoints above.

## Storage, Backup, and Upgrade

All archives remain in the existing `data/aicalendar.sqlite`: `conversation_archives` stores manifests/status, `archive_parts` stores gzip-file chunks, and `archive_messages` stores the compressed reading index of message segments. There is no new data directory or backup type. Archives are excluded from daily and pre-release backups: snapshots contain only calendar records, annotations, and import audits, plus empty archive table structures. See [Operations](OPERATIONS.md#backup-and-recovery).

This is additional `archive_schema` version 1 on top of core schema 1. A new `init` creates the tables. For an existing database, `migrate-archives --db <existing>` adds them in an explicit transaction without rewriting old activities/annotations or changing the core `user_version`. An old program can still open and display old data and back up the complete database, but it has no archive API. Rollback does not delete the new tables or archived content.

Before upgrade, use the current program to create a native backup, then run `migrate-archives` from the same-commit candidate as the `aicalendar` identity. Check the backup and extension, then release through the normal `make deploy`. Existing functionality remains available when the extension has not been enabled, while archive endpoints return 503. Migration is repeatable. Do not repair a failure by creating an empty database. `check` accepts the original core database and, when the extension exists, also checks its version, tables, SQLite integrity, and foreign keys.

Original text does not expire or delete automatically. Snapshots grow when content changes. gzip compresses repeated source content, but different snapshots are still stored independently. Any actual deletion or compaction of historical archives requires a separately designed, explicit data-maintenance operation.
