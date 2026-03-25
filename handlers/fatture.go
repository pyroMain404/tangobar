package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"tango-gestionale/models"
	"tango-gestionale/pdf"
	"tango-gestionale/templates"
)

// ListaFatture handles GET /fatture
func (h *Handler) ListaFatture(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`
		SELECT id, numero, data_emissione, socio_id, nome_cliente, totale, pagata, pdf_path
		FROM fatture
		ORDER BY data_emissione DESC
	`)
	if err != nil {
		http.Error(w, "Errore nel caricamento fatture", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var fatture []models.Fattura
	for rows.Next() {
		var f models.Fattura
		if err := rows.Scan(&f.ID, &f.Numero, &f.DataEmissione, &f.SocioID, &f.NomeCliente, &f.Totale, &f.Pagata, &f.PdfPath); err != nil {
			http.Error(w, "Errore nella lettura fatture", http.StatusInternalServerError)
			return
		}
		fatture = append(fatture, f)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.FatturePage(fatture).Render(r.Context(), w)
}

// DettaglioFattura handles GET /fatture/{id}
func (h *Handler) DettaglioFattura(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var fattura models.Fattura
	err := h.DB.QueryRow(`
		SELECT id, numero, data_emissione, socio_id, nome_cliente, totale, pagata, pdf_path
		FROM fatture
		WHERE id = ?
	`, id).Scan(&fattura.ID, &fattura.Numero, &fattura.DataEmissione, &fattura.SocioID, &fattura.NomeCliente, &fattura.Totale, &fattura.Pagata, &fattura.PdfPath)

	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Errore nel caricamento fattura", http.StatusInternalServerError)
		return
	}

	// Get righe
	righeRows, err := h.DB.Query(`
		SELECT id, fattura_id, descrizione, quantita, prezzo_unitario, totale
		FROM righe_fattura
		WHERE fattura_id = ?
		ORDER BY id ASC
	`, id)
	if err != nil {
		http.Error(w, "Errore nel caricamento righe", http.StatusInternalServerError)
		return
	}
	defer righeRows.Close()

	for righeRows.Next() {
		var riga models.RigaFattura
		if err := righeRows.Scan(&riga.ID, &riga.FatturaID, &riga.Descrizione, &riga.Quantita, &riga.PrezzoUnitario, &riga.Totale); err != nil {
			http.Error(w, "Errore nella lettura righe", http.StatusInternalServerError)
			return
		}
		fattura.Righe = append(fattura.Righe, riga)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.FatturaDettaglio(fattura).Render(r.Context(), w)
}

// NuovaFatturaForm handles GET /fatture/nuova
func (h *Handler) NuovaFatturaForm(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`SELECT id, nome FROM soci ORDER BY nome ASC`)
	if err != nil {
		http.Error(w, "Errore nel caricamento soci", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var soci []models.Socio
	for rows.Next() {
		var s models.Socio
		if err := rows.Scan(&s.ID, &s.Nome); err != nil {
			http.Error(w, "Errore nella lettura soci", http.StatusInternalServerError)
			return
		}
		soci = append(soci, s)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.FatturaForm(nil, soci).Render(r.Context(), w)
}

// CreaFattura handles POST /fatture
func (h *Handler) CreaFattura(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Errore nel parsing form", http.StatusBadRequest)
		return
	}

	socioIDStr := r.FormValue("socio_id")
	nomeCliente := r.FormValue("nome_cliente")

	// Parse arrays from form
	descrizioni := r.Form["righe[descrizione][]"]
	quantitaStrs := r.Form["righe[quantita][]"]
	prezziStrs := r.Form["righe[prezzo_unit][]"]

	if len(descrizioni) == 0 || len(descrizioni) != len(quantitaStrs) || len(quantitaStrs) != len(prezziStrs) {
		http.Error(w, "Errore: righe non valide", http.StatusBadRequest)
		return
	}

	// Parse righe
	var righe []models.RigaFattura
	var totale float64

	for i := range descrizioni {
		quantita, err := strconv.ParseFloat(quantitaStrs[i], 64)
		if err != nil {
			http.Error(w, "Quantita non valida", http.StatusBadRequest)
			return
		}

		prezzoUnitario, err := strconv.ParseFloat(prezziStrs[i], 64)
		if err != nil {
			http.Error(w, "Prezzo unitario non valido", http.StatusBadRequest)
			return
		}

		rigaTotal := quantita * prezzoUnitario
		totale += rigaTotal

		righe = append(righe, models.RigaFattura{
			Descrizione:     descrizioni[i],
			Quantita:        quantita,
			PrezzoUnitario:  prezzoUnitario,
			Totale:          rigaTotal,
		})
	}

	// Generate numero as YYYY/NNN
	year := time.Now().Year()
	var count int
	err := h.DB.QueryRow(`
		SELECT COUNT(*) FROM fatture WHERE YEAR(data_emissione) = ?
	`, year).Scan(&count)
	if err != nil {
		http.Error(w, "Errore nel calcolo numero fattura", http.StatusInternalServerError)
		return
	}

	numero := fmt.Sprintf("%d/%03d", year, count+1)

	// Start transaction
	tx, err := h.DB.Begin()
	if err != nil {
		http.Error(w, "Errore inizio transazione", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Insert fattura
	var socioID sql.NullInt64
	if socioIDStr != "" {
		id, err := strconv.ParseInt(socioIDStr, 10, 64)
		if err == nil {
			socioID = sql.NullInt64{Int64: id, Valid: true}
		}
	}

	var fatturaID int64
	result, err := tx.Exec(`
		INSERT INTO fatture (numero, data_emissione, socio_id, nome_cliente, totale, pagata)
		VALUES (?, ?, ?, ?, ?, ?)
	`, numero, time.Now(), socioID, nomeCliente, totale, false)
	if err != nil {
		http.Error(w, "Errore creazione fattura", http.StatusInternalServerError)
		return
	}

	fatturaID, err = result.LastInsertId()
	if err != nil {
		http.Error(w, "Errore recupero ID fattura", http.StatusInternalServerError)
		return
	}

	// Insert righe
	for _, riga := range righe {
		_, err := tx.Exec(`
			INSERT INTO righe_fattura (fattura_id, descrizione, quantita, prezzo_unitario, totale)
			VALUES (?, ?, ?, ?, ?)
		`, fatturaID, riga.Descrizione, riga.Quantita, riga.PrezzoUnitario, riga.Totale)
		if err != nil {
			http.Error(w, "Errore creazione riga", http.StatusInternalServerError)
			return
		}
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, "Errore commit transazione", http.StatusInternalServerError)
		return
	}

	// Create PDF
	fattura := models.Fattura{
		ID:            int(fatturaID),
		Numero:        numero,
		DataEmissione: time.Now(),
		SocioID:       socioID,
		NomeCliente:   nomeCliente,
		Totale:        totale,
		Righe:         righe,
	}

	pdfBytes, err := pdf.GeneraFattura(fattura)
	if err != nil {
		http.Error(w, "Errore generazione PDF", http.StatusInternalServerError)
		return
	}

	// Save PDF to disk
	pdfPath := fmt.Sprintf("%s/fattura_%s.pdf", os.Getenv("PDF_PATH"), numero)
	if err := os.WriteFile(pdfPath, pdfBytes, 0644); err != nil {
		http.Error(w, "Errore salvataggio PDF", http.StatusInternalServerError)
		return
	}

	// Update fattura with pdf_path
	_, err = h.DB.Exec(`
		UPDATE fatture SET pdf_path = ? WHERE id = ?
	`, pdfPath, fatturaID)
	if err != nil {
		http.Error(w, "Errore aggiornamento pdf_path", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/fatture/%d", fatturaID), http.StatusSeeOther)
}

// TogglePagataFattura handles POST /fatture/{id}/pagata
func (h *Handler) TogglePagataFattura(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Get current pagata status
	var pagata bool
	err := h.DB.QueryRow(`SELECT pagata FROM fatture WHERE id = ?`, id).Scan(&pagata)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Errore nel caricamento fattura", http.StatusInternalServerError)
		return
	}

	// Toggle pagata
	_, err = h.DB.Exec(`UPDATE fatture SET pagata = ? WHERE id = ?`, !pagata, id)
	if err != nil {
		http.Error(w, "Errore aggiornamento fattura", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

// DownloadFatturaPDF handles GET /fatture/{id}/pdf
func (h *Handler) DownloadFatturaPDF(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var pdfPath sql.NullString
	var numero string
	err := h.DB.QueryRow(`
		SELECT pdf_path, numero FROM fatture WHERE id = ?
	`, id).Scan(&pdfPath, &numero)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Errore nel caricamento fattura", http.StatusInternalServerError)
		return
	}

	// If PDF doesn't exist, regenerate it
	if !pdfPath.Valid || pdfPath.String == "" {
		var fatturaID int
		var dataEmissione time.Time
		var socioID sql.NullInt64
		var nomeCliente string
		var totale float64

		err := h.DB.QueryRow(`
			SELECT id, data_emissione, socio_id, nome_cliente, totale
			FROM fatture WHERE id = ?
		`, id).Scan(&fatturaID, &dataEmissione, &socioID, &nomeCliente, &totale)
		if err != nil {
			http.Error(w, "Errore nel caricamento fattura", http.StatusInternalServerError)
			return
		}

		// Get righe
		righeRows, err := h.DB.Query(`
			SELECT descrizione, quantita, prezzo_unitario, totale
			FROM righe_fattura
			WHERE fattura_id = ?
		`, id)
		if err != nil {
			http.Error(w, "Errore nel caricamento righe", http.StatusInternalServerError)
			return
		}
		defer righeRows.Close()

		var righe []models.RigaFattura
		for righeRows.Next() {
			var riga models.RigaFattura
			if err := righeRows.Scan(&riga.Descrizione, &riga.Quantita, &riga.PrezzoUnitario, &riga.Totale); err != nil {
				http.Error(w, "Errore nella lettura righe", http.StatusInternalServerError)
				return
			}
			righe = append(righe, riga)
		}

		fattura := models.Fattura{
			ID:            fatturaID,
			Numero:        numero,
			DataEmissione: dataEmissione,
			SocioID:       socioID,
			NomeCliente:   nomeCliente,
			Totale:        totale,
			Righe:         righe,
		}

		pdfBytes, err := pdf.GeneraFattura(fattura)
		if err != nil {
			http.Error(w, "Errore generazione PDF", http.StatusInternalServerError)
			return
		}

		newPdfPath := fmt.Sprintf("%s/fattura_%s.pdf", os.Getenv("PDF_PATH"), numero)
		if err := os.WriteFile(newPdfPath, pdfBytes, 0644); err != nil {
			http.Error(w, "Errore salvataggio PDF", http.StatusInternalServerError)
			return
		}

		_, err = h.DB.Exec(`UPDATE fatture SET pdf_path = ? WHERE id = ?`, newPdfPath, id)
		if err != nil {
			http.Error(w, "Errore aggiornamento pdf_path", http.StatusInternalServerError)
			return
		}

		pdfPath = sql.NullString{String: newPdfPath, Valid: true}
	}

	// Serve PDF file
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"fattura_%s.pdf\"", numero))
	http.ServeFile(w, r, pdfPath.String)
}
