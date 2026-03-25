package models

import "time"

type Lezione struct {
	ID         int64
	Titolo     string
	Insegnante string
	DataOra    time.Time
	DurataMin  int
	MaxPosti   *int
	Prezzo     float64
	Iscritti   int
}

type Iscrizione struct {
	ID           int64
	SocioID      int64
	LezioneID    int64
	Pagato       bool
	NomeSocio    string
	CognomeSocio string
}
