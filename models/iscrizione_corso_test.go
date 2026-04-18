package models

import (
	"testing"
	"time"
)

func TestCalcola_Custom(t *testing.T) {
	custom := 50.0
	iscr := IscrizioneCorso{PrezzoCustom: &custom}
	corso := Corso{PrezzoLezione: 20}
	socio := Socio{}
	prezzo, etichetta := CalcolaPrezzo(corso, iscr, socio)
	if prezzo != 50 || etichetta != "custom" {
		t.Errorf("got %.0f %s, want 50 custom", prezzo, etichetta)
	}
}

func TestCalcola_GiovaniForzatoTrue(t *testing.T) {
	forzato := true
	prezzoGiovani := 10.0
	etaMax := 25
	iscr := IscrizioneCorso{ScontoGiovaniForzato: &forzato}
	corso := Corso{PrezzoLezione: 20, EtaMaxGiovani: &etaMax, PrezzoGiovani: &prezzoGiovani}
	socio := Socio{DataNascita: time.Now().AddDate(-30, 0, 0)}
	prezzo, etichetta := CalcolaPrezzo(corso, iscr, socio)
	if prezzo != 10 || etichetta != "giovani (forzato)" {
		t.Errorf("got %.0f %s, want 10 giovani (forzato)", prezzo, etichetta)
	}
}

func TestCalcola_GiovaniForzatoFalse(t *testing.T) {
	forzato := false
	prezzoGiovani := 10.0
	etaMax := 25
	iscr := IscrizioneCorso{ScontoGiovaniForzato: &forzato}
	corso := Corso{PrezzoLezione: 20, EtaMaxGiovani: &etaMax, PrezzoGiovani: &prezzoGiovani}
	socio := Socio{DataNascita: time.Now().AddDate(-20, 0, 0)}
	prezzo, etichetta := CalcolaPrezzo(corso, iscr, socio)
	if prezzo != 20 || etichetta != "standard" {
		t.Errorf("got %.0f %s, want 20 standard", prezzo, etichetta)
	}
}

func TestCalcola_GiovaniAuto_SottoSoglia(t *testing.T) {
	prezzoGiovani := 10.0
	etaMax := 25
	iscr := IscrizioneCorso{}
	corso := Corso{PrezzoLezione: 20, EtaMaxGiovani: &etaMax, PrezzoGiovani: &prezzoGiovani}
	socio := Socio{DataNascita: time.Now().AddDate(-20, 0, 0)}
	prezzo, etichetta := CalcolaPrezzo(corso, iscr, socio)
	if prezzo != 10 || etichetta != "giovani" {
		t.Errorf("got %.0f %s, want 10 giovani", prezzo, etichetta)
	}
}

func TestCalcola_GiovaniAuto_SopraSoglia(t *testing.T) {
	prezzoGiovani := 10.0
	etaMax := 25
	iscr := IscrizioneCorso{}
	corso := Corso{PrezzoLezione: 20, EtaMaxGiovani: &etaMax, PrezzoGiovani: &prezzoGiovani}
	socio := Socio{DataNascita: time.Now().AddDate(-30, 0, 0)}
	prezzo, etichetta := CalcolaPrezzo(corso, iscr, socio)
	if prezzo != 20 || etichetta != "standard" {
		t.Errorf("got %.0f %s, want 20 standard", prezzo, etichetta)
	}
}

func TestCalcola_Standard(t *testing.T) {
	iscr := IscrizioneCorso{}
	corso := Corso{PrezzoLezione: 20}
	socio := Socio{}
	prezzo, etichetta := CalcolaPrezzo(corso, iscr, socio)
	if prezzo != 20 || etichetta != "standard" {
		t.Errorf("got %.0f %s, want 20 standard", prezzo, etichetta)
	}
}
