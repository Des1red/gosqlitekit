CREATE TABLE IF NOT EXISTS tokens (
    uuid TEXT NOT NULL,
    jti TEXT PRIMARY KEY,
    token_type TEXT NOT NULL,
    expires_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_uuid ON tokens(uuid);