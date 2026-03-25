package models

import "time"

type Fattura struct {
	ID            int64
	Numero        string
	SocioID       *int64
	NomeCliente   string
	DataEmissione time.Time
	Totale        float64
	Pagata        bool
	PDFPath       string
	Righe         []RigaFattura
}

type RigaFattura struct {
	ID          int64
	FatturaID   int64
	Descrizione string
	Quantita    float64
	PrezzoUnit  float64
	Totale      float64
}
