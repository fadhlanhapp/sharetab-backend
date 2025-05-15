// repository/expense_repository.go
package repository

import (
	"database/sql"
	"fmt"

	"github.com/fadhlanhapp/sharetab-backend/models"
)

// ExpenseRepository handles database operations for expenses
type ExpenseRepository struct {
	DB *sql.DB
}

// NewExpenseRepository creates a new ExpenseRepository
func NewExpenseRepository() *ExpenseRepository {
	return &ExpenseRepository{
		DB: GetDB(),
	}
}

// StoreExpense saves an expense to the database
func (r *ExpenseRepository) StoreExpense(expense *models.Expense) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Insert expense
	_, err = tx.Exec(
		`INSERT INTO expenses 
         (id, trip_id, description, amount, subtotal, tax, service_charge, total_discount, 
          paid_by, split_type, creation_time, receipt_image) 
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		expense.ID, expense.TripID, expense.Description, expense.Amount, expense.Subtotal,
		expense.Tax, expense.ServiceCharge, expense.TotalDiscount, expense.PaidBy,
		expense.SplitType, expense.CreationTime, expense.ReceiptImage,
	)
	if err != nil {
		return fmt.Errorf("failed to insert expense: %v", err)
	}

	// Insert participants or items based on split type
	if expense.SplitType == "equal" {
		for _, participant := range expense.SplitAmong {
			_, err = tx.Exec(
				"INSERT INTO expense_participants (expense_id, participant) VALUES ($1, $2)",
				expense.ID, participant,
			)
			if err != nil {
				return fmt.Errorf("failed to insert expense participant: %v", err)
			}
		}
	} else if expense.SplitType == "items" {
		for _, item := range expense.Items {
			var itemID int
			err = tx.QueryRow(
				`INSERT INTO expenses_items 
                 (expense_id, description, unit_price, quantity, amount, item_discount, paid_by) 
                 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
				expense.ID, item.Description, item.UnitPrice, item.Quantity, item.Amount,
				item.ItemDiscount, item.PaidBy,
			).Scan(&itemID)
			if err != nil {
				return fmt.Errorf("failed to insert expense item: %v", err)
			}

			// Insert consumers for the item
			for _, consumer := range item.Consumers {
				_, err = tx.Exec(
					"INSERT INTO item_consumers (item_id, consumer) VALUES ($1, $2)",
					itemID, consumer,
				)
				if err != nil {
					return fmt.Errorf("failed to insert item consumer: %v", err)
				}
			}
		}
	}

	return tx.Commit()
}

// GetExpenses retrieves all expenses for a trip
func (r *ExpenseRepository) GetExpenses(tripID string) ([]*models.Expense, error) {
	// Query expenses
	rows, err := r.DB.Query(
		`SELECT id, trip_id, description, amount, subtotal, tax, service_charge, 
          total_discount, paid_by, split_type, creation_time, receipt_image 
         FROM expenses WHERE trip_id = $1 ORDER BY creation_time ASC`,
		tripID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get expenses: %v", err)
	}
	defer rows.Close()

	var expenses []*models.Expense
	for rows.Next() {
		var expense models.Expense
		var receiptImage sql.NullString

		err = rows.Scan(
			&expense.ID, &expense.TripID, &expense.Description, &expense.Amount,
			&expense.Subtotal, &expense.Tax, &expense.ServiceCharge, &expense.TotalDiscount,
			&expense.PaidBy, &expense.SplitType, &expense.CreationTime, &receiptImage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan expense: %v", err)
		}

		if receiptImage.Valid {
			expense.ReceiptImage = receiptImage.String
		}

		// Load participants or items based on split type
		if expense.SplitType == "equal" {
			// Get participants
			pRows, err := r.DB.Query(
				"SELECT participant FROM expense_participants WHERE expense_id = $1",
				expense.ID,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to get expense participants: %v", err)
			}
			defer pRows.Close()

			for pRows.Next() {
				var participant string
				if err := pRows.Scan(&participant); err != nil {
					return nil, fmt.Errorf("failed to scan participant: %v", err)
				}
				expense.SplitAmong = append(expense.SplitAmong, participant)
			}
		} else if expense.SplitType == "items" {
			// Get items
			iRows, err := r.DB.Query(
				`SELECT id, description, unit_price, quantity, amount, item_discount, paid_by
                 FROM expenses_items WHERE expense_id = $1`,
				expense.ID,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to get expense items: %v", err)
			}
			defer iRows.Close()

			for iRows.Next() {
				var item models.Item
				var itemID int
				if err := iRows.Scan(&itemID, &item.Description, &item.UnitPrice, &item.Quantity,
					&item.Amount, &item.ItemDiscount, &item.PaidBy); err != nil {
					return nil, fmt.Errorf("failed to scan item: %v", err)
				}

				// Get consumers for this item
				cRows, err := r.DB.Query(
					"SELECT consumer FROM item_consumers WHERE item_id = $1",
					itemID,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to get item consumers: %v", err)
				}
				defer cRows.Close()

				for cRows.Next() {
					var consumer string
					if err := cRows.Scan(&consumer); err != nil {
						return nil, fmt.Errorf("failed to scan consumer: %v", err)
					}
					item.Consumers = append(item.Consumers, consumer)
				}

				expense.Items = append(expense.Items, item)
			}
		}

		expenses = append(expenses, &expense)
	}

	return expenses, nil
}

// RemoveExpense removes an expense
func (r *ExpenseRepository) RemoveExpense(tripID string, expenseID string) (bool, error) {
	// First check if expense exists and belongs to the trip
	var count int
	err := r.DB.QueryRow(
		"SELECT COUNT(*) FROM expenses WHERE id = $1 AND trip_id = $2",
		expenseID, tripID,
	).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("failed to check expense: %v", err)
	}

	if count == 0 {
		return false, nil // Expense not found or doesn't belong to trip
	}

	// Delete expense (cascade will delete participants and items)
	_, err = r.DB.Exec("DELETE FROM expenses WHERE id = $1", expenseID)
	if err != nil {
		return false, fmt.Errorf("failed to delete expense: %v", err)
	}

	return true, nil
}
