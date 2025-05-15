// repository/trip_repository.go
package repository

import (
	"database/sql"
	"fmt"

	"github.com/fadhlanhapp/sharetab-backend/models"
)

// TripRepository handles database operations for trips
type TripRepository struct {
	DB *sql.DB
}

// NewTripRepository creates a new TripRepository
func NewTripRepository() *TripRepository {
	return &TripRepository{
		DB: GetDB(),
	}
}

// StoreTrip saves a trip to the database
func (r *TripRepository) StoreTrip(trip *models.Trip) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Insert trip
	_, err = tx.Exec(
		"INSERT INTO trips (id, code, name, creation_time) VALUES ($1, $2, $3, $4)",
		trip.ID, trip.Code, trip.Name, trip.CreationTime,
	)
	if err != nil {
		return fmt.Errorf("failed to insert trip: %v", err)
	}

	// Insert participants
	for _, participant := range trip.Participants {
		_, err = tx.Exec(
			"INSERT INTO trip_participants (trip_id, participant) VALUES ($1, $2)",
			trip.ID, participant,
		)
		if err != nil {
			return fmt.Errorf("failed to insert trip participant: %v", err)
		}
	}

	return tx.Commit()
}

// GetTripByCode retrieves a trip by its code
func (r *TripRepository) GetTripByCode(code string) (*models.Trip, error) {
	// Query trip
	var trip models.Trip
	err := r.DB.QueryRow(
		"SELECT id, code, name, creation_time FROM trips WHERE code = $1",
		code,
	).Scan(&trip.ID, &trip.Code, &trip.Name, &trip.CreationTime)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("trip not found")
		}
		return nil, fmt.Errorf("failed to get trip: %v", err)
	}

	// Query participants
	rows, err := r.DB.Query(
		"SELECT participant FROM trip_participants WHERE trip_id = $1",
		trip.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get trip participants: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var participant string
		if err := rows.Scan(&participant); err != nil {
			return nil, fmt.Errorf("failed to scan participant: %v", err)
		}
		trip.Participants = append(trip.Participants, participant)
	}

	return &trip, nil
}

// AddParticipant adds a participant to a trip
func (r *TripRepository) AddParticipant(tripID string, participant string) error {
	// Check if participant already exists
	var count int
	err := r.DB.QueryRow(
		"SELECT COUNT(*) FROM trip_participants WHERE trip_id = $1 AND participant = $2",
		tripID, participant,
	).Scan(&count)

	if err != nil {
		return fmt.Errorf("failed to check participant: %v", err)
	}

	if count > 0 {
		// Participant already exists
		return nil
	}

	// Add participant
	_, err = r.DB.Exec(
		"INSERT INTO trip_participants (trip_id, participant) VALUES ($1, $2)",
		tripID, participant,
	)
	if err != nil {
		return fmt.Errorf("failed to insert participant: %v", err)
	}

	return nil
}
