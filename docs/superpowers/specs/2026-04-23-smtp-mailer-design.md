# Design: SMTP Mailer configurabile

**Data:** 2026-04-23
**Stato:** approvato

## Obiettivo

Configurare il progetto per inviare email tramite Brevo (SMTP relay esterno), con indirizzo mittente `no-reply@tb.gcoding.it` e soggetto configurabile per ogni tipo di email via variabili d'ambiente.

## Contesto

Il package `mailer` esiste già con supporto `net/smtp`. Attualmente invia solo il magic link di login con soggetto e mittente hardcoded. Non c'è nessun server SMTP configurato in docker-compose.

## Scelte di design

- **Provider SMTP:** Brevo (`smtp-relay.brevo.com:587`, STARTTLS). Free tier 300 email/giorno, supporto dominio custom, nessun obbligo di template.
- **Configurabilità:** ogni tipo di email ha un campo dedicato nel `Mailer` e la relativa env var. Il corpo del messaggio resta in codice Go (non ha senso metterlo in una env var).
- **Nessun server SMTP in Docker:** si usa Brevo come relay esterno; nessun container aggiuntivo nel docker-compose.

## Modifiche

### `mailer/mailer.go`

Aggiunge `LoginSubject string` al `Mailer`. `SendLoginLink` usa `m.LoginSubject` al posto del valore hardcoded. Pattern da replicare per ogni futuro tipo di email: aggiungere un campo `<Tipo>Subject string`.

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

### `main.go`

- `SMTP_FROM` default cambia da `adminEmail` a `no-reply@tb.gcoding.it`
- Aggiunge `MAIL_LOGIN_SUBJECT` con default `"TangoBar · Accesso"`
- `ADMIN_EMAIL` rimane invariato (solo per seed primo utente admin)

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

### `docker-compose.yml`

Sostituisce i placeholder SMTP vuoti con la configurazione Brevo:

```yaml
SMTP_HOST: smtp-relay.brevo.com
SMTP_PORT: ${SMTP_PORT:-587}
SMTP_USER: ${SMTP_USER:-}
SMTP_PASS: ${SMTP_PASS:-}
SMTP_FROM: ${SMTP_FROM:-no-reply@tb.gcoding.it}
MAIL_LOGIN_SUBJECT: ${MAIL_LOGIN_SUBJECT:-TangoBar · Accesso}
```

`SMTP_USER` e `SMTP_PASS` restano vuoti nel file committato — vanno in un `.env` locale non versionato.

## Prerequisiti operativi (fuori scope del codice)

1. Creare account Brevo e recuperare la chiave SMTP da *Settings → SMTP & API → SMTP*.
2. Aggiungere il dominio `tb.gcoding.it` in Brevo e verificarlo (record DNS TXT).
3. Popolare `SMTP_USER` e `SMTP_PASS` nel `.env` locale o nei secret di produzione.

## Estensibilità

Per aggiungere un nuovo tipo di email in futuro:
1. Aggiungere campo `<Tipo>Subject string` al `Mailer`
2. Aggiungere env var `MAIL_<TIPO>_SUBJECT` in `main.go` e `docker-compose.yml`
3. Implementare il metodo `Send<Tipo>` in `mailer.go`
