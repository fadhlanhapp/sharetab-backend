// services/trip_service.go (updated for database)
package services

import (
	"math/rand"
	"time"

	"github.com/fadhlanhapp/sharetab-backend/models"
	"github.com/fadhlanhapp/sharetab-backend/repository"
)

var tripRepo *repository.TripRepository

// InitTripService initializes the trip service
func InitTripService() {
	tripRepo = repository.NewTripRepository()
}

// GenerateID generates a random ID
func GenerateID() string {
	rand.Seed(time.Now().UnixNano())
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	const length = 20

	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

// GenerateCode generates a random trip code
func GenerateCode() string {
	rand.Seed(time.Now().UnixNano())
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 6

	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

// GetTripByCode retrieves a trip by its code
func GetTripByCode(code string) (*models.Trip, error) {
	return tripRepo.GetTripByCode(code)
}

// StoreTrip stores a trip
func StoreTrip(trip *models.Trip) error {
	return tripRepo.StoreTrip(trip)
}

// AddParticipant adds a participant to a trip if they don't exist already
func AddParticipant(tripID string, participant string) error {
	return tripRepo.AddParticipant(tripID, participant)
}
