package models

import "time"

type Socio struct {
	ID            int64
	Nome          string
	Cognome       string
	Email         string
	Telefono      string
	DataNascita   time.Time
	Note          string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	TesseraAttiva bool
}

type SocioIngresso struct {
	Socio
	Entrato bool
}
