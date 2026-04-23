# Corsi, Lezioni, Presenze e Millesimi — Design Spec

## Contesto

TangoBar è un gestionale per scuola di tango. Stack: Go + templ + chi + HTMX + SQLite.
Attualmente `lezioni` è una tabella piatta (titolo, insegnante stringa libera, data, posti, prezzo). Si vuole ristrutturare in un modello a due livelli con corsi ricorrenti, presenze, e un sistema di valutazione maestro→socio.

## Glossario

- **Corso**: template ricorrente settimanale (es. "Principianti Lunedì 20:30, Set–Dic").
- **Lezione**: singola istanza concreta di un corso, con data specifica.
- **Socio**: allievo della scuola (tabella `soci`). Non accede al gestionale.
- **Utente**: account che accede al gestionale (tabella `utenti`). Ruoli: admin, staff, maestro.
- **Maestro**: ruolo utente. Può segnare presenze e gestire i propri millesimi.
- **Millesimi**: voto globale (0–1000) assegnato da un maestro a un socio. Un voto per coppia (maestro, socio).

---

## 1. Modello Dati

### 1.1 Ruoli utenti

Aggiornare il CHECK constraint di `utenti.ruolo`:

```sql
CHECK(ruolo IN ('admin', 'staff', 'maestro'))
```

| Ruolo   | Capacità |
|---------|----------|
| admin   | Tutto, inclusa gestione utenti e vista globale millesimi |
| staff   | Tutto tranne gestione utenti |
| maestro | Sola lettura su corsi/lezioni + segnare presenze + gestire millesimi propri |

### 1.2 Nuova tabella `corsi`

```sql
CREATE TABLE IF NOT EXISTS corsi (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    titolo TEXT NOT NULL,
    giorno_settimana INTEGER NOT NULL CHECK(giorno_settimana BETWEEN 0 AND 6),  -- 0=lunedì
    ora TEXT NOT NULL,                   -- "20:30"
    durata_min INTEGER NOT NULL DEFAULT 60,
    max_posti INTEGER NOT NULL,
    prezzo_lezione REAL NOT NULL DEFAULT 0,
    maestro_id INTEGER REFERENCES utenti(id) ON DELETE SET NULL,
    data_inizio DATE NOT NULL,
    data_fine DATE NOT NULL,
    eta_max_giovani INTEGER,             -- NULL = nessuno sconto giovani
    prezzo_giovani REAL,                 -- prezzo fisso per giovani (usato se eta_max_giovani != NULL)
    attivo BOOLEAN NOT NULL DEFAULT TRUE,
    creato_il DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 1.3 Ristrutturazione tabella `lezioni`

Drop e recreate (DB appena deployato, nessun dato reale):

```sql
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
```

### 1.4 Nuova tabella `iscrizioni_corso`

```sql
CREATE TABLE IF NOT EXISTS iscrizioni_corso (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    socio_id INTEGER NOT NULL REFERENCES soci(id) ON DELETE CASCADE,
    corso_id INTEGER NOT NULL REFERENCES corsi(id) ON DELETE CASCADE,
    prezzo_custom REAL,                  -- NULL = usa regola standard
    sconto_giovani_forzato BOOLEAN,      -- NULL = auto (calcola da età), TRUE = forzato sì, FALSE = forzato no
    creato_il DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(socio_id, corso_id)
);
```

### 1.5 Nuova tabella `presenze`

```sql
CREATE TABLE IF NOT EXISTS presenze (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    socio_id INTEGER NOT NULL REFERENCES soci(id) ON DELETE CASCADE,
    lezione_id INTEGER NOT NULL REFERENCES lezioni(id) ON DELETE CASCADE,
    segnata_da INTEGER NOT NULL REFERENCES utenti(id),
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(socio_id, lezione_id)
);
CREATE INDEX IF NOT EXISTS idx_presenze_lezione ON presenze(lezione_id);
```

### 1.6 Nuova tabella `millesimi`

```sql
CREATE TABLE IF NOT EXISTS millesimi (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    maestro_id INTEGER NOT NULL REFERENCES utenti(id) ON DELETE CASCADE,
    socio_id INTEGER NOT NULL REFERENCES soci(id) ON DELETE CASCADE,
    voto INTEGER NOT NULL CHECK(voto BETWEEN 0 AND 1000),
    aggiornato_il DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(maestro_id, socio_id)
);
CREATE INDEX IF NOT EXISTS idx_millesimi_maestro ON millesimi(maestro_id);
```

### 1.7 Rimozione tabelle legacy

```sql
DROP TABLE IF EXISTS iscrizioni_lezione;
```

La vecchia tabella `iscrizioni_lezione` viene sostituita da `iscrizioni_corso` (iscrizione al corso) + `presenze` (registro per singola lezione).

### 1.8 Migrazione

In `db/db.go`, rilevare la colonna legacy `lezioni.titolo` (presente nella vecchia schema):
- Se esiste: `DROP TABLE iscrizioni_lezione; DROP TABLE lezioni;`
- Il blocco `CREATE TABLE IF NOT EXISTS` in schema.sql creerà le nuove tabelle.

### 1.9 Logica prezzo

Priorità per determinare il prezzo di un socio in un corso:

1. `iscrizioni_corso.prezzo_custom` — se NOT NULL, vince sempre (accordo economico specifico)
2. Sconto giovani — risolto tramite `iscrizioni_corso.sconto_giovani_forzato`:
   - `TRUE` → applica `corsi.prezzo_giovani` (forzato manualmente)
   - `FALSE` → salta sconto, usa `corsi.prezzo_lezione` (escluso manualmente)
   - `NULL` (auto) → calcola da età: se `corsi.eta_max_giovani` IS NOT NULL e l'età del socio (da `soci.data_nascita`) ≤ soglia → `corsi.prezzo_giovani`, altrimenti `corsi.prezzo_lezione`
3. `corsi.prezzo_lezione` — default

Funzione Go helper `CalcolaPrezzo(corso, iscrizione, socio) → (importo float64, etichetta string)`:
- Restituisce importo e etichetta ("custom" / "giovani" / "giovani (forzato)" / "standard") per la UI.

### 1.10 UI dello sconto giovani nella tabella iscritti

Ogni riga socio nella tabella iscritti del corso mostra:

- **Checkbox sconto giovani**: riflette lo stato corrente (checked = sconto applicato).
  - Stato **auto** (NULL): la checkbox riflette il calcolo automatico per età. Aspetto normale.
  - Stato **forzato** (TRUE/FALSE): la checkbox è nello stato forzato. Appare un'**icona reset** (es. ↺) subito a fianco, minimalista, solo icona, nessun testo. Al click rimette `sconto_giovani_forzato = NULL` e torna in auto.
- **Toggle manuale**: quando l'utente clicca la checkbox, il valore viene scritto staticamente (`TRUE` o `FALSE`), e l'icona reset appare. Da quel momento il calcolo per età è ignorato fino al reset.
- **Interazione HTMX**: il toggle della checkbox fa POST inline; il reset fa POST inline. Entrambi restituiscono la riga aggiornata.

---

## 2. Generazione Lezioni

Alla creazione di un corso, il backend:

1. Calcola tutte le date tra `data_inizio` e `data_fine` che cadono su `giorno_settimana`.
2. Inserisce una riga `lezioni` per ogni data con: `ora`, `durata_min`, `max_posti`, `prezzo` ereditati dal corso; `stato = 'programmata'`.

### Gestione eccezioni

Ogni lezione generata è una riga indipendente modificabile:
- **Spostare**: cambiare `data` e/o `ora` sulla singola lezione. Campo `nota` per annotare il motivo (es. "Spostata da lun 25/12 per Natale").
- **Annullare**: impostare `stato = 'annullata'`.
- **Aggiungere**: inserire manualmente una lezione extra dentro il corso.
- **Completare**: `stato = 'completata'` (impostabile dopo che le presenze sono segnate, o automaticamente dopo la data).

---

## 3. Routing

### 3.1 Corsi e Lezioni

```
GET    /corsi                                Lista corsi (tutti i logged-in)
GET    /corsi/nuovo                          Form nuovo corso (admin/staff)
POST   /corsi                                Crea corso + genera lezioni (admin/staff)
GET    /corsi/{id}                           Dettaglio corso (tutti)
GET    /corsi/{id}/modifica                  Form modifica corso (admin/staff)
PUT    /corsi/{id}                           Aggiorna corso (admin/staff)
DELETE /corsi/{id}                           Elimina corso + cascade (admin/staff)

POST   /corsi/{id}/iscrizioni               Iscrivi socio (admin/staff)
DELETE /corsi/{id}/iscrizioni/{socioID}      Rimuovi iscrizione (admin/staff)
PUT    /corsi/{id}/iscrizioni/{socioID}      Aggiorna prezzo_custom (admin/staff)
POST   /corsi/{id}/iscrizioni/{socioID}/giovani       Toggle sconto giovani forzato (admin/staff)
DELETE /corsi/{id}/iscrizioni/{socioID}/giovani       Reset sconto giovani ad auto (admin/staff)

GET    /corsi/{id}/lezioni/{lid}             Dettaglio lezione + presenze (tutti)
PUT    /corsi/{id}/lezioni/{lid}             Modifica singola lezione (admin/staff)
POST   /corsi/{id}/lezioni/{lid}/presenze   Toggle presenza socio (admin/staff/maestro)
```

### 3.2 Millesimi

```
GET    /millesimi                            Maestro: propri voti. Admin: tutti (con filtro maestro)
POST   /millesimi                            Crea/aggiorna voto (solo maestro per i propri)
```

### 3.3 Legacy redirect

```
GET    /lezioni   → 301 → /corsi
```

### 3.4 Rimozione route legacy

Rimuovere tutte le route `/lezioni/*` da main.go e il file `handlers/lezioni.go`.

---

## 4. Permessi

### 4.1 Middleware

Aggiungere `RequireStaffOrAbove` middleware (permette admin + staff, blocca maestro).
`RequireAdmin` esiste già.
Le route presenze usano `RequireAuth` (tutti i logged-in: admin, staff, maestro).

### 4.2 Matrice permessi

| Azione | admin | staff | maestro |
|--------|-------|-------|---------|
| CRUD corsi | ✓ | ✓ | — |
| Iscrivere soci a corsi | ✓ | ✓ | — |
| Spostare/annullare lezione | ✓ | ✓ | — |
| Segnare presenze | ✓ | ✓ | ✓ |
| Vedere lista corsi/lezioni | ✓ | ✓ | ✓ |
| Gestire millesimi propri | — | — | ✓ |
| Vedere tutti i millesimi | ✓ | — | — |
| Gestione utenti | ✓ | — | — |

---

## 5. Sidebar

```
GESTIONE
  I    Dashboard
  II   Soci
  III  Tessere

ATTIVITÀ
  IV   Corsi        (/corsi)     ← sostituisce "Lezioni"
  V    Eventi

BUSINESS
  VI   Fatture
  VII  Bar

INSEGNAMENTO              ← visibile solo a ruolo maestro
  ✦    Millesimi    (/millesimi)

AMMINISTRAZIONE           ← visibile solo a ruolo admin
  ✦    Utenti
  ✦    Millesimi    (/millesimi?view=admin)
```

---

## 6. UI Pagine

### 6.1 Lista Corsi (`/corsi`)

Tabella con colonne: Titolo, Giorno+Ora, Maestro, Periodo (inizio–fine), Iscritti/Posti, Prezzo (con badge "Sconto giovani" se `eta_max_giovani` impostato). Bottone "+ Nuovo Corso" (admin/staff). Filtro per stato attivo/inattivo.

### 6.2 Form Nuovo Corso

Campi: titolo, giorno della settimana (select lun–dom), ora, durata, max posti, prezzo, maestro (select tra utenti con ruolo maestro), data inizio, data fine, sconto giovani (checkbox toggle che mostra campi età max + prezzo giovani). Preview: "Verranno generate N lezioni".

### 6.3 Dettaglio Corso (`/corsi/{id}`)

Header con info corso + maestro.

Due sezioni:

**Calendario Lezioni** — tabella/griglia di tutte le lezioni generate. Colonne: Data, Ora, Stato (badge: programmata/completata/annullata), Presenti/Iscritti, Nota. Azioni inline: modifica data/ora, annulla, link al dettaglio lezione. Le annullate sono visivamente attenuate.

**Iscritti al Corso** — tabella soci iscritti. Colonne: Nome, Cognome, Sconto Giovani (checkbox + icona reset ↺ se forzato), Prezzo (con badge derivazione: custom/giovani/giovani forzato/standard), campo per impostare override prezzo custom. Azioni: rimuovi iscrizione. Bottone "+ Iscrivi socio" con ricerca socio. Checkbox e reset sono inline HTMX, la riga si aggiorna al toggle.

### 6.4 Dettaglio Lezione (`/corsi/{id}/lezioni/{lid}`)

Info lezione (data, ora, stato, nota). Lista di tutti gli iscritti al corso con checkbox presenza. Toggle via HTMX POST. Chi ha segnato la presenza viene mostrato discretamente. Possibilità di modificare data/ora/stato se admin/staff.

### 6.5 Millesimi — Vista Maestro (`/millesimi`)

Lista di tutti i soci della scuola. Per ognuno: nome, cognome, campo input numerico 0–1000 con il voto corrente (o vuoto se mai votato). Salvataggio inline via HTMX POST al blur o invio. Feedback visivo: glow dorato al salvataggio.

### 6.6 Millesimi — Vista Admin (`/millesimi?view=admin`)

Select filtro per maestro in alto. Tabella: Socio, Voto, Aggiornato il. Sola lettura.

---

## 7. Modelli Go

```go
// models/corso.go
type Corso struct {
    ID              int
    Titolo          string
    GiornoSettimana int       // 0=lunedì
    Ora             string    // "20:30"
    DurataMin       int
    MaxPosti        int
    PrezzoLezione   float64
    MaestroID       *int
    MaestroNome     string    // join
    DataInizio      time.Time
    DataFine        time.Time
    EtaMaxGiovani   *int
    PrezzoGiovani   *float64
    Attivo          bool
    CreatoIl        time.Time
    Iscritti        int       // count join
    TotaleLezioni   int       // count join
}

// models/lezione.go (ristrutturata)
type Lezione struct {
    ID        int
    CorsoID   int
    Data      time.Time
    Ora       string
    DurataMin int
    MaxPosti  int
    Prezzo    float64
    Stato     string  // "programmata" | "completata" | "annullata"
    Nota      string
    Presenti  int     // count join
}

type Presenza struct {
    ID         int
    SocioID    int
    LezioneID  int
    SegnataDa  int
    Timestamp  time.Time
    NomeSocio  string  // join
    CognomeSocio string // join
    NomeOperatore string // join (chi ha segnato)
}

// models/iscrizione_corso.go
type IscrizioneCorso struct {
    ID                   int
    SocioID              int
    CorsoID              int
    PrezzoCustom         *float64
    ScontoGiovaniForzato *bool    // NULL=auto, TRUE=forzato sì, FALSE=forzato no
    CreatoIl             time.Time
    NomeSocio            string    // join
    CognomeSocio         string    // join
    DataNascita          time.Time // join — per calcolo giovani
    PrezzoEffettivo      float64   // calcolato
    EtichettaPrezzo      string    // "custom" | "giovani" | "giovani (forzato)" | "standard"
    GiovaniAuto          bool      // risultato calcolo età (per mostrare lo stato auto nella UI)
}

// models/millesimo.go
type Millesimo struct {
    ID           int
    MaestroID    int
    SocioID      int
    Voto         int
    AggiornatoIl time.Time
    NomeSocio    string  // join
    CognomeSocio string  // join
    NomeMaestro  string  // join
}
```

---

## 8. File coinvolti

### Nuovi
- `models/corso.go`
- `models/iscrizione_corso.go`
- `models/millesimo.go`
- `handlers/corsi.go`
- `handlers/millesimi.go`
- `templates/corsi.templ`
- `templates/millesimi.templ`

### Modificati
- `db/schema.sql` — nuove tabelle, drop `iscrizioni_lezione`, ristruttura `lezioni`, aggiorna CHECK utenti
- `db/db.go` — migrazione legacy `lezioni.titolo`
- `models/lezione.go` — ristrutturata
- `handlers/auth.go` — `RequireStaffOrAbove` middleware, aggiornare `maestro` come ruolo valido
- `main.go` — nuove route `/corsi/*`, `/millesimi`, redirect `/lezioni` → `/corsi`, rimuovere route legacy `/lezioni/*`
- `templates/layout.templ` — sidebar: "Corsi" al posto di "Lezioni", sezione "Insegnamento" per maestro, millesimi in admin

### Rimossi
- `handlers/lezioni.go` — sostituito da `handlers/corsi.go`
- `templates/lezioni.templ` — sostituito da `templates/corsi.templ`
