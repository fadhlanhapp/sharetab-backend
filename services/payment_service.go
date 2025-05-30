package services

import (
	"errors"
	"github.com/fadhlanhapp/sharetab-backend/models"
	"github.com/fadhlanhapp/sharetab-backend/repository"
	"strings"
	"time"
)

// PaymentService handles payment business logic
type PaymentService struct {
	paymentRepo *repository.PaymentRepository
	tripRepo    *repository.TripRepository
}

// NewPaymentService creates a new payment service
func NewPaymentService(paymentRepo *repository.PaymentRepository, tripRepo *repository.TripRepository) *PaymentService {
	return &PaymentService{
		paymentRepo: paymentRepo,
		tripRepo:    tripRepo,
	}
}

// CreatePayment creates a new payment record
func (s *PaymentService) CreatePayment(req *models.PaymentRequest) (*models.Payment, error) {
	// Validate input
	if strings.TrimSpace(req.FromPerson) == "" {
		return nil, errors.New("from_person is required")
	}
	if strings.TrimSpace(req.ToPerson) == "" {
		return nil, errors.New("to_person is required")
	}
	if req.FromPerson == req.ToPerson {
		return nil, errors.New("cannot pay to yourself")
	}
	if req.Amount <= 0 {
		return nil, errors.New("amount must be greater than 0")
	}

	// Get trip by code
	trip, err := s.tripRepo.GetTripByCode(req.Code)
	if err != nil {
		return nil, errors.New("trip not found")
	}

	// Create payment
	payment := &models.Payment{
		TripID:      trip.ID, // trip.ID is string, payment.TripID is now string
		FromPerson:  strings.TrimSpace(req.FromPerson),
		ToPerson:    strings.TrimSpace(req.ToPerson),
		Amount:      req.Amount,
		Description: strings.TrimSpace(req.Description),
		PaymentDate: time.Now(),
		CreatedAt:   time.Now(),
	}

	err = s.paymentRepo.CreatePayment(payment)
	if err != nil {
		return nil, err
	}

	return payment, nil
}

// GetPaymentsByTripCode retrieves all payments for a trip by code
func (s *PaymentService) GetPaymentsByTripCode(code string) ([]models.Payment, error) {
	// Get trip by code
	trip, err := s.tripRepo.GetTripByCode(code)
	if err != nil {
		return nil, errors.New("trip not found")
	}

	return s.paymentRepo.GetPaymentsByTripID(trip.ID)
}

// DeletePayment deletes a payment by ID
func (s *PaymentService) DeletePayment(paymentID int) error {
	// Check if payment exists
	_, err := s.paymentRepo.GetPaymentByID(paymentID)
	if err != nil {
		return errors.New("payment not found")
	}

	return s.paymentRepo.DeletePayment(paymentID)
}

// CalculateBalancesWithPayments calculates balances including payments
func (s *PaymentService) CalculateBalancesWithPayments(tripCode string, originalBalances map[string]float64) (map[string]float64, error) {
	// Get all payments for this trip
	payments, err := s.GetPaymentsByTripCode(tripCode)
	if err != nil {
		return originalBalances, err
	}

	// Create a copy of original balances
	adjustedBalances := make(map[string]float64)
	for person, balance := range originalBalances {
		adjustedBalances[person] = balance
	}

	// Apply payments to balances
	for _, payment := range payments {
		// The person who paid reduces their debt (becomes less negative or more positive)
		adjustedBalances[payment.FromPerson] += payment.Amount
		// The person who received payment increases their debt (becomes more negative or less positive)
		adjustedBalances[payment.ToPerson] -= payment.Amount
	}

	return adjustedBalances, nil
}