# Operations

## Service Shape

The project is AICalendar, listening on loopback `127.0.0.1:18086` at URL `/aicalendar/`, with root directory `/opt/aicalendar`. It has dedicated runtime user `aicalendar` and restricted deployment user `aicalendar-deploy`. The migration target is one application instance on `nas`, reached directly over LAN HTTPS and through `ali` HTTPS or `vps` HTTPS on port 8443. Shared server current-state records whether production has switched; source changes do not imply deployment.

- `bin/aicalendar`: single-file Go service with the frontend and API schemas embedded.
- `config/aicalendar.env`: database, listener, prefix, allowed HTTPS Origins, and token-hash file location.
- `config/import-token.sha256`: SHA-256 of the import token, root:aicalendar 0640; plaintext exists only in a private directory on the maintenance side.
- `data/aicalendar.sqlite`: records, annotations, import audits, full-conversation archives/upload chunks/reading indexes; `data` and `backups` are 0700.
- `backups/`: preserved legacy native snapshots; no routine NAS backup or upgrade protection is created here.
- `releases/`: program release materials, separate from temporary data protection; NAS has no scheduled release pruning.
- `/var/lib/server-upgrades/aicalendar/<release>/`: root-private complete cold `data`, `config`, and `bin` protection. Remove only the accepted upgrade's copy after runtime and affected-data validation; retain failed or unverified protection.

The service process has `MemoryMax=192M` and `GOMEMLIMIT=144MiB`, and the application writes only its own `data`. The frontend is built on the maintenance computer; Node/Go is not installed on the server.

## Startup and Diagnostics

`serve` does not create or migrate a database automatically. The full-conversation extension is added explicitly with `migrate-archives`; see [ARCHIVES.md](ARCHIVES.md) for upgrade and old-program compatibility. Only `init` may explicitly create a new database, and the destination must not exist. Diagnose first with `systemctl status aicalendar`, `journalctl -u aicalendar`, loopback `/healthz`, and configuration paths. Do not “repair” data loss by creating an empty database.

An unauthenticated public request to a page/API should receive a login redirect or 401 from the portal. `/ingest/v1/activities` must return 401 without a token and may return application data only with a token. Web writes require one exact allowed HTTPS Origin, including any explicit port. `AICALENDAR_ORIGIN` remains the primary Origin; `AICALENDAR_ALLOWED_ORIGINS` adds a comma-separated list. Either may supply the allowed set, and invalid entries fail startup. Configure the `ali` address, `vps` address with `:8443`, and `nas` LAN address during coexistence. There is no wildcard, subnet-based authentication, cross-Origin cookie sharing, or CORS exception. `--demo` alone retains loopback HTTP development.

Token rotation: in a private location, use `aicalendar token --out <new-private-token>` to generate a new token and hash. After authorization is confirmed, replace only the server hash file, restart this service, and update the maintenance-side `token_file`. The old token becomes invalid. Portal device credentials do not change.

## Backup and Recovery

Use the agent-owned [backup and recovery runbook](https://github.com/Crashmere/agent-config/blob/main/skills/server-operations/references/backup-recovery.md), also at `/opt/server-context/references/backup-recovery.md` on NAS. Portal provides no backup or recovery operation.

On `nas`, routine backups are manual, compressed, complete recovery sets written to a verified external filesystem through agent-only server-operations tooling. The controlled job pauses business writers, cold-copies complete data/config/program state externally, resumes previously running services, then compresses and verifies that immutable copy. An unavailable external volume disables the job without an internal fallback. Staging and archive verification stay external; there is no schedule or rotation. The legacy `aicalendar-backup.timer` remains disabled, and preserved `ali` backups are not automatically deleted. The native commands below remain complete database tools for legacy packages and isolated recovery; they do not create a whole-application recovery set or replace the shared NAS upgrade helper.

```sh
aicalendar backup --db /path/aicalendar.sqlite --out /new/path/backup.sqlite
aicalendar check --db /path/backup.sqlite
aicalendar restore --from /path/backup.sqlite --db /new/path/restored.sqlite
```

`backup` uses SQLite `VACUUM INTO` to capture one complete consistent database, including committed WAL data, every table/index, annotations, import audit, conversation manifests, original-material chunks, message indexes, and unfinished uploads. It does not add or enable an archive extension missing from the source. The destination must not exist, has mode 0600, is checked for schema/integrity/foreign keys and synchronized before success; a failed new snapshot is removed. `check` uses a read-only connection. `restore` applies the same complete-copy checks to a new path. Never copy only the main file of a running WAL database.

Older binaries with summary-only backup behavior cannot protect archives during the first upgrade to this version. NAS upgrade protection must include the complete cold application tree, not an old native snapshot. Validate runtime behavior and affected data before deleting that upgrade's private protection; health alone is insufficient. Failures or incomplete verification retain protection. Program rollback does not restore data; an agent must confirm the recovery point and restore into an isolated path before replacing production.

The agent-owned NAS external recovery set includes programs, private configuration, token hashes, identities, and shared NAS recovery materials, including its local CA and tunnel configuration. The agent captures NAS and VPS gateway/proxy state separately; frozen ALI materials remain unchanged. A native SQLite file alone is not a whole-application rebuild package. Material not yet uploaded and maintenance-side plaintext tokens remain outside the database; retain them separately or explicitly include them in an agent backup.

## Portal, Icons, and Documentation

The resource declaration's maintained source is `deploy/portal.json`. A normal release preflights and synchronizes it to `/opt/aicalendar/config/portal.json`, registers it through `registry.d`, and verifies the load receipt. When APIs or resources change, update the declaration and docs together.

The site `favicon.svg` and opaque 180×180 `apple-touch-icon.png` are in `web/public`; the latter can be rebuilt with `python3 scripts/generate-icon.py`. Nginx allows anonymous GET/HEAD only for these two files. After release, verify them with the shared `check-site-icons.py`.

Read shared rules, release, authentication, and retention policies from [server-operations](https://github.com/Crashmere/agent-config/tree/main/skills/server-operations). After committing and pushing managed documentation, run `SYNC_HOST=nas ~/agent-config/skills/server-operations/scripts/sync-docs.sh AICalendar`; use `nas-via-vps` when the VPS management transport is selected. An ordinary binary release does not synchronize docs. Synchronize changed shared gateway context separately to `vps`; frozen `ali` receives no further documentation maintenance.
