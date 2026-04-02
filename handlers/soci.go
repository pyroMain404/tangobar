package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"tango-gestionale/models"
	"tango-gestionale/templates"
)

// ListaSoci handles GET /soci - returns all soci ordered by cognome, nome
func (h *Handler) ListaSoci(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT id, nome, cognome, email, telefono, data_nascita, note, created_at, updated_at
		FROM soci
		ORDER BY cognome, nome
	`)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var soci []models.Socio
	for rows.Next() {
		var s models.Socio
		err := rows.Scan(&s.ID, &s.Nome, &s.Cognome, &s.Email, &s.Telefono, &s.DataNascita, &s.Note, &s.CreatedAt, &s.UpdatedAt)
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

	renderTempl(w, r, templates.SociPage(soci))
}

// CercaSoci handles GET /soci/cerca?q=... - HTMX search endpoint
func (h *Handler) CercaSoci(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		renderTempl(w, r, templates.SociTabella([]models.Socio{}))
		return
	}

	searchTerm := "%" + q + "%"
	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT id, nome, cognome, email, telefono, data_nascita, note, created_at, updated_at
		FROM soci
		WHERE nome LIKE ? OR cognome LIKE ?
		ORDER BY cognome, nome
	`, searchTerm, searchTerm)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var soci []models.Socio
	for rows.Next() {
		var s models.Socio
		err := rows.Scan(&s.ID, &s.Nome, &s.Cognome, &s.Email, &s.Telefono, &s.DataNascita, &s.Note, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			http.Error(w, "Scan error", http.StatusInternalServerError)
			return
		}
		soci = append(soci, s)
	}

	renderTempl(w, r, templates.SociTabella(soci))
}

// DettaglioSocio handles GET /soci/{id} - returns socio details with tessere and iscrizioni
func (h *Handler) DettaglioSocio(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Get socio
	var socio models.Socio
	err := h.DB.QueryRowContext(r.Context(), `
		SELECT id, nome, cognome, email, telefono, data_nascita, note, created_at, updated_at
		FROM soci
		WHERE id = ?
	`, id).Scan(&socio.ID, &socio.Nome, &socio.Cognome, &socio.Email, &socio.Telefono, &socio.DataNascita, &socio.Note, &socio.CreatedAt, &socio.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Socio not found", http.StatusNotFound)
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	// Get tessere
	tessereRows, err := h.DB.QueryContext(r.Context(), `
		SELECT id, socio_id, tipo, emessa_il, valida_fino, importo, pagato, created_at, updated_at
		FROM tessere
		WHERE socio_id = ?
		ORDER BY emessa_il DESC
	`, id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tessereRows.Close()

	var tessere []models.Tessera
	for tessereRows.Next() {
		var t models.Tessera
		err := tessereRows.Scan(&t.ID, &t.SocioID, &t.Tipo, &t.EmessaIl, &t.ValidaFino, &t.Importo, &t.Pagato, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			http.Error(w, "Scan error", http.StatusInternalServerError)
			return
		}
		tessere = append(tessere, t)
	}

	// Get iscrizioni with lezione titles
	iscrizioniRows, err := h.DB.QueryContext(r.Context(), `
		SELECT i.id, i.socio_id, i.lezione_id, l.titolo, i.data_iscrizione, i.created_at, i.updated_at
		FROM iscrizioni i
		JOIN lezioni l ON i.lezione_id = l.id
		WHERE i.socio_id = ?
		ORDER BY i.data_iscrizione DESC
	`, id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer iscrizioniRows.Close()

	var iscrizioni []models.Iscrizione
	for iscrizioniRows.Next() {
		var isc models.Iscrizione
		var lezioneTitle string
		err := iscrizioniRows.Scan(&isc.ID, &isc.SocioID, &isc.LezioneID, &lezioneTitle, &isc.DataIscrizione, &isc.CreatedAt, &isc.UpdatedAt)
		if err != nil {
			http.Error(w, "Scan error", http.StatusInternalServerError)
			return
		}
		isc.LezioneTitolo = lezioneTitle
		iscrizioni = append(iscrizioni, isc)
	}

	renderTempl(w, r, templates.SocioDettaglio(socio, tessere, iscrizioni))
}

// NuovoSocioForm handles GET /soci/nuovo - returns empty form
func (h *Handler) NuovoSocioForm(w http.ResponseWriter, r *http.Request) {
	renderTempl(w, r, templates.SocioForm(nil))
}

// ModificaSocioForm handles GET /soci/{id}/modifica - returns pre-filled form
func (h *Handler) ModificaSocioForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var socio models.Socio
	err := h.DB.QueryRowContext(r.Context(), `
		SELECT id, nome, cognome, email, telefono, data_nascita, note, created_at, updated_at
		FROM soci
		WHERE id = ?
	`, id).Scan(&socio.ID, &socio.Nome, &socio.Cognome, &socio.Email, &socio.Telefono, &socio.DataNascita, &socio.Note, &socio.CreatedAt, &socio.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Socio not found", http.StatusNotFound)
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	renderTempl(w, r, templates.SocioForm(&socio))
}

// CreaSocio handles POST /soci - creates a new socio
func (h *Handler) CreaSocio(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	nome := r.PostFormValue("nome")
	cognome := r.PostFormValue("cognome")
	email := r.PostFormValue("email")
	telefono := r.PostFormValue("telefono")
	dataNascitaStr := r.PostFormValue("data_nascita")
	note := r.PostFormValue("note")

	var dataNascita *time.Time
	if dataNascitaStr != "" {
		t, err := time.Parse("2006-01-02", dataNascitaStr)
		if err == nil {
			dataNascita = &t
		}
	}

	result, err := h.DB.ExecContext(r.Context(), `
		INSERT INTO soci (nome, cognome, email, telefono, data_nascita, note, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, nome, cognome, email, telefono, dataNascita, note, time.Now(), time.Now())
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Check if this is an HTMX request
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/soci/"+strconv.FormatInt(id, 10))
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/soci/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

// AggiornaSocio handles PUT /soci/{id} - updates socio data
func (h *Handler) AggiornaSocio(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	nome := r.PostFormValue("nome")
	cognome := r.PostFormValue("cognome")
	email := r.PostFormValue("email")
	telefono := r.PostFormValue("telefono")
	dataNascitaStr := r.PostFormValue("data_nascita")
	note := r.PostFormValue("note")

	var dataNascita *time.Time
	if dataNascitaStr != "" {
		t, err := time.Parse("2006-01-02", dataNascitaStr)
		if err == nil {
			dataNascita = &t
		}
	}

	_, err = h.DB.ExecContext(r.Context(), `
		UPDATE soci
		SET nome = ?, cognome = ?, email = ?, telefono = ?, data_nascita = ?, note = ?, updated_at = ?
		WHERE id = ?
	`, nome, cognome, email, telefono, dataNascita, note, time.Now(), id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/soci/"+id, http.StatusSeeOther)
}

// EliminaSocio handles DELETE /soci/{id} - deletes a socio
func (h *Handler) EliminaSocio(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	_, err := h.DB.ExecContext(r.Context(), `
		DELETE FROM soci
		WHERE id = ?
	`, id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/soci", http.StatusSeeOther)
}

// renderTempl is a helper function to render templ components
func renderTempl(w http.ResponseWriter, r *http.Request, component templ.Component) error {
	return component.Render(r.Context(), w)
}
