// routes/routes.go
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

	// API routes
	api := router.Group("/api")
	{
		// Trip endpoints
		api.POST("/trips/create", handlers.CreateTrip)
		api.POST("/trips/getByCode", handlers.GetTripByCodeHandler)

		// Expense endpoints
		api.POST("/expenses/calculateSingleBill", handlers.CalculateSingleBill)
		api.POST("/expenses/addEqual", handlers.AddEqualExpense)
		api.POST("/expenses/addItems", handlers.AddItemsExpense)
		api.POST("/expenses/remove", handlers.RemoveExpense)
		api.POST("/expenses/list", handlers.ListExpenses)
		api.POST("/expenses/calculateSettlements", handlers.CalculateSettlements)

		// Receipt processing endpoint
		api.POST("/process-receipt", handlers.HandleProcessReceipt)

		// Add expense from receipt endpoint
		api.POST("/expenses/addFromReceipt", handlers.AddExpenseFromReceipt)
	}
}
