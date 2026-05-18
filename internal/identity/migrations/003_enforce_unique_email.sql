-- Migration 003: Enforce unique email on users table
-- Version: 3

-- SQLite does not support ALTER TABLE to add a UNIQUE constraint,
-- so we create a UNIQUE INDEX instead (same enforcement effect).
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique ON users(email);
