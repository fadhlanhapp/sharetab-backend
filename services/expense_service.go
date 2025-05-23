package services

import (
	"fmt"

	"github.com/fadhlanhapp/sharetab-backend/models"
	"github.com/fadhlanhapp/sharetab-backend/repository"
	"github.com/fadhlanhapp/sharetab-backend/utils"
)

var expenseRepo *repository.ExpenseRepository

// InitExpenseService initializes the expense service
func InitExpenseService() {
	expenseRepo = repository.NewExpenseRepository()
}

// ExpenseService handles expense-related business logic
type ExpenseService struct {
	repo *repository.ExpenseRepository
}

// NewExpenseService creates a new expense service instance
func NewExpenseService() *ExpenseService {
	return &ExpenseService{
		repo: repository.NewExpenseRepository(),
	}
}

// GetExpenses returns all expenses for a trip with formatted names
func (s *ExpenseService) GetExpenses(tripID string) ([]*models.Expense, error) {
	expenses, err := s.repo.GetExpenses(tripID)
	if err != nil {
		return nil, utils.NewInternalError("Failed to retrieve expenses")
	}

	// Format names for display
	formattedExpenses := make([]*models.Expense, len(expenses))
	for i, expense := range expenses {
		formattedExpenses[i] = s.formatExpenseForDisplay(expense)
	}

	return formattedExpenses, nil
}

// StoreExpense stores an expense for a trip
func (s *ExpenseService) StoreExpense(expense *models.Expense) error {
	if err := s.repo.StoreExpense(expense); err != nil {
		return utils.NewInternalError("Failed to store expense")
	}
	return nil
}

// RemoveExpense removes an expense from a trip
func (s *ExpenseService) RemoveExpense(tripID, expenseID string) error {
	found, err := s.repo.RemoveExpense(tripID, expenseID)
	if err != nil {
		return utils.NewInternalError("Failed to remove expense")
	}
	if !found {
		return utils.NewNotFoundError("Expense")
	}
	return nil
}

// CreateEqualExpense creates an equal split expense with validation
func (s *ExpenseService) CreateEqualExpense(request *models.AddEqualExpenseRequest) (*models.Expense, error) {
	if err := s.validateEqualExpenseRequest(request); err != nil {
		return nil, err
	}

	// Normalize names
	normalizedPaidBy := utils.NormalizeName(request.PaidBy)
	normalizedSplitAmong := utils.NormalizeNames(request.SplitAmong)

	// Create expense
	expenseID := utils.GenerateID()
	expense := models.NewEqualExpense(
		expenseID,
		"", // Will be set by caller
		request.Description,
		utils.Round(request.Subtotal),
		utils.Round(request.Tax),
		utils.Round(request.ServiceCharge),
		utils.Round(request.TotalDiscount),
		normalizedPaidBy,
		normalizedSplitAmong,
	)

	return expense, nil
}

// CreateItemsExpense creates an items-based expense with validation
func (s *ExpenseService) CreateItemsExpense(request *models.AddItemsExpenseRequest) (*models.Expense, error) {
	if err := s.validateItemsExpenseRequest(request); err != nil {
		return nil, err
	}

	// Process and normalize items
	processedItems, subtotal, paidBy, err := s.processExpenseItems(request.Items)
	if err != nil {
		return nil, err
	}

	// Create expense
	expenseID := utils.GenerateID()
	expense := models.NewItemExpense(
		expenseID,
		"", // Will be set by caller
		request.Description,
		subtotal,
		utils.Round(request.Tax),
		utils.Round(request.ServiceCharge),
		utils.Round(request.TotalDiscount),
		paidBy,
		processedItems,
	)

	return expense, nil
}

// formatExpenseForDisplay formats expense names for display
func (s *ExpenseService) formatExpenseForDisplay(expense *models.Expense) *models.Expense {
	formatted := *expense
	formatted.PaidBy = utils.FormatNameForDisplay(expense.PaidBy)

	if len(expense.SplitAmong) > 0 {
		formatted.SplitAmong = utils.FormatNamesForDisplay(expense.SplitAmong)
	}

	if len(expense.Items) > 0 {
		formattedItems := make([]models.Item, len(expense.Items))
		for j, item := range expense.Items {
			formattedItems[j] = models.Item{
				Description:  item.Description,
				UnitPrice:    item.UnitPrice,
				Quantity:     item.Quantity,
				Amount:       item.Amount,
				ItemDiscount: item.ItemDiscount,
				PaidBy:       utils.FormatNameForDisplay(item.PaidBy),
				Consumers:    utils.FormatNamesForDisplay(item.Consumers),
			}
		}
		formatted.Items = formattedItems
	}

	return &formatted
}

// processExpenseItems processes items for an expense and returns processed items, subtotal and paidBy
func (s *ExpenseService) processExpenseItems(items []models.Item) ([]models.Item, float64, string, error) {
	var subtotal float64
	var paidBy string
	processedItems := make([]models.Item, len(items))

	for i, item := range items {
		if err := utils.ValidateItemData(item.UnitPrice, item.Quantity, item.Description); err != nil {
			return nil, 0, "", utils.NewValidationError(fmt.Sprintf("Item %d: %s", i+1, err.Error()))
		}

		if item.PaidBy == "" || len(item.Consumers) == 0 {
			return nil, 0, "", utils.NewValidationError(fmt.Sprintf("Item %d: missing paidBy or consumers", i+1))
		}

		// Normalize names
		normalizedPaidBy := utils.NormalizeName(item.PaidBy)
		normalizedConsumers := utils.NormalizeNames(item.Consumers)

		// Set paidBy if not set yet
		if paidBy == "" {
			paidBy = normalizedPaidBy
		}

		// Calculate item amount
		itemAmount := item.UnitPrice*float64(item.Quantity) - item.ItemDiscount
		itemAmount = utils.Round(itemAmount)
		subtotal += itemAmount

		// Store processed item
		processedItems[i] = models.Item{
			Description:  item.Description,
			UnitPrice:    item.UnitPrice,
			Quantity:     item.Quantity,
			Amount:       itemAmount,
			ItemDiscount: item.ItemDiscount,
			PaidBy:       normalizedPaidBy,
			Consumers:    normalizedConsumers,
		}
	}

	return processedItems, utils.Round(subtotal), paidBy, nil
}

// validateEqualExpenseRequest validates an equal expense request
func (s *ExpenseService) validateEqualExpenseRequest(request *models.AddEqualExpenseRequest) error {
	if err := utils.ValidateRequired(request.Code, "trip code"); err != nil {
		return err
	}
	if err := utils.ValidateRequired(request.Description, "description"); err != nil {
		return err
	}
	if err := utils.ValidateNonNegative(request.Subtotal, "subtotal"); err != nil {
		return err
	}
	if err := utils.ValidateNonNegative(request.Tax, "tax"); err != nil {
		return err
	}
	if err := utils.ValidateNonNegative(request.ServiceCharge, "service charge"); err != nil {
		return err
	}
	if err := utils.ValidateNonNegative(request.TotalDiscount, "discount"); err != nil {
		return err
	}
	if err := utils.ValidateRequired(request.PaidBy, "paidBy"); err != nil {
		return err
	}
	if err := utils.ValidateNotEmpty(request.SplitAmong, "splitAmong"); err != nil {
		return err
	}
	if err := utils.ValidateParticipantNames(request.SplitAmong); err != nil {
		return err
	}
	return nil
}

// validateItemsExpenseRequest validates an items expense request
func (s *ExpenseService) validateItemsExpenseRequest(request *models.AddItemsExpenseRequest) error {
	if err := utils.ValidateRequired(request.Code, "trip code"); err != nil {
		return err
	}
	if err := utils.ValidateRequired(request.Description, "description"); err != nil {
		return err
	}
	if err := utils.ValidateNonNegative(request.Tax, "tax"); err != nil {
		return err
	}
	if err := utils.ValidateNonNegative(request.ServiceCharge, "service charge"); err != nil {
		return err
	}
	if err := utils.ValidateNonNegative(request.TotalDiscount, "discount"); err != nil {
		return err
	}
	if err := utils.ValidateNotEmpty(request.Items, "items"); err != nil {
		return err
	}

	// Validate each item
	for i, item := range request.Items {
		if err := utils.ValidateItemData(item.UnitPrice, item.Quantity, item.Description); err != nil {
			return utils.NewValidationError(fmt.Sprintf("Item %d: %s", i+1, err.Error()))
		}
		if err := utils.ValidateRequired(item.PaidBy, "item paidBy"); err != nil {
			return utils.NewValidationError(fmt.Sprintf("Item %d: %s", i+1, err.Error()))
		}
		if err := utils.ValidateNotEmpty(item.Consumers, "item consumers"); err != nil {
			return utils.NewValidationError(fmt.Sprintf("Item %d: %s", i+1, err.Error()))
		}
		if err := utils.ValidateParticipantNames(item.Consumers); err != nil {
			return utils.NewValidationError(fmt.Sprintf("Item %d: %s", i+1, err.Error()))
		}
	}

	return nil
}

// Legacy functions for backward compatibility
func GetExpenses(tripID string) ([]*models.Expense, error) {
	return expenseRepo.GetExpenses(tripID)
}

func StoreExpense(expense *models.Expense) error {
	return expenseRepo.StoreExpense(expense)
}

func RemoveExpense(tripID string, expenseID string) (bool, error) {
	return expenseRepo.RemoveExpense(tripID, expenseID)
}

func Round(num float64) float64 {
	return utils.Round(num)
}

// Legacy ProcessExpenseItems function for backward compatibility
func ProcessExpenseItems(tripID string, items []models.Item) (float64, string, error) {
	service := NewExpenseService()
	processedItems, subtotal, paidBy, err := service.processExpenseItems(items)
	if err != nil {
		return 0, "", err
	}

	// Add participants to trip
	for _, item := range processedItems {
		AddParticipant(tripID, item.PaidBy)
		for _, consumer := range item.Consumers {
			AddParticipant(tripID, consumer)
		}
	}

	return subtotal, paidBy, nil
}

// CalculateSettlements calculates settlements for a trip (legacy function for backward compatibility)
func CalculateSettlements(tripID string) (*models.SettlementResult, error) {
	expenseService := NewExpenseService()
	settlementService := NewSettlementService(expenseService)
	return settlementService.CalculateSettlements(tripID)
}

// ConvertReceiptItemToExpenseItem converts a receipt item to an expense item
func ConvertReceiptItemToExpenseItem(receiptItem models.ReceiptItem, paidBy string, consumers []string) models.Item {
	return models.Item{
		Description:  receiptItem.Name,
		UnitPrice:    receiptItem.Price,
		Quantity:     int(receiptItem.Quantity),
		ItemDiscount: receiptItem.Discount,
		PaidBy:       utils.NormalizeName(paidBy),
		Consumers:    utils.NormalizeNames(consumers),
		Amount:       utils.Round(receiptItem.Price*receiptItem.Quantity - receiptItem.Discount),
	}
}