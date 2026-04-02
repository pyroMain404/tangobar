package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"tango-gestionale/models"
	"tango-gestionale/pdf"
	"tango-gestionale/templates"
)

// ListaTessere handles GET /tessere - returns all tessere with socio info
func (h *Handler) ListaTessere(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT t.id, t.socio_id, s.nome, s.cognome, t.tipo, t.emessa_il, t.valida_fino, t.importo, t.pagato, t.created_at, t.updated_at
		FROM tessere t
		JOIN soci s ON t.socio_id = s.id
		ORDER BY t.emessa_il DESC
	`)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tessere []models.Tessera
	for rows.Next() {
		var t models.Tessera
		err := rows.Scan(&t.ID, &t.SocioID, &t.NomeSocio, &t.CognomeSocio, &t.Tipo, &t.EmessaIl, &t.ValidaFino, &t.Importo, &t.Pagato, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			http.Error(w, "Scan error", http.StatusInternalServerError)
			return
		}
		tessere = append(tessere, t)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, "Query error", http.StatusInternalServerError)
		return
	}

	renderTempl(w, r, templates.TesserePage(tessere))
}

// NuovaTesseraForm handles GET /tessere/nuova?socio_id=X - returns form with socio info
func (h *Handler) NuovaTesseraForm(w http.ResponseWriter, r *http.Request) {
	socioIDStr := r.URL.Query().Get("socio_id")
	if socioIDStr == "" {
		http.Error(w, "Missing socio_id parameter", http.StatusBadRequest)
		return
	}

	var socioNome, socioCognome string
	err := h.DB.QueryRowContext(r.Context(), `
		SELECT nome, cognome
		FROM soci
		WHERE id = ?
	`, socioIDStr).Scan(&socioNome, &socioCognome)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Socio not found", http.StatusNotFound)
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	socioID, _ := strconv.ParseInt(socioIDStr, 10, 64)
	renderTempl(w, r, templates.TesseraForm(socioID, socioNome+" "+socioCognome))
}

// CreaTessera handles POST /tessere - creates a new tessera
func (h *Handler) CreaTessera(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	socioIDStr := r.PostFormValue("socio_id")
	tipo := r.PostFormValue("tipo")
	validaFinoStr := r.PostFormValue("valida_fino")
	importoStr := r.PostFormValue("importo")
	pagatoStr := r.PostFormValue("pagato")

	if socioIDStr == "" || tipo == "" || validaFinoStr == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	socioID, err := strconv.ParseInt(socioIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid socio_id", http.StatusBadRequest)
		return
	}

	validaFino, err := time.Parse("2006-01-02", validaFinoStr)
	if err != nil {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	var importo float64
	if importoStr != "" {
		importo, _ = strconv.ParseFloat(importoStr, 64)
	}

	pagato := pagatoStr == "on" || pagatoStr == "true"
	emoissaIl := time.Now()

	result, err := h.DB.ExecContext(r.Context(), `
		INSERT INTO tessere (socio_id, tipo, emessa_il, valida_fino, importo, pagato, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, socioID, tipo, emoissaIl, validaFino, importo, pagato, emoissaIl, emoissaIl)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	_ = result

	http.Redirect(w, r, "/soci/"+socioIDStr, http.StatusSeeOther)
}

// RinnovaTesseraForm handles GET /tessere/{id}/rinnova - returns renewal form
func (h *Handler) RinnovaTesseraForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var tessera models.Tessera
	err := h.DB.QueryRowContext(r.Context(), `
		SELECT id, socio_id, tipo, emessa_il, valida_fino, importo, pagato, created_at, updated_at
		FROM tessere
		WHERE id = ?
	`, id).Scan(&tessera.ID, &tessera.SocioID, &tessera.Tipo, &tessera.EmessaIl, &tessera.ValidaFino, &tessera.Importo, &tessera.Pagato, &tessera.CreatedAt, &tessera.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Tessera not found", http.StatusNotFound)
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	renderTempl(w, r, templates.TesseraRinnovo(tessera))
}

// RinnovaTessera handles POST /tessere/{id}/rinnova - creates renewed tessera
func (h *Handler) RinnovaTessera(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	// Get old tessera
	var oldTessera models.Tessera
	err = h.DB.QueryRowContext(r.Context(), `
		SELECT id, socio_id, tipo, emessa_il, valida_fino, importo, pagato, created_at, updated_at
		FROM tessere
		WHERE id = ?
	`, id).Scan(&oldTessera.ID, &oldTessera.SocioID, &oldTessera.Tipo, &oldTessera.EmessaIl, &oldTessera.ValidaFino, &oldTessera.Importo, &oldTessera.Pagato, &oldTessera.CreatedAt, &oldTessera.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Tessera not found", http.StatusNotFound)
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	validaFinoStr := r.PostFormValue("valida_fino")
	if validaFinoStr == "" {
		http.Error(w, "Missing valida_fino", http.StatusBadRequest)
		return
	}

	validaFino, err := time.Parse("2006-01-02", validaFinoStr)
	if err != nil {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	now := time.Now()
	pagatoStr := r.PostFormValue("pagato")
	pagato := pagatoStr == "on" || pagatoStr == "true"

	// Create new tessera with updated dates
	result, err := h.DB.ExecContext(r.Context(), `
		INSERT INTO tessere (socio_id, tipo, emessa_il, valida_fino, importo, pagato, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, oldTessera.SocioID, oldTessera.Tipo, now, validaFino, oldTessera.Importo, pagato, now, now)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	_ = result

	http.Redirect(w, r, "/soci/"+strconv.FormatInt(oldTessera.SocioID, 10), http.StatusSeeOther)
}

// TogglePagatoTessera handles POST /tessere/{id}/pagato - toggles payment status
func (h *Handler) TogglePagatoTessera(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Get current pagato status
	var pagato bool
	var socioID int64
	err := h.DB.QueryRowContext(r.Context(), `
		SELECT pagato, socio_id
		FROM tessere
		WHERE id = ?
	`, id).Scan(&pagato, &socioID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Tessera not found", http.StatusNotFound)
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	// Toggle pagato
	_, err = h.DB.ExecContext(r.Context(), `
		UPDATE tessere
		SET pagato = ?, updated_at = ?
		WHERE id = ?
	`, !pagato, time.Now(), id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Redirect to referrer or to soci detail page
	referrer := r.Header.Get("Referer")
	if referrer != "" {
		http.Redirect(w, r, referrer, http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/soci/"+strconv.FormatInt(socioID, 10), http.StatusSeeOther)
	}
}

// StampaTessera handles GET /tessere/{id}/pdf - generates and returns PDF
func (h *Handler) StampaTessera(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Get tessera and socio data
	var tessera models.Tessera
	var socio models.Socio
	err := h.DB.QueryRowContext(r.Context(), `
		SELECT t.id, t.socio_id, t.tipo, t.emessa_il, t.valida_fino, t.importo, t.pagato, t.created_at, t.updated_at,
		       s.id, s.nome, s.cognome, s.email, s.telefono, s.data_nascita, s.note, s.created_at, s.updated_at
		FROM tessere t
		JOIN soci s ON t.socio_id = s.id
		WHERE t.id = ?
	`, id).Scan(&tessera.ID, &tessera.SocioID, &tessera.Tipo, &tessera.EmessaIl, &tessera.ValidaFino, &tessera.Importo, &tessera.Pagato, &tessera.CreatedAt, &tessera.UpdatedAt,
		&socio.ID, &socio.Nome, &socio.Cognome, &socio.Email, &socio.Telefono, &socio.DataNascita, &socio.Note, &socio.CreatedAt, &socio.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Tessera not found", http.StatusNotFound)
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	// Generate PDF
	pdfData, err := pdf.GeneraTessera(socio, tessera)
	if err != nil {
		http.Error(w, "PDF generation error", http.StatusInternalServerError)
		return
	}

	// Write PDF response
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\"tessera_"+strconv.FormatInt(tessera.ID, 10)+".pdf\"")
	w.Header().Set("Content-Length", strconv.Itoa(len(pdfData)))
	_, _ = w.Write(pdfData)
}
