package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"tango-gestionale/templates"
)

var store *sessions.CookieStore

const (
	tokenTTL    = 15 * time.Minute
	tokenBytes  = 32
	sessionName = "auth"
)

// InitAuth initializes the session cookie store.
func InitAuth(sessionKey string) {
	store = sessions.NewCookieStore([]byte(sessionKey))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
}

// LoginPage handles GET /login — renders the email entry form.
func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.LoginPage("").Render(r.Context(), w)
}

// Login handles POST /login — looks up the email in the allowlist, issues a
// one-time token, emails it as a magic link, and always responds with the same
// "check your inbox" screen (to avoid email enumeration).
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginPage("Richiesta non valida").Render(r.Context(), w)
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	if email == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginPage("Inserisci un indirizzo email").Render(r.Context(), w)
		return
	}

	// Allowlist lookup — silent on miss to prevent enumeration.
	var userID int
	err := h.DB.QueryRow(`SELECT id FROM utenti WHERE email = ?`, email).Scan(&userID)
	switch {
	case err == sql.ErrNoRows:
		// No user — still show the same confirmation screen.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginLinkSent(email).Render(r.Context(), w)
		return
	case err != nil:
		log.Printf("[auth] lookup error for %s: %v", email, err)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginPage("Errore interno, riprova più tardi").Render(r.Context(), w)
		return
	}

	// Issue token
	token, err := newToken()
	if err != nil {
		log.Printf("[auth] token gen error: %v", err)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginPage("Errore interno, riprova più tardi").Render(r.Context(), w)
		return
	}
	expiresAt := time.Now().Add(tokenTTL)

	_, err = h.DB.Exec(
		`INSERT INTO login_tokens (token, user_id, expires_at) VALUES (?, ?, ?)`,
		token, userID, expiresAt,
	)
	if err != nil {
		log.Printf("[auth] token insert error: %v", err)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginPage("Errore interno, riprova più tardi").Render(r.Context(), w)
		return
	}

	// Opportunistic cleanup of expired tokens (cheap, runs inline).
	_, _ = h.DB.Exec(`DELETE FROM login_tokens WHERE expires_at < datetime('now', '-1 day')`)

	// Build link and send
	link := strings.TrimRight(h.BaseURL, "/") + "/login/verify?token=" + token
	if err := h.Mailer.SendLoginLink(email, link); err != nil {
		log.Printf("[auth] mailer error: %v", err)
		// Still show the confirmation screen — the link was logged to stdout
		// as a fallback so the admin can recover it.
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.LoginLinkSent(email).Render(r.Context(), w)
}

// VerifyLogin handles GET /login/verify?token=... — consumes the token and
// creates the session.
func (h *Handler) VerifyLogin(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginPage("Link di accesso mancante o non valido").Render(r.Context(), w)
		return
	}

	var (
		userID    int
		email     string
		nome      string
		ruolo     string
		expiresAt time.Time
		usedAt    sql.NullTime
	)
	err := h.DB.QueryRow(`
		SELECT t.user_id, u.email, u.nome, u.ruolo, t.expires_at, t.used_at
		FROM login_tokens t JOIN utenti u ON u.id = t.user_id
		WHERE t.token = ?
	`, token).Scan(&userID, &email, &nome, &ruolo, &expiresAt, &usedAt)

	if err == sql.ErrNoRows {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginPage("Link non valido. Richiedi un nuovo accesso.").Render(r.Context(), w)
		return
	}
	if err != nil {
		log.Printf("[auth] verify lookup error: %v", err)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginPage("Errore interno, riprova più tardi").Render(r.Context(), w)
		return
	}

	if usedAt.Valid {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginPage("Link già utilizzato. Richiedi un nuovo accesso.").Render(r.Context(), w)
		return
	}
	if time.Now().After(expiresAt) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginPage("Link scaduto. Richiedi un nuovo accesso.").Render(r.Context(), w)
		return
	}

	// Consume the token
	if _, err := h.DB.Exec(`UPDATE login_tokens SET used_at = CURRENT_TIMESTAMP WHERE token = ?`, token); err != nil {
		log.Printf("[auth] verify consume error: %v", err)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginPage("Errore interno, riprova più tardi").Render(r.Context(), w)
		return
	}

	// Create session
	session, err := store.Get(r, sessionName)
	if err != nil {
		log.Printf("[auth] session get error: %v", err)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginPage("Errore nella creazione della sessione").Render(r.Context(), w)
		return
	}
	session.Values["user_id"] = userID
	session.Values["email"] = email
	session.Values["nome"] = nome
	session.Values["ruolo"] = ruolo
	if err := session.Save(r, w); err != nil {
		log.Printf("[auth] session save error: %v", err)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginPage("Errore nel salvataggio della sessione").Render(r.Context(), w)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Logout handles POST /logout (and GET /logout for convenience).
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, sessionName)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	session.Options.MaxAge = -1
	_ = session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// RequireAuth is middleware that checks for authentication.
func (h *Handler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, sessionName)
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

// WithUserContext threads the current session user's role onto the request
// context so templates can branch on it (e.g. show admin nav items).
// Must be installed after RequireAuth in the middleware chain.
func (h *Handler) WithUserContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, sessionName)
		if err == nil && !session.IsNew {
			if ruolo, ok := session.Values["ruolo"].(string); ok && ruolo != "" {
				r = r.WithContext(templates.WithRole(r.Context(), ruolo))
			}
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin is middleware that requires the current session user to have
// the 'admin' role. Must be used after RequireAuth.
func (h *Handler) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			http.Error(w, "Accesso negato: richiesti privilegi di amministratore", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isAdmin reports whether the current request's session belongs to an admin.
func isAdmin(r *http.Request) bool {
	session, err := store.Get(r, sessionName)
	if err != nil || session.IsNew {
		return false
	}
	ruolo, _ := session.Values["ruolo"].(string)
	return ruolo == "admin"
}

// IsAdmin exposes the session admin check to templates via the handler.
func (h *Handler) IsAdmin(r *http.Request) bool { return isAdmin(r) }

// Dashboard handles GET /
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.Page("Dashboard", templates.DashboardPage()).Render(r.Context(), w)
}

// RequireStaffOrAbove è middleware che richiede ruolo admin o staff.
func (h *Handler) RequireStaffOrAbove(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isStaffOrAbove(r) {
			http.Error(w, "Accesso negato: richiesti privilegi staff o superiori", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isStaffOrAbove(r *http.Request) bool {
	session, err := store.Get(r, sessionName)
	if err != nil || session.IsNew {
		return false
	}
	ruolo, _ := session.Values["ruolo"].(string)
	return ruolo == "admin" || ruolo == "staff"
}

// userIDFromRequest estrae l'ID utente dalla sessione corrente.
func userIDFromRequest(r *http.Request) (int, bool) {
	session, err := store.Get(r, sessionName)
	if err != nil || session.IsNew {
		return 0, false
	}
	id, ok := session.Values["user_id"]
	if !ok {
		return 0, false
	}
	switch v := id.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	}
	return 0, false
}

// newToken returns a URL-safe random token.
func newToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
