# Strip to Auth Skeleton

**Data:** 2026-04-23
**Branch:** `skeleton` (da `master`)
**Primo modulo post-skeleton:** soci + corsi

## Obiettivo

Portare il progetto a uno stato minimo funzionante: solo autenticazione magic-link, niente altro. Tutto il codice applicativo viene eliminato. Il database riparte vuoto. Il layout visivo riparte da zero. Si ricostruisce modulo per modulo, con decisioni deliberate su ogni scelta.

## Branch strategy

Si crea il branch `skeleton` da `master`. Il vecchio `master` rimane intatto come riferimento storico (handlers, templates, modelli precedenti). Tutto il lavoro di rebuild avviene su `skeleton`.

## File sopravvissuti (modificati o intatti)

| File | Stato |
|---|---|
| `main.go` | Modificato — solo route auth + dashboard stub, rimosso `pdfPath` |
| `handlers/auth.go` | Intatto |
| `handlers/handler.go` | Intatto |
| `templates/auth.templ` | Riscritto — HTML funzionale puro, zero stili |
| `templates/layout.templ` | Riscritto da zero — html/head/body/main, style minimale (flex/grid only) |
| `templates/context.go` | Intatto |
| `mailer/mailer.go` | Intatto |
| `mailer/mailer_test.go` | Intatto |
| `models/utente.go` | Intatto |
| `db/db.go` | Intatto |
| `db/schema.sql` | Riscritto — solo `utenti` + `login_tokens` |
| `docker-compose.yml` | Intatto |
| `Dockerfile` | Intatto |
| `.env.example` | Intatto |
| `go.mod` / `go.sum` | Intatti |

## File eliminati

**handlers/:** `admin.go`, `bar.go`, `corsi.go`, `eventi.go`, `fatture.go`, `lezioni.go`, `soci.go`, `tessere.go`

**templates/:** `admin.templ`, `bar.templ`, `eventi.templ`, `fatture.templ`, `lezioni.templ`, `soci.templ`, `tessere.templ`

**models/:** `bar.go`, `corso.go`, `evento.go`, `fattura.go`, `iscrizione_corso.go`, `iscrizione_corso_test.go`, `lezione.go`, `millesimo.go`, `socio.go`, `tessera.go`

**pdf/:** intero package (`fattura.go`, `tessera.go`)

**data/gestionale.db:** eliminato (riparte da zero al boot successivo)

## Database

Schema ridotto a due tabelle:

```sql
utenti (id, email, nome, ruolo, creato_il)
login_tokens (token, user_id, expires_at, used_at, created_at)
```

Nessuna migrazione separata — `db.InitDB` applica `schema.sql` all'avvio. Il file SQLite viene eliminato; il seed dell'admin viene ricreato automaticamente dal boot.

## Route post-strip

```
GET  /login          → LoginPage
POST /login          → Login
GET  /login/verify   → VerifyLogin

[RequireAuth]
GET  /               → Dashboard (stub minimale)
POST /logout         → Logout
GET  /logout         → Logout
```

## Layout e CSS

`layout.templ` nuovo da zero:
- `<html>`, `<head>` (charset, viewport, `<style>` inline), `<body>`, `<main>`
- CSS ammesso: `display: flex`, `display: grid`, e proprietà di posizionamento (gap, align-items, justify-content, grid-template-*)
- Zero colori applicati, zero tipografia, zero bordi decorativi — tutto browser default
- Ogni scelta stilistica verrà presa modulo per modulo durante il rebuild

`auth.templ` riscritto: HTML semantico puro (form, label, input, button), niente classi, niente stili inline.

## Prossimi passi

Dopo lo skeleton approvato, il primo modulo da costruire è **soci + corsi** (gestione anagrafica soci e corsi a cui partecipano).
