package handlers

import (
	"net/http"

	"github.com/fadhlanhapp/sharetab-backend/models"
	"github.com/fadhlanhapp/sharetab-backend/services"
	"github.com/fadhlanhapp/sharetab-backend/utils"

	"github.com/gin-gonic/gin"
)

// HandlerServices contains all service dependencies
type HandlerServices struct {
	TripService       *services.TripService
	ExpenseService    *services.ExpenseService
	CalculationService *services.CalculationService
	SettlementService *services.SettlementService
}

// NewHandlerServices creates a new handler services instance
func NewHandlerServices() *HandlerServices {
	expenseService := services.NewExpenseService()
	return &HandlerServices{
		TripService:       services.NewTripService(),
		ExpenseService:    expenseService,
		CalculationService: services.NewCalculationService(),
		SettlementService: services.NewSettlementService(expenseService),
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
	trip, err := services.GetTripByCode(request.Code)
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
		if err := services.AddParticipant(trip.ID, participant); err != nil {
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
	trip, err := services.GetTripByCode(request.Code)
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
		if err := services.AddParticipant(trip.ID, item.PaidBy); err != nil {
			utils.HandleError(c, utils.NewInternalError("Failed to add participant"))
			return
		}
		for _, consumer := range item.Consumers {
			if err := services.AddParticipant(trip.ID, consumer); err != nil {
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
	trip, err := services.GetTripByCode(request.Code)
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
	trip, err := services.GetTripByCode(request.Code)
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
	trip, err := services.GetTripByCode(request.Code)
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