# SMTP Mailer configurabile — Piano d'implementazione

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rendere il `Mailer` configurabile per tipo di email via env var, impostare Brevo come relay SMTP e `no-reply@tb.gcoding.it` come mittente di default.

**Architecture:** Aggiunta del campo `LoginSubject string` al `Mailer`, popolato da `MAIL_LOGIN_SUBJECT` in `main.go`. Il docker-compose viene aggiornato con i parametri Brevo. Nessun nuovo package, nessuna dipendenza esterna.

**Tech Stack:** Go 1.23, `net/smtp` (stdlib), Brevo SMTP relay (`smtp-relay.brevo.com:587`)

---

## File coinvolti

| File | Operazione |
|---|---|
| `mailer/mailer.go` | Modifica — aggiunge `LoginSubject`, lo usa in `SendLoginLink` |
| `mailer/mailer_test.go` | Creazione — unit test per `Configured`, `buildMIME`, `buildLoginBody` |
| `main.go` | Modifica — aggiunge `MAIL_LOGIN_SUBJECT`, cambia default `SMTP_FROM` |
| `docker-compose.yml` | Modifica — configurazione Brevo |

---

## Task 1: Test unitari per il mailer

**Files:**
- Create: `mailer/mailer_test.go`

- [ ] **Step 1: Scrivere i test (falliranno il test sul soggetto custom)**

Crea `mailer/mailer_test.go`:

```go
package mailer

import (
	"strings"
	"testing"
)

func TestConfigured(t *testing.T) {
	if (&Mailer{}).Configured() {
		t.Fatal("mailer senza host non deve essere configurato")
	}
	m := &Mailer{Host: "smtp.example.com", From: "noreply@example.com"}
	if !m.Configured() {
		t.Fatal("mailer con host e from deve essere configurato")
	}
}

func TestBuildMIMESubject(t *testing.T) {
	mime := buildMIME("from@example.com", "to@example.com", "Soggetto Custom", "corpo email")
	if !strings.Contains(mime, "Subject: Soggetto Custom\r\n") {
		t.Fatalf("MIME non contiene il soggetto atteso:\n%s", mime)
	}
}

func TestBuildLoginBodyContainsLink(t *testing.T) {
	link := "https://tb.gcoding.it/login/verify?token=abc123"
	body := buildLoginBody(link)
	if !strings.Contains(body, link) {
		t.Fatalf("body non contiene il link:\n%s", body)
	}
}
```

- [ ] **Step 2: Eseguire i test e verificare che passino**

```bash
go test ./mailer/... -v
```

Output atteso: tutti e 3 i test PASS (le funzioni `buildMIME` e `buildLoginBody` esistono già, `Configured` anche).

Se qualche test fallisce, c'è un bug nei test stessi — rileggi il codice di `mailer.go` prima di procedere.

- [ ] **Step 3: Commit**

```bash
git add mailer/mailer_test.go
git commit -m "test: unit test per Mailer, buildMIME, buildLoginBody"
```

---

## Task 2: Aggiungere LoginSubject al Mailer

**Files:**
- Modify: `mailer/mailer.go`

- [ ] **Step 1: Aggiungere il campo e usarlo in SendLoginLink**

In `mailer/mailer.go`, modifica la struct `Mailer` aggiungendo il campo `LoginSubject`:

```go
type Mailer struct {
	Host         string
	Port         string
	User         string
	Pass         string
	From         string
	LoginSubject string
}
```

Poi modifica `SendLoginLink` per usare `m.LoginSubject` invece del valore hardcoded:

```go
func (m *Mailer) SendLoginLink(to, link string) error {
	subject := m.LoginSubject
	body := buildLoginBody(link)

	if !m.Configured() {
		logFallback(to, subject, body, link)
		return nil
	}

	msg := buildMIME(m.From, to, subject, body)
	addr := m.Host + ":" + m.Port
	var auth smtp.Auth
	if m.User != "" {
		auth = smtp.PlainAuth("", m.User, m.Pass, m.Host)
	}
	if err := smtp.SendMail(addr, auth, m.From, []string{to}, []byte(msg)); err != nil {
		logFallback(to, subject, body, link)
		return fmt.Errorf("smtp send: %w", err)
	}
	log.Printf("[mailer] login link sent to %s via %s", to, addr)
	return nil
}
```

- [ ] **Step 2: Eseguire i test**

```bash
go test ./mailer/... -v
```

Output atteso: tutti e 3 i test PASS.

- [ ] **Step 3: Verificare che il progetto compili**

```bash
go build ./...
```

Output atteso: nessun errore.

- [ ] **Step 4: Commit**

```bash
git add mailer/mailer.go
git commit -m "feat: LoginSubject configurabile nel Mailer"
```

---

## Task 3: Aggiornare main.go

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Aggiungere MAIL_LOGIN_SUBJECT e cambiare il default di SMTP_FROM**

In `main.go`, trova il blocco di inizializzazione del `Mailer` (righe 26-32 circa) e sostituiscilo con:

```go
m := &mailer.Mailer{
	Host:         os.Getenv("SMTP_HOST"),
	Port:         env("SMTP_PORT", "587"),
	User:         os.Getenv("SMTP_USER"),
	Pass:         os.Getenv("SMTP_PASS"),
	From:         env("SMTP_FROM", "no-reply@tb.gcoding.it"),
	LoginSubject: env("MAIL_LOGIN_SUBJECT", "TangoBar · Accesso"),
}
```

Nota: `ADMIN_EMAIL` rimane invariato — serve solo per il seed del primo utente admin, non come mittente email.

- [ ] **Step 2: Verificare che il progetto compili**

```bash
go build ./...
```

Output atteso: nessun errore.

- [ ] **Step 3: Commit**

```bash
git add main.go
git commit -m "feat: SMTP_FROM default no-reply@tb.gcoding.it, MAIL_LOGIN_SUBJECT configurabile"
```

---

## Task 4: Aggiornare docker-compose.yml per Brevo

**Files:**
- Modify: `docker-compose.yml`

- [ ] **Step 1: Sostituire la sezione SMTP con la configurazione Brevo**

In `docker-compose.yml`, trova il blocco SMTP nel service `gestionale` e sostituiscilo con:

```yaml
      # SMTP — Brevo (smtp-relay.brevo.com)
      # Credenziali: Brevo → Settings → SMTP & API → SMTP
      # SMTP_USER = email account Brevo
      # SMTP_PASS = chiave SMTP generata dalla dashboard
      SMTP_HOST: smtp-relay.brevo.com
      SMTP_PORT: ${SMTP_PORT:-587}
      SMTP_USER: ${SMTP_USER:-}
      SMTP_PASS: ${SMTP_PASS:-}
      SMTP_FROM: ${SMTP_FROM:-no-reply@tb.gcoding.it}
      MAIL_LOGIN_SUBJECT: ${MAIL_LOGIN_SUBJECT:-TangoBar · Accesso}
```

Le righe da sostituire sono quelle con `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM` già presenti.

- [ ] **Step 2: Verificare che il file YAML sia valido**

```bash
docker compose config --quiet
```

Output atteso: nessun errore.

- [ ] **Step 3: Commit**

```bash
git add docker-compose.yml
git commit -m "feat: configurazione Brevo SMTP in docker-compose"
```

---

## Note operative post-deploy

Questi passi non fanno parte del codice ma sono necessari prima che le email vengano inviate in produzione:

1. **Account Brevo:** creare account su brevo.com, andare in *Settings → SMTP & API → SMTP*, generare una chiave SMTP.
2. **Verifica dominio:** in Brevo aggiungere `tb.gcoding.it` come sender domain e completare la verifica DNS (record TXT + DKIM).
3. **Env var di produzione:** popolare `SMTP_USER` e `SMTP_PASS` nel file `.env` locale (non committare mai questo file).

Senza `SMTP_HOST` impostato, il sistema continua a funzionare normalmente — i link di login vengono stampati su stdout (comportamento di fallback già esistente).
