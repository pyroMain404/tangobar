PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS soci (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nome TEXT NOT NULL,
    cognome TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    telefono TEXT,
    data_nascita DATE,
    note TEXT,
    creato_il DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tessere (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    socio_id INTEGER NOT NULL REFERENCES soci(id),
    tipo TEXT NOT NULL CHECK(tipo IN ('base', 'premium', 'annuale')),
    emessa_il DATE NOT NULL,
    valida_fino DATE NOT NULL,
    pagato BOOLEAN DEFAULT FALSE,
    importo REAL NOT NULL
);

CREATE TABLE IF NOT EXISTS corsi (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    titolo TEXT NOT NULL,
    giorno_settimana INTEGER NOT NULL CHECK(giorno_settimana BETWEEN 0 AND 6),
    ora TEXT NOT NULL,
    durata_min INTEGER NOT NULL DEFAULT 60,
    max_posti INTEGER NOT NULL,
    prezzo_lezione REAL NOT NULL DEFAULT 0,
    maestro_id INTEGER REFERENCES utenti(id) ON DELETE SET NULL,
    data_inizio DATE NOT NULL,
    data_fine DATE NOT NULL,
    eta_max_giovani INTEGER,
    prezzo_giovani REAL,
    attivo BOOLEAN NOT NULL DEFAULT TRUE,
    creato_il DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS lezioni (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    corso_id INTEGER NOT NULL REFERENCES corsi(id) ON DELETE CASCADE,
    data DATE NOT NULL,
    ora TEXT NOT NULL,
    durata_min INTEGER NOT NULL DEFAULT 60,
    max_posti INTEGER NOT NULL,
    prezzo REAL NOT NULL DEFAULT 0,
    stato TEXT NOT NULL DEFAULT 'programmata' CHECK(stato IN ('programmata', 'completata', 'annullata')),
    nota TEXT
);
CREATE INDEX IF NOT EXISTS idx_lezioni_corso ON lezioni(corso_id);
CREATE INDEX IF NOT EXISTS idx_lezioni_data ON lezioni(data);

CREATE TABLE IF NOT EXISTS iscrizioni_corso (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    socio_id INTEGER NOT NULL REFERENCES soci(id) ON DELETE CASCADE,
    corso_id INTEGER NOT NULL REFERENCES corsi(id) ON DELETE CASCADE,
    prezzo_custom REAL,
    sconto_giovani_forzato BOOLEAN,
    creato_il DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(socio_id, corso_id)
);

CREATE TABLE IF NOT EXISTS presenze (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    socio_id INTEGER NOT NULL REFERENCES soci(id) ON DELETE CASCADE,
    lezione_id INTEGER NOT NULL REFERENCES lezioni(id) ON DELETE CASCADE,
    segnata_da INTEGER NOT NULL REFERENCES utenti(id),
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(socio_id, lezione_id)
);
CREATE INDEX IF NOT EXISTS idx_presenze_lezione ON presenze(lezione_id);

CREATE TABLE IF NOT EXISTS millesimi (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    maestro_id INTEGER NOT NULL REFERENCES utenti(id) ON DELETE CASCADE,
    socio_id INTEGER NOT NULL REFERENCES soci(id) ON DELETE CASCADE,
    voto INTEGER NOT NULL CHECK(voto BETWEEN 0 AND 1000),
    aggiornato_il DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(maestro_id, socio_id)
);
CREATE INDEX IF NOT EXISTS idx_millesimi_maestro ON millesimi(maestro_id);

CREATE TABLE IF NOT EXISTS eventi (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nome TEXT NOT NULL,
    data_ora DATETIME NOT NULL,
    luogo TEXT NOT NULL,
    prezzo_base REAL DEFAULT 0,
    note TEXT
);

CREATE TABLE IF NOT EXISTS ingressi_milonga (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    socio_id INTEGER REFERENCES soci(id),
    nome_ospite TEXT,
    evento_id INTEGER NOT NULL REFERENCES eventi(id),
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    importo REAL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS fatture (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    numero TEXT UNIQUE NOT NULL,
    socio_id INTEGER REFERENCES soci(id),
    nome_cliente TEXT,
    data_emissione DATE NOT NULL,
    totale REAL NOT NULL,
    pagata BOOLEAN DEFAULT FALSE,
    pdf_path TEXT
);

CREATE TABLE IF NOT EXISTS righe_fattura (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    fattura_id INTEGER NOT NULL REFERENCES fatture(id),
    descrizione TEXT NOT NULL,
    quantita REAL DEFAULT 1,
    prezzo_unit REAL NOT NULL,
    totale REAL NOT NULL
);

CREATE TABLE IF NOT EXISTS bar_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nome TEXT NOT NULL,
    categoria TEXT NOT NULL,
    quantita INTEGER DEFAULT 0,
    soglia_min INTEGER DEFAULT 5,
    prezzo REAL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS bar_movimenti (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    item_id INTEGER NOT NULL REFERENCES bar_items(id),
    delta INTEGER NOT NULL,
    nota TEXT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

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
