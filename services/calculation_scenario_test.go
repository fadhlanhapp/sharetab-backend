package services

import (
	"testing"

	"github.com/fadhlanhapp/sharetab-backend/models"
	"github.com/stretchr/testify/assert"
)

func TestCalculationService_UserProvidedScenario(t *testing.T) {
	service := NewCalculationService()

	// Test case based on user's exact scenario:
	// Kwetiaw (36k): joj
	// Baso (18k): everyone  
	// Kriuk (5k): everyone
	// Es teh tawar (15k): Del, Ji, Dik
	// Extra topping (5k): everyone
	// Bakmi ayam (108k): tash, ji, dik
	// Pangsit (20k): everyone
	// Bakmi lebar (36k): del
	request := &models.CalculateSingleBillRequest{
		Items: []models.Item{
			{
				Description:  "Kwetiaw",
				UnitPrice:    36000,
				Quantity:     1,
				ItemDiscount: 0,
				PaidBy:       "joj",
				Consumers:    []string{"joj"},
			},
			{
				Description:  "Baso",
				UnitPrice:    18000,
				Quantity:     1,
				ItemDiscount: 0,
				PaidBy:       "del",
				Consumers:    []string{"del", "joj", "tash", "ji", "dik"},
			},
			{
				Description:  "Kriuk",
				UnitPrice:    5000,
				Quantity:     1,
				ItemDiscount: 0,
				PaidBy:       "del",
				Consumers:    []string{"del", "joj", "tash", "ji", "dik"},
			},
			{
				Description:  "Es teh tawar",
				UnitPrice:    15000,
				Quantity:     1,
				ItemDiscount: 0,
				PaidBy:       "del",
				Consumers:    []string{"del", "ji", "dik"},
			},
			{
				Description:  "Extra topping",
				UnitPrice:    5000,
				Quantity:     1,
				ItemDiscount: 0,
				PaidBy:       "del",
				Consumers:    []string{"del", "joj", "tash", "ji", "dik"},
			},
			{
				Description:  "Bakmi ayam",
				UnitPrice:    108000,
				Quantity:     1,
				ItemDiscount: 0,
				PaidBy:       "tash",
				Consumers:    []string{"tash", "ji", "dik"},
			},
			{
				Description:  "Pangsit",
				UnitPrice:    20000,
				Quantity:     1,
				ItemDiscount: 0,
				PaidBy:       "del",
				Consumers:    []string{"del", "joj", "tash", "ji", "dik"},
			},
			{
				Description:  "Bakmi lebar",
				UnitPrice:    36000,
				Quantity:     1,
				ItemDiscount: 0,
				PaidBy:       "del",
				Consumers:    []string{"del"},
			},
		},
		Tax:           0,
		ServiceCharge: 0,
		TotalDiscount: 0,
	}

	result, err := service.CalculateSingleBill(request)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Manual calculation:
	// Del: Baso(18k/5=3.6k) + Kriuk(5k/5=1k) + Es teh(15k/3=5k) + Extra topping(5k/5=1k) + Pangsit(20k/5=4k) + Bakmi lebar(36k) = 3.6 + 1 + 5 + 1 + 4 + 36 = 50.6k
	// Joj: Kwetiaw(36k) + Baso(18k/5=3.6k) + Kriuk(5k/5=1k) + Extra topping(5k/5=1k) + Pangsit(20k/5=4k) = 36 + 3.6 + 1 + 1 + 4 = 45.6k
	// Tash: Baso(18k/5=3.6k) + Kriuk(5k/5=1k) + Extra topping(5k/5=1k) + Bakmi ayam(108k/3=36k) + Pangsit(20k/5=4k) = 3.6 + 1 + 1 + 36 + 4 = 45.6k
	// Ji: Baso(18k/5=3.6k) + Kriuk(5k/5=1k) + Es teh(15k/3=5k) + Extra topping(5k/5=1k) + Bakmi ayam(108k/3=36k) + Pangsit(20k/5=4k) = 3.6 + 1 + 5 + 1 + 36 + 4 = 50.6k
	// Dik: Baso(18k/5=3.6k) + Kriuk(5k/5=1k) + Es teh(15k/3=5k) + Extra topping(5k/5=1k) + Bakmi ayam(108k/3=36k) + Pangsit(20k/5=4k) = 3.6 + 1 + 5 + 1 + 36 + 4 = 50.6k

	expectedSubtotals := map[string]float64{
		"Del":  50600,
		"Joj":  45600,
		"Tash": 45600,
		"Ji":   50600,
		"Dik":  50600,
	}

	// Total should be sum of all items: 36k + 18k + 5k + 15k + 5k + 108k + 20k + 36k = 243k
	expectedTotal := float64(243000)

	assert.Equal(t, expectedTotal, result.Subtotal, "Total subtotal should be 243k")

	// Check individual subtotals
	for person, expectedSubtotal := range expectedSubtotals {
		actualBreakdown, exists := result.PerPersonBreakdown[person]
		assert.True(t, exists, "Person %s should exist in breakdown", person)
		assert.Equal(t, expectedSubtotal, actualBreakdown.Subtotal, "Subtotal for %s should be %.0f", person, expectedSubtotal)
		
		// Since no tax/service/discount, total should equal subtotal
		assert.Equal(t, expectedSubtotal, actualBreakdown.Total, "Total for %s should equal subtotal when no extras", person)
	}

	// Verify the sum of individual subtotals equals the total subtotal
	var sumOfIndividualSubtotals float64
	for _, breakdown := range result.PerPersonBreakdown {
		sumOfIndividualSubtotals += breakdown.Subtotal
	}
	assert.Equal(t, expectedTotal, sumOfIndividualSubtotals, "Sum of individual subtotals should equal total subtotal")

	// Print actual results for verification
	t.Logf("Results:")
	t.Logf("Total subtotal: %.0f (expected: %.0f)", result.Subtotal, expectedTotal)
	for person, breakdown := range result.PerPersonBreakdown {
		expected := expectedSubtotals[person]
		t.Logf("%s: %.0f (expected: %.0f)", person, breakdown.Subtotal, expected)
	}
}