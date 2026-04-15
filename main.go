package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/crypto/bcrypt"
	"tango-gestionale/db"
	"tango-gestionale/handlers"
)

func main() {
	// Read environment variables
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/gestionale.db"
	}

	pdfPath := os.Getenv("PDF_PATH")
	if pdfPath == "" {
		pdfPath = "./data/pdf"
	}

	sessionKey := os.Getenv("SESSION_KEY")
	if sessionKey == "" {
		sessionKey = "cambia-questa-chiave"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create directories if they don't exist
	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	if err := os.MkdirAll(pdfPath, 0755); err != nil {
		log.Fatalf("Failed to create pdf directory: %v", err)
	}

	// Initialize database
	database, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Seed admin user if utenti table is empty
	if err := seedAdminUser(database); err != nil {
		log.Fatalf("Failed to seed admin user: %v", err)
	}

	// Initialize handlers
	h := &handlers.Handler{DB: database}

	// Initialize auth
	handlers.InitAuth(sessionKey)

	// Create router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// Public routes
	r.Post("/login", h.Login)
	r.Get("/login", h.LoginPage)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(h.RequireAuth)

		r.Get("/", h.Dashboard)
		r.Post("/logout", h.Logout)

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
	})

	// Start server
	log.Printf("Starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func seedAdminUser(database *sql.DB) error {
	// Check if utenti table is empty
	var count int
	err := database.QueryRow("SELECT COUNT(*) FROM utenti").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	// Hash password for "admin"
	hashedPassword, err := hashPassword("admin")
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Insert admin user
	_, err = database.Exec(
		"INSERT INTO utenti (username, password_hash) VALUES (?, ?)",
		"admin",
		hashedPassword,
	)
	if err != nil {
		return fmt.Errorf("failed to insert admin user: %w", err)
	}

	log.Println("Admin user seeded successfully")
	return nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
