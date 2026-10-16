# AICalendar Maintenance Entry Point

This project is a personal AI activity calendar responsible for storing, presenting, searching, and idempotently importing activity records, full conversations, and original-source archives. Platform exports and material organization belong to an external agent and the `ai-calendar-import` skill; do not add platform connectors, hooks, or automatic collection to the server.

- Read `docs/README.md` first. Before server work, use server-operations and explicitly read remote `/opt/AGENTS.md`.
- Code and configuration are stored in this public repository. Real records, original chats, credentials, production addresses, and backups must not enter Git.
- Import counts are snapshot values; `source + external_id` is unique. Preserve version checks, idempotency, whole-batch transactions, unknown-value semantics, and separation between manual annotations and source data.
- The import token authorizes only activity/archive import, download, and verification under `/ingest/v1/`. Browser pages and `/api/v1/` inherit portal device authentication. Follow the server-operations authorization table for change boundaries.
- Only `init` explicitly creates a database. `serve`, `check`, and `backup` must not create an empty database that hides missing data. SQLite backup uses a consistent snapshot; restoration writes only to a new path that does not exist.
- Build and release after validating the current functionality locally. There is no full-regression gate, and permanent tests are not retained by default. Validate import coverage and backup restoration with isolated copies; production acceptance is read-only.
- By default, web pages block Tab / Shift+Tab focus traversal and remove focus decoration. Preserve mouse behavior, text input, selection, modifier keys, and IME behavior. Check desktop and 375×667 narrow-screen layouts.
- When APIs, data directories, ports, units, backups, or paths change, update `deploy/portal.json`, project docs, and the shared server inventory together. GitHub backs up only source; local release and documentation synchronization are separate operations.
- When maintaining API contracts, update `api/import.schema.json`, `api/archive.schema.json`, `api/openapi.json`, and `docs/API.md`, then check the personal skill client. Do not maintain duplicate schemas on both sides.

Agent specifications, maintenance documentation, technical plans, handoffs, and similar agent-authored operational documents default to English unless the user explicitly requests another language. User-facing product content follows the language requested by the user.
