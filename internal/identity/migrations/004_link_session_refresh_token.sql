-- Migration 004: Link sessions to their paired refresh tokens
-- Version: 4
-- Fixes: Logout must revoke the paired refresh token alongside the session.

-- SQLite does not support ALTER TABLE ADD COLUMN with DEFAULT for non-nullable,
-- so we use the two-step migrate: rename, recreate, copy.

ALTER TABLE sessions RENAME TO _sessions_old;

CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    token_hash TEXT NOT NULL UNIQUE,
    ip_address TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    last_used_at TEXT NOT NULL,
    revoked INTEGER NOT NULL DEFAULT 0,
    refresh_token_id TEXT REFERENCES refresh_tokens(id)
);

INSERT INTO sessions (id, user_id, token_hash, ip_address, user_agent, expires_at, created_at, last_used_at, revoked)
SELECT id, user_id, token_hash, ip_address, user_agent, expires_at, created_at, last_used_at, revoked FROM _sessions_old;

DROP TABLE _sessions_old;

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_sessions_refresh_token_id ON sessions(refresh_token_id);
