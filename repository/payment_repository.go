package repository

import (
	"database/sql"
	"github.com/fadhlanhapp/sharetab-backend/models"
)

// PaymentRepository handles payment data operations
type PaymentRepository struct {
	db *sql.DB
}

// NewPaymentRepository creates a new payment repository
func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// CreatePayment creates a new payment record
func (r *PaymentRepository) CreatePayment(payment *models.Payment) error {
	query := `
		INSERT INTO payments (trip_id, from_person, to_person, amount, description, payment_date)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	err := r.db.QueryRow(query, payment.TripID, payment.FromPerson, payment.ToPerson, 
		payment.Amount, payment.Description, payment.PaymentDate).Scan(&payment.ID)
	if err != nil {
		return err
	}

	return nil
}

// GetPaymentsByTripID retrieves all payments for a specific trip
func (r *PaymentRepository) GetPaymentsByTripID(tripID string) ([]models.Payment, error) {
	query := `
		SELECT id, trip_id, from_person, to_person, amount, description, payment_date, created_at
		FROM payments
		WHERE trip_id = $1
		ORDER BY payment_date DESC
	`
	rows, err := r.db.Query(query, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []models.Payment
	for rows.Next() {
		var payment models.Payment
		err := rows.Scan(&payment.ID, &payment.TripID, &payment.FromPerson, &payment.ToPerson,
			&payment.Amount, &payment.Description, &payment.PaymentDate, &payment.CreatedAt)
		if err != nil {
			return nil, err
		}
		payments = append(payments, payment)
	}

	return payments, nil
}

// DeletePayment deletes a payment by ID
func (r *PaymentRepository) DeletePayment(paymentID int) error {
	query := `DELETE FROM payments WHERE id = $1`
	_, err := r.db.Exec(query, paymentID)
	return err
}

// GetPaymentByID retrieves a payment by its ID
func (r *PaymentRepository) GetPaymentByID(paymentID int) (*models.Payment, error) {
	query := `
		SELECT id, trip_id, from_person, to_person, amount, description, payment_date, created_at
		FROM payments
		WHERE id = $1
	`
	var payment models.Payment
	err := r.db.QueryRow(query, paymentID).Scan(
		&payment.ID, &payment.TripID, &payment.FromPerson, &payment.ToPerson,
		&payment.Amount, &payment.Description, &payment.PaymentDate, &payment.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &payment, nil
}