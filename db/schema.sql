PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS utenti (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT UNIQUE NOT NULL,
    nome TEXT NOT NULL,
    ruolo TEXT NOT NULL DEFAULT 'staff' CHECK(ruolo IN ('admin', 'staff', 'maestro')),
    creato_il DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS login_tokens (
    token TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES utenti(id) ON DELETE CASCADE,
    expires_at DATETIME NOT NULL,
    used_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_login_tokens_expires ON login_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_login_tokens_user ON login_tokens(user_id);
