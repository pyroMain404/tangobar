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

	session.MaxAge = -1
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.Layout("Dashboard", templates.DashboardPage()).Render(r.Context(), w)
}
