package routes

import (
	"os"

	"github.com/fadhlanhapp/sharetab-backend/handlers"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all API routes for the application
func SetupRoutes(router *gin.Engine) {
	// Create uploads directory if not exists
	os.MkdirAll("uploads", os.ModePerm)

	// Initialize refactored handlers
	handlers.InitHandlers()

	// API v1 routes (refactored)
	v1 := router.Group("/api/v1")
	{
		// Trip endpoints
		v1.POST("/trips/create", handlers.CreateTripRefactored)
		v1.POST("/trips/getByCode", handlers.GetTripByCodeRefactored)

		// Expense endpoints
		v1.POST("/expenses/calculateSingleBill", handlers.CalculateSingleBillRefactored)
		v1.POST("/expenses/addEqual", handlers.AddEqualExpenseRefactored)
		v1.POST("/expenses/addItems", handlers.AddItemsExpenseRefactored)
		v1.POST("/expenses/remove", handlers.RemoveExpenseRefactored)
		v1.POST("/expenses/list", handlers.ListExpensesRefactored)
		v1.POST("/expenses/calculateSettlements", handlers.CalculateSettlementsRefactored)
	}

	// Legacy API routes (for backward compatibility)
	// Trip endpoints
	router.POST("/trips/create", handlers.CreateTrip)
	router.POST("/trips/getByCode", handlers.GetTripByCodeHandler)

	// Expense endpoints
	router.POST("/expenses/calculateSingleBill", handlers.CalculateSingleBill)
	router.POST("/expenses/addEqual", handlers.AddEqualExpense)
	router.POST("/expenses/addItems", handlers.AddItemsExpense)
	router.POST("/expenses/remove", handlers.RemoveExpense)
	router.POST("/expenses/list", handlers.ListExpenses)
	router.POST("/expenses/calculateSettlements", handlers.CalculateSettlements)

	// Receipt processing endpoint
	router.POST("/process-receipt", handlers.HandleProcessReceipt)

	// Add expense from receipt endpoint
	router.POST("/expenses/addFromReceipt", handlers.AddExpenseFromReceipt)
}
