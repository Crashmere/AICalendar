# Local Deployment

Follow [server-operations release](https://github.com/Crashmere/agent-config/blob/main/skills/server-operations/references/release.md). This repository is public. GitHub stores only source and configuration; it has no Actions release or production Secrets.

## First Installation

Complete local validation, commit and push, and run `make release` first. Under the shared rules, confirm the exact installation of new runtime/deployment identities, import authentication, and Nginx routing. First read remote `/opt/AGENTS.md` and verify that port 18086, the application root, and usernames are all unused.

Generate the import token privately on the maintenance side; install only its SHA-256 file on the server. Transfer the verified Linux amd64 program, deploy materials, hash, and existing deployment public key, but not the administrator private key. Verify the program's SHA-256.

```sh
sudo bash deploy/install.sh <linux-program> <https-origin> <token-hash-file> <deploy-public-key>
```

The script is for an explicitly new empty service, not migration of existing production data. It rejects an existing directory, account, or port, creates the database and restricted deployment identity, starts only the application unit, verifies loopback health, then adds this application's Nginx location link and validates the reload. It installs legacy backup unit files but does not enable the timer or create a local daily snapshot. It does not modify the shared Nginx server itself or restart other applications. If Nginx syntax validation fails, remove the new link and do not force a reload.

After first installation, use normal `make deploy` to register the committed program version and portal declaration. Update the shared inventory, push, and synchronize project and shared documentation. Keep initialization materials and local credentials outside Git.

## Updates

```sh
npm --prefix web ci
make build
# Validate the current feature, then commit and push to origin/main.
make release
make deploy
```

`release.sh` reuses `server-operations/scripts/release.py`, with configuration in `release.json`. It builds only committed files. Artifacts are stored in `.local/releases/<commit>/` and include the program, declaration, hashes, and build manifest. Release requires a clean tracked working tree matching the remote backup.

Temporary upgrade backups follow the shared [database-only policy](https://github.com/Crashmere/agent-config/blob/main/skills/server-operations/references/retention.md#temporary-upgrade-backups): skip them unless the change can delete or overwrite existing database data, and never include non-database files. The release helper records selected database paths and reasons; [database selection](https://github.com/Crashmere/agent-config/blob/main/skills/server-operations/references/retention.md#tool-alignment) owns capture and acceptance.

The restricted SSH user may execute only `deploy`, `portal-check` and `portal`; it cannot edit root-owned scripts. On `nas`, releases take shared maintenance and application deployment locks, verify the upload, stop AICalendar and check existing data as `aicalendar`. Ordinary releases create no temporary copy. A declared deletion/overwrite operation protects only selected SQLite databases under `/var/lib/server-upgrades/aicalendar/<release>/`; the complete snapshot includes archive BLOBs and committed WAL content. Runtime health permits immediate acceptance without a backup. Selected databases additionally pass cold preservation checks and final runtime health before removal of only the accepted protection. Failures retain captured databases; program rollback does not restore data. Logs record `recovery` and `result`; see the [shared release workflow](https://github.com/Crashmere/agent-config/blob/main/skills/server-operations/references/release.md).

For the `ali` to `nas` migration, retain the existing paths, runtime identity, database schema, and import-token hash. Transfer a final consistent database only after `ali` writes stop; do not use `init` for that migration. The same `nas` instance serves all entrances. Configure `AICALENDAR_ORIGIN` for the retained primary address and `AICALENDAR_ALLOWED_ORIGINS` for the `vps` HTTPS address with `:8443` and the `nas` LAN HTTPS address. The application remains loopback-only; Nginx and the reverse tunnel belong to shared server configuration. Release tooling must distinguish the deployment host `nas` from public checks on `ali` and `vps`.

When database protection is required, its consistent snapshot includes every conversation/archive table, index, and unfinished upload stored in that database; an older summary-only backup is not suitable protection. No backup timer or scheduled pruning is enabled on `nas`, and preserved historical `ali` backups are not part of routine cleanup.

### First Activation of Full Conversation Archives

This version adds `archive_schema` 1 while core `user_version` remains 1. First validate migration and old-program compatibility against an isolated database, then perform one explicit extension under shared authorization. Ordinary `make deploy` does not migrate the database automatically.

1. Complete current validation, commit and push, and run `make release`. Through administrator SSH, upload the same-commit `program` into a separate staging directory, verify the SHA-256 in the manifest, and make the candidate executable by the `aicalendar` runtime user.
2. Hold the shared maintenance and application deployment locks and stop affected writes. This extension only adds tables without rewriting existing data, so it does not itself require a temporary backup. If the actual candidate adds a concrete deletion or overwrite risk, capture only a consistent complete `aicalendar.sqlite` snapshot under `/var/lib/server-upgrades/aicalendar/<release>/`; a summary-only old binary cannot protect existing archives. Record the old activity/annotation counts and content digests for the affected-data checks.
3. As the `aicalendar` identity, run the candidate's `migrate-archives --db /opt/aicalendar/data/aicalendar.sqlite`, then run `check` with both the candidate and old program. Verify that core `user_version` is unchanged, old activities/annotations are unchanged, and the new tables contain no historical records. Migration only adds tables and may be repeated.
4. Release the administrative lock handles, then run normal `make deploy`, which reacquires the release locks; retain any database protection required by the explicit extension through final acceptance. Verify the archive schema, authenticated listing, unauthenticated rejection, portal declaration, and affected data. Only after that acceptance, remove this extension's staged program, any accepted extension database protection, and the accepted program upgrade's temporary copies. If verification fails or is incomplete, retain protection. Compatible old programs can retain archive tables and bytes during program rollback, but archive endpoints and native backup coverage depend on their version.

This extension reuses the original database and every backup. It adds no storage root and changes neither Nginx request limits, runtime identity, nor the root release script. See [ARCHIVES.md](ARCHIVES.md) for format and integrity constraints.

## Rollback and Configuration

`make rollback COMMIT=<full-old-commit>` uses the same release protections. First confirm that the old program is compatible with the schema. Real data recovery requires separate confirmation and uses a verified snapshot restored to a new directory.

Units, env, token hashes, Nginx, and root release scripts do not change with an ordinary program update; inspect and install them under the shared authorization table when necessary. Do not store a production address, token, or database in the repository.

Release acceptance covers loopback health, web deep link, unauthenticated page/API, invalid token, public icons, authenticated import read query, portal declaration receipt, and runtime/affected-data preservation. For the migration, also verify all configured Origins and the initial independent complete recovery set; actual external-volume acceptance is a separate check once a drive is configured. Do not start the legacy backup unit merely as an acceptance check. After pushing managed documentation, synchronize it with `SYNC_HOST=nas ~/agent-config/skills/server-operations/scripts/sync-docs.sh AICalendar`.
