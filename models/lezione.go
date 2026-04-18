package models

import "time"

type Lezione struct {
	ID        int
	CorsoID   int
	Data      time.Time
	Ora       string
	DurataMin int
	MaxPosti  int
	Prezzo    float64
	Stato     string // "programmata" | "completata" | "annullata"
	Nota      string
	Presenti  int // count join
}

type Presenza struct {
	ID            int
	SocioID       int
	LezioneID     int
	SegnataDa     int
	Timestamp     time.Time
	NomeSocio     string
	CognomeSocio  string
	NomeOperatore string
}
