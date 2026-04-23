# Strip to Auth Skeleton — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ridurre il progetto al solo scheletro di autenticazione magic-link, eliminando tutti i moduli applicativi, il database esistente e ogni stile visivo non posizionale.

**Architecture:** Branch `skeleton` da `master`. Si eliminano file, si riscrivono layout e template auth da zero (HTML puro + flex/grid CSS only), si resetta lo schema SQL a sole due tabelle. Il vecchio `master` rimane intatto come archivio storico.

**Tech Stack:** Go 1.23, chi v5, gorilla/sessions, templ v0.2.778, SQLite (modernc)

---

## File map

| Stato | Percorso | Note |
|---|---|---|
| Elimina | `handlers/admin.go` | |
| Elimina | `handlers/bar.go` | contiene `renderTempl` — vedi Task 5 |
| Elimina | `handlers/corsi.go` | |
| Elimina | `handlers/eventi.go` | |
| Elimina | `handlers/fatture.go` | |
| Elimina | `handlers/lezioni.go` | |
| Elimina | `handlers/soci.go` | contiene `renderTempl` — vedi Task 5 |
| Elimina | `handlers/tessere.go` | |
| Elimina | `templates/admin.templ` | |
| Elimina | `templates/bar.templ` | |
| Elimina | `templates/eventi.templ` | |
| Elimina | `templates/fatture.templ` | |
| Elimina | `templates/lezioni.templ` | |
| Elimina | `templates/soci.templ` | |
| Elimina | `templates/tessere.templ` | |
| Elimina | `models/bar.go` | |
| Elimina | `models/corso.go` | |
| Elimina | `models/evento.go` | |
| Elimina | `models/fattura.go` | |
| Elimina | `models/iscrizione_corso.go` | |
| Elimina | `models/iscrizione_corso_test.go` | |
| Elimina | `models/lezione.go` | |
| Elimina | `models/millesimo.go` | |
| Elimina | `models/socio.go` | |
| Elimina | `models/tessera.go` | |
| Elimina | `pdf/fattura.go` | |
| Elimina | `pdf/tessera.go` | |
| Elimina | `data/gestionale.db` | se esiste |
| Riscrivi | `db/schema.sql` | solo `utenti` + `login_tokens` |
| Riscrivi | `templates/layout.templ` | nuovo da zero, flex/grid only |
| Riscrivi | `templates/auth.templ` | HTML puro, zero stili |
| Modifica | `handlers/auth.go` | strip Dashboard (stats + renderTempl) |
| Modifica | `main.go` | strip route e pdfPath |
| Intatto | `handlers/handler.go` | |
| Intatto | `templates/context.go` | |
| Intatto | `mailer/mailer.go` | |
| Intatto | `mailer/mailer_test.go` | |
| Intatto | `models/utente.go` | |
| Intatto | `db/db.go` | |
| Intatto | `docker-compose.yml` | |
| Intatto | `Dockerfile` | |
| Intatto | `.env.example` | |

---

## Task 1: Crea branch skeleton

**Files:** nessuno

- [ ] **Step 1: Crea e checkout del branch skeleton**

```bash
git checkout -b skeleton
```

Expected output: `Switched to a new branch 'skeleton'`

- [ ] **Step 2: Verifica**

```bash
git branch
```

Expected: `* skeleton` evidenziato.

---

## Task 2: Elimina i file non-auth

**Files:** tutti i file nella colonna "Elimina" del file map.

- [ ] **Step 1: Elimina handlers**

```bash
rm handlers/admin.go handlers/bar.go handlers/corsi.go handlers/eventi.go \
   handlers/fatture.go handlers/lezioni.go handlers/soci.go handlers/tessere.go
```

- [ ] **Step 2: Elimina templates**

```bash
rm templates/admin.templ templates/bar.templ templates/eventi.templ \
   templates/fatture.templ templates/lezioni.templ templates/soci.templ \
   templates/tessere.templ
```

- [ ] **Step 3: Elimina models**

```bash
rm models/bar.go models/corso.go models/evento.go models/fattura.go \
   models/iscrizione_corso.go models/iscrizione_corso_test.go \
   models/lezione.go models/millesimo.go models/socio.go models/tessera.go
```

- [ ] **Step 4: Elimina package pdf**

```bash
rm pdf/fattura.go pdf/tessera.go && rmdir pdf
```

- [ ] **Step 5: Elimina DB se esiste**

```bash
rm -f data/gestionale.db
```

---

## Task 3: Reset schema SQL

**Files:**
- Modifica: `db/schema.sql`

- [ ] **Step 1: Sovrascrivi schema.sql**

Sostituisci l'intero contenuto del file con:

```sql
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
```

---

## Task 4: Nuovo layout.templ

**Files:**
- Riscrivi: `templates/layout.templ`

Il `layout.templ` attuale è 1143 righe con Tailwind CDN, Google Fonts, HTMX, sidebar, custom Art Deco CSS. Va tutto via.

- [ ] **Step 1: Sovrascrivi layout.templ**

Sostituisci l'intero contenuto del file con:

```templ
package templates

// Page wraps content nel layout principale.
templ Page(title string, content templ.Component) {
	@Layout(title) {
		@content
	}
}

templ Layout(title string) {
	<!DOCTYPE html>
	<html lang="it">
		<head>
			<meta charset="UTF-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
			<title>{ title } · TangoBar</title>
			<style>
				*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
				body { min-height: 100vh; display: flex; flex-direction: column; }
				main { flex: 1; }
			</style>
		</head>
		<body>
			<main>
				{ children... }
			</main>
		</body>
	</html>
}
```

**Nota:** la firma di `Page` cambia da `Page(title, activePage string, content)` a `Page(title string, content)`. Il parametro `activePage` viene rimosso perché non c'è più navigazione — verrà reintrodotto quando servirà.

---

## Task 5: Nuovo auth.templ

**Files:**
- Riscrivi: `templates/auth.templ`

Il file attuale importa `"fmt"` e usa classi CSS Art Deco. Va tutto via.

- [ ] **Step 1: Sovrascrivi auth.templ**

Sostituisci l'intero contenuto del file con:

```templ
package templates

templ AuthLayout(title string) {
	<!DOCTYPE html>
	<html lang="it">
		<head>
			<meta charset="UTF-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
			<title>{ title } · TangoBar</title>
			<style>
				*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
				body { min-height: 100vh; display: flex; align-items: center; justify-content: center; }
			</style>
		</head>
		<body>
			{ children... }
		</body>
	</html>
}

templ LoginPage(errMsg string) {
	@AuthLayout("Accesso") {
		<div>
			<h1>TangoBar</h1>
			<p>Inserisci la tua email. Ti invieremo un link di accesso.</p>
			if errMsg != "" {
				<p role="alert">{ errMsg }</p>
			}
			<form method="POST" action="/login">
				<label for="email">Email</label>
				<input
					type="email"
					id="email"
					name="email"
					required
					autofocus
					autocomplete="email"
					inputmode="email"
				/>
				<button type="submit">Invia link</button>
			</form>
		</div>
	}
}

templ LoginLinkSent(email string) {
	@AuthLayout("Controlla la posta") {
		<div>
			<h1>Link inviato</h1>
			<p>
				Se l'indirizzo <strong>{ email }</strong> è autorizzato,
				riceverai a breve un link per accedere. Valido 15 minuti.
			</p>
			<a href="/login">Torna indietro</a>
		</div>
	}
}

templ DashboardPage() {
	<p>Accesso effettuato.</p>
	<form method="POST" action="/logout">
		<button type="submit">Esci</button>
	</form>
}
```

**Note:**
- `AuthLayout` si separa da `Layout` perché la pagina di login non usa il layout principale (ha la propria struttura centrata).
- `DashboardPage` ora non prende argomenti (niente stats da tabelle che non esistono più).
- Rimossa l'importazione di `"fmt"` (non più necessaria).

---

## Task 6: Aggiorna handlers/auth.go

**Files:**
- Modifica: `handlers/auth.go`

Due problemi da risolvere:
1. `Dashboard` interroga tabelle eliminate (`soci`, `tessere`, `lezioni`, `bar_items`, `ingressi_milonga`, `bar_movimenti`) e chiama `renderTempl` (definita in `soci.go` che stiamo eliminando).
2. `renderTempl` non esiste più da nessuna parte.

La soluzione: riscrivere `Dashboard` come stub che chiama `.Render` direttamente (come fanno già tutti gli altri handler auth), e smettere di usare `renderTempl` qui.

- [ ] **Step 1: Sostituisci la funzione Dashboard in handlers/auth.go**

Trova il blocco che inizia con `// Dashboard handles GET /` (riga ~257) e sostituisci l'intera funzione con:

```go
// Dashboard handles GET /
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.Page("Dashboard", templates.DashboardPage()).Render(r.Context(), w)
}
```

**Cosa cambia:** rimosso il `session.Get` ridondante (RequireAuth lo ha già verificato), rimossi tutti i `QueryRow` su tabelle non più esistenti, rimossa la mappa `stats`, cambiata la chiamata da `renderTempl(...)` a `.Render(r.Context(), w)`, firma di `templates.Page` aggiornata (niente più `activePage`).

---

## Task 7: Aggiorna main.go

**Files:**
- Modifica: `main.go`

- [ ] **Step 1: Rimuovi pdfPath da main.go**

Trova e rimuovi le righe:
```go
pdfPath := env("PDF_PATH", "./data/pdf")
```
e:
```go
if err := os.MkdirAll(pdfPath, 0755); err != nil {
    log.Fatalf("Failed to create pdf directory: %v", err)
}
```

- [ ] **Step 2: Sostituisci il blocco Router con le sole route auth**

Trova il blocco che inizia con `// ----- Router -----` e sostituisci tutto il gruppo di route con:

```go
// ----- Router -----
r := chi.NewRouter()
r.Use(middleware.Logger)
r.Use(middleware.Recoverer)
r.Use(middleware.Compress(5))

// Public
r.Get("/login", h.LoginPage)
r.Post("/login", h.Login)
r.Get("/login/verify", h.VerifyLogin)

// Protected
r.Group(func(r chi.Router) {
    r.Use(h.RequireAuth)
    r.Use(h.WithUserContext)

    r.Get("/", h.Dashboard)
    r.Post("/logout", h.Logout)
    r.Get("/logout", h.Logout)
})
```

---

## Task 8: go mod tidy + templ generate + build

**Files:** `go.mod`, `go.sum` (aggiornati automaticamente)

- [ ] **Step 1: Rimuovi dipendenze orfane (maroto, gopdf, ecc.)**

```bash
go mod tidy
```

Expected: le dipendenze `github.com/johnfercher/maroto`, `github.com/signintech/gopdf` e correlate vengono rimosse da `go.mod` e `go.sum`. Nessun errore.

- [ ] **Step 2: Rigenera i file templ**

```bash
templ generate
```

Expected: output del tipo `(✓) Complete [ Xms] ...` per ogni `.templ` modificato. Nessun errore.

- [ ] **Step 3: Build**

```bash
go build ./...
```

Expected: nessun output (build OK). Se ci sono errori di import o tipo, correggerli prima di procedere.

- [ ] **Step 4: Esegui i test rimasti**

```bash
go test ./...
```

Expected: solo `mailer/mailer_test.go` gira. Output del tipo:
```
ok  	tango-gestionale/mailer	0.XXXs
```

---

## Task 9: Commit skeleton

- [ ] **Step 1: Verifica stato git**

```bash
git status
```

Expected: file eliminati e modificati, nessun untracked inatteso.

- [ ] **Step 2: Stage e commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
chore: strip to auth skeleton

Rimossi tutti i moduli applicativi (soci, tessere, corsi, lezioni,
presenze, millesimi, eventi, milonga, fatture, bar, admin, pdf).
Schema SQL ridotto a utenti + login_tokens.
Layout e template auth riscritti da zero: HTML puro, CSS flex/grid only.
DB dati eliminato. Dipendenze PDF rimosse via go mod tidy.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 3: Verifica commit**

```bash
git log --oneline -3
```

Expected: il commit `chore: strip to auth skeleton` in cima.

---

## Self-review

**Spec coverage:**
- [x] Branch `skeleton` da `master` → Task 1
- [x] Tutti i file non-auth eliminati → Task 2
- [x] Schema SQL reset a utenti + login_tokens → Task 3
- [x] layout.templ nuovo da zero con flex/grid only → Task 4
- [x] auth.templ riscritto HTML puro → Task 5
- [x] Dashboard stub senza query su tabelle eliminate → Task 6
- [x] main.go: pdfPath rimosso, route ridotte → Task 7
- [x] DB data file eliminato → Task 2 step 5
- [x] go mod tidy rimuove dipendenze PDF → Task 8
- [x] Build + test verifica → Task 8
- [x] Commit → Task 9

**Placeholder scan:** nessun TBD, nessun "handle edge cases", nessun "similar to Task N".

**Type consistency:**
- `templates.Page(title string, content templ.Component)` — definita in Task 4, usata in Task 6. ✓
- `templates.DashboardPage()` senza argomenti — definita in Task 5, usata in Task 6. ✓
- `templates.AuthLayout(title string)` — definita e usata solo in Task 5. ✓
