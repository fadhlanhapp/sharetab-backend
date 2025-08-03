package services

import (
	"testing"

	"github.com/fadhlanhapp/sharetab-backend/models"
	"github.com/stretchr/testify/assert"
)

func TestCalculationService_CalculateSingleBill_RealWorldScenario(t *testing.T) {
	service := NewCalculationService()

	// Test case based on the actual issue from screenshots
	request := &models.CalculateSingleBillRequest{
		Items: []models.Item{
			{
				Description:  "Bakmi Ayam Kampung",
				UnitPrice:    36000,
				Quantity:     1,
				ItemDiscount: 0,
				PaidBy:       "del",
				Consumers:    []string{"del", "joj", "tash", "ji", "dik"},
			},
			{
				Description:  "Pangsit Goreng (5 Pcs)",
				UnitPrice:    108000,
				Quantity:     1,
				ItemDiscount: 0,
				PaidBy:       "tash",
				Consumers:    []string{"tash", "ji", "dik"},
			},
			{
				Description:  "Bakmi Lebar Ayam Kampung",
				UnitPrice:    20000,
				Quantity:     1,
				ItemDiscount: 0,
				PaidBy:       "del",
				Consumers:    []string{"del", "joj", "tash", "ji", "dik"},
			},
			{
				Description:  "Kwetiaw Ayam Kampung",
				UnitPrice:    36000,
				Quantity:     1,
				ItemDiscount: 0,
				PaidBy:       "del",
				Consumers:    []string{"del"},
			},
			{
				Description:  "Bakso Kuah Isi 5",
				UnitPrice:    18000,
				Quantity:     1,
				ItemDiscount: 0,
				PaidBy:       "del",
				Consumers:    []string{"del", "joj", "tash", "ji", "dik"},
			},
			{
				Description:  "Kriuk (3x)",
				UnitPrice:    5000,
				Quantity:     1,
				ItemDiscount: 0,
				PaidBy:       "del",
				Consumers:    []string{"del", "joj", "tash", "ji", "dik"},
			},
			{
				Description:  "Es Teh Tawar",
				UnitPrice:    15000,
				Quantity:     1,
				ItemDiscount: 0,
				PaidBy:       "del",
				Consumers:    []string{"del", "ji", "dik"},
			},
			{
				Description:  "Extra Topping Ayam Kecap",
				UnitPrice:    5000,
				Quantity:     1,
				ItemDiscount: 0,
				PaidBy:       "del",
				Consumers:    []string{"del", "joj", "tash", "ji", "dik"},
			},
		},
		Tax:           0,
		ServiceCharge: 0,
		TotalDiscount: 0,
	}

	result, err := service.CalculateSingleBill(request)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Expected subtotals:
	// Del: 36000/5 + 20000/5 + 36000 + 18000/5 + 5000/5 + 15000/3 + 5000/5 = 7200 + 4000 + 36000 + 3600 + 1000 + 5000 + 1000 = 57800
	// Joj: 36000/5 + 20000/5 + 18000/5 + 5000/5 + 5000/5 = 7200 + 4000 + 3600 + 1000 + 1000 = 16800
	// Tash: 36000/5 + 108000/3 + 20000/5 + 18000/5 + 5000/5 + 5000/5 = 7200 + 36000 + 4000 + 3600 + 1000 + 1000 = 52800
	// Ji: 36000/5 + 108000/3 + 20000/5 + 18000/5 + 5000/5 + 15000/3 + 5000/5 = 7200 + 36000 + 4000 + 3600 + 1000 + 5000 + 1000 = 57800
	// Dik: 36000/5 + 108000/3 + 20000/5 + 18000/5 + 5000/5 + 15000/3 + 5000/5 = 7200 + 36000 + 4000 + 3600 + 1000 + 5000 + 1000 = 57800

	expectedSubtotals := map[string]float64{
		"Del":  57800,
		"Joj":  16800,
		"Tash": 52800,
		"Ji":   57800,
		"Dik":  57800,
	}

	// Total should be sum of all items
	expectedTotal := float64(36000 + 108000 + 20000 + 36000 + 18000 + 5000 + 15000 + 5000) // 243000

	assert.Equal(t, expectedTotal, result.Subtotal, "Total subtotal should match sum of all items")

	// Check individual subtotals
	for person, expectedSubtotal := range expectedSubtotals {
		actualBreakdown, exists := result.PerPersonBreakdown[person]
		assert.True(t, exists, "Person %s should exist in breakdown", person)
		assert.Equal(t, expectedSubtotal, actualBreakdown.Subtotal, "Subtotal for %s should be correct", person)
		
		// Since no tax/service/discount, total should equal subtotal
		assert.Equal(t, expectedSubtotal, actualBreakdown.Total, "Total for %s should equal subtotal when no extras", person)
	}

	// Verify the sum of individual subtotals equals the total subtotal
	var sumOfIndividualSubtotals float64
	for _, breakdown := range result.PerPersonBreakdown {
		sumOfIndividualSubtotals += breakdown.Subtotal
	}
	assert.Equal(t, expectedTotal, sumOfIndividualSubtotals, "Sum of individual subtotals should equal total subtotal")
}

func TestCalculationService_CalculateSingleBill_SimpleSharedItem(t *testing.T) {
	service := NewCalculationService()

	// Simple test case: one item shared by two people
	request := &models.CalculateSingleBillRequest{
		Items: []models.Item{
			{
				Description:  "Pizza",
				UnitPrice:    100,
				Quantity:     1,
				ItemDiscount: 0,
				PaidBy:       "alice",
				Consumers:    []string{"alice", "bob"},
			},
		},
		Tax:           0,
		ServiceCharge: 0,
		TotalDiscount: 0,
	}

	result, err := service.CalculateSingleBill(request)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, float64(100), result.Subtotal)

	// Each person should pay 50
	for _, person := range []string{"Alice", "Bob"} {
		breakdown, exists := result.PerPersonBreakdown[person]
		assert.True(t, exists, "Person %s should exist", person)
		assert.Equal(t, float64(50), breakdown.Subtotal)
		assert.Equal(t, float64(50), breakdown.Total)
	}
}

func TestCalculationService_CalculateSingleBill_WithTaxAndService(t *testing.T) {
	service := NewCalculationService()

	request := &models.CalculateSingleBillRequest{
		Items: []models.Item{
			{
				Description:  "Meal",
				UnitPrice:    100,
				Quantity:     1,
				ItemDiscount: 0,
				PaidBy:       "alice",
				Consumers:    []string{"alice", "bob"},
			},
		},
		Tax:           10,  // 10% of 100 = 10
		ServiceCharge: 5,   // 5% of 100 = 5  
		TotalDiscount: 0,
	}

	result, err := service.CalculateSingleBill(request)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, float64(100), result.Subtotal)
	assert.Equal(t, float64(115), result.Amount) // 100 + 10 + 5

	// Each person pays 50 for subtotal, plus proportional tax/service
	// Alice: 50 subtotal + 5 tax + 2.5 service = 57.5
	// Bob: 50 subtotal + 5 tax + 2.5 service = 57.5
	for _, person := range []string{"Alice", "Bob"} {
		breakdown, exists := result.PerPersonBreakdown[person]
		assert.True(t, exists, "Person %s should exist", person)
		assert.Equal(t, float64(50), breakdown.Subtotal)
		assert.Equal(t, float64(5), breakdown.Tax)
		assert.Equal(t, float64(2.5), breakdown.ServiceCharge)
		assert.Equal(t, float64(57.5), breakdown.Total)
	}
}