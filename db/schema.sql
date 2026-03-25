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

CREATE TABLE IF NOT EXISTS lezioni (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    titolo TEXT NOT NULL,
    insegnante TEXT NOT NULL,
    data_ora DATETIME NOT NULL,
    durata_min INTEGER DEFAULT 60,
    max_posti INTEGER NOT NULL,
    prezzo REAL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS iscrizioni_lezione (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    socio_id INTEGER NOT NULL REFERENCES soci(id),
    lezione_id INTEGER NOT NULL REFERENCES lezioni(id),
    pagato BOOLEAN DEFAULT FALSE,
    UNIQUE(socio_id, lezione_id)
);

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
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL
);
