PRAGMA user_version=1;
CREATE TABLE activities (
 id TEXT PRIMARY KEY, source TEXT NOT NULL, external_id TEXT NOT NULL,
 activity_date TEXT NOT NULL, record_json TEXT NOT NULL, record_hash TEXT NOT NULL,
 version INTEGER NOT NULL, annotation_json TEXT NOT NULL DEFAULT '{}',
 annotation_version INTEGER NOT NULL DEFAULT 0, updated_at TEXT NOT NULL,
 UNIQUE(source, external_id)
);
CREATE INDEX activities_date ON activities(activity_date, source);
CREATE TABLE import_batches (
 id TEXT PRIMARY KEY, idempotency_key TEXT NOT NULL UNIQUE,
 request_hash TEXT NOT NULL, result_json TEXT NOT NULL, created_at TEXT NOT NULL
);
CREATE TABLE import_changes (
 batch_id TEXT NOT NULL REFERENCES import_batches(id), activity_id TEXT NOT NULL,
 before_json TEXT, after_json TEXT NOT NULL,
 PRIMARY KEY(batch_id, activity_id)
);
