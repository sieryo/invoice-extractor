CREATE TABLE users (
	id TEXT PRIMARY KEY,
	username VARCHAR(100) UNIQUE NOT NULL,
	password_hash TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	expires_at DATETIME NOT NULL,
	FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE jobs (
	id TEXT PRIMARY KEY,

	user_id TEXT,

	type TEXT NOT NULL,
	status TEXT NOT NULL,
	progress INTEGER NOT NULL,

	-- opaque, job-type specific
	input_payload TEXT NOT NULL,

	-- structured JSON
	output_manifest TEXT,

	error_message TEXT,

	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	started_at DATETIME,
	finished_at DATETIME,
	expired_at DATETIME,

	FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE collections (
	id TEXT PRIMARY KEY,

	user_id TEXT NOT NULL,
	status TEXT NOT NULL, -- active, committed, expired

	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	expired_at DATETIME,

	FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE files (
	id TEXT PRIMARY KEY,

	collection_id TEXT NOT NULL,

	name TEXT NOT NULL,
	state TEXT NOT NULL, -- temp | final
	path TEXT NOT NULL,

	size INTEGER,
	mime_type TEXT,

	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

	FOREIGN KEY(collection_id) REFERENCES collections(id) ON DELETE CASCADE
);
