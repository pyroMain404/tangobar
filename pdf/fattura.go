package pdf

import (
	"fmt"
	"os"
	"strconv"
	"tango-gestionale/models"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
)

// GeneraFattura generates an invoice PDF from a Fattura model
func GeneraFattura(f models.Fattura) ([]byte, error) {
	// Configure A4 page with 15mm margins
	cfg := config.NewBuilder().
		WithPageFormat(config.PageFormatA4).
		WithLeftMargin(15).
		WithRightMargin(15).
		WithTopMargin(15).
		WithBottomMargin(15).
		Build()

	m := maroto.New(cfg)

	// Header row: logo left, "FATTURA {numero}" right
	m.AddRow(15, func(row config.Row) {
		// Try to add logo if it exists
		logoPath := "assets/logo.png"
		if _, err := os.Stat(logoPath); err == nil {
			row.Col(6, func(col config.Col) {
				col.Add(image.NewFromFileStr(logoPath, props.RectProp{
					Width:  20,
					Height: 15,
				}))
			})
		} else {
			row.Col(6, func(col config.Col) {
				col.Add(text.New("", props.TextProp{
					Top:   2,
					Size:  10,
					Align: align.Left,
				}))
			})
		}

		// Invoice number on the right
		row.Col(6, func(col config.Col) {
			col.Add(text.New(fmt.Sprintf("FATTURA %s", f.Numero), props.TextProp{
				Top:       5,
				Size:      14,
				Align:     align.Right,
				Style:     fontstyle.Bold,
			}))
		})
	})

	// Client info
	m.AddRow(10, func(row config.Row) {
		row.Col(12, func(col config.Col) {
			col.Add(text.New(fmt.Sprintf("Cliente: %s", f.NomeCliente), props.TextProp{
				Top:   2,
				Size:  10,
				Align: align.Left,
			}))
		})
	})

	m.AddRow(8, func(row config.Row) {
		row.Col(12, func(col config.Col) {
			col.Add(text.New(fmt.Sprintf("Data: %s", f.DataEmissione.Format("02/01/2006")), props.TextProp{
				Top:   2,
				Size:  10,
				Align: align.Left,
			}))
		})
	})

	// Separator line
	m.AddRow(3, func(row config.Row) {
		row.Col(12, func(col config.Col) {
			col.Add(text.New("", props.TextProp{
				Top:   1,
				Size:  1,
				Align: align.Left,
			}))
		})
	})

	// Table headers
	m.AddRow(8, func(row config.Row) {
		row.Col(5, func(col config.Col) {
			col.Add(text.New("Descrizione", props.TextProp{
				Top:   2,
				Size:  9,
				Style: fontstyle.Bold,
				Align: align.Left,
			}))
		})
		row.Col(2, func(col config.Col) {
			col.Add(text.New("Qtà", props.TextProp{
				Top:   2,
				Size:  9,
				Style: fontstyle.Bold,
				Align: align.Center,
			}))
		})
		row.Col(2.5, func(col config.Col) {
			col.Add(text.New("Prezzo Unit.", props.TextProp{
				Top:   2,
				Size:  9,
				Style: fontstyle.Bold,
				Align: align.Right,
			}))
		})
		row.Col(2.5, func(col config.Col) {
			col.Add(text.New("Totale", props.TextProp{
				Top:   2,
				Size:  9,
				Style: fontstyle.Bold,
				Align: align.Right,
			}))
		})
	})

	// Table rows for righe
	for _, riga := range f.Righe {
		m.AddRow(7, func(row config.Row) {
			row.Col(5, func(col config.Col) {
				col.Add(text.New(riga.Descrizione, props.TextProp{
					Top:   2,
					Size:  9,
					Align: align.Left,
				}))
			})
			row.Col(2, func(col config.Col) {
				col.Add(text.New(strconv.FormatFloat(riga.Quantita, 'f', 2, 64), props.TextProp{
					Top:   2,
					Size:  9,
					Align: align.Center,
				}))
			})
			row.Col(2.5, func(col config.Col) {
				col.Add(text.New(fmt.Sprintf("€ %.2f", riga.PrezzoUnit), props.TextProp{
					Top:   2,
					Size:  9,
					Align: align.Right,
				}))
			})
			row.Col(2.5, func(col config.Col) {
				col.Add(text.New(fmt.Sprintf("€ %.2f", riga.Totale), props.TextProp{
					Top:   2,
					Size:  9,
					Align: align.Right,
				}))
			})
		})
	}

	// Separator before total
	m.AddRow(3, func(row config.Row) {
		row.Col(12, func(col config.Col) {
			col.Add(text.New("", props.TextProp{
				Top:   1,
				Size:  1,
				Align: align.Left,
			}))
		})
	})

	// Total row
	m.AddRow(8, func(row config.Row) {
		row.Col(9.5, func(col config.Col) {
			col.Add(text.New("", props.TextProp{
				Top:   2,
				Size:  9,
				Align: align.Right,
			}))
		})
		row.Col(2.5, func(col config.Col) {
			col.Add(text.New(fmt.Sprintf("TOTALE: € %.2f", f.Totale), props.TextProp{
				Top:   2,
				Size:  10,
				Style: fontstyle.Bold,
				Align: align.Right,
			}))
		})
	})

	// Footer spacer
	m.AddRow(10, func(row config.Row) {
		row.Col(12, func(col config.Col) {
			col.Add(text.New("", props.TextProp{
				Top:   2,
				Size:  8,
				Align: align.Center,
			}))
		})
	})

	// Footer
	m.AddRow(5, func(row config.Row) {
		row.Col(12, func(col config.Col) {
			col.Add(text.New("Scuola di Tango TangoBar", props.TextProp{
				Top:   2,
				Size:  8,
				Align: align.Center,
			}))
		})
	})

	doc, err := m.Generate()
	if err != nil {
		return nil, err
	}

	return doc.GetBytes(), nil
}
