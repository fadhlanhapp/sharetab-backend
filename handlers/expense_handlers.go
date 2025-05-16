// handlers/expense_handlers.go
package handlers

import (
	"fmt"
	"log"
	"math"
	"net/http"

	"github.com/fadhlanhapp/sharetab-backend/models"
	"github.com/fadhlanhapp/sharetab-backend/services"

	"github.com/gin-gonic/gin"
)

// Updated CalculateSingleBill handler with detailed breakdown
// Replace your current calculatePersonalCharges function with this

func calculatePersonalCharges(
	items []models.Item,
	tax float64,
	serviceCharge float64,
	totalDiscount float64,
	participants []string,
) (map[string]float64, map[string]models.PersonChargeBreakdown) {
	// Initialize charges map
	charges := make(map[string]float64)
	breakdown := make(map[string]models.PersonChargeBreakdown)

	// Initialize each participant's breakdown
	for _, participant := range participants {
		charges[participant] = 0
		breakdown[participant] = models.PersonChargeBreakdown{
			Subtotal:      0,
			Tax:           0,
			ServiceCharge: 0,
			Total:         0,
		}
	}

	// Calculate each person's share of items (subtotal)
	for _, item := range items {
		// Calculate item price
		itemAmount := item.UnitPrice*float64(item.Quantity) - item.ItemDiscount
		itemAmount = Round(itemAmount)

		// Divide equally among consumers
		if len(item.Consumers) > 0 {
			sharePerPerson := itemAmount / float64(len(item.Consumers))
			sharePerPerson = Round(sharePerPerson)

			for _, consumer := range item.Consumers {
				// Add to participant's subtotal
				breakdown[consumer] = models.PersonChargeBreakdown{
					Subtotal:      breakdown[consumer].Subtotal + sharePerPerson,
					Tax:           breakdown[consumer].Tax,
					ServiceCharge: breakdown[consumer].ServiceCharge,
					Total:         breakdown[consumer].Total + sharePerPerson,
				}
			}
		}

		// Calculate share per person for logging
		var sharePerPersonLog float64
		if len(item.Consumers) > 0 {
			sharePerPersonLog = Round(itemAmount / float64(len(item.Consumers)))
		} else {
			sharePerPersonLog = 0
		}

		log.Printf("Item: %s, price=%.2f, consumers=%v, sharePerPerson=%.2f",
			item.Description, itemAmount, item.Consumers, sharePerPersonLog)
	}

	// Calculate total subtotal for proportion calculation
	var totalSubtotal float64
	for _, person := range participants {
		totalSubtotal += breakdown[person].Subtotal
	}

	// Calculate extras (tax, service charge, discount)
	if totalSubtotal > 0 && len(participants) > 0 {
		for _, person := range participants {
			// Calculate proportional tax and service charge based on person's subtotal
			proportion := breakdown[person].Subtotal / totalSubtotal
			personTax := tax * proportion
			personService := serviceCharge * proportion
			personDiscount := totalDiscount * proportion

			// Update breakdown with tax and service
			breakdown[person] = models.PersonChargeBreakdown{
				Subtotal:      breakdown[person].Subtotal,
				Tax:           Round(personTax),
				ServiceCharge: Round(personService),
				Total:         Round(breakdown[person].Subtotal + personTax + personService - personDiscount),
			}

			// Update total charges for backward compatibility
			charges[person] = breakdown[person].Total
		}

		log.Printf("Extras calculation: tax=%.2f, service=%.2f, discount=%.2f, totalSubtotal=%.2f",
			tax, serviceCharge, totalDiscount, totalSubtotal)
	} else if totalSubtotal == 0 && len(participants) > 0 {
		// If subtotal is 0 but we have extras, divide them equally
		extraCharges := tax + serviceCharge - totalDiscount
		extraPerPerson := extraCharges / float64(len(participants))
		extraPerPerson = Round(extraPerPerson)

		for _, person := range participants {
			// Divide tax and service equally
			personTax := tax / float64(len(participants))
			personService := serviceCharge / float64(len(participants))

			// Update breakdown
			breakdown[person] = models.PersonChargeBreakdown{
				Subtotal:      0,
				Tax:           Round(personTax),
				ServiceCharge: Round(personService),
				Total:         Round(extraPerPerson),
			}

			// Update total charges
			charges[person] = extraPerPerson
		}

		log.Printf("Equal extras distribution: perPerson=%.2f", extraPerPerson)
	}

	// Round all values in the breakdown
	for person := range breakdown {
		breakdown[person] = models.PersonChargeBreakdown{
			Subtotal:      Round(breakdown[person].Subtotal),
			Tax:           Round(breakdown[person].Tax),
			ServiceCharge: Round(breakdown[person].ServiceCharge),
			Total:         Round(breakdown[person].Total),
		}
	}

	// Log the breakdown
	for person, bd := range breakdown {
		log.Printf("Person %s breakdown: subtotal=%.2f, tax=%.2f, service=%.2f, total=%.2f",
			person, bd.Subtotal, bd.Tax, bd.ServiceCharge, bd.Total)
	}

	return charges, breakdown
}

// Update your CalculateSingleBill handler to use the new breakdown
func CalculateSingleBill(c *gin.Context) {
	// Parse the request exactly as sent by the frontend
	var request struct {
		Items         []models.Item `json:"items"`
		Tax           float64       `json:"tax"`
		ServiceCharge float64       `json:"serviceCharge"`
		TotalDiscount float64       `json:"totalDiscount"`
	}

	// Parse the request
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Extract all unique participants
	participants := make(map[string]bool)
	for _, item := range request.Items {
		for _, consumer := range item.Consumers {
			participants[consumer] = true
		}
	}

	// Convert participants map to slice
	var allParticipants []string
	for participant := range participants {
		allParticipants = append(allParticipants, participant)
	}

	log.Printf("Request: %d items, tax=%.2f, serviceCharge=%.2f, participants=%v",
		len(request.Items), request.Tax, request.ServiceCharge, allParticipants)

	// Calculate how much each person owes with detailed breakdown
	perPersonCharges, perPersonBreakdown := calculatePersonalCharges(
		request.Items,
		request.Tax,
		request.ServiceCharge,
		request.TotalDiscount,
		allParticipants,
	)

	// Calculate subtotal and total
	subtotal := calculateSubtotal(request.Items)
	total := subtotal + request.Tax + request.ServiceCharge - request.TotalDiscount

	// Create result
	result := &models.SingleBillCalculation{
		Amount:             Round(total),
		Subtotal:           Round(subtotal),
		Tax:                Round(request.Tax),
		ServiceCharge:      Round(request.ServiceCharge),
		TotalDiscount:      Round(request.TotalDiscount),
		PerPersonCharges:   perPersonCharges,
		PerPersonBreakdown: perPersonBreakdown, // Add the breakdown to the result
	}

	// Log the result
	log.Printf("Calculation result: total=%.2f", result.Amount)
	for person, amount := range result.PerPersonCharges {
		log.Printf("  %s owes: %.2f", person, amount)
	}

	c.JSON(http.StatusOK, result)
}

// calculateSubtotal calculates the sum of all items
func calculateSubtotal(items []models.Item) float64 {
	var subtotal float64
	for _, item := range items {
		itemAmount := item.UnitPrice*float64(item.Quantity) - item.ItemDiscount
		subtotal += itemAmount
	}
	return subtotal
}

// Round rounds a number to 2 decimal places
func Round(num float64) float64 {
	return math.Round(num*100) / 100
}

// Simplified CalculateBill function that works with the actual request format
func CalculateBill(items []models.Item, tax, serviceCharge, totalDiscount float64, splitType string, splitAmong []string) (*models.SingleBillCalculation, error) {
	// Calculate subtotal from items
	var subtotal float64
	perPersonCharges := make(map[string]float64)

	// Initialize all participants with zero balance
	for _, person := range splitAmong {
		perPersonCharges[person] = 0
	}

	log.Printf("Processing %d items with split type: %s", len(items), splitType)

	// Process each item
	for i, item := range items {
		if item.UnitPrice < 0 || item.Quantity <= 0 {
			return nil, fmt.Errorf("invalid item price or quantity for item: %s", item.Description)
		}

		if item.PaidBy == "" || len(item.Consumers) == 0 {
			return nil, fmt.Errorf("missing paidBy or consumers for item: %s", item.Description)
		}

		// Calculate item amount
		itemAmount := item.UnitPrice*float64(item.Quantity) - item.ItemDiscount
		itemAmount = Round(itemAmount)
		subtotal += itemAmount

		// The payer pays the full amount for this item
		if _, exists := perPersonCharges[item.PaidBy]; !exists {
			perPersonCharges[item.PaidBy] = 0
		}
		perPersonCharges[item.PaidBy] += itemAmount

		// Each consumer owes their share of this item
		sharePerPerson := itemAmount / float64(len(item.Consumers))
		sharePerPerson = Round(sharePerPerson)

		for _, consumer := range item.Consumers {
			if _, exists := perPersonCharges[consumer]; !exists {
				perPersonCharges[consumer] = 0
			}
			perPersonCharges[consumer] -= sharePerPerson
		}

		log.Printf("Item %d: %s, Amount=%.2f, PaidBy=%s, Consumers=%v, SharePerPerson=%.2f",
			i, item.Description, itemAmount, item.PaidBy, item.Consumers, sharePerPerson)
	}

	// Round the subtotal
	subtotal = Round(subtotal)

	// Process tax, service charge, and discount
	tax = Round(tax)
	serviceCharge = Round(serviceCharge)
	totalDiscount = Round(totalDiscount)

	// Calculate total
	totalAmount := subtotal + tax + serviceCharge - totalDiscount
	totalAmount = Round(totalAmount)

	// Process extras (tax, service, discount)
	extraCharges := tax + serviceCharge - totalDiscount
	if extraCharges != 0 && len(splitAmong) > 0 {
		// Find the payer (person who paid the most items)
		payerCounts := make(map[string]int)
		for _, item := range items {
			payerCounts[item.PaidBy]++
		}

		var payer string
		maxCount := 0
		for p, count := range payerCounts {
			if count > maxCount {
				maxCount = count
				payer = p
			}
		}

		// If we couldn't determine payer, use the first participant
		if payer == "" {
			payer = splitAmong[0]
		}

		// Add the extra charges to the payer
		if _, exists := perPersonCharges[payer]; !exists {
			perPersonCharges[payer] = 0
		}
		perPersonCharges[payer] += extraCharges

		// Calculate per-person share of extras
		extraPerPerson := extraCharges / float64(len(splitAmong))
		extraPerPerson = Round(extraPerPerson)

		// Subtract each person's share of extras
		for _, person := range splitAmong {
			if _, exists := perPersonCharges[person]; !exists {
				perPersonCharges[person] = 0
			}
			perPersonCharges[person] -= extraPerPerson
		}

		log.Printf("Extra charges: Total=%.2f, PerPerson=%.2f, Payer=%s",
			extraCharges, extraPerPerson, payer)
	}

	// Round all final balances
	for person, amount := range perPersonCharges {
		perPersonCharges[person] = Round(amount)
	}

	// Create result
	result := &models.SingleBillCalculation{
		Amount:           totalAmount,
		Subtotal:         subtotal,
		Tax:              tax,
		ServiceCharge:    serviceCharge,
		TotalDiscount:    totalDiscount,
		PerPersonCharges: perPersonCharges,
	}

	return result, nil
}

// AddEqualExpense adds an equal-split expense
func AddEqualExpense(c *gin.Context) {
	var request models.AddEqualExpenseRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Get trip
	trip, err := services.GetTripByCode(request.Code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Add participants
	for _, participant := range request.SplitAmong {
		err := services.AddParticipant(trip.ID, participant)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add participant: " + err.Error()})
			return
		}
	}

	// Create expense
	expenseID := services.GenerateID()

	// Round all monetary values
	subtotal := services.Round(request.Subtotal)
	tax := services.Round(request.Tax)
	serviceCharge := services.Round(request.ServiceCharge)
	totalDiscount := services.Round(request.TotalDiscount)

	expense := models.NewEqualExpense(
		expenseID,
		trip.ID,
		request.Description,
		subtotal,
		tax,
		serviceCharge,
		totalDiscount,
		request.PaidBy,
		request.SplitAmong,
	)

	// Store expense
	err = services.StoreExpense(expense)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store expense: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, expense)
}

// AddItemsExpense adds an item-based expense
func AddItemsExpense(c *gin.Context) {
	var request models.AddItemsExpenseRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Get trip
	trip, err := services.GetTripByCode(request.Code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Process items and calculate subtotal
	subtotal, paidBy, err := services.ProcessExpenseItems(trip.ID, request.Items)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Round monetary values
	subtotal = services.Round(subtotal)
	tax := services.Round(request.Tax)
	serviceCharge := services.Round(request.ServiceCharge)
	totalDiscount := services.Round(request.TotalDiscount)

	// Create expense
	expenseID := services.GenerateID()

	expense := models.NewItemExpense(
		expenseID,
		trip.ID,
		request.Description,
		subtotal,
		tax,
		serviceCharge,
		totalDiscount,
		paidBy,
		request.Items,
	)

	// Store expense
	err = services.StoreExpense(expense)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store expense: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, expense)
}

// RemoveExpense removes an expense
func RemoveExpense(c *gin.Context) {
	var request models.RemoveExpenseRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Get trip
	trip, err := services.GetTripByCode(request.Code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Remove expense
	found, err := services.RemoveExpense(trip.ID, request.ExpenseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Expense not found"})
		return
	}

	c.JSON(http.StatusOK, true)
}

// ListExpenses lists all expenses for a trip
func ListExpenses(c *gin.Context) {
	var request models.GetTripByCodeRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Get trip
	trip, err := services.GetTripByCode(request.Code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Get expenses
	tripExpenses, err := services.GetExpenses(trip.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get expenses: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, tripExpenses)
}

// CalculateSettlements calculates settlements for a trip
func CalculateSettlements(c *gin.Context) {
	var request models.GetTripByCodeRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Get trip
	trip, err := services.GetTripByCode(request.Code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Calculate settlements
	settlementResult, err := services.CalculateSettlements(trip.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate settlements: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, settlementResult)
}
