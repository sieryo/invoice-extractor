PRAGMA foreign_keys = OFF;

DROP TABLE IF EXISTS collection_action_outputs;
DROP TABLE IF EXISTS collection_action_items;
DROP TABLE IF EXISTS collection_actions;
DROP TABLE IF EXISTS collection_history_items;
DROP TABLE IF EXISTS collection_history;
DROP TABLE IF EXISTS upload_session_chunks;
DROP TABLE IF EXISTS upload_sessions;
DROP TABLE IF EXISTS documents;
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS collections;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;

PRAGMA foreign_keys = ON;

CREATE TABLE users (
	id TEXT PRIMARY KEY,
	username TEXT UNIQUE NOT NULL,
	password_hash TEXT,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	expires_at DATETIME NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE collections (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	parent_id TEXT,
	name TEXT NOT NULL,
	node_type TEXT NOT NULL CHECK (node_type IN ('folder', 'collection')),
	document_type TEXT CHECK (document_type IN ('pdf_invoice', 'pdf_tax_invoice', 'pdf_bppu', 'pdf_bp21', 'pdf_bpa1')),
	phase TEXT NOT NULL DEFAULT 'ready' CHECK (phase IN ('ready', 'uploading', 'processing')),
	total_count INTEGER NOT NULL DEFAULT 0,
	ready_count INTEGER NOT NULL DEFAULT 0,
	warning_count INTEGER NOT NULL DEFAULT 0,
	failed_count INTEGER NOT NULL DEFAULT 0,
	duplicate_count INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	deleted_at DATETIME,
	deleted_by TEXT,
	delete_reason TEXT,
	FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY(parent_id) REFERENCES collections(id) ON DELETE CASCADE,
	CHECK (id <> parent_id),
	CHECK (
		(node_type = 'folder' AND document_type IS NULL) OR
		(node_type = 'collection' AND document_type IS NOT NULL)
	)
);

CREATE UNIQUE INDEX idx_collections_unique_name_active
ON collections (user_id, COALESCE(parent_id, ''), name)
WHERE deleted_at IS NULL;

CREATE INDEX idx_collections_parent_active
ON collections (parent_id, deleted_at);

CREATE INDEX idx_collections_user_active
ON collections (user_id, deleted_at);

CREATE TABLE documents (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	collection_id TEXT NOT NULL,
	document_type TEXT NOT NULL CHECK (document_type IN ('pdf_invoice', 'pdf_tax_invoice', 'pdf_bppu', 'pdf_bp21', 'pdf_bpa1')),
	document_tag TEXT,
	source_name TEXT NOT NULL,
	source_size_bytes INTEGER,
	source_mime TEXT,
	source_sha256 TEXT NOT NULL,
	source_order INTEGER NOT NULL,
	status TEXT NOT NULL CHECK (status IN ('ready', 'warning')),
	message TEXT,
	normalized_ref TEXT NOT NULL,
	audit_ref TEXT,
	raw_ref TEXT,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	deleted_at DATETIME,
	deleted_by TEXT,
	delete_reason TEXT,
	FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY(collection_id) REFERENCES collections(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_documents_dedup_active
ON documents (collection_id, document_type, source_sha256)
WHERE deleted_at IS NULL;

CREATE INDEX idx_documents_collection_status_active
ON documents (collection_id, status, deleted_at);

CREATE INDEX idx_documents_source_order_active
ON documents (collection_id, source_order, deleted_at);

CREATE TABLE upload_sessions (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	collection_id TEXT NOT NULL,
	document_type TEXT NOT NULL CHECK (document_type IN ('pdf_invoice', 'pdf_tax_invoice', 'pdf_bppu', 'pdf_bp21', 'pdf_bpa1')),
	status TEXT NOT NULL CHECK (
		status IN (
			'created',
			'receiving',
			'processing',
			'finalized',
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

CREATE UNIQUE INDEX idx_upload_sessions_client_key
ON upload_sessions (collection_id, client_session_key)
WHERE client_session_key IS NOT NULL;

CREATE INDEX idx_upload_sessions_collection_status
ON upload_sessions (collection_id, status);

CREATE TABLE upload_session_chunks (
	id TEXT PRIMARY KEY,
	session_id TEXT NOT NULL,
	chunk_index INTEGER NOT NULL,
	status TEXT NOT NULL CHECK (
		status IN ('uploaded', 'queued', 'processing', 'done', 'failed', 'duplicate')
	),
	idempotency_key TEXT NOT NULL,
	request_checksum TEXT,
	file_count INTEGER NOT NULL DEFAULT 0,
	size_bytes INTEGER NOT NULL DEFAULT 0,
	job_id TEXT,
	error_message TEXT,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	started_at DATETIME,
	finished_at DATETIME,
	FOREIGN KEY(session_id) REFERENCES upload_sessions(id) ON DELETE CASCADE,
	UNIQUE(session_id, chunk_index),
	UNIQUE(session_id, idempotency_key)
);

CREATE INDEX idx_upload_session_chunks_status
ON upload_session_chunks (session_id, status, chunk_index);

CREATE TABLE collection_history (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	collection_id TEXT NOT NULL,
	action_type TEXT NOT NULL CHECK (
		action_type IN ('upload', 'delete', 'retry', 'dismiss')
	),
	session_id TEXT,
	triggered_by TEXT NOT NULL DEFAULT 'user',
	status TEXT NOT NULL DEFAULT 'running' CHECK (
		status IN ('running', 'success', 'warning', 'partial', 'failed', 'canceled')
	),
	started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	finished_at DATETIME,
	total_count INTEGER NOT NULL DEFAULT 0,
	ready_count INTEGER NOT NULL DEFAULT 0,
	warning_count INTEGER NOT NULL DEFAULT 0,
	failed_count INTEGER NOT NULL DEFAULT 0,
	duplicate_count INTEGER NOT NULL DEFAULT 0,
	metadata_json TEXT,
	FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY(collection_id) REFERENCES collections(id) ON DELETE CASCADE,
	FOREIGN KEY(session_id) REFERENCES upload_sessions(id) ON DELETE SET NULL
);

CREATE INDEX idx_collection_history_collection_time
ON collection_history (collection_id, started_at DESC);

CREATE INDEX idx_collection_history_status
ON collection_history (collection_id, status);

CREATE TABLE collection_history_items (
	id TEXT PRIMARY KEY,
	history_id TEXT NOT NULL,
	user_id TEXT NOT NULL,
	collection_id TEXT NOT NULL,
	document_type TEXT NOT NULL CHECK (document_type IN ('pdf_invoice', 'pdf_tax_invoice', 'pdf_bppu', 'pdf_bp21', 'pdf_bpa1')),
	source_name TEXT NOT NULL,
	source_size_bytes INTEGER,
	source_mime TEXT,
	source_sha256 TEXT,
	source_order INTEGER,
	item_status TEXT NOT NULL CHECK (
		item_status IN ('ready', 'warning', 'failed', 'duplicate', 'deleted', 'dismissed')
	),
	message TEXT,
	document_id TEXT,
	duplicate_of_id TEXT,
	duplicate_key TEXT,
	warnings_json TEXT,
	errors_json TEXT,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(history_id) REFERENCES collection_history(id) ON DELETE CASCADE,
	FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY(collection_id) REFERENCES collections(id) ON DELETE CASCADE,
	FOREIGN KEY(document_id) REFERENCES documents(id) ON DELETE SET NULL,
	FOREIGN KEY(duplicate_of_id) REFERENCES documents(id) ON DELETE SET NULL
);

CREATE INDEX idx_collection_history_items_history
ON collection_history_items (history_id, created_at ASC);

CREATE INDEX idx_collection_history_items_status
ON collection_history_items (history_id, item_status);

CREATE TABLE collection_actions (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	collection_id TEXT NOT NULL,
	document_type TEXT NOT NULL CHECK (document_type IN ('pdf_invoice', 'pdf_tax_invoice', 'pdf_bppu', 'pdf_bp21', 'pdf_bpa1')),
	action_type TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'queued' CHECK (
		status IN ('queued', 'running', 'success', 'warning', 'partial', 'failed', 'canceled')
	),
	message TEXT,
	params_json TEXT,
	snapshot_json TEXT NOT NULL,
	snapshot_hash TEXT NOT NULL,
	snapshot_total INTEGER NOT NULL DEFAULT 0,
	rerun_of_action_id TEXT,
	idempotency_key TEXT,
	total_count INTEGER NOT NULL DEFAULT 0,
	success_count INTEGER NOT NULL DEFAULT 0,
	warning_count INTEGER NOT NULL DEFAULT 0,
	failed_count INTEGER NOT NULL DEFAULT 0,
	skipped_count INTEGER NOT NULL DEFAULT 0,
	started_at DATETIME,
	finished_at DATETIME,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY(collection_id) REFERENCES collections(id) ON DELETE CASCADE,
	FOREIGN KEY(rerun_of_action_id) REFERENCES collection_actions(id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX idx_collection_actions_idempotency
ON collection_actions (collection_id, action_type, idempotency_key)
WHERE idempotency_key IS NOT NULL;

CREATE INDEX idx_collection_actions_collection_time
ON collection_actions (collection_id, created_at DESC);

CREATE INDEX idx_collection_actions_status
ON collection_actions (collection_id, status);

CREATE TABLE collection_action_items (
	id TEXT PRIMARY KEY,
	action_id TEXT NOT NULL,
	document_id TEXT,
	status TEXT NOT NULL CHECK (
		status IN ('success', 'warning', 'failed', 'skipped', 'canceled')
	),
	message TEXT,
	warnings_json TEXT,
	error TEXT,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(action_id) REFERENCES collection_actions(id) ON DELETE CASCADE,
	FOREIGN KEY(document_id) REFERENCES documents(id) ON DELETE SET NULL,
	UNIQUE(action_id, document_id)
);

CREATE INDEX idx_collection_action_items_status
ON collection_action_items (action_id, status);

CREATE TABLE collection_action_outputs (
	id TEXT PRIMARY KEY,
	action_id TEXT NOT NULL,
	kind TEXT NOT NULL CHECK (kind IN ('file', 'link', 'payload')),
	name TEXT NOT NULL,
	object_ref TEXT NOT NULL,
	mime_type TEXT,
	size_bytes INTEGER,
	checksum TEXT,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(action_id) REFERENCES collection_actions(id) ON DELETE CASCADE
);

CREATE INDEX idx_collection_action_outputs_action
ON collection_action_outputs (action_id, created_at ASC);
