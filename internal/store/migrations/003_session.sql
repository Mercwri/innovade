-- Migration 003: session state
-- Lightweight key-value store for persisting UI state across restarts.

CREATE TABLE IF NOT EXISTS session (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT ''
);
