package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"tango-gestionale/db"
	"tango-gestionale/handlers"
	"tango-gestionale/mailer"
)

func main() {
	// ----- Environment -----
	dbPath := env("DB_PATH", "./data/gestionale.db")
	sessionKey := env("SESSION_KEY", "cambia-questa-chiave")
	port := env("PORT", "8080")

	appURL := env("APP_URL", "http://localhost:"+port)
	adminEmail := env("ADMIN_EMAIL", "admin@tangobar.local")

	m := &mailer.Mailer{
		Host:         os.Getenv("SMTP_HOST"),
		Port:         env("SMTP_PORT", "587"),
		User:         os.Getenv("SMTP_USER"),
		Pass:         os.Getenv("SMTP_PASS"),
		From:         env("SMTP_FROM", "no-reply@tb.gcoding.it"),
		LoginSubject: env("MAIL_LOGIN_SUBJECT", "TangoBar · Accesso"),
	}
	if !m.Configured() {
		log.Printf("[mailer] SMTP non configurato — i link di accesso saranno stampati su stdout.")
	}

	// ----- Filesystem -----
	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}
	// ----- DB -----
	database, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	if err := seedAdminUser(database, adminEmail); err != nil {
		log.Fatalf("Failed to seed admin user: %v", err)
	}

	// ----- Handlers -----
	h := &handlers.Handler{
		DB:      database,
		Mailer:  m,
		BaseURL: appURL,
	}
	handlers.InitAuth(sessionKey)

	// ----- Router -----
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// Public
	r.Get("/login", h.LoginPage)
	r.Post("/login", h.Login)
	r.Get("/login/verify", h.VerifyLogin)

	// Protected
	r.Group(func(r chi.Router) {
		r.Use(h.RequireAuth)
		r.Use(h.WithUserContext)

		r.Get("/", h.Dashboard)
		r.Post("/logout", h.Logout)
		r.Get("/logout", h.Logout)
	})

	log.Printf("Starting server on :%s (APP_URL=%s)", port, appURL)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// seedAdminUser ensures there is at least one admin user in the database.
// Called on every boot; idempotent when the admin already exists.
func seedAdminUser(database *sql.DB, adminEmail string) error {
	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM utenti").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err := database.Exec(
		"INSERT INTO utenti (email, nome, ruolo) VALUES (?, ?, 'admin')",
		adminEmail, "Admin",
	)
	if err != nil {
		return err
	}
	log.Printf("Admin user seeded: %s", adminEmail)
	return nil
}
