package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"tango-gestionale/models"
	"tango-gestionale/templates"
)

// generaDateLezioni calcola tutte le date tra dataInizio e dataFine che cadono su giornoSettimana.
// Spec: 0=lunedì. Go time.Weekday: Sunday=0, Monday=1.
func generaDateLezioni(dataInizio, dataFine time.Time, giornoSettimana int) []time.Time {
	targetWeekday := time.Weekday((giornoSettimana + 1) % 7)
	var dates []time.Time
	current := dataInizio
	for current.Weekday() != targetWeekday {
		current = current.AddDate(0, 0, 1)
	}
	for !current.After(dataFine) {
		dates = append(dates, current)
		current = current.AddDate(0, 0, 7)
	}
	return dates
}

func (h *Handler) ListaCorsi(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := h.DB.QueryContext(ctx, `
		SELECT c.id, c.titolo, c.giorno_settimana, c.ora, c.durata_min, c.max_posti,
		       c.prezzo_lezione, c.maestro_id, COALESCE(u.nome,'') as maestro_nome,
		       c.data_inizio, c.data_fine, c.eta_max_giovani, c.prezzo_giovani,
		       c.attivo, c.creato_il,
		       COUNT(DISTINCT ic.id) as iscritti,
		       COUNT(DISTINCT l.id) as totale_lezioni
		FROM corsi c
		LEFT JOIN utenti u ON u.id = c.maestro_id
		LEFT JOIN iscrizioni_corso ic ON ic.corso_id = c.id
		LEFT JOIN lezioni l ON l.corso_id = c.id
		GROUP BY c.id
		ORDER BY c.data_inizio DESC
	`)
	if err != nil {
		http.Error(w, "Errore DB", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var corsi []models.Corso
	for rows.Next() {
		var c models.Corso
		var maestroID sql.NullInt64
		var etaMax sql.NullInt64
		var prezzoGiovani sql.NullFloat64
		if err := rows.Scan(
			&c.ID, &c.Titolo, &c.GiornoSettimana, &c.Ora, &c.DurataMin, &c.MaxPosti,
			&c.PrezzoLezione, &maestroID, &c.MaestroNome,
			&c.DataInizio, &c.DataFine, &etaMax, &prezzoGiovani,
			&c.Attivo, &c.CreatoIl, &c.Iscritti, &c.TotaleLezioni,
		); err != nil {
			log.Printf("[corsi] scan: %v", err)
			continue
		}
		if maestroID.Valid {
			id := int(maestroID.Int64)
			c.MaestroID = &id
		}
		if etaMax.Valid {
			v := int(etaMax.Int64)
			c.EtaMaxGiovani = &v
		}
		if prezzoGiovani.Valid {
			c.PrezzoGiovani = &prezzoGiovani.Float64
		}
		corsi = append(corsi, c)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderTempl(w, r, templates.Page("Corsi", "corsi", templates.CorsiPage(corsi)))
}

func (h *Handler) NuovoCorsoForm(w http.ResponseWriter, r *http.Request) {
	maestri := h.fetchMaestri(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderTempl(w, r, templates.Page("Nuovo Corso", "corsi", templates.NuovoCorsoForm(maestri, 0)))
}

func (h *Handler) CreaCorso(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form non valido", http.StatusBadRequest)
		return
	}
	ctx := r.Context()

	titolo := r.FormValue("titolo")
	giornoStr := r.FormValue("giorno_settimana")
	ora := r.FormValue("ora")
	durataStr := r.FormValue("durata_min")
	postiStr := r.FormValue("max_posti")
	prezzoStr := r.FormValue("prezzo_lezione")
	inizioStr := r.FormValue("data_inizio")
	fineStr := r.FormValue("data_fine")
	etaMaxStr := r.FormValue("eta_max_giovani")
	prezzoGiovaniStr := r.FormValue("prezzo_giovani")
	maestroIDStr := r.FormValue("maestro_id")

	giorno, _ := strconv.Atoi(giornoStr)
	durata, _ := strconv.Atoi(durataStr)
	if durata == 0 {
		durata = 60
	}
	posti, _ := strconv.Atoi(postiStr)
	prezzo, _ := strconv.ParseFloat(prezzoStr, 64)
	dataInizio, _ := time.Parse("2006-01-02", inizioStr)
	dataFine, _ := time.Parse("2006-01-02", fineStr)

	var etaMaxGiovani *int
	var prezzoGiovani *float64
	if etaMaxStr != "" {
		v, err := strconv.Atoi(etaMaxStr)
		if err == nil {
			etaMaxGiovani = &v
		}
	}
	if prezzoGiovaniStr != "" && etaMaxGiovani != nil {
		v, err := strconv.ParseFloat(prezzoGiovaniStr, 64)
		if err == nil {
			prezzoGiovani = &v
		}
	}

	var maestroID *int
	if maestroIDStr != "" && maestroIDStr != "0" {
		v, err := strconv.Atoi(maestroIDStr)
		if err == nil {
			maestroID = &v
		}
	}

	res, err := h.DB.ExecContext(ctx, `
		INSERT INTO corsi (titolo, giorno_settimana, ora, durata_min, max_posti, prezzo_lezione,
		                   maestro_id, data_inizio, data_fine, eta_max_giovani, prezzo_giovani, attivo)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, TRUE)
	`, titolo, giorno, ora, durata, posti, prezzo, maestroID,
		dataInizio.Format("2006-01-02"), dataFine.Format("2006-01-02"),
		etaMaxGiovani, prezzoGiovani,
	)
	if err != nil {
		log.Printf("[corsi] insert corso: %v", err)
		http.Error(w, "Errore DB", http.StatusInternalServerError)
		return
	}

	corsoID, _ := res.LastInsertId()

	date := generaDateLezioni(dataInizio, dataFine, giorno)
	for _, d := range date {
		_, err := h.DB.ExecContext(ctx, `
			INSERT INTO lezioni (corso_id, data, ora, durata_min, max_posti, prezzo, stato)
			VALUES (?, ?, ?, ?, ?, ?, 'programmata')
		`, corsoID, d.Format("2006-01-02"), ora, durata, posti, prezzo)
		if err != nil {
			log.Printf("[corsi] insert lezione %s: %v", d.Format("2006-01-02"), err)
		}
	}

	http.Redirect(w, r, fmt.Sprintf("/corsi/%d", corsoID), http.StatusSeeOther)
}

func (h *Handler) DettaglioCorso(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))

	corso, err := h.fetchCorso(ctx, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	rows, err := h.DB.QueryContext(ctx, `
		SELECT l.id, l.corso_id, l.data, l.ora, l.durata_min, l.max_posti, l.prezzo, l.stato, COALESCE(l.nota,''),
		       COUNT(p.id) as presenti
		FROM lezioni l
		LEFT JOIN presenze p ON p.lezione_id = l.id
		WHERE l.corso_id = ?
		GROUP BY l.id
		ORDER BY l.data ASC
	`, id)
	if err != nil {
		http.Error(w, "Errore DB", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var lezioni []models.Lezione
	for rows.Next() {
		var l models.Lezione
		if err := rows.Scan(&l.ID, &l.CorsoID, &l.Data, &l.Ora, &l.DurataMin, &l.MaxPosti,
			&l.Prezzo, &l.Stato, &l.Nota, &l.Presenti); err != nil {
			log.Printf("[corsi] scan lezione: %v", err)
			continue
		}
		lezioni = append(lezioni, l)
	}

	iscritti, err := h.fetchIscrittiCorso(ctx, id, corso)
	if err != nil {
		http.Error(w, "Errore DB", http.StatusInternalServerError)
		return
	}

	soci := h.fetchTuttiSoci(ctx)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderTempl(w, r, templates.Page(corso.Titolo, "corsi",
		templates.DettaglioCorsoPage(corso, lezioni, iscritti, soci)))
}

func (h *Handler) ModificaCorsoForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	corso, err := h.fetchCorso(ctx, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	maestri := h.fetchMaestri(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderTempl(w, r, templates.Page("Modifica Corso", "corsi", templates.NuovoCorsoForm(maestri, 0, &corso)))
}

func (h *Handler) AggiornaCorso(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form non valido", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))

	titolo := r.FormValue("titolo")
	ora := r.FormValue("ora")
	durataStr := r.FormValue("durata_min")
	postiStr := r.FormValue("max_posti")
	prezzoStr := r.FormValue("prezzo_lezione")
	etaMaxStr := r.FormValue("eta_max_giovani")
	prezzoGiovaniStr := r.FormValue("prezzo_giovani")
	maestroIDStr := r.FormValue("maestro_id")
	attivo := r.FormValue("attivo") == "on"

	durata, _ := strconv.Atoi(durataStr)
	if durata == 0 {
		durata = 60
	}
	posti, _ := strconv.Atoi(postiStr)
	prezzo, _ := strconv.ParseFloat(prezzoStr, 64)

	var etaMaxGiovani *int
	var prezzoGiovani *float64
	if etaMaxStr != "" {
		v, _ := strconv.Atoi(etaMaxStr)
		etaMaxGiovani = &v
	}
	if prezzoGiovaniStr != "" && etaMaxGiovani != nil {
		v, _ := strconv.ParseFloat(prezzoGiovaniStr, 64)
		prezzoGiovani = &v
	}
	var maestroID *int
	if maestroIDStr != "" && maestroIDStr != "0" {
		v, _ := strconv.Atoi(maestroIDStr)
		maestroID = &v
	}

	_, err := h.DB.ExecContext(ctx, `
		UPDATE corsi SET titolo=?, ora=?, durata_min=?, max_posti=?, prezzo_lezione=?,
		                 eta_max_giovani=?, prezzo_giovani=?, maestro_id=?, attivo=?
		WHERE id=?
	`, titolo, ora, durata, posti, prezzo, etaMaxGiovani, prezzoGiovani, maestroID, attivo, id)
	if err != nil {
		log.Printf("[corsi] update corso: %v", err)
		http.Error(w, "Errore DB", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/corsi/%d", id), http.StatusSeeOther)
}

func (h *Handler) EliminaCorso(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	if _, err := h.DB.ExecContext(ctx, `DELETE FROM corsi WHERE id=?`, id); err != nil {
		http.Error(w, "Errore DB", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/corsi", http.StatusSeeOther)
}

func (h *Handler) fetchCorso(ctx context.Context, id int) (models.Corso, error) {
	var c models.Corso
	var maestroID sql.NullInt64
	var etaMax sql.NullInt64
	var prezzoGiovani sql.NullFloat64
	err := h.DB.QueryRowContext(ctx, `
		SELECT c.id, c.titolo, c.giorno_settimana, c.ora, c.durata_min, c.max_posti,
		       c.prezzo_lezione, c.maestro_id, COALESCE(u.nome,'') as maestro_nome,
		       c.data_inizio, c.data_fine, c.eta_max_giovani, c.prezzo_giovani, c.attivo, c.creato_il
		FROM corsi c
		LEFT JOIN utenti u ON u.id = c.maestro_id
		WHERE c.id = ?
	`, id).Scan(
		&c.ID, &c.Titolo, &c.GiornoSettimana, &c.Ora, &c.DurataMin, &c.MaxPosti,
		&c.PrezzoLezione, &maestroID, &c.MaestroNome,
		&c.DataInizio, &c.DataFine, &etaMax, &prezzoGiovani, &c.Attivo, &c.CreatoIl,
	)
	if err != nil {
		return c, err
	}
	if maestroID.Valid {
		v := int(maestroID.Int64)
		c.MaestroID = &v
	}
	if etaMax.Valid {
		v := int(etaMax.Int64)
		c.EtaMaxGiovani = &v
	}
	if prezzoGiovani.Valid {
		c.PrezzoGiovani = &prezzoGiovani.Float64
	}
	return c, nil
}

func (h *Handler) fetchMaestri(r *http.Request) []models.Utente {
	rows, err := h.DB.QueryContext(r.Context(), `SELECT id, email, nome, ruolo, creato_il FROM utenti WHERE ruolo='maestro' ORDER BY nome`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []models.Utente
	for rows.Next() {
		var u models.Utente
		rows.Scan(&u.ID, &u.Email, &u.Nome, &u.Ruolo, &u.CreatoIl)
		result = append(result, u)
	}
	return result
}

func (h *Handler) fetchTuttiSoci(ctx context.Context) []models.Socio {
	rows, err := h.DB.QueryContext(ctx, `SELECT id, nome, cognome, email, COALESCE(telefono,''), COALESCE(data_nascita,'0001-01-01'), COALESCE(note,''), creato_il FROM soci ORDER BY cognome, nome`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []models.Socio
	for rows.Next() {
		var s models.Socio
		rows.Scan(&s.ID, &s.Nome, &s.Cognome, &s.Email, &s.Telefono, &s.DataNascita, &s.Note, &s.CreatedAt)
		result = append(result, s)
	}
	return result
}
