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
