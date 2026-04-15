package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/a-h/templ"
	"tango-gestionale/models"
	"tango-gestionale/templates"
)

func nullInt64ToPtr(n sql.NullInt64) *int64 {
	if n.Valid {
		v := n.Int64
		return &v
	}
	return nil
}

func (h *Handler) fetchIngressi(ctx context.Context, eventoID int) ([]models.IngressoMilonga, float64, error) {
	rows, err := h.DB.QueryContext(ctx, `
		SELECT im.id, im.evento_id, im.socio_id, im.nome_ospite, im.importo,
		       CASE WHEN im.socio_id IS NOT NULL THEN s.cognome || ' ' || s.nome ELSE im.nome_ospite END as persona_nome
		FROM ingressi_milonga im
		LEFT JOIN soci s ON im.socio_id = s.id
		WHERE im.evento_id = $1
		ORDER BY im.id DESC
	`, eventoID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var ingressi []models.IngressoMilonga
	var totale float64
	for rows.Next() {
		var ing models.IngressoMilonga
		var eid, sid sql.NullInt64
		var pn sql.NullString
		if err := rows.Scan(&ing.ID, &eid, &sid, &ing.NomeOspite, &ing.Importo, &pn); err != nil {
			return nil, 0, err
		}
		ing.EventoID = nullInt64ToPtr(eid)
		ing.SocioID = nullInt64ToPtr(sid)
		if pn.Valid {
			ing.NomeSocio = pn.String
		}
		totale += ing.Importo
		ingressi = append(ingressi, ing)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, err
	}
	return ingressi, totale, nil
}

// ListaEventi handles GET /eventi
func (h *Handler) ListaEventi(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, titolo, descrizione, data_ora, location
		FROM eventi
		ORDER BY data_ora DESC
	`

	rows, err := h.DB.QueryContext(r.Context(), query)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var eventi []models.Evento
	for rows.Next() {
		var e models.Evento
		err := rows.Scan(
			&e.ID,
			&e.Titolo,
			&e.Descrizione,
			&e.DataOra,
			&e.Location,
		)
		if err != nil {
			http.Error(w, "Scan error", http.StatusInternalServerError)
			return
		}
		eventi = append(eventi, e)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, "Query error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(templates.Page("Eventi", "eventi", templates.EventiPage(eventi))).ServeHTTP(w, r)
}

// DettaglioEvento handles GET /eventi/{id}
func (h *Handler) DettaglioEvento(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Get evento
	var evento models.Evento
	err = h.DB.QueryRowContext(r.Context(), `
		SELECT id, titolo, descrizione, data_ora, location
		FROM eventi
		WHERE id = $1
	`, id).Scan(
		&evento.ID,
		&evento.Titolo,
		&evento.Descrizione,
		&evento.DataOra,
		&evento.Location,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Evento not found", http.StatusNotFound)
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	ingressi, totaleIncassato, err := h.fetchIngressi(r.Context(), id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(templates.Page("Dettaglio Evento", "eventi", templates.EventoDettaglio(evento, ingressi, totaleIncassato))).ServeHTTP(w, r)
}

// NuovoEventoForm handles GET /eventi/nuovo
func (h *Handler) NuovoEventoForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(templates.EventoForm(nil)).ServeHTTP(w, r)
}

// ModificaEventoForm handles GET /eventi/{id}/modifica
func (h *Handler) ModificaEventoForm(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var evento models.Evento
	err = h.DB.QueryRowContext(r.Context(), `
		SELECT id, titolo, descrizione, data_ora, location
		FROM eventi
		WHERE id = $1
	`, id).Scan(
		&evento.ID,
		&evento.Titolo,
		&evento.Descrizione,
		&evento.DataOra,
		&evento.Location,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Evento not found", http.StatusNotFound)
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(templates.EventoForm(&evento)).ServeHTTP(w, r)
}

// CreaEvento handles POST /eventi
func (h *Handler) CreaEvento(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form parse error", http.StatusBadRequest)
		return
	}

	titolo := r.FormValue("titolo")
	descrizione := r.FormValue("descrizione")
	dataOraStr := r.FormValue("data_ora")
	location := r.FormValue("location")

	// Parse data_ora - expecting ISO format
	// Adjust format as needed based on your form input
	dataOra := dataOraStr

	var eventoID int
	err := h.DB.QueryRowContext(r.Context(), `
		INSERT INTO eventi (titolo, descrizione, data_ora, location)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, titolo, descrizione, dataOra, location).Scan(&eventoID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/eventi", http.StatusSeeOther)
}

// AggiornaEvento handles PUT /eventi/{id}
func (h *Handler) AggiornaEvento(w http.ResponseWriter, r *http.Request) {
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
	descrizione := r.FormValue("descrizione")
	dataOraStr := r.FormValue("data_ora")
	location := r.FormValue("location")

	dataOra := dataOraStr

	_, err = h.DB.ExecContext(r.Context(), `
		UPDATE eventi
		SET titolo = $1, descrizione = $2, data_ora = $3, location = $4
		WHERE id = $5
	`, titolo, descrizione, dataOra, location, id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/eventi/"+idStr, http.StatusSeeOther)
}

// EliminaEvento handles DELETE /eventi/{id}
func (h *Handler) EliminaEvento(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Delete ingressi first (foreign key constraint)
	_, err = h.DB.ExecContext(r.Context(), `
		DELETE FROM ingressi_milonga WHERE evento_id = $1
	`, id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Delete evento
	_, err = h.DB.ExecContext(r.Context(), `
		DELETE FROM eventi WHERE id = $1
	`, id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/eventi", http.StatusSeeOther)
}

// RegistraIngresso handles POST /milonga/ingresso/{socioID}
func (h *Handler) RegistraIngresso(w http.ResponseWriter, r *http.Request) {
	socioIDStr := chi.URLParam(r, "socioID")
	socioID, err := strconv.Atoi(socioIDStr)
	if err != nil {
		http.Error(w, "Invalid socio ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form parse error", http.StatusBadRequest)
		return
	}

	eventoIDStr := r.FormValue("evento_id")
	importoStr := r.FormValue("importo")

	eventoID, err := strconv.Atoi(eventoIDStr)
	if err != nil {
		http.Error(w, "Invalid evento ID", http.StatusBadRequest)
		return
	}

	importo, err := strconv.ParseFloat(importoStr, 64)
	if err != nil {
		http.Error(w, "Invalid importo", http.StatusBadRequest)
		return
	}

	_, err = h.DB.ExecContext(r.Context(), `
		INSERT INTO ingressi_milonga (evento_id, socio_id, importo)
		VALUES ($1, $2, $3)
	`, eventoID, socioID, importo)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	ingressi, totaleIncassato, err := h.fetchIngressi(r.Context(), eventoID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(templates.ListaIngressi(ingressi, totaleIncassato)).ServeHTTP(w, r)
}

// RegistraIngressoOspite handles POST /milonga/ingresso-ospite
func (h *Handler) RegistraIngressoOspite(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form parse error", http.StatusBadRequest)
		return
	}

	nomeOspite := r.FormValue("nome_ospite")
	eventoIDStr := r.FormValue("evento_id")
	importoStr := r.FormValue("importo")

	eventoID, err := strconv.Atoi(eventoIDStr)
	if err != nil {
		http.Error(w, "Invalid evento ID", http.StatusBadRequest)
		return
	}

	importo, err := strconv.ParseFloat(importoStr, 64)
	if err != nil {
		http.Error(w, "Invalid importo", http.StatusBadRequest)
		return
	}

	_, err = h.DB.ExecContext(r.Context(), `
		INSERT INTO ingressi_milonga (evento_id, nome_ospite, importo)
		VALUES ($1, $2, $3)
	`, eventoID, nomeOspite, importo)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	ingressi, totaleIncassato, err := h.fetchIngressi(r.Context(), eventoID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templ.Handler(templates.ListaIngressi(ingressi, totaleIncassato)).ServeHTTP(w, r)
}

// CercaSociMilonga handles GET /milonga/cerca?q=...&evento_id=...
func (h *Handler) CercaSociMilonga(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	eventoIDStr := r.URL.Query().Get("evento_id")

	if query == "" {
		http.Error(w, "Search query required", http.StatusBadRequest)
		return
	}

	eventoID, err := strconv.Atoi(eventoIDStr)
	if err != nil {
		http.Error(w, "Invalid evento ID", http.StatusBadRequest)
		return
	}

	// Search soci by name, exclude those already with an ingresso for this evento
	searchTerm := "%" + query + "%"
	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT s.id, s.nome, s.cognome
		FROM soci s
		WHERE (s.nome ILIKE $1 OR s.cognome ILIKE $1)
		AND s.id NOT IN (
			SELECT DISTINCT socio_id FROM ingressi_milonga
			WHERE evento_id = $2 AND socio_id IS NOT NULL
		)
		ORDER BY s.cognome, s.nome
		LIMIT 10
	`, searchTerm, eventoID)
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
	templ.Handler(templates.SociSearchResults(soci, eventoID)).ServeHTTP(w, r)
}
