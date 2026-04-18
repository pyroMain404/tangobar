package models

import "time"

type Millesimo struct {
	ID           int
	MaestroID    int
	SocioID      int
	Voto         int
	AggiornatoIl time.Time
	NomeSocio    string
	CognomeSocio string
	NomeMaestro  string
}
