package pdf

import (
	"fmt"
	"os"
	"tango-gestionale/models"

	"github.com/signintech/gopdf"
)

// GeneraTessera generates a membership card PDF from Socio and Tessera models
func GeneraTessera(s models.Socio, t models.Tessera) ([]byte, error) {
	// Credit card size: 85.6mm x 53.98mm
	cardWidth := 85.6
	cardHeight := 53.98

	// Create a new PDF with credit card dimensions
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{
		PageSize: gopdf.Rect{
			W: cardWidth,
			H: cardHeight,
		},
	})

	// Add page
	pdf.AddPage()

	// Try to add TTF fonts, fall back to Helvetica if not available
	regularFontName := "Inter-Regular"
	boldFontName := "Inter-Bold"

	regularFontPath := "assets/fonts/Inter-Regular.ttf"
	boldFontPath := "assets/fonts/Inter-Bold.ttf"

	// Try to load fonts
	if _, err := os.Stat(regularFontPath); err == nil {
		err = pdf.AddTTFFont(regularFontName, regularFontPath)
		if err != nil {
			regularFontName = "Helvetica"
		}
	} else {
		regularFontName = "Helvetica"
	}

	if _, err := os.Stat(boldFontPath); err == nil {
		err = pdf.AddTTFFont(boldFontName, boldFontPath)
		if err != nil {
			boldFontName = "Helvetica"
		}
	} else {
		boldFontName = "Helvetica"
	}

	// Dark background rectangle (RGB 30, 30, 80) filling entire card
	pdf.SetFillColor(30, 30, 80)
	pdf.RectFromUpperLeftWithStyle(0, 0, cardWidth, cardHeight, "F")

	// Logo at top-left if file exists
	logoPath := "assets/logo.png"
	if _, err := os.Stat(logoPath); err == nil {
		imageWidth := 12.0
		imageHeight := 10.0
		imageX := 5.0
		imageY := 5.0

		err = pdf.Image(logoPath, imageX, imageY, &gopdf.Rect{
			W: imageWidth,
			H: imageHeight,
		})
		_ = err
	}

	// Set text color to white for all text
	pdf.SetTextColor(255, 255, 255)

	// "TANGOBAR" text in white, bold, 10pt at top
	err := pdf.SetFont(boldFontName, "", 10)
	if err != nil {
		pdf.SetFont("Helvetica", "", 10)
	}
	pdf.SetX(5)
	pdf.SetY(8)
	pdf.Text("TANGOBAR")

	// Socio name in white, bold, 10pt
	err = pdf.SetFont(boldFontName, "", 10)
	if err != nil {
		pdf.SetFont("Helvetica", "", 10)
	}
	fullName := fmt.Sprintf("%s %s", s.Nome, s.Cognome)
	pdf.SetX(5)
	pdf.SetY(18)
	pdf.Text(fullName)

	// "Tessera: {tipo}" in white, regular, 8pt
	err = pdf.SetFont(regularFontName, "", 8)
	if err != nil {
		pdf.SetFont("Helvetica", "", 8)
	}
	pdf.SetX(5)
	pdf.SetY(28)
	pdf.Text(fmt.Sprintf("Tessera: %s", t.Tipo))

	// "Valida fino: MM/YYYY" in white, regular, 8pt
	pdf.SetX(5)
	pdf.SetY(35)
	pdf.Text(fmt.Sprintf("Valida fino: %s", t.ValidaFino.Format("01/2006")))

	// Gold accent line (RGB 255, 215, 0) - thin horizontal line near bottom
	pdf.SetStrokeColor(255, 215, 0)
	pdf.SetLineWidth(0.5)
	pdf.Line(5, 45, cardWidth-5, 45)

	// Get PDF bytes
	buf := pdf.GetBytesPdf()

	return buf, nil
}
