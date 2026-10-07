CREATE TABLE IF NOT EXISTS archive_schema (version INTEGER NOT NULL);
INSERT INTO archive_schema(version) SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM archive_schema);
CREATE TABLE IF NOT EXISTS conversation_archives (
 id TEXT PRIMARY KEY, source TEXT NOT NULL, conversation_id TEXT NOT NULL,
 manifest_json TEXT NOT NULL, manifest_hash TEXT NOT NULL, state TEXT NOT NULL,
 created_at TEXT NOT NULL, committed_at TEXT
);
CREATE INDEX IF NOT EXISTS archives_conversation ON conversation_archives(source,conversation_id,committed_at);
CREATE TABLE IF NOT EXISTS archive_parts (
 archive_id TEXT NOT NULL REFERENCES conversation_archives(id), part_index INTEGER NOT NULL,
 sha256 TEXT NOT NULL, data BLOB NOT NULL, PRIMARY KEY(archive_id,part_index)
);
CREATE TABLE IF NOT EXISTS archive_messages (
 archive_id TEXT NOT NULL REFERENCES conversation_archives(id), ordinal INTEGER NOT NULL,
 message_json BLOB NOT NULL, PRIMARY KEY(archive_id,ordinal)
);
