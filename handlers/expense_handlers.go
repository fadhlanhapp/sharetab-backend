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

// Fixed CalculateSingleBill handler to work with the actual request format
func CalculateSingleBill(c *gin.Context) {
	// Define the actual request structure that matches the frontend payload
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

	// Log the parsed request for debugging
	log.Printf("Parsed request: items=%d, tax=%.2f, serviceCharge=%.2f, totalDiscount=%.2f",
		len(request.Items), request.Tax, request.ServiceCharge, request.TotalDiscount)

	// Extract all unique participants
	participants := make(map[string]bool)
	for _, item := range request.Items {
		// Add the payer
		if item.PaidBy != "" {
			participants[item.PaidBy] = true
		}

		// Add all consumers
		for _, consumer := range item.Consumers {
			participants[consumer] = true
		}
	}

	// Convert participants map to slice
	var splitAmong []string
	for participant := range participants {
		splitAmong = append(splitAmong, participant)
	}

	// Determine split type - just use "items" since that's what the payload indicates
	splitType := "items"

	// Log the extracted information
	log.Printf("Extracted participants: %v, splitType: %s", splitAmong, splitType)

	// Calculate the bill
	result, err := CalculateBill(request.Items, request.Tax, request.ServiceCharge, request.TotalDiscount, splitType, splitAmong)
	if err != nil {
		log.Printf("Error calculating bill: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Log the result for debugging
	log.Printf("Calculation result: Amount=%.2f, Subtotal=%.2f", result.Amount, result.Subtotal)
	for person, amount := range result.PerPersonCharges {
		log.Printf("  %s: %.2f", person, amount)
	}

	// Return the result
	c.JSON(http.StatusOK, result)
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

// Helper function to round to 2 decimal places
func Round(num float64) float64 {
	return math.Round(num*100) / 100
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
