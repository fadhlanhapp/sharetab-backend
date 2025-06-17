package services

import (
	"fmt"
	"sort"
	"time"

	"github.com/fadhlanhapp/sharetab-backend/models"
	"github.com/fadhlanhapp/sharetab-backend/utils"
	"github.com/xuri/excelize/v2"
)

// ExcelService handles Excel export functionality
type ExcelService struct {
	tripService       *TripService
	expenseService    *ExpenseService
	settlementService *SettlementService
	paymentService    *PaymentService
}

// NewExcelService creates a new Excel service
func NewExcelService(tripService *TripService, expenseService *ExpenseService, settlementService *SettlementService, paymentService *PaymentService) *ExcelService {
	return &ExcelService{
		tripService:       tripService,
		expenseService:    expenseService,
		settlementService: settlementService,
		paymentService:    paymentService,
	}
}

// PersonSummary represents a person's spending summary
type PersonSummary struct {
	Name         string
	TotalSpent   float64 // How much they paid out
	TotalOwed    float64 // How much they consumed
	NetBalance   float64 // Positive = should receive, Negative = should pay
}

// ExpenseMatrixRow represents a row in the expense matrix
type ExpenseMatrixRow struct {
	Date        string
	BillName    string
	PaidBy      string
	TotalAmount float64
	PersonAmounts map[string]float64 // person name -> amount they owe for this expense
}

// ExportTripToExcel generates an Excel file for a trip
func (s *ExcelService) ExportTripToExcel(tripCode string) (*excelize.File, string, error) {
	// Get trip data
	trip, err := s.tripService.GetTripByCode(tripCode)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get trip: %v", err)
	}

	// Get all expenses
	expenses, err := s.expenseService.GetExpenses(trip.ID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get expenses: %v", err)
	}

	// Get settlements
	settlementResult, err := s.settlementService.CalculateSettlements(trip.ID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to calculate settlements: %v", err)
	}

	// Get payments
	payments, err := s.paymentService.GetPaymentsByTripID(trip.ID)
	if err != nil {
		// If payment service fails, just use empty payments
		payments = []models.Payment{}
	}

	// Create Excel file
	f := excelize.NewFile()

	// Create sheets
	err = s.createSummarySheet(f, trip, expenses, settlementResult)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create summary sheet: %v", err)
	}

	err = s.createExpenseMatrixSheet(f, trip, expenses)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create expense matrix sheet: %v", err)
	}

	err = s.createPaymentSheet(f, payments)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create payment sheet: %v", err)
	}

	// Delete the default sheet if it exists
	f.DeleteSheet("Sheet1")

	filename := fmt.Sprintf("%s_Export_%s.xlsx", 
		utils.CleanFileName(trip.Name), 
		time.Now().Format("2006-01-02"))

	return f, filename, nil
}

// createSummarySheet creates Sheet 1: Summary
func (s *ExcelService) createSummarySheet(f *excelize.File, trip *models.Trip, expenses []*models.Expense, settlementResult *models.SettlementResult) error {
	sheetName := "Summary"
	f.NewSheet(sheetName)
	sheetIndex, _ := f.GetSheetIndex(sheetName)
	f.SetActiveSheet(sheetIndex)

	// Calculate person summaries
	summaries := s.calculatePersonSummaries(expenses)

	// Sort summaries by name for consistent output
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})

	// Set headers
	headers := []string{"Person", "Total Spent", "Total Owed", "Net Balance"}
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		f.SetCellValue(sheetName, cell, header)
	}

	// Style headers
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"E6F3FF"}, Pattern: 1},
	})
	f.SetCellStyle(sheetName, "A1", fmt.Sprintf("%s1", string(rune('A'+len(headers)-1))), headerStyle)

	// Add summary data
	for i, summary := range summaries {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), summary.Name)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), summary.TotalSpent)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), summary.TotalOwed)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), summary.NetBalance)
	}

	// Add settlements section
	settlementsStartRow := len(summaries) + 4
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", settlementsStartRow), "Required Settlements:")
	
	settlementHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	f.SetCellStyle(sheetName, fmt.Sprintf("A%d", settlementsStartRow), fmt.Sprintf("A%d", settlementsStartRow), settlementHeaderStyle)

	// Settlement headers
	settlementsStartRow++
	settlementHeaders := []string{"From", "To", "Amount"}
	for i, header := range settlementHeaders {
		cell := fmt.Sprintf("%s%d", string(rune('A'+i)), settlementsStartRow)
		f.SetCellValue(sheetName, cell, header)
	}
	f.SetCellStyle(sheetName, fmt.Sprintf("A%d", settlementsStartRow), fmt.Sprintf("C%d", settlementsStartRow), headerStyle)

	// Settlement data
	for i, settlement := range settlementResult.Settlements {
		row := settlementsStartRow + 1 + i
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), settlement.From)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), settlement.To)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), settlement.Amount)
	}

	// Auto-fit columns
	f.SetColWidth(sheetName, "A", "D", 15)

	return nil
}

// createExpenseMatrixSheet creates Sheet 2: Expense Matrix
func (s *ExcelService) createExpenseMatrixSheet(f *excelize.File, trip *models.Trip, expenses []*models.Expense) error {
	sheetName := "Expense Matrix"
	f.NewSheet(sheetName)

	// Get all participants
	participantSet := make(map[string]bool)
	for _, expense := range expenses {
		if expense.SplitType == utils.SplitTypeEqual {
			for _, person := range expense.SplitAmong {
				participantSet[utils.FormatNameForDisplay(person)] = true
			}
		} else {
			for _, item := range expense.Items {
				for _, consumer := range item.Consumers {
					participantSet[utils.FormatNameForDisplay(consumer)] = true
				}
			}
		}
	}

	// Convert to sorted slice
	var participants []string
	for participant := range participantSet {
		participants = append(participants, participant)
	}
	sort.Strings(participants)

	// Set headers
	headers := []string{"Date", "Bill Name", "Paid By", "Total Amount"}
	headers = append(headers, participants...)

	for i, header := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		f.SetCellValue(sheetName, cell, header)
	}

	// Style headers
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"E6F3FF"}, Pattern: 1},
	})
	lastCol := string(rune('A' + len(headers) - 1))
	f.SetCellStyle(sheetName, "A1", fmt.Sprintf("%s1", lastCol), headerStyle)

	// Calculate expense matrix
	matrixRows := s.calculateExpenseMatrix(expenses, participants)

	// Sort by date
	sort.Slice(matrixRows, func(i, j int) bool {
		return matrixRows[i].Date < matrixRows[j].Date
	})

	// Add expense data
	for i, row := range matrixRows {
		excelRow := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", excelRow), row.Date)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", excelRow), row.BillName)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", excelRow), row.PaidBy)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", excelRow), row.TotalAmount)

		// Add person amounts
		for j, participant := range participants {
			col := string(rune('E' + j))
			amount := row.PersonAmounts[participant]
			if amount > 0 {
				f.SetCellValue(sheetName, fmt.Sprintf("%s%d", col, excelRow), amount)
			} else {
				f.SetCellValue(sheetName, fmt.Sprintf("%s%d", col, excelRow), 0)
			}
		}
	}

	// Auto-fit columns
	f.SetColWidth(sheetName, "A", lastCol, 12)
	f.SetColWidth(sheetName, "B", "B", 20) // Bill name column wider

	return nil
}

// createPaymentSheet creates Sheet 3: Payment List
func (s *ExcelService) createPaymentSheet(f *excelize.File, payments []models.Payment) error {
	sheetName := "Payments"
	f.NewSheet(sheetName)

	// Set headers
	headers := []string{"From", "To", "Amount"}
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		f.SetCellValue(sheetName, cell, header)
	}

	// Style headers
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"E6F3FF"}, Pattern: 1},
	})
	f.SetCellStyle(sheetName, "A1", "C1", headerStyle)

	// Add payment data
	for i, payment := range payments {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), utils.FormatNameForDisplay(payment.FromPerson))
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), utils.FormatNameForDisplay(payment.ToPerson))
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), payment.Amount)
	}

	// Auto-fit columns
	f.SetColWidth(sheetName, "A", "C", 15)

	return nil
}

// calculatePersonSummaries calculates spending summary for each person
func (s *ExcelService) calculatePersonSummaries(expenses []*models.Expense) []PersonSummary {
	summaryMap := make(map[string]*PersonSummary)

	for _, expense := range expenses {
		if expense.SplitType == utils.SplitTypeEqual {
			s.processEqualExpenseForSummary(expense, summaryMap)
		} else {
			s.processItemExpenseForSummary(expense, summaryMap)
		}
	}

	// Convert map to slice
	var summaries []PersonSummary
	for _, summary := range summaryMap {
		summary.NetBalance = summary.TotalSpent - summary.TotalOwed
		summaries = append(summaries, *summary)
	}

	return summaries
}

// processEqualExpenseForSummary processes equal split expense for summary
func (s *ExcelService) processEqualExpenseForSummary(expense *models.Expense, summaryMap map[string]*PersonSummary) {
	paidBy := utils.FormatNameForDisplay(expense.PaidBy)
	
	// Initialize payer if not exists
	if _, exists := summaryMap[paidBy]; !exists {
		summaryMap[paidBy] = &PersonSummary{Name: paidBy}
	}
	
	// Add to total spent
	summaryMap[paidBy].TotalSpent += expense.Amount

	// Calculate share per person
	sharePerPerson := expense.Amount / float64(len(expense.SplitAmong))

	// Add to each person's owed amount
	for _, person := range expense.SplitAmong {
		formattedName := utils.FormatNameForDisplay(person)
		if _, exists := summaryMap[formattedName]; !exists {
			summaryMap[formattedName] = &PersonSummary{Name: formattedName}
		}
		summaryMap[formattedName].TotalOwed += sharePerPerson
	}
}

// processItemExpenseForSummary processes item-based expense for summary
func (s *ExcelService) processItemExpenseForSummary(expense *models.Expense, summaryMap map[string]*PersonSummary) {
	// Process each item
	for _, item := range expense.Items {
		paidBy := utils.FormatNameForDisplay(item.PaidBy)
		
		// Initialize payer if not exists
		if _, exists := summaryMap[paidBy]; !exists {
			summaryMap[paidBy] = &PersonSummary{Name: paidBy}
		}
		
		// Add to total spent
		summaryMap[paidBy].TotalSpent += item.Amount

		// Calculate share per consumer
		sharePerPerson := item.Amount / float64(len(item.Consumers))

		// Add to each consumer's owed amount
		for _, consumer := range item.Consumers {
			formattedName := utils.FormatNameForDisplay(consumer)
			if _, exists := summaryMap[formattedName]; !exists {
				summaryMap[formattedName] = &PersonSummary{Name: formattedName}
			}
			summaryMap[formattedName].TotalOwed += sharePerPerson
		}
	}

	// Handle extra charges (tax, service, discount)
	extraCharges := expense.Tax + expense.ServiceCharge - expense.TotalDiscount
	if extraCharges != 0 {
		// Find primary payer
		primaryPayer := s.findPrimaryPayerForSummary(expense)
		formattedPayer := utils.FormatNameForDisplay(primaryPayer)
		
		if _, exists := summaryMap[formattedPayer]; !exists {
			summaryMap[formattedPayer] = &PersonSummary{Name: formattedPayer}
		}
		
		// Add extra charges to spending
		summaryMap[formattedPayer].TotalSpent += extraCharges

		// Distribute extra charges proportionally
		personItemTotals := make(map[string]float64)
		var totalItemAmount float64

		// Calculate each person's item consumption
		for _, item := range expense.Items {
			sharePerPerson := item.Amount / float64(len(item.Consumers))
			for _, consumer := range item.Consumers {
				formattedName := utils.FormatNameForDisplay(consumer)
				personItemTotals[formattedName] += sharePerPerson
			}
			totalItemAmount += item.Amount
		}

		// Distribute extra charges proportionally
		if totalItemAmount > 0 {
			for person, itemTotal := range personItemTotals {
				proportion := itemTotal / totalItemAmount
				extraChargeShare := extraCharges * proportion
				
				if _, exists := summaryMap[person]; !exists {
					summaryMap[person] = &PersonSummary{Name: person}
				}
				summaryMap[person].TotalOwed += extraChargeShare
			}
		}
	}
}

// findPrimaryPayerForSummary finds the primary payer for an expense
func (s *ExcelService) findPrimaryPayerForSummary(expense *models.Expense) string {
	payerAmounts := make(map[string]float64)
	for _, item := range expense.Items {
		payerAmounts[item.PaidBy] += item.Amount
	}

	var primaryPayer string
	var maxAmount float64
	for payer, amount := range payerAmounts {
		if amount > maxAmount {
			maxAmount = amount
			primaryPayer = payer
		}
	}

	if primaryPayer == "" {
		primaryPayer = expense.PaidBy
	}

	return primaryPayer
}

// calculateExpenseMatrix calculates the expense matrix data
func (s *ExcelService) calculateExpenseMatrix(expenses []*models.Expense, participants []string) []ExpenseMatrixRow {
	var rows []ExpenseMatrixRow

	for _, expense := range expenses {
		row := ExpenseMatrixRow{
			Date:          time.Unix(expense.CreationTime/1000, 0).Format("2006-01-02"),
			BillName:      expense.Description,
			PaidBy:        utils.FormatNameForDisplay(expense.PaidBy),
			TotalAmount:   expense.Amount,
			PersonAmounts: make(map[string]float64),
		}

		// Initialize all participants with 0
		for _, participant := range participants {
			row.PersonAmounts[participant] = 0
		}

		if expense.SplitType == utils.SplitTypeEqual {
			s.calculateEqualSplitMatrix(expense, &row)
		} else {
			s.calculateItemSplitMatrix(expense, &row)
		}

		rows = append(rows, row)
	}

	return rows
}

// calculateEqualSplitMatrix calculates matrix for equal split expense
func (s *ExcelService) calculateEqualSplitMatrix(expense *models.Expense, row *ExpenseMatrixRow) {
	sharePerPerson := expense.Amount / float64(len(expense.SplitAmong))
	
	for _, person := range expense.SplitAmong {
		formattedName := utils.FormatNameForDisplay(person)
		row.PersonAmounts[formattedName] = sharePerPerson
	}
}

// calculateItemSplitMatrix calculates matrix for item-based split expense
func (s *ExcelService) calculateItemSplitMatrix(expense *models.Expense, row *ExpenseMatrixRow) {
	// Calculate item amounts per person
	for _, item := range expense.Items {
		sharePerPerson := item.Amount / float64(len(item.Consumers))
		for _, consumer := range item.Consumers {
			formattedName := utils.FormatNameForDisplay(consumer)
			row.PersonAmounts[formattedName] += sharePerPerson
		}
	}

	// Handle extra charges proportionally
	extraCharges := expense.Tax + expense.ServiceCharge - expense.TotalDiscount
	if extraCharges != 0 {
		// Calculate each person's proportion of items
		personItemTotals := make(map[string]float64)
		var totalItemAmount float64

		for _, item := range expense.Items {
			sharePerPerson := item.Amount / float64(len(item.Consumers))
			for _, consumer := range item.Consumers {
				formattedName := utils.FormatNameForDisplay(consumer)
				personItemTotals[formattedName] += sharePerPerson
			}
			totalItemAmount += item.Amount
		}

		// Distribute extra charges proportionally
		if totalItemAmount > 0 {
			for person, itemTotal := range personItemTotals {
				proportion := itemTotal / totalItemAmount
				extraChargeShare := extraCharges * proportion
				row.PersonAmounts[person] += extraChargeShare
			}
		}
	}

	// Round all amounts
	for person, amount := range row.PersonAmounts {
		row.PersonAmounts[person] = utils.Round(amount)
	}
}