# Operations

## Service Shape

The project is AICalendar, listening on loopback `127.0.0.1:18086` at URL `/aicalendar/`, with root directory `/opt/aicalendar`. It has dedicated runtime user `aicalendar` and restricted deployment user `aicalendar-deploy`. See the [Verification Record](VERIFICATION.md) for current delivery status.

- `bin/aicalendar`: single-file Go service with the frontend and API schemas embedded.
- `config/aicalendar.env`: database, listener, prefix, production Origin, and token-hash file location.
- `config/import-token.sha256`: SHA-256 of the import token, root:aicalendar 0640; plaintext exists only in a private directory on the maintenance side.
- `data/aicalendar.sqlite`: records, annotations, import audits, full-conversation archives/upload chunks/reading indexes; `data` and `backups` are 0700.
- `backups/`: native consistent SQLite snapshots.
- `releases/`: release materials under the shared 3/5 retention policy.

The service process has `MemoryMax=192M` and `GOMEMLIMIT=144MiB`, and the application writes only its own `data`. The frontend is built on the maintenance computer; Node/Go is not installed on the server.

## Startup and Diagnostics

`serve` does not create or migrate a database automatically. The full-conversation extension is added explicitly with `migrate-archives`; see [ARCHIVES.md](ARCHIVES.md) for upgrade and old-program compatibility. Only `init` may explicitly create a new database, and the destination must not exist. Diagnose first with `systemctl status aicalendar`, `journalctl -u aicalendar`, loopback `/healthz`, and configuration paths. Do not “repair” data loss by creating an empty database.

An unauthenticated public request to a page/API should receive a login redirect or 401 from the portal. `/ingest/v1/activities` must return 401 without a token and may return application data only with a token. The Origin of web write requests must exactly match the production address.

Token rotation: in a private location, use `aicalendar token --out <new-private-token>` to generate a new token and hash. After authorization is confirmed, replace only the server hash file, restart this service, and update the maintenance-side `token_file`. The old token becomes invalid. Portal device credentials do not change.

## Backup and Recovery

`aicalendar-backup.timer` runs daily at 04:15 Beijing time with a 0–5-minute randomized delay. `daily` retains 14 copies, rotating only after a new snapshot succeeds and passes checks. `backup.service` uses the shared retention lock. The portal registry automatically discovers the native SQLite backup contract.

```sh
aicalendar backup --db /path/aicalendar.sqlite --out /new/path/backup.sqlite
aicalendar check --db /path/backup.sqlite
aicalendar restore --from /path/backup.sqlite --db /new/path/restored.sqlite
```

Within one read transaction, `backup` copies `activities`, `import_batches`, and `import_changes` into a new database, including committed WAL data; the destination must not exist. User confirmation on 2026-10-09: full conversation archives are large and have limited review value, so they are not backed up. A snapshot creates only empty archive tables. A restored database can accept new archives, but previous chats and original exports are not restored. `check` uses a read-only connection to check schemas, integrity, foreign keys, and required tables. `restore` uses `VACUUM INTO` to copy the specified snapshot completely to a separate new path. Program rollback does not restore data; real restoration requires confirmation of the recovery point and a write-stop replacement procedure.

Daily snapshots reside on the same disk as production. Off-machine protection follows the server's existing arrangement; do not present a same-disk snapshot as an off-machine backup. Full conversations and original exports exist only in the production database. The user must retain local copies of original material that requires long-term preservation; local material not yet uploaded is also outside the service's backup scope.

## Portal, Icons, and Documentation

The resource declaration's maintained source is `deploy/portal.json`. A normal release preflights and synchronizes it to `/opt/aicalendar/config/portal.json`, registers it through `registry.d`, and verifies the load receipt. When APIs or resources change, update the declaration and docs together.

The site `favicon.svg` and opaque 180×180 `apple-touch-icon.png` are in `web/public`; the latter can be rebuilt with `python3 scripts/generate-icon.py`. Nginx allows anonymous GET/HEAD only for these two files. After release, verify them with the shared `check-site-icons.py`.

Read shared rules, release, authentication, and retention policies from [server-operations](https://github.com/Crashmere/agent-config/tree/main/skills/server-operations). After committing and pushing, run its `scripts/sync-docs.sh AICalendar`; an ordinary binary release does not synchronize docs.
