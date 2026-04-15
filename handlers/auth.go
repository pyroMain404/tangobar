package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	"tango-gestionale/templates"
)

var store *sessions.CookieStore

// InitAuth initializes the session cookie store
func InitAuth(sessionKey string) {
	store = sessions.NewCookieStore([]byte(sessionKey))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	}
}

// LoginPage handles GET /login
func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.LoginPage("").Render(r.Context(), w)
}

// Login handles POST /login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginPage("Errore nel parsing del form").Render(r.Context(), w)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	// Query for user
	var userID int
	var hashedPassword string
	var nome string

	err := h.DB.QueryRow(`
		SELECT id, nome, password_hash
		FROM utenti
		WHERE username = ?
	`, username).Scan(&userID, &nome, &hashedPassword)

	if err == sql.ErrNoRows {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginPage("Credenziali non valide").Render(r.Context(), w)
		return
	}
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginPage("Errore nel caricamento utente").Render(r.Context(), w)
		return
	}

	// Compare password hash
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginPage("Credenziali non valide").Render(r.Context(), w)
		return
	}

	// Create session
	session, err := store.Get(r, "auth")
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginPage("Errore nella creazione della sessione").Render(r.Context(), w)
		return
	}

	session.Values["user_id"] = userID
	session.Values["username"] = username
	session.Values["nome"] = nome

	err = session.Save(r, w)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginPage("Errore nel salvataggio della sessione").Render(r.Context(), w)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Logout handles POST /logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "auth")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	session.Options.MaxAge = -1
	session.Save(r, w)

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// RequireAuth is middleware that checks for authentication
func (h *Handler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "auth")
		if err != nil || session.IsNew {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		userID, ok := session.Values["user_id"]
		if !ok || userID == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Dashboard handles GET /
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "auth")
	if err != nil || session.IsNew {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	stats := map[string]interface{}{}

	// Total soci
	var totalSoci int
	h.DB.QueryRow("SELECT COUNT(*) FROM soci").Scan(&totalSoci)
	stats["total_soci"] = totalSoci

	// Tessere in scadenza (prossimi 30 giorni)
	var tessereScadenza int
	h.DB.QueryRow("SELECT COUNT(*) FROM tessere WHERE valida_fino BETWEEN date('now') AND date('now', '+30 days')").Scan(&tessereScadenza)
	stats["tessere_scadenza"] = tessereScadenza

	// Lezioni prossime
	var lezioniProssime int
	h.DB.QueryRow("SELECT COUNT(*) FROM lezioni WHERE data_ora > datetime('now')").Scan(&lezioniProssime)
	stats["lezioni_prossime"] = lezioniProssime

	// Articoli bar sotto soglia
	var inventoryWarning int
	h.DB.QueryRow("SELECT COUNT(*) FROM bar_items WHERE quantita <= soglia_min").Scan(&inventoryWarning)
	stats["inventory_warning"] = inventoryWarning

	// Incasso oggi - eventi
	var incassoEventi float64
	h.DB.QueryRow("SELECT COALESCE(SUM(importo), 0) FROM ingressi_milonga WHERE date(timestamp) = date('now')").Scan(&incassoEventi)

	// Incasso oggi - tessere pagate oggi
	var incassoTessere float64
	h.DB.QueryRow("SELECT COALESCE(SUM(importo), 0) FROM tessere WHERE date(updated_at) = date('now') AND pagato = 1").Scan(&incassoTessere)

	// Incasso oggi - bar vendite (delta negativo = vendita)
	var incassoBar float64
	h.DB.QueryRow(`SELECT COALESCE(SUM(ABS(m.delta) * b.prezzo), 0)
		FROM bar_movimenti m JOIN bar_items b ON m.item_id = b.id
		WHERE m.delta < 0 AND date(m.timestamp) = date('now')`).Scan(&incassoBar)

	stats["incasso_oggi"] = incassoEventi + incassoTessere + incassoBar

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderTempl(w, r, templates.Page("Dashboard", "dashboard", templates.DashboardPage(stats)))
}
