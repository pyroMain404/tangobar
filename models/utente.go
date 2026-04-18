package models

import "time"

type Utente struct {
	ID       int
	Email    string
	Nome     string
	Ruolo    string // "admin" | "staff" | "maestro"
	CreatoIl time.Time
}
