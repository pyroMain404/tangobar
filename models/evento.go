package models

import "time"

type Evento struct {
	ID          int64
	Titolo      string
	Descrizione string
	DataOra     time.Time
	Location    string
	PrezzoBase  float64
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
