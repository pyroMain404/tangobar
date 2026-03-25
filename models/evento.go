package models

import "time"

type Evento struct {
	ID         int64
	Nome       string
	DataOra    time.Time
	Luogo      string
	PrezzoBase float64
	Note       string
}

type IngressoMilonga struct {
	ID           int64
	SocioID      *int64
	NomeOspite   string
	EventoID     *int64
	Timestamp    time.Time
	Importo      float64
	NomeSocio    string
	CognomeSocio string
}
