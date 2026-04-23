# Corsi, Presenze e Millesimi — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Sostituire il sistema lezioni piatto con un modello a due livelli (corsi ricorrenti + lezioni istanze), aggiungere presenze e millesimi, e introdurre il ruolo `maestro`.

**Architecture:** Schema SQLite migrato da flat-lezioni a corsi→lezioni cascade tramite rilevamento colonna legacy; nuovi handler e template seguono il pattern chi + templ + HTMX esistente; il ruolo `maestro` è aggiunto all'auth senza rompere i ruoli esistenti.

**Tech Stack:** Go 1.23, `github.com/go-chi/chi/v5`, `github.com/a-h/templ v0.2.778`, HTMX, SQLite (`modernc.org/sqlite`), Gorilla sessions.

---

## File Structure

**Nuovi:**
- `models/corso.go` — struct `Corso`
- `models/iscrizione_corso.go` — struct `IscrizioneCorso` + `CalcolaPrezzo`
- `models/iscrizione_corso_test.go` — unit test `CalcolaPrezzo`
- `models/millesimo.go` — struct `Millesimo`
- `handlers/corsi.go` — tutti gli handler corsi/lezioni/iscrizioni/presenze
- `handlers/millesimi.go` — handler millesimi
- `templates/corsi.templ` — componenti UI corsi
- `templates/millesimi.templ` — componenti UI millesimi

**Modificati:**
- `db/schema.sql` — nuove tabelle, rimozione `iscrizioni_lezione`, lezioni ristrutturata, utenti con ruolo `maestro`
- `db/db.go` — due migrazioni legacy
- `models/lezione.go` — struct `Lezione` ristrutturata + `Presenza`
- `models/utente.go` — commento ruolo aggiornato
- `templates/context.go` — aggiunto `IsMaestroCtx`
- `handlers/auth.go` — `RequireStaffOrAbove`, `userIDFromRequest`, fix query dashboard
- `templates/layout.templ` — sidebar: Corsi, sezione Insegnamento per maestro
- `main.go` — nuove route `/corsi/*`, `/millesimi`, redirect `/lezioni` → `/corsi`

**Eliminati:**
- `handlers/lezioni.go`
- `templates/lezioni.templ`

---

### Task 1: Aggiorna db/schema.sql

**Files:**
- Modify: `db/schema.sql`

- [ ] **Step 1: Sostituisci il contenuto di db/schema.sql**

```sql
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
```

- [ ] **Step 2: Verifica che il file non abbia più la tabella `iscrizioni_lezione`**

Cerca nel file: non deve apparire `iscrizioni_lezione`.

- [ ] **Step 3: Commit**

```bash
git add db/schema.sql
git commit -m "feat: nuovo schema corsi/lezioni/presenze/millesimi + ruolo maestro"
```

---

### Task 2: Migrazioni in db/db.go

**Files:**
- Modify: `db/db.go`

- [ ] **Step 1: Sostituisci il contenuto di db/db.go**

```go
package db

import (
	"database/sql"
	_ "embed"
	"strings"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		db.Close()
		return nil, err
	}

	// Migrazione 1: drop tabella utenti con vecchio schema password-based.
	if legacy, err := hasColumn(db, "utenti", "password_hash"); err != nil {
		db.Close()
		return nil, err
	} else if legacy {
		if _, err := db.Exec("DROP TABLE IF EXISTS login_tokens; DROP TABLE utenti;"); err != nil {
			db.Close()
			return nil, err
		}
	}

	// Migrazione 2: drop lezioni/iscrizioni_lezione legacy (schema piatto con colonna titolo).
	if legacy, err := hasColumn(db, "lezioni", "titolo"); err != nil {
		db.Close()
		return nil, err
	} else if legacy {
		if _, err := db.Exec("DROP TABLE IF EXISTS iscrizioni_lezione; DROP TABLE IF EXISTS lezioni;"); err != nil {
			db.Close()
			return nil, err
		}
	}

	// Migrazione 3: aggiorna CHECK constraint utenti.ruolo per includere 'maestro'.
	if needs, err := needsRuoloMigration(db); err != nil {
		db.Close()
		return nil, err
	} else if needs {
		stmts := []string{
			`CREATE TABLE utenti_new (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				email TEXT UNIQUE NOT NULL,
				nome TEXT NOT NULL,
				ruolo TEXT NOT NULL DEFAULT 'staff' CHECK(ruolo IN ('admin', 'staff', 'maestro')),
				creato_il DATETIME DEFAULT CURRENT_TIMESTAMP
			)`,
			`INSERT OR IGNORE INTO utenti_new SELECT id, email, nome, ruolo, creato_il FROM utenti`,
			`DROP TABLE IF EXISTS login_tokens`,
			`DROP TABLE utenti`,
			`ALTER TABLE utenti_new RENAME TO utenti`,
		}
		if _, err := db.Exec("PRAGMA foreign_keys = OFF;"); err != nil {
			db.Close()
			return nil, err
		}
		for _, s := range stmts {
			if _, err := db.Exec(s); err != nil {
				db.Close()
				return nil, err
			}
		}
		if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
			db.Close()
			return nil, err
		}
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func hasColumn(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

// needsRuoloMigration controlla se la tabella utenti non include ancora 'maestro' nel CHECK.
func needsRuoloMigration(db *sql.DB) (bool, error) {
	var createSQL string
	err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='utenti'`).Scan(&createSQL)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return !strings.Contains(createSQL, "'maestro'"), nil
}
```

- [ ] **Step 2: Verifica compilazione**

```bash
go build ./...
```

Expected: nessun errore.

- [ ] **Step 3: Commit**

```bash
git add db/db.go
git commit -m "feat: migrazioni legacy lezioni + aggiornamento CHECK ruolo utenti"
```

---

### Task 3: Modelli Go — Corso, Lezione ristrutturata, Millesimo

**Files:**
- Create: `models/corso.go`
- Modify: `models/lezione.go`
- Create: `models/millesimo.go`
- Modify: `models/utente.go` (solo commento)

- [ ] **Step 1: Crea models/corso.go**

```go
package models

import "time"

type Corso struct {
	ID              int
	Titolo          string
	GiornoSettimana int // 0=lunedì
	Ora             string
	DurataMin       int
	MaxPosti        int
	PrezzoLezione   float64
	MaestroID       *int
	MaestroNome     string // join
	DataInizio      time.Time
	DataFine        time.Time
	EtaMaxGiovani   *int
	PrezzoGiovani   *float64
	Attivo          bool
	CreatoIl        time.Time
	Iscritti        int // count join
	TotaleLezioni   int // count join
}

var GiorniSettimana = []string{"Lunedì", "Martedì", "Mercoledì", "Giovedì", "Venerdì", "Sabato", "Domenica"}

func (c Corso) GiornoLabel() string {
	if c.GiornoSettimana >= 0 && c.GiornoSettimana < 7 {
		return GiorniSettimana[c.GiornoSettimana]
	}
	return ""
}
```

- [ ] **Step 2: Sostituisci models/lezione.go**

```go
package models

import "time"

type Lezione struct {
	ID        int
	CorsoID   int
	Data      time.Time
	Ora       string
	DurataMin int
	MaxPosti  int
	Prezzo    float64
	Stato     string // "programmata" | "completata" | "annullata"
	Nota      string
	Presenti  int // count join
}

type Presenza struct {
	ID             int
	SocioID        int
	LezioneID      int
	SegnataDa      int
	Timestamp      time.Time
	NomeSocio      string
	CognomeSocio   string
	NomeOperatore  string
}
```

- [ ] **Step 3: Crea models/millesimo.go**

```go
package models

import "time"

type Millesimo struct {
	ID           int
	MaestroID    int
	SocioID      int
	Voto         int
	AggiornatoIl time.Time
	NomeSocio    string
	CognomeSocio string
	NomeMaestro  string
}
```

- [ ] **Step 4: Aggiorna commento in models/utente.go**

Nel campo `Ruolo`, aggiorna il commento da `// "admin" | "staff"` a `// "admin" | "staff" | "maestro"`.

```go
Ruolo    string // "admin" | "staff" | "maestro"
```

- [ ] **Step 5: Verifica compilazione**

```bash
go build ./...
```

Expected: errori solo da handlers che usano i vecchi campi di `Lezione` (es. `Titolo`, `Insegnante`) — saranno risolti nel Task 15 con la rimozione di `handlers/lezioni.go`.

- [ ] **Step 6: Commit**

```bash
git add models/corso.go models/lezione.go models/millesimo.go models/utente.go
git commit -m "feat: modelli Corso, Lezione ristrutturata, Millesimo"
```

---

### Task 4: models/iscrizione_corso.go con CalcolaPrezzo

**Files:**
- Create: `models/iscrizione_corso.go`

- [ ] **Step 1: Crea models/iscrizione_corso.go**

```go
package models

import "time"

type IscrizioneCorso struct {
	ID                   int
	SocioID              int
	CorsoID              int
	PrezzoCustom         *float64
	ScontoGiovaniForzato *bool // nil=auto, true=forzato sì, false=forzato no
	CreatoIl             time.Time
	NomeSocio            string
	CognomeSocio         string
	DataNascita          time.Time
	PrezzoEffettivo      float64
	EtichettaPrezzo      string
	GiovaniAuto          bool // risultato calcolo età (per stato auto nella UI)
}

// CalcolaPrezzo determina il prezzo effettivo e l'etichetta per un socio iscritto a un corso.
func CalcolaPrezzo(corso Corso, iscr IscrizioneCorso, socio Socio) (float64, string) {
	if iscr.PrezzoCustom != nil {
		return *iscr.PrezzoCustom, "custom"
	}

	if iscr.ScontoGiovaniForzato != nil {
		if *iscr.ScontoGiovaniForzato && corso.PrezzoGiovani != nil {
			return *corso.PrezzoGiovani, "giovani (forzato)"
		}
		// false forzato: salta sconto, usa standard
		return corso.PrezzoLezione, "standard"
	}

	// auto: calcola da età
	if corso.EtaMaxGiovani != nil && corso.PrezzoGiovani != nil && !socio.DataNascita.IsZero() {
		if calcEta(socio.DataNascita) <= *corso.EtaMaxGiovani {
			return *corso.PrezzoGiovani, "giovani"
		}
	}

	return corso.PrezzoLezione, "standard"
}

func calcEta(dataNascita time.Time) int {
	now := time.Now()
	anni := now.Year() - dataNascita.Year()
	if now.YearDay() < dataNascita.YearDay() {
		anni--
	}
	return anni
}
```

- [ ] **Step 2: Verifica compilazione**

```bash
go build ./models/...
```

Expected: nessun errore.

- [ ] **Step 3: Commit**

```bash
git add models/iscrizione_corso.go
git commit -m "feat: IscrizioneCorso e CalcolaPrezzo"
```

---

### Task 5: Test CalcolaPrezzo

**Files:**
- Create: `models/iscrizione_corso_test.go`

- [ ] **Step 1: Crea il file di test**

```go
package models

import (
	"testing"
	"time"
)

func TestCalcola_Custom(t *testing.T) {
	custom := 50.0
	iscr := IscrizioneCorso{PrezzoCustom: &custom}
	corso := Corso{PrezzoLezione: 20}
	socio := Socio{}
	prezzo, etichetta := CalcolaPrezzo(corso, iscr, socio)
	if prezzo != 50 || etichetta != "custom" {
		t.Errorf("got %.0f %s, want 50 custom", prezzo, etichetta)
	}
}

func TestCalcola_GiovaniForzatoTrue(t *testing.T) {
	forzato := true
	prezzoGiovani := 10.0
	etaMax := 25
	iscr := IscrizioneCorso{ScontoGiovaniForzato: &forzato}
	corso := Corso{PrezzoLezione: 20, EtaMaxGiovani: &etaMax, PrezzoGiovani: &prezzoGiovani}
	socio := Socio{DataNascita: time.Now().AddDate(-30, 0, 0)} // 30 anni
	prezzo, etichetta := CalcolaPrezzo(corso, iscr, socio)
	if prezzo != 10 || etichetta != "giovani (forzato)" {
		t.Errorf("got %.0f %s, want 10 giovani (forzato)", prezzo, etichetta)
	}
}

func TestCalcola_GiovaniForzatoFalse(t *testing.T) {
	forzato := false
	prezzoGiovani := 10.0
	etaMax := 25
	iscr := IscrizioneCorso{ScontoGiovaniForzato: &forzato}
	corso := Corso{PrezzoLezione: 20, EtaMaxGiovani: &etaMax, PrezzoGiovani: &prezzoGiovani}
	socio := Socio{DataNascita: time.Now().AddDate(-20, 0, 0)} // 20 anni, ma forzato false
	prezzo, etichetta := CalcolaPrezzo(corso, iscr, socio)
	if prezzo != 20 || etichetta != "standard" {
		t.Errorf("got %.0f %s, want 20 standard", prezzo, etichetta)
	}
}

func TestCalcola_GiovaniAuto_SottoSoglia(t *testing.T) {
	prezzoGiovani := 10.0
	etaMax := 25
	iscr := IscrizioneCorso{}
	corso := Corso{PrezzoLezione: 20, EtaMaxGiovani: &etaMax, PrezzoGiovani: &prezzoGiovani}
	socio := Socio{DataNascita: time.Now().AddDate(-20, 0, 0)}
	prezzo, etichetta := CalcolaPrezzo(corso, iscr, socio)
	if prezzo != 10 || etichetta != "giovani" {
		t.Errorf("got %.0f %s, want 10 giovani", prezzo, etichetta)
	}
}

func TestCalcola_GiovaniAuto_SopraSoglia(t *testing.T) {
	prezzoGiovani := 10.0
	etaMax := 25
	iscr := IscrizioneCorso{}
	corso := Corso{PrezzoLezione: 20, EtaMaxGiovani: &etaMax, PrezzoGiovani: &prezzoGiovani}
	socio := Socio{DataNascita: time.Now().AddDate(-30, 0, 0)}
	prezzo, etichetta := CalcolaPrezzo(corso, iscr, socio)
	if prezzo != 20 || etichetta != "standard" {
		t.Errorf("got %.0f %s, want 20 standard", prezzo, etichetta)
	}
}

func TestCalcola_Standard(t *testing.T) {
	iscr := IscrizioneCorso{}
	corso := Corso{PrezzoLezione: 20}
	socio := Socio{}
	prezzo, etichetta := CalcolaPrezzo(corso, iscr, socio)
	if prezzo != 20 || etichetta != "standard" {
		t.Errorf("got %.0f %s, want 20 standard", prezzo, etichetta)
	}
}
```

- [ ] **Step 2: Esegui i test**

```bash
go test ./models/... -v -run TestCalcola
```

Expected: tutti e 6 i test PASS.

- [ ] **Step 3: Commit**

```bash
git add models/iscrizione_corso_test.go
git commit -m "test: CalcolaPrezzo unit tests"
```

---

### Task 6: Auth — RequireStaffOrAbove, userIDFromRequest, IsMaestroCtx

**Files:**
- Modify: `handlers/auth.go`
- Modify: `templates/context.go`

- [ ] **Step 1: Aggiungi a handlers/auth.go (in fondo al file, prima della funzione `newToken`)**

```go
// RequireStaffOrAbove è middleware che richiede ruolo admin o staff. I maestri vengono bloccati.
func (h *Handler) RequireStaffOrAbove(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isStaffOrAbove(r) {
			http.Error(w, "Accesso negato: richiesti privilegi staff o superiori", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isStaffOrAbove(r *http.Request) bool {
	session, err := store.Get(r, sessionName)
	if err != nil || session.IsNew {
		return false
	}
	ruolo, _ := session.Values["ruolo"].(string)
	return ruolo == "admin" || ruolo == "staff"
}

// userIDFromRequest estrae l'ID utente dalla sessione corrente.
func userIDFromRequest(r *http.Request) (int, bool) {
	session, err := store.Get(r, sessionName)
	if err != nil || session.IsNew {
		return 0, false
	}
	id, ok := session.Values["user_id"]
	if !ok {
		return 0, false
	}
	switch v := id.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	}
	return 0, false
}
```

- [ ] **Step 2: Aggiorna la query dashboard in handlers/auth.go**

Nella funzione `Dashboard`, sostituisci la query `lezioniProssime`:

Vecchia:
```go
h.DB.QueryRow("SELECT COUNT(*) FROM lezioni WHERE data_ora > datetime('now')").Scan(&lezioniProssime)
```

Nuova:
```go
h.DB.QueryRow("SELECT COUNT(*) FROM lezioni WHERE data >= date('now') AND stato = 'programmata'").Scan(&lezioniProssime)
```

- [ ] **Step 3: Aggiungi IsMaestroCtx a templates/context.go**

```go
// IsMaestroCtx reports whether the context carries a maestro role.
func IsMaestroCtx(ctx context.Context) bool {
	return RoleFromContext(ctx) == "maestro"
}
```

- [ ] **Step 4: Verifica compilazione**

```bash
go build ./...
```

Expected: nessun errore (eccetto eventuali errori da handlers/lezioni.go che sarà rimosso nel Task 15).

- [ ] **Step 5: Commit**

```bash
git add handlers/auth.go templates/context.go
git commit -m "feat: RequireStaffOrAbove, userIDFromRequest, IsMaestroCtx"
```

---

### Task 7: handlers/corsi.go — CRUD corsi + generazione lezioni

**Files:**
- Create: `handlers/corsi.go`

- [ ] **Step 1: Crea handlers/corsi.go con intestazione e helper**

```go
package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"tango-gestionale/models"
	"tango-gestionale/templates"
)

// --- Helper: generazione date lezioni ---

// generaDateLezioni calcola tutte le date tra dataInizio e dataFine che cadono su giornoSettimana.
// Spec: 0=lunedì. Go time.Weekday: Sunday=0, Monday=1.
func generaDateLezioni(dataInizio, dataFine time.Time, giornoSettimana int) []time.Time {
	// Converti: spec 0=Monday → Go Monday=1; spec 6=Sunday → Go Sunday=0
	targetWeekday := time.Weekday((giornoSettimana + 1) % 7)
	var dates []time.Time
	current := dataInizio
	for current.Weekday() != targetWeekday {
		current = current.AddDate(0, 0, 1)
	}
	for !current.After(dataFine) {
		dates = append(dates, current)
		current = current.AddDate(0, 0, 7)
	}
	return dates
}

func contaLezioniNelRange(giornoSettimana int, dataInizio, dataFine time.Time) int {
	return len(generaDateLezioni(dataInizio, dataFine, giornoSettimana))
}
```

- [ ] **Step 2: Aggiungi ListaCorsi a handlers/corsi.go**

```go
func (h *Handler) ListaCorsi(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := h.DB.QueryContext(ctx, `
		SELECT c.id, c.titolo, c.giorno_settimana, c.ora, c.durata_min, c.max_posti,
		       c.prezzo_lezione, c.maestro_id, COALESCE(u.nome,'') as maestro_nome,
		       c.data_inizio, c.data_fine, c.eta_max_giovani, c.prezzo_giovani,
		       c.attivo, c.creato_il,
		       COUNT(DISTINCT ic.id) as iscritti,
		       COUNT(DISTINCT l.id) as totale_lezioni
		FROM corsi c
		LEFT JOIN utenti u ON u.id = c.maestro_id
		LEFT JOIN iscrizioni_corso ic ON ic.corso_id = c.id
		LEFT JOIN lezioni l ON l.corso_id = c.id
		GROUP BY c.id
		ORDER BY c.data_inizio DESC
	`)
	if err != nil {
		http.Error(w, "Errore DB", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var corsi []models.Corso
	for rows.Next() {
		var c models.Corso
		var maestroID sql.NullInt64
		var etaMax sql.NullInt64
		var prezzoGiovani sql.NullFloat64
		if err := rows.Scan(
			&c.ID, &c.Titolo, &c.GiornoSettimana, &c.Ora, &c.DurataMin, &c.MaxPosti,
			&c.PrezzoLezione, &maestroID, &c.MaestroNome,
			&c.DataInizio, &c.DataFine, &etaMax, &prezzoGiovani,
			&c.Attivo, &c.CreatoIl, &c.Iscritti, &c.TotaleLezioni,
		); err != nil {
			log.Printf("[corsi] scan: %v", err)
			continue
		}
		if maestroID.Valid {
			id := int(maestroID.Int64)
			c.MaestroID = &id
		}
		if etaMax.Valid {
			v := int(etaMax.Int64)
			c.EtaMaxGiovani = &v
		}
		if prezzoGiovani.Valid {
			c.PrezzoGiovani = &prezzoGiovani.Float64
		}
		corsi = append(corsi, c)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderTempl(w, r, templates.Page("Corsi", "corsi", templates.CorsiPage(corsi)))
}
```

- [ ] **Step 3: Aggiungi NuovoCorsoForm e CreaCorso**

```go
func (h *Handler) NuovoCorsoForm(w http.ResponseWriter, r *http.Request) {
	maestri := h.fetchMaestri(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderTempl(w, r, templates.Page("Nuovo Corso", "corsi", templates.NuovoCorsoForm(maestri, 0)))
}

func (h *Handler) CreaCorso(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form non valido", http.StatusBadRequest)
		return
	}
	ctx := r.Context()

	titolo := r.FormValue("titolo")
	giornoStr := r.FormValue("giorno_settimana")
	ora := r.FormValue("ora")
	durataStr := r.FormValue("durata_min")
	postiStr := r.FormValue("max_posti")
	prezzoStr := r.FormValue("prezzo_lezione")
	inizioStr := r.FormValue("data_inizio")
	fineStr := r.FormValue("data_fine")
	etaMaxStr := r.FormValue("eta_max_giovani")
	prezzoGiovaniStr := r.FormValue("prezzo_giovani")
	maestroIDStr := r.FormValue("maestro_id")

	giorno, _ := strconv.Atoi(giornoStr)
	durata, _ := strconv.Atoi(durataStr)
	if durata == 0 {
		durata = 60
	}
	posti, _ := strconv.Atoi(postiStr)
	prezzo, _ := strconv.ParseFloat(prezzoStr, 64)
	dataInizio, _ := time.Parse("2006-01-02", inizioStr)
	dataFine, _ := time.Parse("2006-01-02", fineStr)

	var etaMaxGiovani *int
	var prezzoGiovani *float64
	if etaMaxStr != "" {
		v, err := strconv.Atoi(etaMaxStr)
		if err == nil {
			etaMaxGiovani = &v
		}
	}
	if prezzoGiovaniStr != "" && etaMaxGiovani != nil {
		v, err := strconv.ParseFloat(prezzoGiovaniStr, 64)
		if err == nil {
			prezzoGiovani = &v
		}
	}

	var maestroID *int
	if maestroIDStr != "" && maestroIDStr != "0" {
		v, err := strconv.Atoi(maestroIDStr)
		if err == nil {
			maestroID = &v
		}
	}

	res, err := h.DB.ExecContext(ctx, `
		INSERT INTO corsi (titolo, giorno_settimana, ora, durata_min, max_posti, prezzo_lezione,
		                   maestro_id, data_inizio, data_fine, eta_max_giovani, prezzo_giovani, attivo)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, TRUE)
	`, titolo, giorno, ora, durata, posti, prezzo, maestroID,
		dataInizio.Format("2006-01-02"), dataFine.Format("2006-01-02"),
		etaMaxGiovani, prezzoGiovani,
	)
	if err != nil {
		log.Printf("[corsi] insert corso: %v", err)
		http.Error(w, "Errore DB", http.StatusInternalServerError)
		return
	}

	corsoID, _ := res.LastInsertId()

	// Genera lezioni
	date := generaDateLezioni(dataInizio, dataFine, giorno)
	for _, d := range date {
		_, err := h.DB.ExecContext(ctx, `
			INSERT INTO lezioni (corso_id, data, ora, durata_min, max_posti, prezzo, stato)
			VALUES (?, ?, ?, ?, ?, ?, 'programmata')
		`, corsoID, d.Format("2006-01-02"), ora, durata, posti, prezzo)
		if err != nil {
			log.Printf("[corsi] insert lezione %s: %v", d.Format("2006-01-02"), err)
		}
	}

	http.Redirect(w, r, fmt.Sprintf("/corsi/%d", corsoID), http.StatusSeeOther)
}
```

- [ ] **Step 4: Aggiungi DettaglioCorso**

```go
func (h *Handler) DettaglioCorso(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))

	corso, err := h.fetchCorso(ctx, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Lezioni del corso
	rows, err := h.DB.QueryContext(ctx, `
		SELECT l.id, l.corso_id, l.data, l.ora, l.durata_min, l.max_posti, l.prezzo, l.stato, COALESCE(l.nota,''),
		       COUNT(p.id) as presenti
		FROM lezioni l
		LEFT JOIN presenze p ON p.lezione_id = l.id
		WHERE l.corso_id = ?
		GROUP BY l.id
		ORDER BY l.data ASC
	`, id)
	if err != nil {
		http.Error(w, "Errore DB", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var lezioni []models.Lezione
	for rows.Next() {
		var l models.Lezione
		if err := rows.Scan(&l.ID, &l.CorsoID, &l.Data, &l.Ora, &l.DurataMin, &l.MaxPosti,
			&l.Prezzo, &l.Stato, &l.Nota, &l.Presenti); err != nil {
			log.Printf("[corsi] scan lezione: %v", err)
			continue
		}
		lezioni = append(lezioni, l)
	}

	// Iscritti al corso
	iscritti, err := h.fetchIscrittiCorso(ctx, id, corso)
	if err != nil {
		http.Error(w, "Errore DB", http.StatusInternalServerError)
		return
	}

	// Tutti i soci (per form iscrizione)
	soci := h.fetchTuttiSoci(ctx)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderTempl(w, r, templates.Page(corso.Titolo, "corsi",
		templates.DettaglioCorsoPage(corso, lezioni, iscritti, soci)))
}
```

- [ ] **Step 5: Aggiungi ModificaCorsoForm, AggiornaCorso, EliminaCorso e helpers**

```go
func (h *Handler) ModificaCorsoForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	corso, err := h.fetchCorso(ctx, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	maestri := h.fetchMaestri(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderTempl(w, r, templates.Page("Modifica Corso", "corsi", templates.NuovoCorsoForm(maestri, 0, &corso)))
}

func (h *Handler) AggiornaCorso(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form non valido", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))

	titolo := r.FormValue("titolo")
	ora := r.FormValue("ora")
	durataStr := r.FormValue("durata_min")
	postiStr := r.FormValue("max_posti")
	prezzoStr := r.FormValue("prezzo_lezione")
	etaMaxStr := r.FormValue("eta_max_giovani")
	prezzoGiovaniStr := r.FormValue("prezzo_giovani")
	maestroIDStr := r.FormValue("maestro_id")
	attivo := r.FormValue("attivo") == "on"

	durata, _ := strconv.Atoi(durataStr)
	if durata == 0 {
		durata = 60
	}
	posti, _ := strconv.Atoi(postiStr)
	prezzo, _ := strconv.ParseFloat(prezzoStr, 64)

	var etaMaxGiovani *int
	var prezzoGiovani *float64
	if etaMaxStr != "" {
		v, _ := strconv.Atoi(etaMaxStr)
		etaMaxGiovani = &v
	}
	if prezzoGiovaniStr != "" && etaMaxGiovani != nil {
		v, _ := strconv.ParseFloat(prezzoGiovaniStr, 64)
		prezzoGiovani = &v
	}
	var maestroID *int
	if maestroIDStr != "" && maestroIDStr != "0" {
		v, _ := strconv.Atoi(maestroIDStr)
		maestroID = &v
	}

	_, err := h.DB.ExecContext(ctx, `
		UPDATE corsi SET titolo=?, ora=?, durata_min=?, max_posti=?, prezzo_lezione=?,
		                 eta_max_giovani=?, prezzo_giovani=?, maestro_id=?, attivo=?
		WHERE id=?
	`, titolo, ora, durata, posti, prezzo, etaMaxGiovani, prezzoGiovani, maestroID, attivo, id)
	if err != nil {
		log.Printf("[corsi] update corso: %v", err)
		http.Error(w, "Errore DB", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/corsi/%d", id), http.StatusSeeOther)
}

func (h *Handler) EliminaCorso(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	if _, err := h.DB.ExecContext(ctx, `DELETE FROM corsi WHERE id=?`, id); err != nil {
		http.Error(w, "Errore DB", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/corsi", http.StatusSeeOther)
}

// --- Helpers privati ---

func (h *Handler) fetchCorso(ctx interface{ Value(interface{}) interface{} }, id int) (models.Corso, error) {
	// Nota: ctx è r.Context() che implementa context.Context
	return h.fetchCorsoCtx(ctx.(interface {
		Deadline() (deadline time.Time, ok bool)
		Done() <-chan struct{}
		Err() error
		Value(key interface{}) interface{}
	}), id)
}
```

**Nota:** l'helper `fetchCorso` ha bisogno di `context.Context`. Riscrivi il passo precedente usando il tipo corretto. Aggiungi all'import `"context"` e usa questa firma:

```go
// Aggiungi "context" agli import del file.

func (h *Handler) fetchCorso(ctx context.Context, id int) (models.Corso, error) {
	var c models.Corso
	var maestroID sql.NullInt64
	var etaMax sql.NullInt64
	var prezzoGiovani sql.NullFloat64
	err := h.DB.QueryRowContext(ctx, `
		SELECT c.id, c.titolo, c.giorno_settimana, c.ora, c.durata_min, c.max_posti,
		       c.prezzo_lezione, c.maestro_id, COALESCE(u.nome,'') as maestro_nome,
		       c.data_inizio, c.data_fine, c.eta_max_giovani, c.prezzo_giovani, c.attivo, c.creato_il
		FROM corsi c
		LEFT JOIN utenti u ON u.id = c.maestro_id
		WHERE c.id = ?
	`, id).Scan(
		&c.ID, &c.Titolo, &c.GiornoSettimana, &c.Ora, &c.DurataMin, &c.MaxPosti,
		&c.PrezzoLezione, &maestroID, &c.MaestroNome,
		&c.DataInizio, &c.DataFine, &etaMax, &prezzoGiovani, &c.Attivo, &c.CreatoIl,
	)
	if err != nil {
		return c, err
	}
	if maestroID.Valid {
		id := int(maestroID.Int64)
		c.MaestroID = &id
	}
	if etaMax.Valid {
		v := int(etaMax.Int64)
		c.EtaMaxGiovani = &v
	}
	if prezzoGiovani.Valid {
		c.PrezzoGiovani = &prezzoGiovani.Float64
	}
	return c, nil
}

func (h *Handler) fetchMaestri(r *http.Request) []models.Utente {
	rows, err := h.DB.QueryContext(r.Context(), `SELECT id, email, nome, ruolo, creato_il FROM utenti WHERE ruolo='maestro' ORDER BY nome`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []models.Utente
	for rows.Next() {
		var u models.Utente
		rows.Scan(&u.ID, &u.Email, &u.Nome, &u.Ruolo, &u.CreatoIl)
		result = append(result, u)
	}
	return result
}

func (h *Handler) fetchTuttiSoci(ctx interface{}) []models.Socio {
	// Usa r.Context() passato direttamente
	return nil // placeholder — vedi step successivo
}
```

**Nota sul fetchTuttiSoci:** riscrivi usando `context.Context`:

```go
func (h *Handler) fetchTuttiSoci(ctx context.Context) []models.Socio {
	rows, err := h.DB.QueryContext(ctx, `SELECT id, nome, cognome, email, COALESCE(telefono,''), COALESCE(data_nascita,'0001-01-01'), COALESCE(note,''), creato_il FROM soci ORDER BY cognome, nome`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []models.Socio
	for rows.Next() {
		var s models.Socio
		rows.Scan(&s.ID, &s.Nome, &s.Cognome, &s.Email, &s.Telefono, &s.DataNascita, &s.Note, &s.CreatedAt)
		result = append(result, s)
	}
	return result
}
```

- [ ] **Step 6: Verifica compilazione**

```bash
go build ./handlers/...
```

Expected: errori possibili per template non ancora creati — verranno risolti nel Task 11.

- [ ] **Step 7: Commit parziale**

```bash
git add handlers/corsi.go
git commit -m "feat: handlers corsi CRUD + generazione lezioni"
```

---

### Task 8: handlers/corsi.go — Iscrizioni corso + sconto giovani

**Files:**
- Modify: `handlers/corsi.go`

- [ ] **Step 1: Aggiungi fetchIscrittiCorso e IscriviSocio**

```go
func (h *Handler) fetchIscrittiCorso(ctx context.Context, corsoID int, corso models.Corso) ([]models.IscrizioneCorso, error) {
	rows, err := h.DB.QueryContext(ctx, `
		SELECT ic.id, ic.socio_id, ic.corso_id, ic.prezzo_custom, ic.sconto_giovani_forzato,
		       ic.creato_il, s.nome, s.cognome, COALESCE(s.data_nascita,'0001-01-01')
		FROM iscrizioni_corso ic
		JOIN soci s ON s.id = ic.socio_id
		WHERE ic.corso_id = ?
		ORDER BY s.cognome, s.nome
	`, corsoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.IscrizioneCorso
	for rows.Next() {
		var ic models.IscrizioneCorso
		var prezzoCustom sql.NullFloat64
		var scontoForzato sql.NullBool
		if err := rows.Scan(
			&ic.ID, &ic.SocioID, &ic.CorsoID, &prezzoCustom, &scontoForzato,
			&ic.CreatoIl, &ic.NomeSocio, &ic.CognomeSocio, &ic.DataNascita,
		); err != nil {
			log.Printf("[corsi] scan iscritto: %v", err)
			continue
		}
		if prezzoCustom.Valid {
			ic.PrezzoCustom = &prezzoCustom.Float64
		}
		if scontoForzato.Valid {
			ic.ScontoGiovaniForzato = &scontoForzato.Bool
		}

		socio := models.Socio{DataNascita: ic.DataNascita}
		ic.PrezzoEffettivo, ic.EtichettaPrezzo = models.CalcolaPrezzo(corso, ic, socio)

		// Calcola GiovaniAuto (risultato del solo calcolo età, indipendentemente dal forzato)
		if corso.EtaMaxGiovani != nil && !ic.DataNascita.IsZero() {
			age := calcEtaHelper(ic.DataNascita)
			ic.GiovaniAuto = age <= *corso.EtaMaxGiovani
		}

		result = append(result, ic)
	}
	return result, nil
}

// calcEtaHelper replica calcEta da models per uso interno handler.
func calcEtaHelper(dataNascita time.Time) int {
	now := time.Now()
	anni := now.Year() - dataNascita.Year()
	if now.YearDay() < dataNascita.YearDay() {
		anni--
	}
	return anni
}

func (h *Handler) IscriviSocio(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form non valido", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	corsoID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	socioID, _ := strconv.Atoi(r.FormValue("socio_id"))

	_, err := h.DB.ExecContext(ctx,
		`INSERT OR IGNORE INTO iscrizioni_corso (socio_id, corso_id) VALUES (?, ?)`,
		socioID, corsoID,
	)
	if err != nil {
		log.Printf("[corsi] iscrivi socio: %v", err)
		http.Error(w, "Errore DB", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/corsi/%d", corsoID), http.StatusSeeOther)
}

func (h *Handler) RimuoviIscrizioneCorso(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	corsoID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	socioID, _ := strconv.Atoi(chi.URLParam(r, "socioID"))

	h.DB.ExecContext(ctx, `DELETE FROM iscrizioni_corso WHERE corso_id=? AND socio_id=?`, corsoID, socioID)
	http.Redirect(w, r, fmt.Sprintf("/corsi/%d", corsoID), http.StatusSeeOther)
}

func (h *Handler) AggiornaPrezzoCustom(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form non valido", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	corsoID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	socioID, _ := strconv.Atoi(chi.URLParam(r, "socioID"))
	prezzoStr := r.FormValue("prezzo_custom")

	var prezzoCustom *float64
	if prezzoStr != "" {
		v, err := strconv.ParseFloat(prezzoStr, 64)
		if err == nil {
			prezzoCustom = &v
		}
	}

	h.DB.ExecContext(ctx,
		`UPDATE iscrizioni_corso SET prezzo_custom=? WHERE corso_id=? AND socio_id=?`,
		prezzoCustom, corsoID, socioID,
	)

	corso, _ := h.fetchCorso(ctx, corsoID)
	iscritti, _ := h.fetchIscrittiCorso(ctx, corsoID, corso)
	for _, ic := range iscritti {
		if ic.SocioID == socioID {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			templates.IscrittaRiga(ic, corso).Render(ctx, w)
			return
		}
	}
}

func (h *Handler) ToggleScontoGiovani(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form non valido", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	corsoID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	socioID, _ := strconv.Atoi(chi.URLParam(r, "socioID"))
	forzato := r.FormValue("sconto") == "true"

	h.DB.ExecContext(ctx,
		`UPDATE iscrizioni_corso SET sconto_giovani_forzato=? WHERE corso_id=? AND socio_id=?`,
		forzato, corsoID, socioID,
	)

	corso, _ := h.fetchCorso(ctx, corsoID)
	iscritti, _ := h.fetchIscrittiCorso(ctx, corsoID, corso)
	for _, ic := range iscritti {
		if ic.SocioID == socioID {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			templates.IscrittaRiga(ic, corso).Render(ctx, w)
			return
		}
	}
}

func (h *Handler) ResetScontoGiovani(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	corsoID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	socioID, _ := strconv.Atoi(chi.URLParam(r, "socioID"))

	h.DB.ExecContext(ctx,
		`UPDATE iscrizioni_corso SET sconto_giovani_forzato=NULL WHERE corso_id=? AND socio_id=?`,
		corsoID, socioID,
	)

	corso, _ := h.fetchCorso(ctx, corsoID)
	iscritti, _ := h.fetchIscrittiCorso(ctx, corsoID, corso)
	for _, ic := range iscritti {
		if ic.SocioID == socioID {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			templates.IscrittaRiga(ic, corso).Render(ctx, w)
			return
		}
	}
}
```

- [ ] **Step 2: Verifica compilazione**

```bash
go build ./handlers/...
```

- [ ] **Step 3: Commit**

```bash
git add handlers/corsi.go
git commit -m "feat: handlers iscrizioni corso e sconto giovani"
```

---

### Task 9: handlers/corsi.go — Lezioni e Presenze

**Files:**
- Modify: `handlers/corsi.go`

- [ ] **Step 1: Aggiungi DettaglioLezione, AggiornaLezione, TogglePresenza**

```go
func (h *Handler) DettaglioLezione(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	corsoID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	lezioneID, _ := strconv.Atoi(chi.URLParam(r, "lid"))

	var l models.Lezione
	err := h.DB.QueryRowContext(ctx, `
		SELECT id, corso_id, data, ora, durata_min, max_posti, prezzo, stato, COALESCE(nota,'')
		FROM lezioni WHERE id=? AND corso_id=?
	`, lezioneID, corsoID).Scan(
		&l.ID, &l.CorsoID, &l.Data, &l.Ora, &l.DurataMin, &l.MaxPosti, &l.Prezzo, &l.Stato, &l.Nota,
	)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	corso, _ := h.fetchCorso(ctx, corsoID)

	// Iscritti al corso con flag presenza per questa lezione
	iscritti, _ := h.fetchIscrittiCorso(ctx, corsoID, corso)

	// Presenze per questa lezione
	presRows, _ := h.DB.QueryContext(ctx, `
		SELECT p.socio_id, p.segnata_da, p.timestamp, COALESCE(u.nome,'')
		FROM presenze p
		LEFT JOIN utenti u ON u.id = p.segnata_da
		WHERE p.lezione_id=?
	`, lezioneID)
	defer presRows.Close()
	presenti := make(map[int]models.Presenza)
	for presRows.Next() {
		var p models.Presenza
		p.LezioneID = lezioneID
		presRows.Scan(&p.SocioID, &p.SegnataDa, &p.Timestamp, &p.NomeOperatore)
		presenti[p.SocioID] = p
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderTempl(w, r, templates.Page(
		fmt.Sprintf("Lezione %s", l.Data.Format("02/01/2006")), "corsi",
		templates.DettaglioLezioneContent(corso, l, iscritti, presenti),
	))
}

func (h *Handler) AggiornaLezione(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form non valido", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	corsoID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	lezioneID, _ := strconv.Atoi(chi.URLParam(r, "lid"))

	dataStr := r.FormValue("data")
	ora := r.FormValue("ora")
	stato := r.FormValue("stato")
	nota := r.FormValue("nota")

	h.DB.ExecContext(ctx,
		`UPDATE lezioni SET data=?, ora=?, stato=?, nota=? WHERE id=? AND corso_id=?`,
		dataStr, ora, stato, nota, lezioneID, corsoID,
	)
	http.Redirect(w, r, fmt.Sprintf("/corsi/%d/lezioni/%d", corsoID, lezioneID), http.StatusSeeOther)
}

func (h *Handler) TogglePresenza(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form non valido", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	corsoID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	lezioneID, _ := strconv.Atoi(chi.URLParam(r, "lid"))
	socioID, _ := strconv.Atoi(r.FormValue("socio_id"))

	operatoreID, _ := userIDFromRequest(r)

	var presenzaID int
	err := h.DB.QueryRowContext(ctx,
		`SELECT id FROM presenze WHERE socio_id=? AND lezione_id=?`, socioID, lezioneID,
	).Scan(&presenzaID)

	if err == sql.ErrNoRows {
		h.DB.ExecContext(ctx,
			`INSERT INTO presenze (socio_id, lezione_id, segnata_da) VALUES (?,?,?)`,
			socioID, lezioneID, operatoreID,
		)
	} else if err == nil {
		h.DB.ExecContext(ctx, `DELETE FROM presenze WHERE id=?`, presenzaID)
	}

	// Restituisci la riga aggiornata
	corso, _ := h.fetchCorso(ctx, corsoID)
	iscritti, _ := h.fetchIscrittiCorso(ctx, corsoID, corso)

	var presRows, _ = h.DB.QueryContext(ctx, `SELECT socio_id, segnata_da, timestamp, COALESCE(u.nome,'') FROM presenze p LEFT JOIN utenti u ON u.id=p.segnata_da WHERE lezione_id=?`, lezioneID)
	presenti := make(map[int]models.Presenza)
	if presRows != nil {
		defer presRows.Close()
		for presRows.Next() {
			var p models.Presenza
			presRows.Scan(&p.SocioID, &p.SegnataDa, &p.Timestamp, &p.NomeOperatore)
			presenti[p.SocioID] = p
		}
	}

	for _, ic := range iscritti {
		if ic.SocioID == socioID {
			_, presente := presenti[socioID]
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			templates.PresenzaRiga(ic, lezioneID, corsoID, presente, presenti[socioID]).Render(ctx, w)
			return
		}
	}
}
```

- [ ] **Step 2: Verifica compilazione**

```bash
go build ./handlers/...
```

- [ ] **Step 3: Commit**

```bash
git add handlers/corsi.go
git commit -m "feat: handlers lezioni e presenze"
```

---

### Task 10: handlers/millesimi.go

**Files:**
- Create: `handlers/millesimi.go`

- [ ] **Step 1: Crea handlers/millesimi.go**

```go
package handlers

import (
	"log"
	"net/http"
	"strconv"

	"tango-gestionale/models"
	"tango-gestionale/templates"
)

func (h *Handler) ListaMillesimi(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	viewAdmin := r.URL.Query().Get("view") == "admin" && templates.IsAdminCtx(ctx)

	if viewAdmin {
		h.milesimiAdmin(w, r)
		return
	}

	// Vista maestro: tutti i soci + voto del maestro corrente
	maestroID, ok := userIDFromRequest(r)
	if !ok {
		http.Error(w, "Non autenticato", http.StatusUnauthorized)
		return
	}

	soci := h.fetchTuttiSoci(ctx)

	// Mappa socio_id → voto corrente
	rows, err := h.DB.QueryContext(ctx,
		`SELECT socio_id, voto FROM millesimi WHERE maestro_id=?`, maestroID,
	)
	voti := make(map[int]int)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var sid, voto int
			rows.Scan(&sid, &voto)
			voti[sid] = voto
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderTempl(w, r, templates.Page("Millesimi", "millesimi", templates.MillesimiMaestroPage(soci, voti)))
}

func (h *Handler) milesimiAdmin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	filtroMaestroStr := r.URL.Query().Get("maestro_id")
	filtroMaestroID, _ := strconv.Atoi(filtroMaestroStr)

	maestri := h.fetchMaestri(r)

	var millesimi []models.Millesimo
	var err error
	if filtroMaestroID > 0 {
		rows, e := h.DB.QueryContext(ctx, `
			SELECT m.id, m.maestro_id, m.socio_id, m.voto, m.aggiornato_il,
			       s.nome, s.cognome, COALESCE(u.nome,'')
			FROM millesimi m
			JOIN soci s ON s.id = m.socio_id
			LEFT JOIN utenti u ON u.id = m.maestro_id
			WHERE m.maestro_id=?
			ORDER BY s.cognome, s.nome
		`, filtroMaestroID)
		err = e
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var m models.Millesimo
				rows.Scan(&m.ID, &m.MaestroID, &m.SocioID, &m.Voto, &m.AggiornatoIl,
					&m.NomeSocio, &m.CognomeSocio, &m.NomeMaestro)
				millesimi = append(millesimi, m)
			}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderTempl(w, r, templates.Page("Millesimi (Admin)", "millesimi",
		templates.MillesimiAdminPage(maestri, filtroMaestroID, millesimi)))
}

func (h *Handler) SalvaMillesimo(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form non valido", http.StatusBadRequest)
		return
	}
	ctx := r.Context()

	maestroID, ok := userIDFromRequest(r)
	if !ok {
		http.Error(w, "Non autenticato", http.StatusUnauthorized)
		return
	}

	socioID, _ := strconv.Atoi(r.FormValue("socio_id"))
	votoStr := r.FormValue("voto")

	if votoStr == "" {
		// Voto vuoto = cancella
		h.DB.ExecContext(ctx, `DELETE FROM millesimi WHERE maestro_id=? AND socio_id=?`, maestroID, socioID)
		w.WriteHeader(http.StatusOK)
		return
	}

	voto, err := strconv.Atoi(votoStr)
	if err != nil || voto < 0 || voto > 1000 {
		http.Error(w, "Voto non valido (0-1000)", http.StatusBadRequest)
		return
	}

	_, err = h.DB.ExecContext(ctx, `
		INSERT INTO millesimi (maestro_id, socio_id, voto, aggiornato_il)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(maestro_id, socio_id) DO UPDATE SET voto=excluded.voto, aggiornato_il=CURRENT_TIMESTAMP
	`, maestroID, socioID, voto)
	if err != nil {
		log.Printf("[millesimi] salva: %v", err)
		http.Error(w, "Errore DB", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
```

- [ ] **Step 2: Verifica compilazione**

```bash
go build ./handlers/...
```

- [ ] **Step 3: Commit**

```bash
git add handlers/millesimi.go
git commit -m "feat: handlers millesimi (maestro + admin)"
```

---

### Task 11: templates/corsi.templ

**Files:**
- Create: `templates/corsi.templ`

- [ ] **Step 1: Crea templates/corsi.templ**

```templ
package templates

import (
	"tango-gestionale/models"
	"strconv"
	"fmt"
)

// --- Lista Corsi ---

templ CorsiPage(corsi []models.Corso) {
	<div class="page-container">
		<div class="page-header">
			<h1 class="page-title">Corsi</h1>
			if IsAdminCtx(ctx) || !IsMaestroCtx(ctx) {
				<a href="/corsi/nuovo" class="btn-primary">+ Nuovo Corso</a>
			}
		</div>
		if len(corsi) == 0 {
			<p class="empty-state">Nessun corso presente.</p>
		} else {
			<div class="overflow-x-auto">
				<table class="data-table">
					<thead>
						<tr>
							<th>Titolo</th>
							<th>Giorno · Ora</th>
							<th>Maestro</th>
							<th>Periodo</th>
							<th>Iscritti/Posti</th>
							<th>Prezzo</th>
							<th>Stato</th>
							<th></th>
						</tr>
					</thead>
					<tbody>
						for _, c := range corsi {
							<tr class={ templ.KV("opacity-50", !c.Attivo) }>
								<td class="font-semibold">
									<a href={ templ.SafeURL("/corsi/" + strconv.Itoa(c.ID)) }>{ c.Titolo }</a>
								</td>
								<td>{ c.GiornoLabel() } { c.Ora }</td>
								<td>{ c.MaestroNome }</td>
								<td>{ c.DataInizio.Format("02/01") }–{ c.DataFine.Format("02/01/06") }</td>
								<td>
									{ strconv.Itoa(c.Iscritti) }/{ strconv.Itoa(c.MaxPosti) }
								</td>
								<td>
									€{ fmt.Sprintf("%.2f", c.PrezzoLezione) }
									if c.EtaMaxGiovani != nil {
										<span class="badge badge-gold ml-1">Under { strconv.Itoa(*c.EtaMaxGiovani) }</span>
									}
								</td>
								<td>
									if c.Attivo {
										<span class="badge badge-green">Attivo</span>
									} else {
										<span class="badge badge-gray">Inattivo</span>
									}
								</td>
								<td class="text-right">
									if !IsMaestroCtx(ctx) {
										<a href={ templ.SafeURL("/corsi/" + strconv.Itoa(c.ID) + "/modifica") } class="link-action mr-2">Modifica</a>
										<button
											hx-delete={ "/corsi/" + strconv.Itoa(c.ID) }
											hx-confirm="Eliminare il corso e tutte le lezioni?"
											hx-target="closest tr"
											hx-swap="outerHTML"
											class="link-danger">Elimina</button>
									}
								</td>
							</tr>
						}
					</tbody>
				</table>
			</div>
		}
	</div>
}

// --- Form Nuovo/Modifica Corso ---

templ NuovoCorsoForm(maestri []models.Utente, lezioniPreview int, corso ...*models.Corso) {
	<div class="page-container max-w-2xl">
		<h1 class="page-title">
			if len(corso) > 0 && corso[0] != nil {
				Modifica Corso
			} else {
				Nuovo Corso
			}
		</h1>
		<form
			if len(corso) > 0 && corso[0] != nil {
				method="POST"
				hx-put={ "/corsi/" + strconv.Itoa(corso[0].ID) }
			} else {
				method="POST"
				action="/corsi"
			}
			class="form-card"
		>
			<div class="form-group">
				<label class="form-label">Titolo</label>
				<input type="text" name="titolo" class="form-input" required
					if len(corso) > 0 && corso[0] != nil {
						value={ corso[0].Titolo }
					}
				/>
			</div>
			<div class="form-row">
				<div class="form-group">
					<label class="form-label">Giorno della settimana</label>
					<select name="giorno_settimana" class="form-input">
						for i, g := range models.GiorniSettimana {
							<option value={ strconv.Itoa(i) }
								if len(corso) > 0 && corso[0] != nil && corso[0].GiornoSettimana == i {
									selected
								}
							>{ g }</option>
						}
					</select>
				</div>
				<div class="form-group">
					<label class="form-label">Ora (HH:MM)</label>
					<input type="time" name="ora" class="form-input" required
						if len(corso) > 0 && corso[0] != nil {
							value={ corso[0].Ora }
						}
					/>
				</div>
			</div>
			<div class="form-row">
				<div class="form-group">
					<label class="form-label">Durata (min)</label>
					<input type="number" name="durata_min" class="form-input" value="60" min="15"
						if len(corso) > 0 && corso[0] != nil {
							value={ strconv.Itoa(corso[0].DurataMin) }
						}
					/>
				</div>
				<div class="form-group">
					<label class="form-label">Max posti</label>
					<input type="number" name="max_posti" class="form-input" min="1" required
						if len(corso) > 0 && corso[0] != nil {
							value={ strconv.Itoa(corso[0].MaxPosti) }
						}
					/>
				</div>
				<div class="form-group">
					<label class="form-label">Prezzo lezione (€)</label>
					<input type="number" step="0.01" name="prezzo_lezione" class="form-input"
						if len(corso) > 0 && corso[0] != nil {
							value={ fmt.Sprintf("%.2f", corso[0].PrezzoLezione) }
						} else {
							value="0"
						}
					/>
				</div>
			</div>
			<div class="form-group">
				<label class="form-label">Maestro</label>
				<select name="maestro_id" class="form-input">
					<option value="0">— Nessuno —</option>
					for _, m := range maestri {
						<option value={ strconv.Itoa(m.ID) }
							if len(corso) > 0 && corso[0] != nil && corso[0].MaestroID != nil && *corso[0].MaestroID == m.ID {
								selected
							}
						>{ m.Nome }</option>
					}
				</select>
			</div>
			<div class="form-row">
				<div class="form-group">
					<label class="form-label">Data inizio</label>
					<input type="date" name="data_inizio" class="form-input" required
						if len(corso) > 0 && corso[0] != nil {
							value={ corso[0].DataInizio.Format("2006-01-02") }
						}
					/>
				</div>
				<div class="form-group">
					<label class="form-label">Data fine</label>
					<input type="date" name="data_fine" class="form-input" required
						if len(corso) > 0 && corso[0] != nil {
							value={ corso[0].DataFine.Format("2006-01-02") }
						}
					/>
				</div>
			</div>
			<div class="form-group">
				<label class="flex items-center gap-2 cursor-pointer">
					<input type="checkbox" name="sconto_giovani" id="scontoGiovani"
						if len(corso) > 0 && corso[0] != nil && corso[0].EtaMaxGiovani != nil {
							checked
						}
					/>
					<span class="form-label mb-0">Sconto giovani</span>
				</label>
			</div>
			<div id="scontoGiovaniFields" class="form-row"
				if len(corso) == 0 || corso[0] == nil || corso[0].EtaMaxGiovani == nil {
					style="display:none"
				}
			>
				<div class="form-group">
					<label class="form-label">Età max giovani</label>
					<input type="number" name="eta_max_giovani" class="form-input" min="1" max="99"
						if len(corso) > 0 && corso[0] != nil && corso[0].EtaMaxGiovani != nil {
							value={ strconv.Itoa(*corso[0].EtaMaxGiovani) }
						}
					/>
				</div>
				<div class="form-group">
					<label class="form-label">Prezzo giovani (€)</label>
					<input type="number" step="0.01" name="prezzo_giovani" class="form-input"
						if len(corso) > 0 && corso[0] != nil && corso[0].PrezzoGiovani != nil {
							value={ fmt.Sprintf("%.2f", *corso[0].PrezzoGiovani) }
						}
					/>
				</div>
			</div>
			if len(corso) > 0 && corso[0] != nil {
				<div class="form-group">
					<label class="flex items-center gap-2 cursor-pointer">
						<input type="checkbox" name="attivo"
							if corso[0].Attivo {
								checked
							}
						/>
						<span class="form-label mb-0">Corso attivo</span>
					</label>
				</div>
			}
			if lezioniPreview > 0 {
				<p class="text-sm text-amber-700 mt-2">Verranno generate <strong>{ strconv.Itoa(lezioniPreview) }</strong> lezioni.</p>
			}
			<div class="form-actions">
				<a href="/corsi" class="btn-secondary">Annulla</a>
				<button type="submit" class="btn-primary">Salva</button>
			</div>
		</form>
		<script>
			document.getElementById('scontoGiovani').addEventListener('change', function() {
				document.getElementById('scontoGiovaniFields').style.display = this.checked ? 'flex' : 'none';
			});
		</script>
	</div>
}

// --- Dettaglio Corso ---

templ DettaglioCorsoPage(corso models.Corso, lezioni []models.Lezione, iscritti []models.IscrizioneCorso, soci []models.Socio) {
	<div class="page-container">
		<div class="page-header">
			<div>
				<h1 class="page-title">{ corso.Titolo }</h1>
				<p class="text-sm text-gray-500">
					{ corso.GiornoLabel() } { corso.Ora } · { corso.DataInizio.Format("02/01/2006") }–{ corso.DataFine.Format("02/01/2006") }
					if corso.MaestroNome != "" {
						· Maestro: { corso.MaestroNome }
					}
				</p>
			</div>
			if !IsMaestroCtx(ctx) {
				<a href={ templ.SafeURL("/corsi/" + strconv.Itoa(corso.ID) + "/modifica") } class="btn-secondary">Modifica</a>
			}
		</div>

		<!-- Calendario Lezioni -->
		<section class="section-card mt-6">
			<h2 class="section-title">Calendario Lezioni ({ strconv.Itoa(len(lezioni)) })</h2>
			<table class="data-table">
				<thead>
					<tr>
						<th>Data</th>
						<th>Ora</th>
						<th>Stato</th>
						<th>Presenti/Iscritti</th>
						<th>Nota</th>
						<th></th>
					</tr>
				</thead>
				<tbody>
					for _, l := range lezioni {
						<tr class={ templ.KV("opacity-40", l.Stato == "annullata") }>
							<td>{ l.Data.Format("02/01/2006") }</td>
							<td>{ l.Ora }</td>
							<td>
								switch l.Stato {
									case "completata":
										<span class="badge badge-green">Completata</span>
									case "annullata":
										<span class="badge badge-red">Annullata</span>
									default:
										<span class="badge badge-gray">Programmata</span>
								}
							</td>
							<td>{ strconv.Itoa(l.Presenti) }/{ strconv.Itoa(len(iscritti)) }</td>
							<td class="text-sm text-gray-500">{ l.Nota }</td>
							<td>
								<a href={ templ.SafeURL("/corsi/" + strconv.Itoa(corso.ID) + "/lezioni/" + strconv.Itoa(l.ID)) } class="link-action">Apri</a>
							</td>
						</tr>
					}
				</tbody>
			</table>
		</section>

		<!-- Iscritti al Corso -->
		if !IsMaestroCtx(ctx) {
			<section class="section-card mt-6">
				<div class="flex justify-between items-center mb-4">
					<h2 class="section-title">Iscritti ({ strconv.Itoa(len(iscritti)) }/{ strconv.Itoa(corso.MaxPosti) })</h2>
					<form method="POST" action={ templ.SafeURL("/corsi/" + strconv.Itoa(corso.ID) + "/iscrizioni") } class="flex gap-2">
						<select name="socio_id" class="form-input form-input-sm">
							<option value="">— Seleziona socio —</option>
							for _, s := range soci {
								<option value={ strconv.FormatInt(s.ID, 10) }>{ s.Cognome } { s.Nome }</option>
							}
						</select>
						<button type="submit" class="btn-primary btn-sm">Iscrivi</button>
					</form>
				</div>
				<table class="data-table" id="tabellaIscritti">
					<thead>
						<tr>
							<th>Socio</th>
							<th>Sconto Giovani</th>
							<th>Prezzo</th>
							<th>Override Prezzo</th>
							<th></th>
						</tr>
					</thead>
					<tbody>
						for _, ic := range iscritti {
							@IscrittaRiga(ic, corso)
						}
					</tbody>
				</table>
			</section>
		}
	</div>
}

// IscrittaRiga è il partial HTMX per una riga iscritto (aggiornabile inline).
templ IscrittaRiga(ic models.IscrizioneCorso, corso models.Corso) {
	<tr id={ "riga-iscritto-" + strconv.Itoa(ic.SocioID) }>
		<td>{ ic.CognomeSocio } { ic.NomeSocio }</td>
		<td>
			<div class="flex items-center gap-1">
				<input
					type="checkbox"
					hx-post={ "/corsi/" + strconv.Itoa(ic.CorsoID) + "/iscrizioni/" + strconv.Itoa(ic.SocioID) + "/giovani" }
					hx-vals={ `{"sconto": "` + scontoChecked(ic) + `"}` }
					hx-target={ "#riga-iscritto-" + strconv.Itoa(ic.SocioID) }
					hx-swap="outerHTML"
					if scontoIsActive(ic) {
						checked
					}
				/>
				if ic.ScontoGiovaniForzato != nil {
					<button
						hx-delete={ "/corsi/" + strconv.Itoa(ic.CorsoID) + "/iscrizioni/" + strconv.Itoa(ic.SocioID) + "/giovani" }
						hx-target={ "#riga-iscritto-" + strconv.Itoa(ic.SocioID) }
						hx-swap="outerHTML"
						class="text-gray-400 hover:text-amber-600"
						title="Reset ad automatico"
					>↺</button>
				}
			</div>
		</td>
		<td>
			€{ fmt.Sprintf("%.2f", ic.PrezzoEffettivo) }
			<span class="badge badge-gray ml-1">{ ic.EtichettaPrezzo }</span>
		</td>
		<td>
			<form
				hx-put={ "/corsi/" + strconv.Itoa(ic.CorsoID) + "/iscrizioni/" + strconv.Itoa(ic.SocioID) }
				hx-target={ "#riga-iscritto-" + strconv.Itoa(ic.SocioID) }
				hx-swap="outerHTML"
				class="flex gap-1"
			>
				<input type="number" step="0.01" name="prezzo_custom" class="form-input form-input-xs"
					placeholder="override"
					if ic.PrezzoCustom != nil {
						value={ fmt.Sprintf("%.2f", *ic.PrezzoCustom) }
					}
				/>
				<button type="submit" class="btn-xs">✓</button>
			</form>
		</td>
		<td>
			<button
				hx-delete={ "/corsi/" + strconv.Itoa(ic.CorsoID) + "/iscrizioni/" + strconv.Itoa(ic.SocioID) }
				hx-confirm="Rimuovere l'iscrizione?"
				hx-target={ "#riga-iscritto-" + strconv.Itoa(ic.SocioID) }
				hx-swap="outerHTML"
				class="link-danger">✕</button>
		</td>
	</tr>
}

// --- Dettaglio Lezione ---

templ DettaglioLezioneContent(corso models.Corso, lezione models.Lezione, iscritti []models.IscrizioneCorso, presenti map[int]models.Presenza) {
	<div class="page-container">
		<div class="page-header">
			<div>
				<h1 class="page-title">{ lezione.Data.Format("Lunedì 02 January 2006") }</h1>
				<p class="text-sm text-gray-500">{ corso.Titolo } · { lezione.Ora } · { strconv.Itoa(lezione.DurataMin) } min</p>
			</div>
		</div>

		<div class="flex gap-2 mb-4">
			switch lezione.Stato {
				case "completata":
					<span class="badge badge-green">Completata</span>
				case "annullata":
					<span class="badge badge-red">Annullata</span>
				default:
					<span class="badge badge-gray">Programmata</span>
			}
			if lezione.Nota != "" {
				<span class="text-sm text-gray-500 italic">{ lezione.Nota }</span>
			}
		</div>

		if !IsMaestroCtx(ctx) {
			<form method="POST" action={ templ.SafeURL("/corsi/" + strconv.Itoa(corso.ID) + "/lezioni/" + strconv.Itoa(lezione.ID)) } class="form-row mb-6">
				<input type="hidden" name="_method" value="PUT"/>
				<input type="date" name="data" class="form-input form-input-sm" value={ lezione.Data.Format("2006-01-02") }/>
				<input type="time" name="ora" class="form-input form-input-sm" value={ lezione.Ora }/>
				<select name="stato" class="form-input form-input-sm">
					<option value="programmata" if lezione.Stato == "programmata" { selected }>Programmata</option>
					<option value="completata" if lezione.Stato == "completata" { selected }>Completata</option>
					<option value="annullata" if lezione.Stato == "annullata" { selected }>Annullata</option>
				</select>
				<input type="text" name="nota" class="form-input form-input-sm" placeholder="Nota..." value={ lezione.Nota }/>
				<button type="submit" class="btn-primary btn-sm">Salva</button>
			</form>
		}

		<section class="section-card">
			<h2 class="section-title">Presenze ({ strconv.Itoa(len(presenti)) }/{ strconv.Itoa(len(iscritti)) })</h2>
			<table class="data-table">
				<thead>
					<tr><th>Socio</th><th>Presente</th><th>Segnata da</th></tr>
				</thead>
				<tbody id="tabellaPresenze">
					for _, ic := range iscritti {
						@PresenzaRiga(ic, lezione.ID, corso.ID, isPresente(presenti, ic.SocioID), presenti[ic.SocioID])
					}
				</tbody>
			</table>
		</section>
	</div>
}

templ PresenzaRiga(ic models.IscrizioneCorso, lezioneID int, corsoID int, presente bool, presenza models.Presenza) {
	<tr id={ "presenza-" + strconv.Itoa(ic.SocioID) }>
		<td>{ ic.CognomeSocio } { ic.NomeSocio }</td>
		<td>
			<input
				type="checkbox"
				hx-post={ "/corsi/" + strconv.Itoa(corsoID) + "/lezioni/" + strconv.Itoa(lezioneID) + "/presenze" }
				hx-vals={ `{"socio_id":"` + strconv.Itoa(ic.SocioID) + `"}` }
				hx-target={ "#presenza-" + strconv.Itoa(ic.SocioID) }
				hx-swap="outerHTML"
				if presente {
					checked
				}
			/>
		</td>
		<td class="text-sm text-gray-400">
			if presente {
				{ presenza.NomeOperatore }
			}
		</td>
	</tr>
}

// --- Helpers per templates ---

func scontoIsActive(ic models.IscrizioneCorso) bool {
	if ic.ScontoGiovaniForzato != nil {
		return *ic.ScontoGiovaniForzato
	}
	return ic.GiovaniAuto
}

func scontoChecked(ic models.IscrizioneCorso) string {
	if scontoIsActive(ic) {
		return "false"
	}
	return "true"
}

func isPresente(presenti map[int]models.Presenza, socioID int) bool {
	_, ok := presenti[socioID]
	return ok
}
```

- [ ] **Step 2: Genera i template**

```bash
templ generate
```

Expected: nessun errore, crea `templates/corsi_templ.go`.

- [ ] **Step 3: Verifica compilazione**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add templates/corsi.templ templates/corsi_templ.go
git commit -m "feat: template corsi, lezioni e presenze"
```

---

### Task 12: templates/millesimi.templ

**Files:**
- Create: `templates/millesimi.templ`

- [ ] **Step 1: Crea templates/millesimi.templ**

```templ
package templates

import (
	"tango-gestionale/models"
	"strconv"
)

// --- Vista Maestro ---

templ MillesimiMaestroPage(soci []models.Socio, voti map[int]int) {
	<div class="page-container">
		<div class="page-header">
			<h1 class="page-title">I Miei Millesimi</h1>
		</div>
		<p class="text-sm text-gray-500 mb-4">Assegna un voto da 0 a 1000 a ciascun socio. Salvataggio automatico.</p>
		<table class="data-table">
			<thead>
				<tr>
					<th>Socio</th>
					<th>Voto (0–1000)</th>
				</tr>
			</thead>
			<tbody>
				for _, s := range soci {
					<tr id={ "millesimo-row-" + strconv.FormatInt(s.ID, 10) }>
						<td>{ s.Cognome } { s.Nome }</td>
						<td>
							<input
								type="number"
								min="0"
								max="1000"
								name="voto"
								class="form-input form-input-sm millesimo-input"
								data-socio-id={ strconv.FormatInt(s.ID, 10) }
								if v, ok := voti[int(s.ID)]; ok {
									value={ strconv.Itoa(v) }
								}
								hx-post="/millesimi"
								hx-vals={ `{"socio_id":"` + strconv.FormatInt(s.ID, 10) + `"}` }
								hx-include="[name='voto']"
								hx-trigger="change, blur"
								hx-swap="none"
								_="on htmx:afterRequest add .glow-gold to me then wait 1s then remove .glow-gold from me"
							/>
						</td>
					</tr>
				}
			</tbody>
		</table>
	</div>
}

// --- Vista Admin ---

templ MillesimiAdminPage(maestri []models.Utente, filtroMaestroID int, millesimi []models.Millesimo) {
	<div class="page-container">
		<div class="page-header">
			<h1 class="page-title">Millesimi — Vista Admin</h1>
		</div>
		<form method="GET" action="/millesimi" class="flex items-center gap-3 mb-6">
			<input type="hidden" name="view" value="admin"/>
			<label class="form-label mb-0">Filtra per maestro:</label>
			<select name="maestro_id" class="form-input" onchange="this.form.submit()">
				<option value="0">— Tutti —</option>
				for _, m := range maestri {
					<option value={ strconv.Itoa(m.ID) }
						if m.ID == filtroMaestroID {
							selected
						}
					>{ m.Nome }</option>
				}
			</select>
		</form>
		if filtroMaestroID > 0 && len(millesimi) == 0 {
			<p class="empty-state">Nessun voto inserito da questo maestro.</p>
		} else if filtroMaestroID == 0 {
			<p class="text-sm text-gray-400">Seleziona un maestro per visualizzarne i voti.</p>
		} else {
			<table class="data-table">
				<thead>
					<tr>
						<th>Socio</th>
						<th>Voto</th>
						<th>Aggiornato il</th>
					</tr>
				</thead>
				<tbody>
					for _, m := range millesimi {
						<tr>
							<td>{ m.CognomeSocio } { m.NomeSocio }</td>
							<td class="font-semibold">{ strconv.Itoa(m.Voto) }</td>
							<td class="text-sm text-gray-400">{ m.AggiornatoIl.Format("02/01/2006 15:04") }</td>
						</tr>
					}
				</tbody>
			</table>
		}
	</div>
}
```

- [ ] **Step 2: Genera i template**

```bash
templ generate
```

- [ ] **Step 3: Verifica compilazione**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add templates/millesimi.templ templates/millesimi_templ.go
git commit -m "feat: template millesimi (maestro + admin)"
```

---

### Task 13: Layout sidebar — templates/layout.templ

**Files:**
- Modify: `templates/layout.templ`

- [ ] **Step 1: Sostituisci il blocco ATTIVITÀ nella sidebar**

Trova nella sidebar il blocco:
```templ
<div class="nav-group">
    <div class="nav-group-title">Attività</div>
    <a href="/lezioni" class={ "nav-link", templ.KV("active", activePage == "lezioni") }>
        <span class="numeral">IV</span> Lezioni
    </a>
    <a href="/eventi" class={ "nav-link", templ.KV("active", activePage == "eventi") }>
        <span class="numeral">V</span> Eventi
    </a>
</div>
```

Sostituisci con:
```templ
<div class="nav-group">
    <div class="nav-group-title">Attività</div>
    <a href="/corsi" class={ "nav-link", templ.KV("active", activePage == "corsi") }>
        <span class="numeral">IV</span> Corsi
    </a>
    <a href="/eventi" class={ "nav-link", templ.KV("active", activePage == "eventi") }>
        <span class="numeral">V</span> Eventi
    </a>
</div>
```

- [ ] **Step 2: Aggiungi sezione INSEGNAMENTO per maestro — prima del blocco `if IsAdminCtx(ctx)`**

```templ
if IsMaestroCtx(ctx) {
    <div class="nav-group">
        <div class="nav-group-title">Insegnamento</div>
        <a href="/millesimi" class={ "nav-link", templ.KV("active", activePage == "millesimi") }>
            <span class="numeral">✦</span> Millesimi
        </a>
    </div>
}
```

- [ ] **Step 3: Aggiungi link Millesimi nel blocco admin**

Trova il blocco AMMINISTRAZIONE:
```templ
if IsAdminCtx(ctx) {
    <div class="nav-group">
        <div class="nav-group-title">Amministrazione</div>
        <a href="/admin/utenti" class={ "nav-link", templ.KV("active", activePage == "admin-utenti") }>
            <span class="numeral">✦</span> Utenti
        </a>
    </div>
}
```

Sostituisci con:
```templ
if IsAdminCtx(ctx) {
    <div class="nav-group">
        <div class="nav-group-title">Amministrazione</div>
        <a href="/admin/utenti" class={ "nav-link", templ.KV("active", activePage == "admin-utenti") }>
            <span class="numeral">✦</span> Utenti
        </a>
        <a href="/millesimi?view=admin" class={ "nav-link", templ.KV("active", activePage == "millesimi") }>
            <span class="numeral">✦</span> Millesimi
        </a>
    </div>
}
```

- [ ] **Step 4: Genera e compila**

```bash
templ generate && go build ./...
```

Expected: nessun errore.

- [ ] **Step 5: Commit**

```bash
git add templates/layout.templ templates/layout_templ.go
git commit -m "feat: sidebar aggiornata con Corsi, Insegnamento e Millesimi"
```

---

### Task 14: Routing — main.go

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Aggiungi le nuove route nel blocco protected (dopo le route Tessere)**

Individua nel file main.go il commento `// Lezioni` con le route esistenti. Sostituisci l'intero blocco lezioni con:

```go
// Redirect legacy
r.Get("/lezioni", func(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, "/corsi", http.StatusMovedPermanently)
})

// Corsi (admin/staff) + lettura (maestro)
r.Get("/corsi", h.ListaCorsi)
r.Group(func(r chi.Router) {
    r.Use(h.RequireStaffOrAbove)
    r.Get("/corsi/nuovo", h.NuovoCorsoForm)
    r.Post("/corsi", h.CreaCorso)
    r.Get("/corsi/{id}/modifica", h.ModificaCorsoForm)
    r.Put("/corsi/{id}", h.AggiornaCorso)
    r.Delete("/corsi/{id}", h.EliminaCorso)

    r.Post("/corsi/{id}/iscrizioni", h.IscriviSocio)
    r.Delete("/corsi/{id}/iscrizioni/{socioID}", h.RimuoviIscrizioneCorso)
    r.Put("/corsi/{id}/iscrizioni/{socioID}", h.AggiornaPrezzoCustom)
    r.Post("/corsi/{id}/iscrizioni/{socioID}/giovani", h.ToggleScontoGiovani)
    r.Delete("/corsi/{id}/iscrizioni/{socioID}/giovani", h.ResetScontoGiovani)

    r.Put("/corsi/{id}/lezioni/{lid}", h.AggiornaLezione)
})
r.Get("/corsi/{id}", h.DettaglioCorso)
r.Get("/corsi/{id}/lezioni/{lid}", h.DettaglioLezione)
r.Post("/corsi/{id}/lezioni/{lid}/presenze", h.TogglePresenza)

// Millesimi
r.Group(func(r chi.Router) {
    r.Get("/millesimi", h.ListaMillesimi)
    r.Post("/millesimi", h.SalvaMillesimo)
})
```

- [ ] **Step 2: Verifica compilazione**

```bash
go build ./...
```

Expected: nessun errore.

- [ ] **Step 3: Commit**

```bash
git add main.go
git commit -m "feat: routing /corsi, /millesimi, redirect /lezioni"
```

---

### Task 15: Cleanup — rimozione file legacy

**Files:**
- Delete: `handlers/lezioni.go`
- Delete: `templates/lezioni.templ`
- Delete: `templates/lezioni_templ.go` (file generato)

- [ ] **Step 1: Rimuovi i file legacy**

```bash
rm handlers/lezioni.go templates/lezioni.templ templates/lezioni_templ.go
```

- [ ] **Step 2: Verifica compilazione**

```bash
go build ./...
```

Expected: nessun errore. Se compaiono reference a tipi o funzioni vecchie (es. `Lezione.Titolo`, `Lezione.Insegnante`), correggile nel file che le usa.

- [ ] **Step 3: Esegui tutti i test**

```bash
go test ./...
```

Expected: tutti i test PASS (almeno i test CalcolaPrezzo del Task 5).

- [ ] **Step 4: Commit finale**

```bash
git add -A
git commit -m "feat: rimozione handler/template lezioni legacy — completata migrazione a corsi"
```

---

## Checklist finale spec

Dopo aver completato tutti i task, verifica:

- [ ] Schema SQL contiene tutte le tabelle: `corsi`, `lezioni` (nuova), `iscrizioni_corso`, `presenze`, `millesimi`
- [ ] Schema SQL NON contiene `iscrizioni_lezione`
- [ ] `utenti.ruolo` CHECK include `'maestro'`
- [ ] Le 3 migrazioni in `db/db.go` funzionano: password_hash, lezioni.titolo, ruolo maestro
- [ ] `go test ./models/... -run TestCalcola` → PASS
- [ ] Rotta `/lezioni` redirige a `/corsi` con 301
- [ ] Rotta `/corsi/{id}/lezioni/{lid}/presenze` è accessibile a maestro
- [ ] Sidebar mostra "Corsi" al posto di "Lezioni"
- [ ] Sezione "Insegnamento" visibile solo a maestro
- [ ] Sezione "Millesimi" in Amministrazione visibile solo ad admin
- [ ] `go build ./...` → nessun errore
