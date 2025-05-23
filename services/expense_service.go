package services

import (
	"fmt"
	"log"
	"math"

	"github.com/fadhlanhapp/sharetab-backend/models"
	"github.com/fadhlanhapp/sharetab-backend/repository"
)

var expenseRepo *repository.ExpenseRepository

// InitExpenseService initializes the expense service
func InitExpenseService() {
	expenseRepo = repository.NewExpenseRepository()
}

// Round rounds a number to 2 decimal places
func Round(num float64) float64 {
	return math.Round(num*100) / 100
}

// GetExpenses returns all expenses for a trip
func GetExpenses(tripID string) ([]*models.Expense, error) {
	return expenseRepo.GetExpenses(tripID)
}

// StoreExpense stores an expense for a trip
func StoreExpense(expense *models.Expense) error {
	return expenseRepo.StoreExpense(expense)
}

// RemoveExpense removes an expense from a trip
func RemoveExpense(tripID string, expenseID string) (bool, error) {
	return expenseRepo.RemoveExpense(tripID, expenseID)
}

// Updated CalculateBill function to work with both formats
func CalculateBill(items []models.Item, tax, serviceCharge, totalDiscount float64, splitType string, splitAmong []string) (*models.SingleBillCalculation, error) {
	// Calculate subtotal from items
	var subtotal float64
	perPersonCharges := make(map[string]float64)

	// Initialize all participants with zero balance
	for _, person := range splitAmong {
		perPersonCharges[person] = 0
	}

	// Process based on split type
	if splitType == "equal" {
		// For equal split, we just need the total and divide it equally

		// Calculate total from items (should be just one item for equal split)
		for _, item := range items {
			subtotal += item.UnitPrice*float64(item.Quantity) - item.ItemDiscount
		}

		// Get the payer
		var payer string
		if len(items) > 0 && items[0].PaidBy != "" {
			payer = items[0].PaidBy
		} else if len(splitAmong) > 0 {
			payer = splitAmong[0]
		} else {
			return nil, fmt.Errorf("no payer specified")
		}

		// The payer pays the full amount initially
		perPersonCharges[payer] += subtotal

		// Calculate equal share per person
		numPeople := float64(len(splitAmong))
		if numPeople == 0 {
			return nil, fmt.Errorf("no participants to split bill")
		}

		equalShare := subtotal / numPeople
		equalShare = Round(equalShare)

		// Each person (including payer) owes their equal share
		for _, person := range splitAmong {
			perPersonCharges[person] -= equalShare
		}

		// Debug output
		log.Printf("Equal split calculation: subtotal=%.2f, people=%d, share=%.2f, payer=%s",
			subtotal, int(numPeople), equalShare, payer)

	} else {
		// For itemized split, process each item separately
		for i, item := range items {
			if item.UnitPrice < 0 || item.Quantity <= 0 {
				return nil, fmt.Errorf("invalid item price or quantity")
			}

			if item.PaidBy == "" || len(item.Consumers) == 0 {
				return nil, fmt.Errorf("missing paidBy or consumers for item")
			}

			// Calculate item amount
			itemAmount := item.UnitPrice*float64(item.Quantity) - item.ItemDiscount
			itemAmount = Round(itemAmount)
			items[i].Amount = itemAmount
			subtotal += itemAmount

			// The payer pays the full amount for this item
			if _, exists := perPersonCharges[item.PaidBy]; !exists {
				perPersonCharges[item.PaidBy] = 0
			}
			perPersonCharges[item.PaidBy] += itemAmount

			// Each consumer owes their share of this item
			sharePerPerson := itemAmount / float64(len(item.Consumers))
			sharePerPerson = Round(sharePerPerson)

			for _, consumer := range item.Consumers {
				if _, exists := perPersonCharges[consumer]; !exists {
					perPersonCharges[consumer] = 0
				}
				perPersonCharges[consumer] -= sharePerPerson
			}

			// Debug output for this item
			log.Printf("Item %d calculation: description=%s, amount=%.2f, paidBy=%s, numConsumers=%d, sharePerPerson=%.2f",
				i, item.Description, itemAmount, item.PaidBy, len(item.Consumers), sharePerPerson)
		}
	}

	// Round the subtotal
	subtotal = Round(subtotal)

	// Process tax, service charge, and discount
	tax = Round(tax)
	serviceCharge = Round(serviceCharge)
	totalDiscount = Round(totalDiscount)

	// Calculate total
	totalAmount := subtotal + tax + serviceCharge - totalDiscount
	totalAmount = Round(totalAmount)

	// Process extras (tax, service, discount)
	extraCharges := tax + serviceCharge - totalDiscount
	if extraCharges != 0 {
		// Split extras evenly among all participants
		extraPerPerson := extraCharges / float64(len(splitAmong))
		extraPerPerson = Round(extraPerPerson)

		// Find the payer to add the total extra charges
		var payer string
		if len(items) > 0 && items[0].PaidBy != "" {
			payer = items[0].PaidBy
		} else if len(splitAmong) > 0 {
			payer = splitAmong[0]
		}

		// Add the extra charges to the payer
		if payer != "" {
			if _, exists := perPersonCharges[payer]; !exists {
				perPersonCharges[payer] = 0
			}
			perPersonCharges[payer] += extraCharges
		}

		// Subtract each person's share of extras
		for _, person := range splitAmong {
			if _, exists := perPersonCharges[person]; !exists {
				perPersonCharges[person] = 0
			}
			perPersonCharges[person] -= extraPerPerson
		}

		// Debug output for extras
		log.Printf("Extra charges calculation: total=%.2f, perPerson=%.2f, payer=%s",
			extraCharges, extraPerPerson, payer)
	}

	// Round all final balances
	for person, amount := range perPersonCharges {
		perPersonCharges[person] = Round(amount)
	}

	// Debug output for final balances
	log.Printf("Final balances:")
	for person, amount := range perPersonCharges {
		log.Printf("- %s: %.2f", person, amount)
	}

	// Create result
	result := &models.SingleBillCalculation{
		Amount:           totalAmount,
		Subtotal:         subtotal,
		Tax:              tax,
		ServiceCharge:    serviceCharge,
		TotalDiscount:    totalDiscount,
		PerPersonCharges: perPersonCharges,
	}

	return result, nil
}

// ProcessExpenseItems processes items for an expense and returns the subtotal and paidBy
func ProcessExpenseItems(tripID string, items []models.Item) (float64, string, error) {
	var subtotal float64
	var paidBy string
	participants := make(map[string]bool)

	for i, item := range items {
		if item.UnitPrice < 0 || item.Quantity <= 0 {
			return 0, "", fmt.Errorf("Invalid item price or quantity")
		}

		if item.PaidBy == "" || len(item.Consumers) == 0 {
			return 0, "", fmt.Errorf("Missing paidBy or consumers for item")
		}

		// Set paidBy if not set yet
		if paidBy == "" {
			paidBy = item.PaidBy
		}

		// Calculate item amount
		itemAmount := item.UnitPrice*float64(item.Quantity) - item.ItemDiscount
		itemAmount = Round(itemAmount)
		items[i].Amount = itemAmount
		subtotal += itemAmount

		// Add participants if they don't exist
		AddParticipant(tripID, item.PaidBy)
		participants[item.PaidBy] = true

		for _, consumer := range item.Consumers {
			AddParticipant(tripID, consumer)
			participants[consumer] = true
		}
	}

	return subtotal, paidBy, nil
}

// CalculateSettlements calculates settlements for a trip
func CalculateSettlements(tripID string) (*models.SettlementResult, error) {
	tripExpenses, err := GetExpenses(tripID)
	if err != nil {
		return nil, err
	}

	if len(tripExpenses) == 0 {
		return &models.SettlementResult{
			Settlements:        []models.Settlement{},
			IndividualBalances: make(map[string]float64),
		}, nil
	}

	// Calculate how much each person has paid and owes
	balances := make(map[string]float64)

	for _, expense := range tripExpenses {
		if expense.SplitType == "equal" {
			processEqualSplitExpense(expense, balances)
		} else if expense.SplitType == "items" {
			processItemSplitExpense(expense, balances)
		}
	}

	// Round all balances
	for person, balance := range balances {
		balances[person] = Round(balance)
	}

	// Calculate settlements
	settlements := calculateOptimalSettlements(balances)

	return &models.SettlementResult{
		Settlements:        settlements,
		IndividualBalances: balances,
	}, nil
}

// processEqualSplitExpense processes an equal split expense for settlement calculation
func processEqualSplitExpense(expense *models.Expense, balances map[string]float64) {
	// The payer pays the total amount
	if _, exists := balances[expense.PaidBy]; !exists {
		balances[expense.PaidBy] = 0
	}
	balances[expense.PaidBy] += expense.Amount

	// Each person in splitAmong owes their share
	sharePerPerson := expense.Amount / float64(len(expense.SplitAmong))
	sharePerPerson = Round(sharePerPerson)

	for _, person := range expense.SplitAmong {
		if _, exists := balances[person]; !exists {
			balances[person] = 0
		}
		balances[person] -= sharePerPerson
	}
}

// processItemSplitExpense processes an item-based expense for settlement calculation
func processItemSplitExpense(expense *models.Expense, balances map[string]float64) {
	// First, calculate the extra charges (tax, service, discount)
	extraCharges := expense.Tax + expense.ServiceCharge - expense.TotalDiscount

	// Calculate each person's share of items (for proportional extra charge distribution)
	personItemTotals := make(map[string]float64)
	var totalItemAmount float64

	// Process each item first
	for _, item := range expense.Items {
		// The payer pays for the item
		if _, exists := balances[item.PaidBy]; !exists {
			balances[item.PaidBy] = 0
		}
		balances[item.PaidBy] += item.Amount

		// Each consumer owes their share of this item
		sharePerPerson := item.Amount / float64(len(item.Consumers))
		sharePerPerson = Round(sharePerPerson)

		for _, consumer := range item.Consumers {
			if _, exists := balances[consumer]; !exists {
				balances[consumer] = 0
			}
			balances[consumer] -= sharePerPerson
			
			// Track each person's total consumption for proportional extra charges
			if _, exists := personItemTotals[consumer]; !exists {
				personItemTotals[consumer] = 0
			}
			personItemTotals[consumer] += sharePerPerson
		}
		
		totalItemAmount += item.Amount
	}

	// Handle extra charges (tax, service, discount) if any
	if extraCharges != 0 && totalItemAmount > 0 {
		// Find the person who paid for the most items to assign extra charges
		payerCounts := make(map[string]float64)
		for _, item := range expense.Items {
			payerCounts[item.PaidBy] += item.Amount
		}
		
		// Find the payer with the highest total
		var primaryPayer string
		var highestAmount float64
		for payer, amount := range payerCounts {
			if amount > highestAmount {
				highestAmount = amount
				primaryPayer = payer
			}
		}
		
		// If no primary payer found, use expense.PaidBy as fallback
		if primaryPayer == "" {
			primaryPayer = expense.PaidBy
		}

		// The primary payer gets credit for paying the full extra charges
		if _, exists := balances[primaryPayer]; !exists {
			balances[primaryPayer] = 0
		}
		balances[primaryPayer] += extraCharges

		// Each person owes their proportional share of extra charges based on their item consumption
		var totalAllocated float64
		var lastPerson string
		
		for person, itemTotal := range personItemTotals {
			proportion := itemTotal / totalItemAmount
			extraChargeShare := extraCharges * proportion
			extraChargeShare = Round(extraChargeShare)
			
			if _, exists := balances[person]; !exists {
				balances[person] = 0
			}
			balances[person] -= extraChargeShare
			totalAllocated += extraChargeShare
			lastPerson = person
		}
		
		// Handle rounding discrepancy by adjusting the last person
		roundingDiff := Round(extraCharges - totalAllocated)
		if roundingDiff != 0 && lastPerson != "" {
			balances[lastPerson] -= roundingDiff
		}
	}
}

// calculateOptimalSettlements calculates the optimal settlements
func calculateOptimalSettlements(balances map[string]float64) []models.Settlement {
	// Separate creditors and debtors
	var creditors []struct {
		Person  string
		Balance float64
	}

	var debtors []struct {
		Person  string
		Balance float64
	}

	for person, balance := range balances {
		if balance > 0 {
			creditors = append(creditors, struct {
				Person  string
				Balance float64
			}{Person: person, Balance: balance})
		} else if balance < 0 {
			debtors = append(debtors, struct {
				Person  string
				Balance float64
			}{Person: person, Balance: -balance}) // Store as positive for simplicity
		}
	}

	// Sort creditors and debtors by balance (descending)
	sort := func(slice interface{}, less func(i, j int) bool) {
		switch v := slice.(type) {
		case []struct {
			Person  string
			Balance float64
		}:
			for i := 0; i < len(v); i++ {
				for j := i + 1; j < len(v); j++ {
					if less(i, j) {
						v[i], v[j] = v[j], v[i]
					}
				}
			}
		}
	}

	sort(creditors, func(i, j int) bool {
		return creditors[i].Balance > creditors[j].Balance
	})

	sort(debtors, func(i, j int) bool {
		return debtors[i].Balance > debtors[j].Balance
	})

	// Calculate settlements
	var settlements []models.Settlement

	i, j := 0, 0
	for i < len(creditors) && j < len(debtors) {
		creditor := creditors[i]
		debtor := debtors[j]

		// Calculate the settlement amount
		amount := math.Min(creditor.Balance, debtor.Balance)
		amount = Round(amount)

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
		if Round(creditors[i].Balance) == 0 {
			i++
		}
		if Round(debtors[j].Balance) == 0 {
			j++
		}
	}

	return settlements
}

// ConvertReceiptItemToExpenseItem converts a receipt item to an expense item
func ConvertReceiptItemToExpenseItem(receiptItem models.ReceiptItem, paidBy string, consumers []string) models.Item {
	return models.Item{
		Description:  receiptItem.Name,
		UnitPrice:    receiptItem.Price,
		Quantity:     int(receiptItem.Quantity),
		ItemDiscount: receiptItem.Discount,
		PaidBy:       paidBy,
		Consumers:    consumers,
		Amount:       Round(receiptItem.Price*receiptItem.Quantity - receiptItem.Discount),
	}
}
