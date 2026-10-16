# Local Deployment

Follow [server-operations release](https://github.com/Crashmere/agent-config/blob/main/skills/server-operations/references/release.md). This repository is public. GitHub stores only source and configuration; it has no Actions release or production Secrets.

## First Installation

Complete local validation, commit and push, and run `make release` first. Under the shared rules, confirm the exact installation of new runtime/deployment identities, import authentication, and Nginx routing. First read remote `/opt/AGENTS.md` and verify that port 18086, the application root, and usernames are all unused.

Generate the import token privately on the maintenance side; install only its SHA-256 file on the server. Transfer the verified Linux amd64 program, deploy materials, hash, and existing deployment public key, but not the administrator private key. Verify the program's SHA-256.

```sh
sudo bash deploy/install.sh <linux-program> <https-origin> <token-hash-file> <deploy-public-key>
```

The script rejects an existing directory, account, or port. It creates an empty database, starts the unit and 04:15 backup timer, creates the restricted deployment identity, verifies loopback health and the first backup, then adds this application's Nginx location link and validates the reload. It does not modify the shared Nginx server itself or restart other applications. If Nginx syntax validation fails, remove the new link and do not force a reload.

After first installation, use normal `make deploy` to register the committed program version and portal declaration. Update the shared inventory, push, and synchronize project and shared documentation. Keep initialization materials and local credentials outside Git.

## Updates

```sh
npm --prefix web ci
make build
# 进行本次功能验证后，提交并 push origin main
make release
make deploy
```

`release.sh` reuses `server-operations/scripts/release.py`, with configuration in `release.json`. It builds only committed files. Artifacts are stored in `.local/releases/<commit>/` and include the program, declaration, hashes, and build manifest. Release requires a clean tracked working tree matching the remote backup.

The restricted SSH user may execute only `deploy`, `portal-check`, and `portal`. Uploaders cannot modify the fixed root script. A release verifies size/hash, stops this application, backs up the old program, checks the candidate as the runtime user, then atomically replaces and starts it. On failure it restores the old program without rolling back the database. Logs record `recovery` and `result`.

### First Activation of Full Conversation Archives

This version adds `archive_schema` 1 while core `user_version` remains 1. First validate migration and old-program compatibility against an isolated database, then perform one explicit extension under shared authorization. Ordinary `make deploy` does not migrate the database automatically.

1. Complete current validation, commit and push, and run `make release`. Through administrator SSH, upload the same-commit `program` into a separate staging directory, verify the SHA-256 in the manifest, and make the candidate executable by the `aicalendar` runtime user.
2. Use the current production program's `backup` command to create a consistent `backups/before-archive-extension-<timestamp>.sqlite` snapshot. Validate it with the old program's `check` and record the old activity/annotation counts and content digests.
3. As the `aicalendar` identity, run the candidate's `migrate-archives --db /opt/aicalendar/data/aicalendar.sqlite`, then run `check` with both the candidate and old program. Verify that core `user_version` is unchanged, old activities/annotations are unchanged, and the new tables contain no historical records. Migration only adds tables and may be repeated.
4. Run normal `make deploy`. Verify the archive schema, authenticated listing, unauthenticated rejection, and portal declaration. Remove this task's staged program and retain the pre-upgrade snapshot. The old program can be restored and will retain archive tables and bytes, but it has no archive endpoints.

This extension reuses the original database and every backup. It adds no storage root and changes neither Nginx request limits, runtime identity, nor the root release script. See [ARCHIVES.md](ARCHIVES.md) for format and integrity constraints.

## Rollback and Configuration

`make rollback COMMIT=<完整旧提交>` uses the same release protections. First confirm that the old program is compatible with the schema. Real data recovery requires separate confirmation and uses a verified snapshot restored to a new directory.

Units, env, token hashes, Nginx, and root release scripts do not change with an ordinary program update; inspect and install them under the shared authorization table when necessary. Do not store a production address, token, or database in the repository.

Release acceptance: loopback health, web deep link, unauthenticated page/API, invalid token, public icons, authenticated import read query, portal declaration receipt, and `backup.service` Result. Synchronize documentation separately with the shared `sync-docs.sh`.
