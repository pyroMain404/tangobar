# TangoBar — Gestionale

Gestionale per scuola di tango: soci, corsi, milonghe, bar, eventi.

Stack: Go 1.23 · chi v5 · templ · SQLite · Docker

---

## Sviluppo locale

### Prima installazione

```bash
go mod download
go install github.com/a-h/templ/cmd/templ@v0.2.778
```

### Avvio

Ogni volta che modifichi un file `.templ`, rigenera prima di avviare:

```bash
templ generate
go run .
```

Il server parte su `http://localhost:8080`.

I magic link di login vengono stampati su stdout quando SMTP non è configurato — copia il link direttamente dal terminale.

### Live-reload opzionale (air)

```bash
go install github.com/air-verse/air@latest
```

Crea `.air.toml` nella root:

```toml
[build]
  cmd = "templ generate && go build -o ./tmp/main ."
  bin = "./tmp/main"
  include_ext = ["go", "templ"]
  exclude_dir = ["tmp", "docs", "data"]
```

Poi avvia con:

```bash
air
```

Ad ogni salvataggio di `.go` o `.templ`, air rigenera e riavvia automaticamente.

### Test

```bash
go test ./...
```

---

## Deploy (Docker Compose)

### 1. Configura l'ambiente

```bash
cp .env.example .env
```

Modifica `.env` con i valori reali:

```env
SESSION_KEY=stringa-casuale-lunga-almeno-32-caratteri
APP_URL=https://tuo-dominio.it
ADMIN_EMAIL=tua@email.it

# SMTP — Brevo (o altro provider)
SMTP_USER=tuaemail@brevo.com
SMTP_PASS=chiave-smtp-dalla-dashboard
```

`ADMIN_EMAIL` è l'utente admin creato automaticamente al primo avvio.

### 2. Avvia

```bash
docker compose up -d --build
```

L'app gira su `127.0.0.1:8090` — da proxare con nginx o Caddy verso l'esterno.

### 3. Log

```bash
docker compose logs -f
```

Al primo avvio: `Admin user seeded: tua@email.it`.

### Aggiornare dopo modifiche

```bash
docker compose up -d --build
```

Il volume `./data` persiste il database SQLite tra i rebuild.

---

## Struttura

```
handlers/    handler HTTP (auth + moduli)
templates/   componenti templ (HTML)
models/      struct dati
mailer/      invio magic link via SMTP
db/          init SQLite + schema.sql
main.go      router, configurazione, bootstrap
data/        database e file generati (gitignored)
```

## Autenticazione

Login passwordless via magic link email. Gli utenti interni (`utenti`) sono creati manualmente dall'admin — non c'è registrazione pubblica. Al primo avvio viene creato un utente `admin` con l'email definita in `ADMIN_EMAIL`.

Ruoli: `admin` · `staff` · `maestro`
