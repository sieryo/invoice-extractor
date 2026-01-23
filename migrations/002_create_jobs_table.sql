CREATE TABLE jobs (
    id TEXT PRIMARY KEY,
    user_id TEXT,
    type TEXT NOT NULL,
    status TEXT NOT NULL,
    progress INTEGER DEFAULT 0,
    input_payload TEXT,
    output_payload TEXT,
    error_message TEXT,
    created_at DATETIME NOT NULL,
    started_at DATETIME,
    finished_at DATETIME,
    expired_at DATETIME
);
