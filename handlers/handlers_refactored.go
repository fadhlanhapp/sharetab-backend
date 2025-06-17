package handlers

import (
	"fmt"
	
	"github.com/fadhlanhapp/sharetab-backend/models"
	"github.com/fadhlanhapp/sharetab-backend/services"
	"github.com/fadhlanhapp/sharetab-backend/repository"
	"github.com/fadhlanhapp/sharetab-backend/utils"

	"github.com/gin-gonic/gin"
)

// HandlerServices contains all service dependencies
type HandlerServices struct {
	TripService       *services.TripService
	ExpenseService    *services.ExpenseService
	CalculationService *services.CalculationService
	SettlementService *services.SettlementService
	PaymentService    *services.PaymentService
}

// NewHandlerServices creates a new handler services instance
func NewHandlerServices() *HandlerServices {
	expenseService := services.NewExpenseService()
	tripService := services.NewTripService()
	
	// Initialize repositories and services for payments
	paymentRepo := repository.NewPaymentRepository(repository.GetDB())
	tripRepo := repository.NewTripRepository()
	paymentService := services.NewPaymentService(paymentRepo, tripRepo)
	
	return &HandlerServices{
		TripService:       tripService,
		ExpenseService:    expenseService,
		CalculationService: services.NewCalculationService(),
		SettlementService: services.NewSettlementService(expenseService, paymentService),
		PaymentService:    paymentService,
	}
}

var handlerServices *HandlerServices

// InitHandlers initializes the handler services
func InitHandlers() {
	handlerServices = NewHandlerServices()
}

// CreateTripRefactored handles the creation of a new trip
func CreateTripRefactored(c *gin.Context) {
	var request models.CreateTripRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.HandleError(c, utils.NewBadRequestError(utils.ErrInvalidRequest))
		return
	}

	trip, err := handlerServices.TripService.CreateTrip(request.Name, request.Participant)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	response := models.CreateTripResponse{
		TripID: trip.ID,
		Code:   trip.Code,
	}

	utils.HandleSuccess(c, response)
}

// GetTripByCodeRefactored handles retrieving a trip by its code
func GetTripByCodeRefactored(c *gin.Context) {
	var request models.GetTripByCodeRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.HandleError(c, utils.NewBadRequestError(utils.ErrInvalidRequest))
		return
	}

	trip, err := handlerServices.TripService.GetTripByCode(request.Code)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.HandleSuccess(c, trip)
}

// CalculateSingleBillRefactored handles single bill calculation
func CalculateSingleBillRefactored(c *gin.Context) {
	var request models.CalculateSingleBillRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.HandleError(c, utils.NewBadRequestError(utils.ErrInvalidRequest))
		return
	}

	result, err := handlerServices.CalculationService.CalculateSingleBill(&request)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.HandleSuccess(c, result)
}

// AddEqualExpenseRefactored adds an equal-split expense
func AddEqualExpenseRefactored(c *gin.Context) {
	var request models.AddEqualExpenseRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.HandleError(c, utils.NewBadRequestError(utils.ErrInvalidRequest))
		return
	}

	// Get trip to validate and get trip ID
	trip, err := handlerServices.TripService.GetTripByCode(request.Code)
	if err != nil {
		utils.HandleError(c, utils.NewNotFoundError("Trip"))
		return
	}

	// Create expense
	expense, err := handlerServices.ExpenseService.CreateEqualExpense(&request)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	// Set trip ID
	expense.TripID = trip.ID

	// Add participants to trip
	for _, participant := range request.SplitAmong {
		if err := handlerServices.TripService.AddParticipant(trip.ID, participant); err != nil {
			utils.HandleError(c, utils.NewInternalError("Failed to add participant"))
			return
		}
	}

	// Store expense
	if err := handlerServices.ExpenseService.StoreExpense(expense); err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.HandleSuccess(c, expense)
}

// AddItemsExpenseRefactored adds an item-based expense
func AddItemsExpenseRefactored(c *gin.Context) {
	var request models.AddItemsExpenseRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.HandleError(c, utils.NewBadRequestError(utils.ErrInvalidRequest))
		return
	}

	// Get trip to validate and get trip ID
	trip, err := handlerServices.TripService.GetTripByCode(request.Code)
	if err != nil {
		utils.HandleError(c, utils.NewNotFoundError("Trip"))
		return
	}

	// Create expense
	expense, err := handlerServices.ExpenseService.CreateItemsExpense(&request)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	// Set trip ID
	expense.TripID = trip.ID

	// Add participants to trip
	for _, item := range expense.Items {
		if err := handlerServices.TripService.AddParticipant(trip.ID, item.PaidBy); err != nil {
			utils.HandleError(c, utils.NewInternalError("Failed to add participant"))
			return
		}
		for _, consumer := range item.Consumers {
			if err := handlerServices.TripService.AddParticipant(trip.ID, consumer); err != nil {
				utils.HandleError(c, utils.NewInternalError("Failed to add participant"))
				return
			}
		}
	}

	// Store expense
	if err := handlerServices.ExpenseService.StoreExpense(expense); err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.HandleSuccess(c, expense)
}

// RemoveExpenseRefactored removes an expense
func RemoveExpenseRefactored(c *gin.Context) {
	var request models.RemoveExpenseRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.HandleError(c, utils.NewBadRequestError(utils.ErrInvalidRequest))
		return
	}

	// Get trip to validate
	trip, err := handlerServices.TripService.GetTripByCode(request.Code)
	if err != nil {
		utils.HandleError(c, utils.NewNotFoundError("Trip"))
		return
	}

	// Remove expense
	if err := handlerServices.ExpenseService.RemoveExpense(trip.ID, request.ExpenseID); err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.HandleSuccess(c, true)
}

// ListExpensesRefactored lists all expenses for a trip
func ListExpensesRefactored(c *gin.Context) {
	var request models.GetTripByCodeRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.HandleError(c, utils.NewBadRequestError(utils.ErrInvalidRequest))
		return
	}

	// Get trip to validate
	trip, err := handlerServices.TripService.GetTripByCode(request.Code)
	if err != nil {
		utils.HandleError(c, utils.NewNotFoundError("Trip"))
		return
	}

	// Get expenses
	expenses, err := handlerServices.ExpenseService.GetExpenses(trip.ID)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.HandleSuccess(c, expenses)
}

// CalculateSettlementsRefactored calculates settlements for a trip
func CalculateSettlementsRefactored(c *gin.Context) {
	var request models.GetTripByCodeRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.HandleError(c, utils.NewBadRequestError(utils.ErrInvalidRequest))
		return
	}

	// Get trip to validate
	trip, err := handlerServices.TripService.GetTripByCode(request.Code)
	if err != nil {
		utils.HandleError(c, utils.NewNotFoundError("Trip"))
		return
	}

	// Calculate settlements
	result, err := handlerServices.SettlementService.CalculateSettlements(trip.ID)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.HandleSuccess(c, result)
}

// Payment handler functions
func CreatePaymentHandler(c *gin.Context) {
	var req models.PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("Payment binding error: %v\n", err)
		c.JSON(400, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	fmt.Printf("Payment request received: %+v\n", req)

	// Check if payment service is properly initialized
	if handlerServices.PaymentService == nil {
		c.JSON(500, gin.H{"error": "Payment service not initialized"})
		return
	}

	payment, err := handlerServices.PaymentService.CreatePayment(&req)
	if err != nil {
		fmt.Printf("Payment service error: %v\n", err)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("Payment created successfully: %+v\n", payment)
	c.JSON(201, payment)
}

func GetPaymentsByTripHandler(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.HandleError(c, utils.NewBadRequestError(err.Error()))
		return
	}

	payments, err := handlerServices.PaymentService.GetPaymentsByTripCode(req.Code)
	if err != nil {
		utils.HandleError(c, utils.NewNotFoundError(err.Error()))
		return
	}

	utils.HandleSuccess(c, payments)
}

func DeletePaymentHandler(c *gin.Context) {
	paymentID := c.Param("id")
	if paymentID == "" {
		utils.HandleError(c, utils.NewBadRequestError("Payment ID is required"))
		return
	}

	// Convert to int
	id := 0
	if _, err := fmt.Sscanf(paymentID, "%d", &id); err != nil {
		utils.HandleError(c, utils.NewBadRequestError("Invalid payment ID"))
		return
	}

	err := handlerServices.PaymentService.DeletePayment(id)
	if err != nil {
		utils.HandleError(c, utils.NewNotFoundError(err.Error()))
		return
	}

	utils.HandleSuccess(c, gin.H{"message": "Payment deleted successfully"})
}