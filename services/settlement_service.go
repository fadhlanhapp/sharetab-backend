package services

import (
	"github.com/fadhlanhapp/sharetab-backend/models"
	"github.com/fadhlanhapp/sharetab-backend/utils"
)

// SettlementService handles settlement calculation logic
type SettlementService struct {
	expenseService *ExpenseService
}

// NewSettlementService creates a new settlement service
func NewSettlementService(expenseService *ExpenseService) *SettlementService {
	return &SettlementService{
		expenseService: expenseService,
	}
}

// CalculateSettlements calculates settlements for a trip
func (s *SettlementService) CalculateSettlements(tripID string) (*models.SettlementResult, error) {
	tripExpenses, err := s.expenseService.GetExpenses(tripID)
	if err != nil {
		return nil, utils.NewInternalError("Failed to retrieve expenses")
	}

	if len(tripExpenses) == 0 {
		return &models.SettlementResult{
			Settlements:        []models.Settlement{},
			IndividualBalances: make(map[string]float64),
		}, nil
	}

	// Calculate balances
	balances := s.calculateBalances(tripExpenses)

	// Calculate settlements
	settlements := s.calculateOptimalSettlements(balances)

	// Format names for display
	formattedBalances := utils.FormatNameMapKeys(balances)
	formattedSettlements := s.formatSettlements(settlements)

	return &models.SettlementResult{
		Settlements:        formattedSettlements,
		IndividualBalances: formattedBalances,
	}, nil
}

// calculateBalances calculates how much each person has paid and owes
func (s *SettlementService) calculateBalances(expenses []*models.Expense) map[string]float64 {
	balances := make(map[string]float64)

	for _, expense := range expenses {
		switch expense.SplitType {
		case utils.SplitTypeEqual:
			s.processEqualSplitExpense(expense, balances)
		case utils.SplitTypeItems:
			s.processItemSplitExpense(expense, balances)
		}
	}

	// Round all balances
	for person, balance := range balances {
		balances[person] = utils.Round(balance)
	}

	return balances
}

// processEqualSplitExpense processes an equal split expense
func (s *SettlementService) processEqualSplitExpense(expense *models.Expense, balances map[string]float64) {
	// The payer pays the total amount
	if _, exists := balances[expense.PaidBy]; !exists {
		balances[expense.PaidBy] = 0
	}
	balances[expense.PaidBy] += expense.Amount

	// Each person in splitAmong owes their share
	sharePerPerson := expense.Amount / float64(len(expense.SplitAmong))
	sharePerPerson = utils.Round(sharePerPerson)

	for _, person := range expense.SplitAmong {
		if _, exists := balances[person]; !exists {
			balances[person] = 0
		}
		balances[person] -= sharePerPerson
	}
}

// processItemSplitExpense processes an item-based expense
func (s *SettlementService) processItemSplitExpense(expense *models.Expense, balances map[string]float64) {
	extraCharges := expense.Tax + expense.ServiceCharge - expense.TotalDiscount

	// Calculate each person's share of items
	personItemTotals := make(map[string]float64)
	var totalItemAmount float64

	// Process each item
	for _, item := range expense.Items {
		// The payer pays for the item
		if _, exists := balances[item.PaidBy]; !exists {
			balances[item.PaidBy] = 0
		}
		balances[item.PaidBy] += item.Amount

		// Each consumer owes their share
		sharePerPerson := item.Amount / float64(len(item.Consumers))
		sharePerPerson = utils.Round(sharePerPerson)

		for _, consumer := range item.Consumers {
			if _, exists := balances[consumer]; !exists {
				balances[consumer] = 0
			}
			balances[consumer] -= sharePerPerson

			// Track consumption for proportional extra charges
			if _, exists := personItemTotals[consumer]; !exists {
				personItemTotals[consumer] = 0
			}
			personItemTotals[consumer] += sharePerPerson
		}

		totalItemAmount += item.Amount
	}

	// Handle extra charges proportionally
	if extraCharges != 0 && totalItemAmount > 0 {
		primaryPayer := s.findPrimaryPayer(expense)

		// Primary payer gets credit for paying extra charges
		if _, exists := balances[primaryPayer]; !exists {
			balances[primaryPayer] = 0
		}
		balances[primaryPayer] += extraCharges

		// Distribute extra charges proportionally
		var totalAllocated float64
		var lastPerson string

		for person, itemTotal := range personItemTotals {
			proportion := itemTotal / totalItemAmount
			extraChargeShare := extraCharges * proportion
			extraChargeShare = utils.Round(extraChargeShare)

			if _, exists := balances[person]; !exists {
				balances[person] = 0
			}
			balances[person] -= extraChargeShare
			totalAllocated += extraChargeShare
			lastPerson = person
		}

		// Handle rounding discrepancy
		roundingDiff := utils.Round(extraCharges - totalAllocated)
		if roundingDiff != 0 && lastPerson != "" {
			balances[lastPerson] -= roundingDiff
		}
	}
}

// findPrimaryPayer finds the person who paid for the most items
func (s *SettlementService) findPrimaryPayer(expense *models.Expense) string {
	payerCounts := make(map[string]float64)
	for _, item := range expense.Items {
		payerCounts[item.PaidBy] += item.Amount
	}

	var primaryPayer string
	var highestAmount float64
	for payer, amount := range payerCounts {
		if amount > highestAmount {
			highestAmount = amount
			primaryPayer = payer
		}
	}

	if primaryPayer == "" {
		primaryPayer = expense.PaidBy
	}

	return primaryPayer
}

// calculateOptimalSettlements calculates the optimal settlements
func (s *SettlementService) calculateOptimalSettlements(balances map[string]float64) []models.Settlement {
	creditors := s.extractCreditors(balances)
	debtors := s.extractDebtors(balances)

	s.sortByBalance(creditors)
	s.sortByBalance(debtors)

	return s.generateSettlements(creditors, debtors)
}

// extractCreditors extracts people who are owed money
func (s *SettlementService) extractCreditors(balances map[string]float64) []PersonBalance {
	var creditors []PersonBalance
	for person, balance := range balances {
		if balance > 0 {
			creditors = append(creditors, PersonBalance{
				Person:  person,
				Balance: balance,
			})
		}
	}
	return creditors
}

// extractDebtors extracts people who owe money
func (s *SettlementService) extractDebtors(balances map[string]float64) []PersonBalance {
	var debtors []PersonBalance
	for person, balance := range balances {
		if balance < 0 {
			debtors = append(debtors, PersonBalance{
				Person:  person,
				Balance: -balance, // Store as positive for simplicity
			})
		}
	}
	return debtors
}

// sortByBalance sorts PersonBalance slice by balance in descending order
func (s *SettlementService) sortByBalance(slice []PersonBalance) {
	for i := 0; i < len(slice); i++ {
		for j := i + 1; j < len(slice); j++ {
			if slice[i].Balance < slice[j].Balance {
				slice[i], slice[j] = slice[j], slice[i]
			}
		}
	}
}

// generateSettlements creates the actual settlement transactions
func (s *SettlementService) generateSettlements(creditors, debtors []PersonBalance) []models.Settlement {
	var settlements []models.Settlement

	i, j := 0, 0
	for i < len(creditors) && j < len(debtors) {
		creditor := creditors[i]
		debtor := debtors[j]

		amount := utils.Min(creditor.Balance, debtor.Balance)
		amount = utils.Round(amount)

		if amount > 0 {
			settlement := models.Settlement{
				From:   debtor.Person,
				To:     creditor.Person,
				Amount: amount,
			}
			settlements = append(settlements, settlement)
		}

		// Update balances
		creditors[i].Balance -= amount
		debtors[j].Balance -= amount

		// Move to next creditor/debtor if balance is settled
		if utils.Round(creditors[i].Balance) == 0 {
			i++
		}
		if utils.Round(debtors[j].Balance) == 0 {
			j++
		}
	}

	return settlements
}

// formatSettlements formats settlement names for display
func (s *SettlementService) formatSettlements(settlements []models.Settlement) []models.Settlement {
	formatted := make([]models.Settlement, len(settlements))
	for i, settlement := range settlements {
		formatted[i] = models.Settlement{
			From:   utils.FormatNameForDisplay(settlement.From),
			To:     utils.FormatNameForDisplay(settlement.To),
			Amount: settlement.Amount,
		}
	}
	return formatted
}

// PersonBalance represents a person and their balance
type PersonBalance struct {
	Person  string
	Balance float64
}