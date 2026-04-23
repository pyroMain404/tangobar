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
	pdfPath := env("PDF_PATH", "./data/pdf")
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
	if err := os.MkdirAll(pdfPath, 0755); err != nil {
		log.Fatalf("Failed to create pdf directory: %v", err)
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

		// Soci
		r.Get("/soci", h.ListaSoci)
		r.Get("/soci/cerca", h.CercaSoci)
		r.Get("/soci/nuovo", h.NuovoSocioForm)
		r.Post("/soci", h.CreaSocio)
		r.Get("/soci/{id}", h.DettaglioSocio)
		r.Get("/soci/{id}/modifica", h.ModificaSocioForm)
		r.Put("/soci/{id}", h.AggiornaSocio)
		r.Delete("/soci/{id}", h.EliminaSocio)

		// Tessere
		r.Get("/tessere", h.ListaTessere)
		r.Get("/tessere/nuova", h.NuovaTesseraForm)
		r.Post("/tessere", h.CreaTessera)
		r.Get("/tessere/{id}/rinnova", h.RinnovaTesseraForm)
		r.Post("/tessere/{id}/rinnova", h.RinnovaTessera)
		r.Post("/tessere/{id}/pagato", h.TogglePagatoTessera)
		r.Get("/tessere/{id}/pdf", h.StampaTessera)

		// Lezioni
		r.Get("/lezioni", h.ListaLezioni)
		r.Get("/lezioni/nuova", h.NuovaLezioneForm)
		r.Post("/lezioni", h.CreaLezione)
		r.Get("/lezioni/{id}", h.DettaglioLezione)
		r.Get("/lezioni/{id}/modifica", h.ModificaLezioneForm)
		r.Put("/lezioni/{id}", h.AggiornaLezione)
		r.Delete("/lezioni/{id}", h.EliminaLezione)
		r.Get("/lezioni/{id}/iscrizioni/nuova", h.FormIscrizione)
		r.Post("/lezioni/{id}/iscrizioni", h.AggiungiIscrizione)
		r.Delete("/lezioni/{lezioneID}/iscrizioni/{iscrizioneID}", h.RimuoviIscrizione)

		// Eventi
		r.Get("/eventi", h.ListaEventi)
		r.Get("/eventi/nuovo", h.NuovoEventoForm)
		r.Post("/eventi", h.CreaEvento)
		r.Get("/eventi/{id}", h.DettaglioEvento)
		r.Get("/eventi/{id}/modifica", h.ModificaEventoForm)
		r.Put("/eventi/{id}", h.AggiornaEvento)
		r.Delete("/eventi/{id}", h.EliminaEvento)

		// Milonga
		r.Get("/milonga/cerca", h.CercaSociMilonga)
		r.Post("/milonga/ingresso/{socioID}", h.RegistraIngresso)
		r.Post("/milonga/ingresso-ospite", h.RegistraIngressoOspite)

		// Fatture
		r.Get("/fatture", h.ListaFatture)
		r.Get("/fatture/nuova", h.NuovaFatturaForm)
		r.Post("/fatture", h.CreaFattura)
		r.Get("/fatture/{id}", h.DettaglioFattura)
		r.Post("/fatture/{id}/pagata", h.TogglePagataFattura)
		r.Get("/fatture/{id}/pdf", h.DownloadFatturaPDF)

		// Bar
		r.Get("/bar", h.ListaBar)
		r.Get("/bar/tabella", h.BarTabellaPartial)
		r.Get("/bar/nuovo", h.NuovoBarItemForm)
		r.Post("/bar", h.CreaBarItem)
		r.Get("/bar/{id}/modifica", h.ModificaBarItemForm)
		r.Put("/bar/{id}", h.AggiornaBarItem)
		r.Post("/bar/{id}/movimento", h.RegistraMovimento)
		r.Get("/bar/{id}/movimenti", h.StoricoMovimenti)

		// Admin-only
		r.Group(func(r chi.Router) {
			r.Use(h.RequireAdmin)

			r.Get("/admin/utenti", h.ListaUtenti)
			r.Get("/admin/utenti/nuovo", h.NuovoUtenteForm)
			r.Post("/admin/utenti", h.CreaUtente)
			r.Post("/admin/utenti/{id}/ruolo", h.CambiaRuoloUtente)
			r.Delete("/admin/utenti/{id}", h.EliminaUtente)
		})
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
