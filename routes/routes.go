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

		// Payment endpoints
		v1.POST("/payments/create", handlers.CreatePaymentHandler)
		v1.POST("/payments/getByTrip", handlers.GetPaymentsByTripHandler)
		v1.DELETE("/payments/:id", handlers.DeletePaymentHandler)

		// Receipt processing endpoints
		v1.POST("/receipts/process", handlers.HandleProcessReceiptV1)
		v1.POST("/receipts/addExpense", handlers.AddExpenseFromReceiptV1)

		// Export endpoints
		v1.POST("/trips/exportToExcel", handlers.ExportTripToExcel)
	}

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
			"service": "sharetab-api",
		})
	})
}
