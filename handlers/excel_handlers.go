package handlers

import (
	"fmt"
	"net/http"

	"github.com/fadhlanhapp/sharetab-backend/models"
	"github.com/fadhlanhapp/sharetab-backend/repository"
	"github.com/fadhlanhapp/sharetab-backend/services"
	"github.com/gin-gonic/gin"
)

// ExportTripToExcel exports a trip's data to Excel format
func ExportTripToExcel(c *gin.Context) {
	var request models.GetTripByCodeRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Initialize services
	tripService := services.NewTripService()
	expenseService := services.NewExpenseService()
	
	// Initialize repositories and services for payments
	paymentRepo := repository.NewPaymentRepository(repository.GetDB())
	tripRepo := repository.NewTripRepository()
	paymentService := services.NewPaymentService(paymentRepo, tripRepo)
	
	settlementService := services.NewSettlementService(expenseService, paymentService)
	excelService := services.NewExcelService(tripService, expenseService, settlementService, paymentService)

	// Generate Excel file
	excelFile, filename, err := excelService.ExportTripToExcel(request.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export trip: " + err.Error()})
		return
	}

	// Set headers for file download
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Transfer-Encoding", "binary")

	// Write Excel file to response
	if err := excelFile.Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write Excel file: " + err.Error()})
		return
	}
}