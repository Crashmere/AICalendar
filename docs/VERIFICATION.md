# Verification Scope

## Migration Contracts: 2026-10-29

Local synthetic checks verified exact HTTPS Origins for the retained `ali` entrance, `vps` on port 8443, and the `nas` LAN entrance, plus legacy single-Origin configuration. Unrelated hosts, incorrect ports, HTTP, missing Origin, and malformed configured Origins were rejected. The independent ingest Bearer gate remained in place.

Complete native backup, restore to a new path, and integrity checks passed while the source retained committed WAL data. Calendar/import tables, archive manifests, compressed conversation bytes with a 512 KiB binary original, message indexes, and incomplete uploaded chunks survived recovery. Existing backup destinations were not overwritten and missing sources were not initialized. These checks used temporary synthetic data, not production writes; migration and real-device acceptance are separate operations.

## Initial Delivery

2026-10-07: the first version was deployed to `/aicalendar/` on `ali`, and the public source repository was created. The production service used a newly created empty database.

Verified: frontend type checking and build, Go build; generation of a fixed plan by the skill from synthetic JSON, batch import, persisted receipts, and record read-back; deduplication when retrying the same key, rejection of different requests under the same key, optimistic-version updates, whole-batch rollback on conflict, rejection of reducing a known count, preservation of manual title/tags and hidden state across reimport, and Origin/token rejection paths.

SQLite `backup` → `restore` to a new path → `check` was run against synthetic data. Real business data was not used in write tests.

The personal skill passed official format validation. Local read-only Trae extraction can generate private candidate files and fragment context and explicitly reports missing logs, fallback behavior, and incomplete coverage. Extracted results were not imported into production.

Chromium verified 1440px desktop and 375×667 phone layouts: no horizontal overflow or page-script errors; search, calendar, editing, and import dialogs worked; ordinary Tab / Shift+Tab retained input focus with no focus outline. The phone check is narrow-screen browser verification and must not be presented as real-iPhone Home Screen acceptance.

An independent agent used a dedicated synthetic source to exercise the skill end to end: first import, replay of the same plan, regeneration of a plan from the same input, updating a count from 2 to 3 without adding it to become 5, an unknown count, replaying an old plan after the update, and a count-reduction conflict all matched the protocol. When an old receipt encounters a newer version it reports `superseded_records` rather than a false failure; conflict preview exits with code 2. Real network interruption and large-scale concurrency were not simulated.

Completed in production: normal release through the restricted identity, program-version registration, portal-declaration preflight and loaded-hash receipt, read-only queries and previews through the independent import credential, systemd boot enablement, and the first 04:15 daily backup plus integrity check. The production database was still empty; none of the real chats from research were imported. Loopback health for all seven sites and public unauthenticated API checks passed. Page navigation redirected to portal login, and the import Bearer could not access the browser API. Anonymous icon GET/HEAD plus type/dimension checks passed. Shared Nginx configuration validation and graceful reload succeeded, and existing applications were not restarted.

Backup recovery was validated with a local synthetic database; it must not be described as restoration of real production data or whole-server recovery acceptance. The previously deferred portal-backup download/recovery task was not started.

## Full Conversations and Source Labels

Local verification on 2026-10-08: full archives preserved normalized messages and original materials. A 468000-byte Unicode/emoji long message, an empty message, structured tool text, and a 1200000-byte binary original were compared byte-for-byte or message-by-message after download and unpacking. Original filenames were preserved correctly. Repeated uploads of identical content did not add a snapshot, while appending a message retained both old and new versions.

An independent agent completed pack → prepare → submit → download → unpack through the skill, followed by calendar-summary import. After the first chunk was accepted, process exit was simulated; the new process uploaded only missing chunks. Incomplete snapshots did not appear in the list, and download returned 409. Tampering with the fixed plan or original file, and overwriting an existing plan or unpack directory, were all rejected. An archive deliberately constructed with a correct outer hash but an incorrect original-source hash returned 400 on commit and left no visible message index.

Source labels were displayed in calendar, archive list, and details; calendar `q`/`tag` and archive `label` filtering worked. Leading/trailing whitespace normalization worked. Updating only a label on the same archive left its ID and archive count unchanged. After a calendar-label upsert, record ID and counts remained unchanged. Chromium desktop and 375×667 phone emulation covered opening linked conversations, the archive pagination interface, source filtering, Tab/Shift+Tab, no focus outline, no horizontal overflow, and HTML escaping, with no page-script errors.

After SQLite consistent backup → restoration to a new path → `check`, the archived compressed-byte hash was unchanged. Repeated additive migration of an old synthetic database left core `user_version` at 1 and every activity and annotation column unchanged. Separately, one small real Trae conversation was used only locally to verify three user messages, three AI messages, and the hashes of two source files; it was not uploaded to the service.

Frontend type checking/Vite, Go build, and `go vet` passed. The 128 MiB/512 MiB capacity limits, large-scale concurrency, and a real public-network outage were not tested; simulated interruption is not equivalent to these scenarios.

Commit `6159b2cfc77633d0a3faa72c30590ba938969101` was released on 2026-10-08. Before upgrade, the old program created an independent consistent backup. A candidate from the same commit ran the additive migration as the runtime user, and both new and old programs passed `check`. Core `user_version` remained 1. Counts and content digests for the activity, batch, and change tables were identical, and `archive_schema` was 1. The normal restricted release created another backup containing the extension, which passed integrity checks with both the candidate and deployed program.

Production loopback health, the deployed frontend's conversation archive/source-label functionality, and import-credential reads of both schemas and empty lists all passed.

## Time Range, Presentation, and Backup Scope

Local verification on 2026-10-09: through a read-only tunnel, the frontend read production data. Chromium desktop at 1440px and a 375×667 layout checked month/quarter/year/custom/all-time ranges, previous/next shifting, URL restoration, search across all time with out-of-range notices, the annual heatmap, tag multi-select, read-only details, the archived-conversation list, and the import-coverage timeline. There was no horizontal overflow, text was at least 11px, Tab / Shift+Tab was intercepted, and there was no focus outline. Reader spot checks covered TraeX, ChatGPT, and Cursor conversations: JSON bodies, reasoning and tool calls, and citation markers were handled as expected, and Markdown did not overflow. Editing and saving were verified only with a local synthetic database: it sent only PATCH and updated the list in place.

The backend was verified against a temporary synthetic database: archive title/source-label `q`, message pagination with `limit=200` and `has_more`, the approximately 2 MiB page cutoff, JSON gzip and `Vary`, and uncompressed downloads. Backups copied only the activity, batch, and change tables. Those tables matched the source, archive tables were empty, `check` passed, restoration to a new path passed, and backup worked while the service was running.

Commit `d66d65fd4251b1f29ac8ac55470c3c16cfe72c26` was released on 2026-10-09. The old program took approximately two minutes to create a complete pre-release backup, so release-script backup/verification timeouts were increased to 600/300 seconds and reinstalled. Loopback health, gzip (activity list 2.1 MB → 354 KB), archive `q`, long-term resource caching, unauthenticated page/API 401 responses, anonymous icon access, and the portal loaded-hash receipt all passed. A manual run of the daily backup had `Result=success`: 7.8 MB, passed `check`, and had the same three-table row counts as production. After user confirmation, the old complete daily backup and this complete pre-release backup were deleted, reducing the backup directory to approximately 7.6 MB. Unauthenticated archive list/schema/download and calendar API requests returned 401. The import credential could not access the browser API. The portal loaded new declaration hash `42e536a64f4545274ddd1fd65dbbea9ea49eed6301d39969c96294467d62d1bc`. The service and daily backup timer were normal, with the most recent backup showing `Result=success`. Production activities and archives were still both 0; this work did not upload real history or use production writes for testing.
