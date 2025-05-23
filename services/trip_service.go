package services

import (
	"github.com/fadhlanhapp/sharetab-backend/models"
	"github.com/fadhlanhapp/sharetab-backend/repository"
	"github.com/fadhlanhapp/sharetab-backend/utils"
)

var tripRepo *repository.TripRepository

// InitTripService initializes the trip service
func InitTripService() {
	tripRepo = repository.NewTripRepository()
}

// TripService handles trip-related business logic
type TripService struct {
	repo *repository.TripRepository
}

// NewTripService creates a new trip service instance
func NewTripService() *TripService {
	return &TripService{
		repo: repository.NewTripRepository(),
	}
}

// CreateTrip creates a new trip with validation
func (s *TripService) CreateTrip(name, participant string) (*models.Trip, error) {
	if err := utils.ValidateRequired(name, "trip name"); err != nil {
		return nil, err
	}
	if err := utils.ValidateRequired(participant, "participant name"); err != nil {
		return nil, err
	}

	tripID := utils.GenerateID()
	code := utils.GenerateCode()
	normalizedParticipant := utils.NormalizeName(participant)

	trip := models.NewTrip(tripID, code, name, normalizedParticipant)
	if err := s.repo.StoreTrip(trip); err != nil {
		return nil, utils.NewInternalError("Failed to create trip")
	}

	return trip, nil
}

// GetTripByCode retrieves a trip by its code with formatted participant names
func (s *TripService) GetTripByCode(code string) (*models.Trip, error) {
	if err := utils.ValidateRequired(code, "trip code"); err != nil {
		return nil, err
	}

	trip, err := s.repo.GetTripByCode(code)
	if err != nil {
		return nil, utils.NewNotFoundError("Trip")
	}

	// Format participant names for display
	trip.Participants = utils.FormatNamesForDisplay(trip.Participants)
	return trip, nil
}

// AddParticipant adds a participant to a trip if they don't exist already
func (s *TripService) AddParticipant(tripID, participant string) error {
	if err := utils.ValidateRequired(participant, "participant name"); err != nil {
		return err
	}

	normalizedName := utils.NormalizeName(participant)
	if err := s.repo.AddParticipant(tripID, normalizedName); err != nil {
		return utils.NewInternalError("Failed to add participant")
	}
	return nil
}

// Legacy functions for backward compatibility
func GetTripByCode(code string) (*models.Trip, error) {
	return tripRepo.GetTripByCode(code)
}

func StoreTrip(trip *models.Trip) error {
	return tripRepo.StoreTrip(trip)
}

func AddParticipant(tripID string, participant string) error {
	normalizedName := utils.NormalizeName(participant)
	return tripRepo.AddParticipant(tripID, normalizedName)
}
