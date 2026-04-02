PRAGMA foreign_keys = OFF;

CREATE TABLE upload_sessions_new (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	collection_id TEXT NOT NULL,
	collection_kind TEXT NOT NULL,
	source_format TEXT NOT NULL CHECK (source_format IN ('pdf', 'xlsx', 'csv')),
	document_type TEXT,
	status TEXT NOT NULL CHECK (
		status IN (
			'created',
			'receiving',
			'processing',
			'finalized',
			'awaiting_resolution',
			'completed',
			'failed',
			'interrupted',
			'expired'
		)
	),
	total_chunks INTEGER NOT NULL DEFAULT 0,
	uploaded_chunks INTEGER NOT NULL DEFAULT 0,
	processed_chunks INTEGER NOT NULL DEFAULT 0,
	failed_chunks INTEGER NOT NULL DEFAULT 0,
	duplicate_chunks INTEGER NOT NULL DEFAULT 0,
	last_heartbeat_at DATETIME,
	started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	finished_at DATETIME,
	expires_at DATETIME,
	client_session_key TEXT,
	metadata_json TEXT,
	FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY(collection_id) REFERENCES collections(id) ON DELETE CASCADE
);

INSERT INTO upload_sessions_new (
	id, user_id, collection_id, collection_kind, source_format, document_type, status,
	total_chunks, uploaded_chunks, processed_chunks, failed_chunks, duplicate_chunks,
	last_heartbeat_at, started_at, finished_at, expires_at, client_session_key, metadata_json
)
SELECT
	id, user_id, collection_id, collection_kind, source_format, document_type, status,
	total_chunks, uploaded_chunks, processed_chunks, failed_chunks, duplicate_chunks,
	last_heartbeat_at, started_at, finished_at, expires_at, client_session_key, metadata_json
FROM upload_sessions;

DROP TABLE upload_sessions;

ALTER TABLE upload_sessions_new RENAME TO upload_sessions;

CREATE UNIQUE INDEX idx_upload_sessions_client_key
ON upload_sessions (collection_id, client_session_key)
WHERE client_session_key IS NOT NULL;

CREATE INDEX idx_upload_sessions_collection_status
ON upload_sessions (collection_id, status);

CREATE INDEX idx_upload_sessions_kind_format
ON upload_sessions (collection_id, collection_kind, source_format, status);

PRAGMA foreign_keys = ON;
