package utils

import (
	"fmt"
	"strings"
)

// ValidateRequired checks if a string field is not empty
func ValidateRequired(value, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return NewValidationError(fmt.Sprintf("%s is required", fieldName))
	}
	return nil
}

// ValidatePositive checks if a number is positive
func ValidatePositive(value float64, fieldName string) error {
	if value <= 0 {
		return NewValidationError(fmt.Sprintf("%s must be positive", fieldName))
	}
	return nil
}

// ValidateNonNegative checks if a number is non-negative
func ValidateNonNegative(value float64, fieldName string) error {
	if value < 0 {
		return NewValidationError(fmt.Sprintf("%s cannot be negative", fieldName))
	}
	return nil
}

// ValidateNotEmpty checks if a slice is not empty
func ValidateNotEmpty[T any](slice []T, fieldName string) error {
	if len(slice) == 0 {
		return NewValidationError(fmt.Sprintf("%s cannot be empty", fieldName))
	}
	return nil
}

// ValidateItemData validates basic item data
func ValidateItemData(unitPrice float64, quantity int, description string) error {
	if err := ValidateRequired(description, "item description"); err != nil {
		return err
	}
	if err := ValidatePositive(unitPrice, "item price"); err != nil {
		return err
	}
	if quantity <= 0 {
		return NewValidationError("item quantity must be positive")
	}
	return nil
}

// ValidateParticipantNames validates that all participant names are not empty
func ValidateParticipantNames(participants []string) error {
	for i, participant := range participants {
		if strings.TrimSpace(participant) == "" {
			return NewValidationError(fmt.Sprintf("participant %d name cannot be empty", i+1))
		}
	}
	return nil
}