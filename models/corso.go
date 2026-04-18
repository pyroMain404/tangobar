package models

import "time"

type Corso struct {
	ID              int
	Titolo          string
	GiornoSettimana int // 0=lunedì
	Ora             string
	DurataMin       int
	MaxPosti        int
	PrezzoLezione   float64
	MaestroID       *int
	MaestroNome     string // join
	DataInizio      time.Time
	DataFine        time.Time
	EtaMaxGiovani   *int
	PrezzoGiovani   *float64
	Attivo          bool
	CreatoIl        time.Time
	Iscritti        int // count join
	TotaleLezioni   int // count join
}

var GiorniSettimana = []string{"Lunedì", "Martedì", "Mercoledì", "Giovedì", "Venerdì", "Sabato", "Domenica"}

func (c Corso) GiornoLabel() string {
	if c.GiornoSettimana >= 0 && c.GiornoSettimana < 7 {
		return GiorniSettimana[c.GiornoSettimana]
	}
	return ""
}
