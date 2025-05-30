package models

import (
	"time"
)

// Payment represents a payment made between people in a trip
type Payment struct {
	ID          int       `json:"id" db:"id"`
	TripID      string    `json:"trip_id" db:"trip_id"`          // VARCHAR(36) to match trips.id
	FromPerson  string    `json:"from_person" db:"from_person"`   // VARCHAR(255) to match schema
	ToPerson    string    `json:"to_person" db:"to_person"`       // VARCHAR(255) to match schema  
	Amount      float64   `json:"amount" db:"amount"`             // DECIMAL(10,2) as float64
	Description string    `json:"description" db:"description"`   // TEXT field
	PaymentDate time.Time `json:"payment_date" db:"payment_date"` // TIMESTAMP
	CreatedAt   time.Time `json:"created_at" db:"created_at"`     // TIMESTAMP
}

// PaymentRequest represents the request body for creating a payment
type PaymentRequest struct {
	Code        string  `json:"code" binding:"required"`
	FromPerson  string  `json:"from_person" binding:"required"`
	ToPerson    string  `json:"to_person" binding:"required"`
	Amount      float64 `json:"amount" binding:"required,gt=0"`
	Description string  `json:"description"`
}