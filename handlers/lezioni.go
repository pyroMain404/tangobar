package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/a-h/templ"
	"tango-gestionale/models"
	"tango-gestionale/templates"
)

// ListaLezioni handles GET /lezioni
func (h *Handler) ListaLezioni(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT
			l.id,
			l.titolo,
			l.insegnante,
			l.data_ora,
			l.durata_min,
			l.max_posti,
			l.prezzo,
			COALESCE(COUNT(il.id), 0) as iscritti
		FROM lezioni l
		LEFT JOIN iscrizioni_lezione il ON l.id = il.lezione_id
		GROUP BY l.id, l.titolo, l.insegnante, l.data_ora, l.durata_min, l.max_posti, l.prezzo
		ORDER BY l.data_ora DESC
	`

	rows, err := h.DB.QueryContext(r.Context(), query)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var lezioni []models.Lezione
	for rows.Next() {
		var l models.Lezione
		err := rows.Scan(
			&l.ID,
			&l.Titolo,
			&l.Insegnante,
			&l.DataOra,
			&l.DurataMin,
			&l.MaxPosti,
			&l.Prezzo,
			&l.Iscritti,
		)
		if err != nil {
			http.Error(w, "Scan error", http.StatusInternalServerError)
			return
		}
		lezioni = append(lezioni, l)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, "Query error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(templates.LezioniPage(lezioni)).ServeHTTP(w, r)
}

// DettaglioLezione handles GET /lezioni/{id}
func (h *Handler) DettaglioLezione(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Get lezione
	var lezione models.Lezione
	err = h.DB.QueryRowContext(r.Context(), `
		SELECT id, titolo, insegnante, data_ora, durata_min, max_posti, prezzo
		FROM lezioni
		WHERE id = $1
	`, id).Scan(
		&lezione.ID,
		&lezione.Titolo,
		&lezione.Insegnante,
		&lezione.DataOra,
		&lezione.DurataMin,
		&lezione.MaxPosti,
		&lezione.Prezzo,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Lezione not found", http.StatusNotFound)
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	// Get iscrizioni with socio names
	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT il.id, il.lezione_id, il.socio_id, s.nome, s.cognome
		FROM iscrizioni_lezione il
		JOIN soci s ON il.socio_id = s.id
		WHERE il.lezione_id = $1
		ORDER BY s.cognome, s.nome
	`, id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var iscrizioni []models.Iscrizione
	for rows.Next() {
		var isc models.Iscrizione
		var nome, cognome string
		err := rows.Scan(
			&isc.ID,
			&isc.LezioneID,
			&isc.SocioID,
			&nome,
			&cognome,
		)
		if err != nil {
			http.Error(w, "Scan error", http.StatusInternalServerError)
			return
		}
		isc.NomeSocio = nome
		isc.CognomeSocio = cognome
		iscrizioni = append(iscrizioni, isc)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, "Query error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(templates.LezioneDettaglio(lezione, iscrizioni)).ServeHTTP(w, r)
}

// NuovaLezioneForm handles GET /lezioni/nuova
func (h *Handler) NuovaLezioneForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(templates.LezioneForm(nil)).ServeHTTP(w, r)
}

// ModificaLezioneForm handles GET /lezioni/{id}/modifica
func (h *Handler) ModificaLezioneForm(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var lezione models.Lezione
	err = h.DB.QueryRowContext(r.Context(), `
		SELECT id, titolo, insegnante, data_ora, durata_min, max_posti, prezzo
		FROM lezioni
		WHERE id = $1
	`, id).Scan(
		&lezione.ID,
		&lezione.Titolo,
		&lezione.Insegnante,
		&lezione.DataOra,
		&lezione.DurataMin,
		&lezione.MaxPosti,
		&lezione.Prezzo,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Lezione not found", http.StatusNotFound)
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(templates.LezioneForm(&lezione)).ServeHTTP(w, r)
}

// CreaLezione handles POST /lezioni
func (h *Handler) CreaLezione(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form parse error", http.StatusBadRequest)
		return
	}

	titolo := r.FormValue("titolo")
	insegnante := r.FormValue("insegnante")
	dataOraStr := r.FormValue("data_ora")
	durataMinStr := r.FormValue("durata_min")
	maxPostiStr := r.FormValue("max_posti")
	prezzoStr := r.FormValue("prezzo")

	durataMin, err := strconv.Atoi(durataMinStr)
	if err != nil {
		http.Error(w, "Invalid duration", http.StatusBadRequest)
		return
	}

	maxPosti, err := strconv.Atoi(maxPostiStr)
	if err != nil {
		http.Error(w, "Invalid max posti", http.StatusBadRequest)
		return
	}

	prezzo, err := strconv.ParseFloat(prezzoStr, 64)
	if err != nil {
		http.Error(w, "Invalid price", http.StatusBadRequest)
		return
	}

	dataOra, err := time.Parse("2006-01-02T15:04", dataOraStr)
	if err != nil {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	var lezioneID int
	err = h.DB.QueryRowContext(r.Context(), `
		INSERT INTO lezioni (titolo, insegnante, data_ora, durata_min, max_posti, prezzo)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, titolo, insegnante, dataOra, durataMin, maxPosti, prezzo).Scan(&lezioneID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/lezioni", http.StatusSeeOther)
}

// AggiornaLezione handles PUT /lezioni/{id}
func (h *Handler) AggiornaLezione(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form parse error", http.StatusBadRequest)
		return
	}

	titolo := r.FormValue("titolo")
	insegnante := r.FormValue("insegnante")
	dataOraStr := r.FormValue("data_ora")
	durataMinStr := r.FormValue("durata_min")
	maxPostiStr := r.FormValue("max_posti")
	prezzoStr := r.FormValue("prezzo")

	durataMin, err := strconv.Atoi(durataMinStr)
	if err != nil {
		http.Error(w, "Invalid duration", http.StatusBadRequest)
		return
	}

	maxPosti, err := strconv.Atoi(maxPostiStr)
	if err != nil {
		http.Error(w, "Invalid max posti", http.StatusBadRequest)
		return
	}

	prezzo, err := strconv.ParseFloat(prezzoStr, 64)
	if err != nil {
		http.Error(w, "Invalid price", http.StatusBadRequest)
		return
	}

	dataOra, err := time.Parse("2006-01-02T15:04", dataOraStr)
	if err != nil {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	_, err = h.DB.ExecContext(r.Context(), `
		UPDATE lezioni
		SET titolo = $1, insegnante = $2, data_ora = $3, durata_min = $4, max_posti = $5, prezzo = $6
		WHERE id = $7
	`, titolo, insegnante, dataOra, durataMin, maxPosti, prezzo, id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/lezioni/"+idStr, http.StatusSeeOther)
}

// EliminaLezione handles DELETE /lezioni/{id}
func (h *Handler) EliminaLezione(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Delete iscrizioni first (foreign key constraint)
	_, err = h.DB.ExecContext(r.Context(), `
		DELETE FROM iscrizioni_lezione WHERE lezione_id = $1
	`, id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Delete lezione
	_, err = h.DB.ExecContext(r.Context(), `
		DELETE FROM lezioni WHERE id = $1
	`, id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/lezioni", http.StatusSeeOther)
}

// AggiungiIscrizione handles POST /lezioni/{id}/iscrizioni
func (h *Handler) AggiungiIscrizione(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	lezioneID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid lezione ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form parse error", http.StatusBadRequest)
		return
	}

	socioIDStr := r.FormValue("socio_id")
	socioID, err := strconv.Atoi(socioIDStr)
	if err != nil {
		http.Error(w, "Invalid socio ID", http.StatusBadRequest)
		return
	}

	_, err = h.DB.ExecContext(r.Context(), `
		INSERT INTO iscrizioni_lezione (lezione_id, socio_id)
		VALUES ($1, $2)
	`, lezioneID, socioID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/lezioni/"+idStr, http.StatusSeeOther)
}

// RimuoviIscrizione handles DELETE /lezioni/{lezioneID}/iscrizioni/{iscrizioneID}
func (h *Handler) RimuoviIscrizione(w http.ResponseWriter, r *http.Request) {
	lezioneIDStr := chi.URLParam(r, "lezioneID")
	iscrizioneIDStr := chi.URLParam(r, "iscrizioneID")

	lezioneID, err := strconv.Atoi(lezioneIDStr)
	if err != nil {
		http.Error(w, "Invalid lezione ID", http.StatusBadRequest)
		return
	}

	iscrizioneID, err := strconv.Atoi(iscrizioneIDStr)
	if err != nil {
		http.Error(w, "Invalid iscrizione ID", http.StatusBadRequest)
		return
	}

	_, err = h.DB.ExecContext(r.Context(), `
		DELETE FROM iscrizioni_lezione WHERE id = $1 AND lezione_id = $2
	`, iscrizioneID, lezioneID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/lezioni/"+lezioneIDStr, http.StatusSeeOther)
}

// FormIscrizione handles GET /lezioni/{id}/iscrizioni/nuova
func (h *Handler) FormIscrizione(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	lezioneID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Get list of soci not already enrolled in this lezione
	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT s.id, s.nome, s.cognome
		FROM soci s
		WHERE s.id NOT IN (
			SELECT socio_id FROM iscrizioni_lezione WHERE lezione_id = $1
		)
		ORDER BY s.cognome, s.nome
	`, lezioneID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var soci []models.Socio
	for rows.Next() {
		var s models.Socio
		err := rows.Scan(&s.ID, &s.Nome, &s.Cognome)
		if err != nil {
			http.Error(w, "Scan error", http.StatusInternalServerError)
			return
		}
		soci = append(soci, s)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, "Query error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(templates.IscrizioneForm(lezioneID, soci)).ServeHTTP(w, r)
}
