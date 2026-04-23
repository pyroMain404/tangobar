package mailer

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

// Mailer sends transactional emails. When SMTP is not configured, it falls
// back to logging a loud banner with the message to stdout so development
// and bootstrap scenarios still work end-to-end.
type Mailer struct {
	Host         string
	Port         string
	User         string
	Pass         string
	From         string
	LoginSubject string
}

// Configured reports whether SMTP credentials are available.
func (m *Mailer) Configured() bool {
	return m.Host != "" && m.From != ""
}

// SendLoginLink sends the magic-link email to the given address.
func (m *Mailer) SendLoginLink(to, link string) error {
	subject := m.LoginSubject
	if subject == "" {
		subject = "TangoBar · Accesso"
	}
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
		// Still surface the link in logs so an admin can recover.
		logFallback(to, subject, body, link)
		return fmt.Errorf("smtp send: %w", err)
	}
	log.Printf("[mailer] login link sent to %s via %s", to, addr)
	return nil
}

func buildLoginBody(link string) string {
	return "Benvenuto in TangoBar.\r\n\r\n" +
		"Per accedere al gestionale, clicca il link qui sotto entro 15 minuti:\r\n\r\n" +
		link + "\r\n\r\n" +
		"Se non hai richiesto l'accesso, ignora questa email.\r\n" +
		"\r\n— TangoBar"
}

func buildMIME(from, to, subject, body string) string {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return b.String()
}

func logFallback(to, subject, body, link string) {
	const bar = "============================================================"
	log.Printf("\n%s\n[mailer] SMTP non configurato — email NON inviata.\nA:      %s\nOggetto: %s\nLink:    %s\n%s\n%s", bar, to, subject, link, body, bar)
}
