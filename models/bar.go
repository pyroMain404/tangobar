package models

import "time"

type BarItem struct {
	ID        int64
	Nome      string
	Categoria string
	Quantita  int
	SogliaMin int
	Prezzo    float64
}

type BarMovimento struct {
	ID        int64
	ItemID    int64
	Delta     int
	Nota      string
	Timestamp time.Time
	NomeItem  string
}
