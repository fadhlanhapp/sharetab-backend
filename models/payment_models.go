package models

import (
	"time"
)

// Payment represents a payment made between people in a trip
type Payment struct {
	ID          int       `json:"id" db:"id"`
	TripID      string    `json:"trip_id" db:"trip_id"`
	FromPerson  string    `json:"from_person" db:"from_person"`
	ToPerson    string    `json:"to_person" db:"to_person"`
	Amount      float64   `json:"amount" db:"amount"`
	Description string    `json:"description" db:"description"`
	PaymentDate time.Time `json:"payment_date" db:"payment_date"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// PaymentRequest represents the request body for creating a payment
type PaymentRequest struct {
	Code        string  `json:"code" binding:"required"`
	FromPerson  string  `json:"from_person" binding:"required"`
	ToPerson    string  `json:"to_person" binding:"required"`
	Amount      float64 `json:"amount" binding:"required,gt=0"`
	Description string  `json:"description"`
}