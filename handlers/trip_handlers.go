// handlers/trip_handlers.go
package handlers

import (
	"net/http"

	"github.com/fadhlanhapp/sharetab-backend/models"
	"github.com/fadhlanhapp/sharetab-backend/services"

	"github.com/gin-gonic/gin"
)

// CreateTrip handles the creation of a new trip
func CreateTrip(c *gin.Context) {
	var request models.CreateTripRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Generate ID and code
	tripID := services.GenerateID()
	code := services.GenerateCode()

	// Create trip with normalized participant name
	normalizedParticipant := services.NormalizeName(request.Participant)
	trip := models.NewTrip(tripID, code, request.Name, normalizedParticipant)

	// Store trip
	services.StoreTrip(trip)

	// Return response
	response := models.CreateTripResponse{
		TripID: tripID,
		Code:   code,
	}

	c.JSON(http.StatusOK, response)
}

// GetTripByCodeHandler handles retrieving a trip by its code
func GetTripByCodeHandler(c *gin.Context) {
	var request models.GetTripByCodeRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if request.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Code is required"})
		return
	}

	trip, err := services.GetTripByCode(request.Code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Format participant names for display
	formattedTrip := *trip
	formattedParticipants := make([]string, len(trip.Participants))
	for i, participant := range trip.Participants {
		formattedParticipants[i] = services.FormatNameForDisplay(participant)
	}
	formattedTrip.Participants = formattedParticipants

	c.JSON(http.StatusOK, &formattedTrip)
}
