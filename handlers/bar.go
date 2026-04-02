package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"tango-gestionale/models"
	"tango-gestionale/templates"
)

// ListaBar handles GET /bar
func (h *Handler) ListaBar(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`
		SELECT id, categoria, nome, quantita, prezzo_unitario
		FROM bar_items
		ORDER BY categoria, nome
	`)
	if err != nil {
		http.Error(w, "Errore nel caricamento bar items", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []models.BarItem
	for rows.Next() {
		var item models.BarItem
		if err := rows.Scan(&item.ID, &item.Categoria, &item.Nome, &item.Quantita, &item.Prezzo); err != nil {
			http.Error(w, "Errore nella lettura bar items", http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.BarPage(items).Render(r.Context(), w)
}

// BarTabellaPartial handles GET /bar/tabella (HTMX)
func (h *Handler) BarTabellaPartial(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`
		SELECT id, categoria, nome, quantita, prezzo_unitario
		FROM bar_items
		ORDER BY categoria, nome
	`)
	if err != nil {
		http.Error(w, "Errore nel caricamento bar items", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []models.BarItem
	for rows.Next() {
		var item models.BarItem
		if err := rows.Scan(&item.ID, &item.Categoria, &item.Nome, &item.Quantita, &item.Prezzo); err != nil {
			http.Error(w, "Errore nella lettura bar items", http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.BarTabella(items).Render(r.Context(), w)
}

// NuovoBarItemForm handles GET /bar/nuovo
func (h *Handler) NuovoBarItemForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.BarItemForm(nil).Render(r.Context(), w)
}

// ModificaBarItemForm handles GET /bar/{id}/modifica
func (h *Handler) ModificaBarItemForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var item models.BarItem
	err := h.DB.QueryRow(`
		SELECT id, categoria, nome, quantita, prezzo_unitario
		FROM bar_items
		WHERE id = ?
	`, id).Scan(&item.ID, &item.Categoria, &item.Nome, &item.Quantita, &item.Prezzo)

	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Errore nel caricamento item", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.BarItemForm(&item).Render(r.Context(), w)
}

// CreaBarItem handles POST /bar
func (h *Handler) CreaBarItem(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Errore nel parsing form", http.StatusBadRequest)
		return
	}

	categoria := r.FormValue("categoria")
	nome := r.FormValue("nome")
	quantitaStr := r.FormValue("quantita")
	prezzoStr := r.FormValue("prezzo_unitario")

	quantita, err := strconv.ParseFloat(quantitaStr, 64)
	if err != nil {
		http.Error(w, "Quantita non valida", http.StatusBadRequest)
		return
	}

	prezzo, err := strconv.ParseFloat(prezzoStr, 64)
	if err != nil {
		http.Error(w, "Prezzo non valido", http.StatusBadRequest)
		return
	}

	_, err = h.DB.Exec(`
		INSERT INTO bar_items (categoria, nome, quantita, prezzo_unitario)
		VALUES (?, ?, ?, ?)
	`, categoria, nome, quantita, prezzo)
	if err != nil {
		http.Error(w, "Errore creazione item", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/bar", http.StatusSeeOther)
}

// AggiornaBarItem handles PUT /bar/{id}
func (h *Handler) AggiornaBarItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Errore nel parsing form", http.StatusBadRequest)
		return
	}

	categoria := r.FormValue("categoria")
	nome := r.FormValue("nome")
	quantitaStr := r.FormValue("quantita")
	prezzoStr := r.FormValue("prezzo_unitario")

	quantita, err := strconv.ParseFloat(quantitaStr, 64)
	if err != nil {
		http.Error(w, "Quantita non valida", http.StatusBadRequest)
		return
	}

	prezzo, err := strconv.ParseFloat(prezzoStr, 64)
	if err != nil {
		http.Error(w, "Prezzo non valido", http.StatusBadRequest)
		return
	}

	result, err := h.DB.Exec(`
		UPDATE bar_items
		SET categoria = ?, nome = ?, quantita = ?, prezzo_unitario = ?
		WHERE id = ?
	`, categoria, nome, quantita, prezzo, id)
	if err != nil {
		http.Error(w, "Errore aggiornamento item", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, "/bar", http.StatusSeeOther)
}

// RegistraMovimento handles POST /bar/{id}/movimento (HTMX)
func (h *Handler) RegistraMovimento(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Errore nel parsing form", http.StatusBadRequest)
		return
	}

	deltaStr := r.FormValue("delta")
	nota := r.FormValue("nota")

	delta, err := strconv.ParseInt(deltaStr, 10, 64)
	if err != nil {
		http.Error(w, "Delta non valido", http.StatusBadRequest)
		return
	}

	// Start transaction
	tx, err := h.DB.Begin()
	if err != nil {
		http.Error(w, "Errore inizio transazione", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Insert movimento
	_, err = tx.Exec(`
		INSERT INTO bar_movimenti (bar_item_id, delta, nota)
		VALUES (?, ?, ?)
	`, id, delta, nota)
	if err != nil {
		http.Error(w, "Errore creazione movimento", http.StatusInternalServerError)
		return
	}

	// Update quantita
	_, err = tx.Exec(`
		UPDATE bar_items
		SET quantita = quantita + ?
		WHERE id = ?
	`, delta, id)
	if err != nil {
		http.Error(w, "Errore aggiornamento quantita", http.StatusInternalServerError)
		return
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, "Errore commit transazione", http.StatusInternalServerError)
		return
	}

	// Re-fetch all items and render table for HTMX swap
	rows, err := h.DB.Query(`
		SELECT id, categoria, nome, quantita, prezzo_unitario
		FROM bar_items
		ORDER BY categoria, nome
	`)
	if err != nil {
		http.Error(w, "Errore nel caricamento bar items", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []models.BarItem
	for rows.Next() {
		var item models.BarItem
		if err := rows.Scan(&item.ID, &item.Categoria, &item.Nome, &item.Quantita, &item.Prezzo); err != nil {
			http.Error(w, "Errore nella lettura bar items", http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.BarTabella(items).Render(r.Context(), w)
}

// StoricoMovimenti handles GET /bar/{id}/movimenti
func (h *Handler) StoricoMovimenti(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Get item
	var item models.BarItem
	err := h.DB.QueryRow(`
		SELECT id, categoria, nome, quantita, prezzo_unitario
		FROM bar_items
		WHERE id = ?
	`, id).Scan(&item.ID, &item.Categoria, &item.Nome, &item.Quantita, &item.Prezzo)

	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Errore nel caricamento item", http.StatusInternalServerError)
		return
	}

	// Get movimenti with item name via JOIN
	movimentiRows, err := h.DB.Query(`
		SELECT m.id, m.bar_item_id, m.delta, m.nota, m.data_registrazione, b.nome
		FROM bar_movimenti m
		JOIN bar_items b ON m.bar_item_id = b.id
		WHERE m.bar_item_id = ?
		ORDER BY m.data_registrazione DESC
	`, id)
	if err != nil {
		http.Error(w, "Errore nel caricamento movimenti", http.StatusInternalServerError)
		return
	}
	defer movimentiRows.Close()

	var movimenti []models.BarMovimento
	for movimentiRows.Next() {
		var m models.BarMovimento
		var itemName string
		if err := movimentiRows.Scan(&m.ID, &m.ItemID, &m.Delta, &m.Nota, &m.Timestamp, &itemName); err != nil {
			http.Error(w, "Errore nella lettura movimenti", http.StatusInternalServerError)
			return
		}
		movimenti = append(movimenti, m)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.BarMovimenti(item, movimenti).Render(r.Context(), w)
}
