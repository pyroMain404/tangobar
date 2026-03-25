package handlers

import (
	"database/sql"
)

// Handler struct contains the database connection
type Handler struct {
	DB *sql.DB
}

// NewHandler creates a new Handler with the provided database connection
func NewHandler(db *sql.DB) *Handler {
	return &Handler{
		DB: db,
	}
}
