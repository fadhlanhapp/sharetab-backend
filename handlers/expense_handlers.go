// handlers/expense_handlers.go
package handlers

import (
	"log"
	"net/http"

	"github.com/fadhlanhapp/sharetab-backend/models"
	"github.com/fadhlanhapp/sharetab-backend/services"

	"github.com/gin-gonic/gin"
)

// Updated CalculateSingleBill handler that handles both formats
func CalculateSingleBill(c *gin.Context) {
	// Create a variable to hold the raw request as a map
	var rawRequest map[string]interface{}

	// First bind the raw request
	if err := c.ShouldBindJSON(&rawRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Check if we have "splitAmong" in the request, which indicates an equal split
	splitType := "items" // Default to items
	var splitAmong []string
	var items []models.Item
	var subtotal float64
	var tax float64
	var serviceCharge float64
	var totalDiscount float64

	// Log the raw request for debugging
	log.Printf("Raw request: %+v", rawRequest)

	// Check if this is an equal split request
	if _, hasSplitAmong := rawRequest["splitAmong"]; hasSplitAmong {
		splitType = "equal"

		// Parse equal split request
		var equalRequest struct {
			Subtotal      float64  `json:"subtotal"`
			Tax           float64  `json:"tax"`
			ServiceCharge float64  `json:"serviceCharge"`
			TotalDiscount float64  `json:"totalDiscount"`
			PaidBy        string   `json:"paidBy"`
			SplitAmong    []string `json:"splitAmong"`
		}

		if err := c.ShouldBindJSON(&equalRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid equal split request"})
			return
		}

		// Extract values
		subtotal = equalRequest.Subtotal
		tax = equalRequest.Tax
		serviceCharge = equalRequest.ServiceCharge
		totalDiscount = equalRequest.TotalDiscount
		splitAmong = equalRequest.SplitAmong

		// Create a dummy item representing the full bill
		paidBy := equalRequest.PaidBy
		if paidBy == "" && len(splitAmong) > 0 {
			paidBy = splitAmong[0]
		}

		items = []models.Item{
			{
				Description:  "Total Bill",
				UnitPrice:    subtotal,
				Quantity:     1,
				ItemDiscount: 0,
				PaidBy:       paidBy,
				Consumers:    splitAmong,
			},
		}

		log.Printf("Parsed equal split request: subtotal=%.2f, tax=%.2f, serviceCharge=%.2f, paidBy=%s, splitAmong=%v",
			subtotal, tax, serviceCharge, paidBy, splitAmong)

	} else {
		// Parse itemized split request
		var itemRequest struct {
			Items         []models.Item `json:"items"`
			Tax           float64       `json:"tax"`
			ServiceCharge float64       `json:"serviceCharge"`
			TotalDiscount float64       `json:"totalDiscount"`
		}

		if err := c.ShouldBindJSON(&itemRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid itemized split request"})
			return
		}

		// Extract values
		items = itemRequest.Items
		tax = itemRequest.Tax
		serviceCharge = itemRequest.ServiceCharge
		totalDiscount = itemRequest.TotalDiscount

		// Collect all participants
		participantsMap := make(map[string]bool)
		for _, item := range items {
			if item.PaidBy != "" {
				participantsMap[item.PaidBy] = true
			}
			for _, consumer := range item.Consumers {
				participantsMap[consumer] = true
			}
		}

		// Convert participants map to slice
		for participant := range participantsMap {
			splitAmong = append(splitAmong, participant)
		}

		log.Printf("Parsed itemized split request: items=%d, tax=%.2f, serviceCharge=%.2f, participants=%v",
			len(items), tax, serviceCharge, splitAmong)
	}

	// Now calculate the bill using our improved CalculateBill function
	result, err := services.CalculateBill(items, tax, serviceCharge, totalDiscount, splitType, splitAmong)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Log the result for debugging
	log.Printf("Calculation result: %+v", result)

	c.JSON(http.StatusOK, result)
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
