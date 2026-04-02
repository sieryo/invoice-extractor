CREATE TABLE collection_action_artifacts (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	collection_id TEXT NOT NULL,
	action_type TEXT NOT NULL,
	artifact_key TEXT NOT NULL,
	object_ref TEXT NOT NULL,
	original_name TEXT NOT NULL,
	mime_type TEXT,
	size_bytes INTEGER NOT NULL DEFAULT 0,
	preview_json TEXT,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY(collection_id) REFERENCES collections(id) ON DELETE CASCADE
);

CREATE INDEX idx_collection_action_artifacts_lookup
ON collection_action_artifacts (collection_id, action_type, artifact_key, created_at DESC);

CREATE INDEX idx_collection_action_artifacts_collection_time
ON collection_action_artifacts (collection_id, created_at DESC);
