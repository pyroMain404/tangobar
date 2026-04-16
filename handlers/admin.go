package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"tango-gestionale/models"
	"tango-gestionale/templates"
)

// ListaUtenti handles GET /admin/utenti.
func (h *Handler) ListaUtenti(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`
		SELECT id, email, nome, ruolo, creato_il
		FROM utenti
		ORDER BY creato_il DESC
	`)
	if err != nil {
		http.Error(w, "Query error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var utenti []models.Utente
	for rows.Next() {
		var u models.Utente
		if err := rows.Scan(&u.ID, &u.Email, &u.Nome, &u.Ruolo, &u.CreatoIl); err != nil {
			http.Error(w, "Scan error", http.StatusInternalServerError)
			return
		}
		utenti = append(utenti, u)
	}

	currentID := currentUserID(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderTempl(w, r, templates.Page("Utenti", "admin-utenti",
		templates.UtentiPage(utenti, currentID, "")))
}

// NuovoUtenteForm handles GET /admin/utenti/nuovo — returns the form partial.
func (h *Handler) NuovoUtenteForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.UtenteForm("").Render(r.Context(), w)
}

// CreaUtente handles POST /admin/utenti.
func (h *Handler) CreaUtente(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	nome := strings.TrimSpace(r.FormValue("nome"))
	ruolo := strings.TrimSpace(r.FormValue("ruolo"))
	if ruolo != "admin" && ruolo != "staff" {
		ruolo = "staff"
	}

	if email == "" || nome == "" {
		templates.UtenteForm("Email e nome sono obbligatori").Render(r.Context(), w)
		return
	}

	_, err := h.DB.Exec(
		`INSERT INTO utenti (email, nome, ruolo) VALUES (?, ?, ?)`,
		email, nome, ruolo,
	)
	if err != nil {
		log.Printf("[admin] insert utente error: %v", err)
		templates.UtenteForm("Email già registrata o errore interno").Render(r.Context(), w)
		return
	}

	// Refresh the full list (HTMX target swaps the table body).
	rows, err := h.DB.Query(`SELECT id, email, nome, ruolo, creato_il FROM utenti ORDER BY creato_il DESC`)
	if err != nil {
		http.Error(w, "Query error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var utenti []models.Utente
	for rows.Next() {
		var u models.Utente
		if err := rows.Scan(&u.ID, &u.Email, &u.Nome, &u.Ruolo, &u.CreatoIl); err != nil {
			http.Error(w, "Scan error", http.StatusInternalServerError)
			return
		}
		utenti = append(utenti, u)
	}
	currentID := currentUserID(r)
	templates.UtentiTabella(utenti, currentID).Render(r.Context(), w)
}

// CambiaRuoloUtente handles POST /admin/utenti/{id}/ruolo.
func (h *Handler) CambiaRuoloUtente(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	ruolo := r.FormValue("ruolo")
	if ruolo != "admin" && ruolo != "staff" {
		http.Error(w, "Ruolo non valido", http.StatusBadRequest)
		return
	}

	// Guardrail: prevent demoting the last admin.
	if ruolo == "staff" {
		var adminCount int
		if err := h.DB.QueryRow(`SELECT COUNT(*) FROM utenti WHERE ruolo = 'admin' AND id != ?`, id).Scan(&adminCount); err != nil {
			http.Error(w, "Query error", http.StatusInternalServerError)
			return
		}
		if adminCount == 0 {
			http.Error(w, "Non puoi rimuovere l'ultimo amministratore", http.StatusConflict)
			return
		}
	}

	if _, err := h.DB.Exec(`UPDATE utenti SET ruolo = ? WHERE id = ?`, ruolo, id); err != nil {
		http.Error(w, "Update error", http.StatusInternalServerError)
		return
	}

	// Return the updated row (HTMX swap target is the single <tr>).
	var u models.Utente
	if err := h.DB.QueryRow(
		`SELECT id, email, nome, ruolo, creato_il FROM utenti WHERE id = ?`, id,
	).Scan(&u.ID, &u.Email, &u.Nome, &u.Ruolo, &u.CreatoIl); err != nil {
		http.Error(w, "Query error", http.StatusInternalServerError)
		return
	}
	currentID := currentUserID(r)
	templates.UtenteRiga(u, currentID).Render(r.Context(), w)
}

// EliminaUtente handles DELETE /admin/utenti/{id}.
func (h *Handler) EliminaUtente(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	// Prevent self-deletion
	if id == currentUserID(r) {
		http.Error(w, "Non puoi eliminare il tuo stesso account", http.StatusConflict)
		return
	}

	// Prevent deleting the last admin
	var ruolo string
	if err := h.DB.QueryRow(`SELECT ruolo FROM utenti WHERE id = ?`, id).Scan(&ruolo); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Utente non trovato", http.StatusNotFound)
			return
		}
		http.Error(w, "Query error", http.StatusInternalServerError)
		return
	}
	if ruolo == "admin" {
		var adminCount int
		if err := h.DB.QueryRow(`SELECT COUNT(*) FROM utenti WHERE ruolo = 'admin' AND id != ?`, id).Scan(&adminCount); err != nil {
			http.Error(w, "Query error", http.StatusInternalServerError)
			return
		}
		if adminCount == 0 {
			http.Error(w, "Non puoi eliminare l'ultimo amministratore", http.StatusConflict)
			return
		}
	}

	if _, err := h.DB.Exec(`DELETE FROM utenti WHERE id = ?`, id); err != nil {
		http.Error(w, "Delete error", http.StatusInternalServerError)
		return
	}
	// Return empty body — the HTMX target is the <tr> itself, which will be removed.
	w.WriteHeader(http.StatusOK)
}

// currentUserID extracts the logged-in user id from the session.
func currentUserID(r *http.Request) int {
	session, err := store.Get(r, sessionName)
	if err != nil || session.IsNew {
		return 0
	}
	if id, ok := session.Values["user_id"].(int); ok {
		return id
	}
	return 0
}
