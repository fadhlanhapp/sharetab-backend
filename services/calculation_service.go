package services

import (
	"fmt"

	"github.com/fadhlanhapp/sharetab-backend/models"
	"github.com/fadhlanhapp/sharetab-backend/utils"
)

// CalculationService handles bill calculation logic
type CalculationService struct{}

// NewCalculationService creates a new calculation service
func NewCalculationService() *CalculationService {
	return &CalculationService{}
}

// CalculateSingleBill calculates how much each person owes for a bill
func (s *CalculationService) CalculateSingleBill(request *models.CalculateSingleBillRequest) (*models.SingleBillCalculation, error) {
	// Validate request
	if err := s.validateCalculationRequest(request); err != nil {
		return nil, err
	}

	// Normalize names in items
	normalizedItems := s.normalizeItemNames(request.Items)
	
	// Extract participants
	participants := s.extractParticipants(normalizedItems)

	// Calculate personal charges
	perPersonCharges, perPersonBreakdown := s.calculatePersonalCharges(
		normalizedItems,
		request.Tax,
		request.ServiceCharge,
		request.TotalDiscount,
		participants,
	)

	// Calculate totals
	subtotal := s.calculateSubtotal(normalizedItems)
	total := subtotal + request.Tax + request.ServiceCharge - request.TotalDiscount

	// Format names for display
	formattedCharges := utils.FormatNameMapKeys(perPersonCharges)
	formattedBreakdown := utils.FormatNameMapKeys(perPersonBreakdown)

	return &models.SingleBillCalculation{
		Amount:             utils.Round(total),
		Subtotal:           utils.Round(subtotal),
		Tax:                utils.Round(request.Tax),
		ServiceCharge:      utils.Round(request.ServiceCharge),
		TotalDiscount:      utils.Round(request.TotalDiscount),
		PerPersonCharges:   formattedCharges,
		PerPersonBreakdown: formattedBreakdown,
	}, nil
}

// validateCalculationRequest validates the calculation request
func (s *CalculationService) validateCalculationRequest(request *models.CalculateSingleBillRequest) error {
	if err := utils.ValidateNotEmpty(request.Items, "items"); err != nil {
		return err
	}
	if err := utils.ValidateNonNegative(request.Tax, "tax"); err != nil {
		return err
	}
	if err := utils.ValidateNonNegative(request.ServiceCharge, "service charge"); err != nil {
		return err
	}
	if err := utils.ValidateNonNegative(request.TotalDiscount, "discount"); err != nil {
		return err
	}

	// Validate each item
	for i, item := range request.Items {
		if err := utils.ValidateItemData(item.UnitPrice, item.Quantity, item.Description); err != nil {
			return utils.NewValidationError(fmt.Sprintf("Item %d: %s", i+1, err.Error()))
		}
		if err := utils.ValidateRequired(item.PaidBy, "item paidBy"); err != nil {
			return utils.NewValidationError(fmt.Sprintf("Item %d: %s", i+1, err.Error()))
		}
		if err := utils.ValidateNotEmpty(item.Consumers, "item consumers"); err != nil {
			return utils.NewValidationError(fmt.Sprintf("Item %d: %s", i+1, err.Error()))
		}
		if err := utils.ValidateParticipantNames(item.Consumers); err != nil {
			return utils.NewValidationError(fmt.Sprintf("Item %d: %s", i+1, err.Error()))
		}
	}

	return nil
}

// normalizeItemNames normalizes all names in items
func (s *CalculationService) normalizeItemNames(items []models.Item) []models.Item {
	normalized := make([]models.Item, len(items))
	for i, item := range items {
		normalized[i] = item
		normalized[i].PaidBy = utils.NormalizeName(item.PaidBy)
		normalized[i].Consumers = utils.NormalizeNames(item.Consumers)
	}
	return normalized
}

// extractParticipants extracts all unique participants from items
func (s *CalculationService) extractParticipants(items []models.Item) []string {
	participants := make(map[string]bool)
	for _, item := range items {
		for _, consumer := range item.Consumers {
			participants[consumer] = true
		}
	}

	var allParticipants []string
	for participant := range participants {
		allParticipants = append(allParticipants, participant)
	}
	return allParticipants
}

// calculateSubtotal calculates the sum of all items
func (s *CalculationService) calculateSubtotal(items []models.Item) float64 {
	var subtotal float64
	for _, item := range items {
		itemAmount := item.UnitPrice*float64(item.Quantity) - item.ItemDiscount
		subtotal += itemAmount
	}
	return subtotal
}

// calculatePersonalCharges calculates how much each person owes
func (s *CalculationService) calculatePersonalCharges(
	items []models.Item,
	tax float64,
	serviceCharge float64,
	totalDiscount float64,
	participants []string,
) (map[string]float64, map[string]models.PersonChargeBreakdown) {
	
	charges := make(map[string]float64)
	breakdown := make(map[string]models.PersonChargeBreakdown)

	// Initialize each participant's breakdown
	for _, participant := range participants {
		charges[participant] = 0
		breakdown[participant] = models.PersonChargeBreakdown{
			Subtotal:      0,
			Tax:           0,
			ServiceCharge: 0,
			Discount:      0,
			Total:         0,
		}
	}

	// Calculate each person's share of items (subtotal)
	for _, item := range items {
		itemAmount := item.UnitPrice*float64(item.Quantity) - item.ItemDiscount
		itemAmount = utils.Round(itemAmount)

		if len(item.Consumers) > 0 {
			sharePerPerson := itemAmount / float64(len(item.Consumers))
			sharePerPerson = utils.Round(sharePerPerson)

			for _, consumer := range item.Consumers {
				breakdown[consumer] = models.PersonChargeBreakdown{
					Subtotal:      breakdown[consumer].Subtotal + sharePerPerson,
					Tax:           breakdown[consumer].Tax,
					ServiceCharge: breakdown[consumer].ServiceCharge,
					Discount:      breakdown[consumer].Discount,
					Total:         breakdown[consumer].Total + sharePerPerson,
				}
			}
		}
	}

	// Calculate total subtotal for proportion calculation
	var totalSubtotal float64
	for _, person := range participants {
		totalSubtotal += breakdown[person].Subtotal
	}

	// Calculate extras (tax, service charge, discount)
	if totalSubtotal > 0 && len(participants) > 0 {
		for _, person := range participants {
			proportion := breakdown[person].Subtotal / totalSubtotal
			personTax := tax * proportion
			personService := serviceCharge * proportion
			personDiscount := totalDiscount * proportion

			breakdown[person] = models.PersonChargeBreakdown{
				Subtotal:      breakdown[person].Subtotal,
				Tax:           utils.Round(personTax),
				ServiceCharge: utils.Round(personService),
				Discount:      utils.Round(personDiscount),
				Total:         utils.Round(breakdown[person].Subtotal + personTax + personService - personDiscount),
			}

			charges[person] = breakdown[person].Total
		}
	} else if totalSubtotal == 0 && len(participants) > 0 {
		// If subtotal is 0 but we have extras, divide them equally
		extraCharges := tax + serviceCharge - totalDiscount
		extraPerPerson := extraCharges / float64(len(participants))
		extraPerPerson = utils.Round(extraPerPerson)

		for _, person := range participants {
			personTax := tax / float64(len(participants))
			personService := serviceCharge / float64(len(participants))

			breakdown[person] = models.PersonChargeBreakdown{
				Subtotal:      0,
				Tax:           utils.Round(personTax),
				ServiceCharge: utils.Round(personService),
				Discount:      utils.Round(totalDiscount / float64(len(participants))),
				Total:         utils.Round(extraPerPerson),
			}

			charges[person] = extraPerPerson
		}
	}

	// Round all values in the breakdown
	for person := range breakdown {
		breakdown[person] = models.PersonChargeBreakdown{
			Subtotal:      utils.Round(breakdown[person].Subtotal),
			Tax:           utils.Round(breakdown[person].Tax),
			ServiceCharge: utils.Round(breakdown[person].ServiceCharge),
			Discount:      utils.Round(breakdown[person].Discount),
			Total:         utils.Round(breakdown[person].Total),
		}
	}

	return charges, breakdown
}