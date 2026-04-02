package models

import "time"

type Tessera struct {
	ID           int64
	SocioID      int64
	Tipo         string // "base" | "premium" | "annuale"
	EmessaIl     time.Time
	ValidaFino   time.Time
	Pagato       bool
	Importo      float64
	NomeSocio    string
	CognomeSocio string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
