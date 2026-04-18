package models

import "time"

type IscrizioneCorso struct {
	ID                   int
	SocioID              int
	CorsoID              int
	PrezzoCustom         *float64
	ScontoGiovaniForzato *bool // nil=auto, true=forzato sì, false=forzato no
	CreatoIl             time.Time
	NomeSocio            string
	CognomeSocio         string
	DataNascita          time.Time
	PrezzoEffettivo      float64
	EtichettaPrezzo      string
	GiovaniAuto          bool // risultato calcolo età (per stato auto nella UI)
}

// CalcolaPrezzo determina il prezzo effettivo e l'etichetta per un socio iscritto a un corso.
func CalcolaPrezzo(corso Corso, iscr IscrizioneCorso, socio Socio) (float64, string) {
	if iscr.PrezzoCustom != nil {
		return *iscr.PrezzoCustom, "custom"
	}

	if iscr.ScontoGiovaniForzato != nil {
		if *iscr.ScontoGiovaniForzato && corso.PrezzoGiovani != nil {
			return *corso.PrezzoGiovani, "giovani (forzato)"
		}
		// false forzato: salta sconto, usa standard
		return corso.PrezzoLezione, "standard"
	}

	// auto: calcola da età
	if corso.EtaMaxGiovani != nil && corso.PrezzoGiovani != nil && !socio.DataNascita.IsZero() {
		if calcEta(socio.DataNascita) <= *corso.EtaMaxGiovani {
			return *corso.PrezzoGiovani, "giovani"
		}
	}

	return corso.PrezzoLezione, "standard"
}

func calcEta(dataNascita time.Time) int {
	now := time.Now()
	anni := now.Year() - dataNascita.Year()
	if now.YearDay() < dataNascita.YearDay() {
		anni--
	}
	return anni
}
