package models

import "time"

type Utente struct {
	ID       int
	Email    string
	Nome     string
	Ruolo    string // "admin" | "staff"
	CreatoIl time.Time
}
