package handlers

import (
	"database/sql"

	"tango-gestionale/mailer"
)

// Handler struct contains shared dependencies injected by main.
type Handler struct {
	DB      *sql.DB
	Mailer  *mailer.Mailer
	BaseURL string // e.g. https://tb.gcoding.it, used to build magic-link URLs
}
