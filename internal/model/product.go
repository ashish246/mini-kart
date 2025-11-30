package model

import "time"

// Product represents a food product in the catalogue.
type Product struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Price     float64   `json:"price" db:"price"`
	Category  string    `json:"category" db:"category"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}
