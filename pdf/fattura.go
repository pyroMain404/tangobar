package pdf

import (
	"fmt"
	"os"
	"strconv"
	"tango-gestionale/models"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// GeneraFattura generates an invoice PDF from a Fattura model
func GeneraFattura(f models.Fattura) ([]byte, error) {
	cfg := config.NewBuilder().
		WithLeftMargin(15).
		WithRightMargin(15).
		WithTopMargin(15).
		WithBottomMargin(15).
		Build()

	m := maroto.New(cfg)

	// Header row: logo left, invoice number right
	var logoCol core.Col
	logoPath := "assets/logo.png"
	if _, err := os.Stat(logoPath); err == nil {
		logoCol = col.New(6).Add(image.NewFromFile(logoPath, props.Rect{
			Width: 20, Height: 15,
		}))
	} else {
		logoCol = col.New(6)
	}

	m.AddRows(
		row.New(15).Add(
			logoCol,
			col.New(6).Add(text.New(fmt.Sprintf("FATTURA %s", f.Numero), props.Text{
				Top: 5, Size: 14, Align: align.Right, Style: fontstyle.Bold,
			})),
		),
		row.New(10).Add(
			col.New(12).Add(text.New(fmt.Sprintf("Cliente: %s", f.NomeCliente), props.Text{
				Top: 2, Size: 10, Align: align.Left,
			})),
		),
		row.New(8).Add(
			col.New(12).Add(text.New(fmt.Sprintf("Data: %s", f.DataEmissione.Format("02/01/2006")), props.Text{
				Top: 2, Size: 10, Align: align.Left,
			})),
		),
		// Table header
		row.New(8).Add(
			col.New(5).Add(text.New("Descrizione", props.Text{Top: 2, Size: 9, Style: fontstyle.Bold, Align: align.Left})),
			col.New(2).Add(text.New("Qtà", props.Text{Top: 2, Size: 9, Style: fontstyle.Bold, Align: align.Center})),
			col.New(2).Add(text.New("Prezzo Unit.", props.Text{Top: 2, Size: 9, Style: fontstyle.Bold, Align: align.Right})),
			col.New(3).Add(text.New("Totale", props.Text{Top: 2, Size: 9, Style: fontstyle.Bold, Align: align.Right})),
		),
	)

	// Table rows for righe
	for _, riga := range f.Righe {
		r := riga
		m.AddRows(
			row.New(7).Add(
				col.New(5).Add(text.New(r.Descrizione, props.Text{Top: 2, Size: 9, Align: align.Left})),
				col.New(2).Add(text.New(strconv.FormatFloat(r.Quantita, 'f', 2, 64), props.Text{Top: 2, Size: 9, Align: align.Center})),
				col.New(2).Add(text.New(fmt.Sprintf("€ %.2f", r.PrezzoUnit), props.Text{Top: 2, Size: 9, Align: align.Right})),
				col.New(3).Add(text.New(fmt.Sprintf("€ %.2f", r.Totale), props.Text{Top: 2, Size: 9, Align: align.Right})),
			),
		)
	}

	m.AddRows(
		row.New(8).Add(
			col.New(9).Add(text.New("", props.Text{Top: 2, Size: 9, Align: align.Right})),
			col.New(3).Add(text.New(fmt.Sprintf("TOTALE: € %.2f", f.Totale), props.Text{
				Top: 2, Size: 10, Style: fontstyle.Bold, Align: align.Right,
			})),
		),
		row.New(5).Add(
			col.New(12).Add(text.New("Scuola di Tango TangoBar", props.Text{
				Top: 2, Size: 8, Align: align.Center,
			})),
		),
	)

	doc, err := m.Generate()
	if err != nil {
		return nil, err
	}

	return doc.GetBytes(), nil
}
