// models/models.go
package models

import "time"

// Trip represents a group of people sharing expenses
type Trip struct {
	ID           string   `json:"_id"`
	CreationTime int64    `json:"_creationTime"`
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	Participants []string `json:"participants"`
}

// Expense represents a shared expense
type Expense struct {
	ID            string   `json:"_id"`
	CreationTime  int64    `json:"_creationTime"`
	TripID        string   `json:"tripId"`
	Description   string   `json:"description"`
	Amount        float64  `json:"amount"`
	Subtotal      float64  `json:"subtotal"`
	Tax           float64  `json:"tax"`
	ServiceCharge float64  `json:"serviceCharge"`
	TotalDiscount float64  `json:"totalDiscount"`
	PaidBy        string   `json:"paidBy"`
	SplitType     string   `json:"splitType"`
	SplitAmong    []string `json:"splitAmong,omitempty"`
	Items         []Item   `json:"items,omitempty"`
	ReceiptImage  string   `json:"receiptImage,omitempty"`
}

// Item represents an individual item in an expense
type Item struct {
	Description  string   `json:"description"`
	UnitPrice    float64  `json:"unitPrice"`
	Quantity     int      `json:"quantity"`
	Amount       float64  `json:"amount,omitempty"`
	ItemDiscount float64  `json:"itemDiscount,omitempty"`
	PaidBy       string   `json:"paidBy"`
	Consumers    []string `json:"consumers"`
}

// Settlement represents a payment from one person to another
type Settlement struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Amount float64 `json:"amount"`
}

// SingleBillCalculation represents the result of calculating a single bill
type SingleBillCalculation struct {
	Amount           float64            `json:"amount"`
	Subtotal         float64            `json:"subtotal"`
	Tax              float64            `json:"tax"`
	ServiceCharge    float64            `json:"serviceCharge"`
	TotalDiscount    float64            `json:"totalDiscount"`
	PerPersonCharges map[string]float64 `json:"perPersonCharges"`
}

// SettlementResult represents the result of calculating settlements
type SettlementResult struct {
	Settlements        []Settlement       `json:"settlements"`
	IndividualBalances map[string]float64 `json:"individualBalances"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Models for receipt processing
type ClaudeResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

type ProcessedReceipt struct {
	Merchant  string        `json:"merchant"`
	Date      string        `json:"date"`
	Items     []ReceiptItem `json:"items"`
	Subtotal  float64       `json:"subtotal"`
	Tax       float64       `json:"tax"`
	Service   float64       `json:"service"`
	Discount  float64       `json:"discount"`
	Total     float64       `json:"total"`
	ImagePath string        `json:"image_path,omitempty"`
}

type ReceiptItem struct {
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
	Discount float64 `json:"discount"`
}

// CreateTrip request model
type CreateTripRequest struct {
	Name        string `json:"name" binding:"required"`
	Participant string `json:"participant" binding:"required"`
}

// GetTripByCodeRequest request model
type GetTripByCodeRequest struct {
	Code string `json:"code" binding:"required"`
}

// AddEqualExpenseRequest request model
type AddEqualExpenseRequest struct {
	Code          string   `json:"code" binding:"required"`
	Description   string   `json:"description" binding:"required"`
	Subtotal      float64  `json:"subtotal" binding:"min=0"`
	Tax           float64  `json:"tax" binding:"min=0"`
	ServiceCharge float64  `json:"serviceCharge" binding:"min=0"`
	TotalDiscount float64  `json:"totalDiscount" binding:"min=0"`
	PaidBy        string   `json:"paidBy" binding:"required"`
	SplitAmong    []string `json:"splitAmong" binding:"required,min=1"`
}

// AddItemsExpenseRequest request model
type AddItemsExpenseRequest struct {
	Code          string  `json:"code" binding:"required"`
	Description   string  `json:"description" binding:"required"`
	Tax           float64 `json:"tax" binding:"min=0"`
	ServiceCharge float64 `json:"serviceCharge" binding:"min=0"`
	TotalDiscount float64 `json:"totalDiscount" binding:"min=0"`
	Items         []Item  `json:"items" binding:"required,min=1"`
}

// RemoveExpenseRequest request model
type RemoveExpenseRequest struct {
	Code      string `json:"code" binding:"required"`
	ExpenseID string `json:"expenseId" binding:"required"`
}

// CalculateSingleBillRequest request model
type CalculateSingleBillRequest struct {
	Items         []Item  `json:"items" binding:"required,min=1"`
	Tax           float64 `json:"tax" binding:"min=0"`
	ServiceCharge float64 `json:"serviceCharge" binding:"min=0"`
	TotalDiscount float64 `json:"totalDiscount" binding:"min=0"`
}

// CreateTripResponse response model
type CreateTripResponse struct {
	TripID string `json:"tripId"`
	Code   string `json:"code"`
}

// NewTrip creates a new Trip instance
func NewTrip(id, code, name string, participant string) *Trip {
	return &Trip{
		ID:           id,
		CreationTime: time.Now().UnixMilli(),
		Code:         code,
		Name:         name,
		Participants: []string{participant},
	}
}

// NewExpense creates a new Expense instance for equal splits
func NewEqualExpense(id, tripID, description string, subtotal, tax, serviceCharge, totalDiscount float64, paidBy string, splitAmong []string) *Expense {
	totalAmount := subtotal + tax + serviceCharge - totalDiscount

	return &Expense{
		ID:            id,
		CreationTime:  time.Now().UnixMilli(),
		TripID:        tripID,
		Description:   description,
		Amount:        totalAmount,
		Subtotal:      subtotal,
		Tax:           tax,
		ServiceCharge: serviceCharge,
		TotalDiscount: totalDiscount,
		PaidBy:        paidBy,
		SplitType:     "equal",
		SplitAmong:    splitAmong,
	}
}

// NewItemExpense creates a new Expense instance for item-based splits
func NewItemExpense(id, tripID, description string, subtotal, tax, serviceCharge, totalDiscount float64, paidBy string, items []Item) *Expense {
	totalAmount := subtotal + tax + serviceCharge - totalDiscount

	return &Expense{
		ID:            id,
		CreationTime:  time.Now().UnixMilli(),
		TripID:        tripID,
		Description:   description,
		Amount:        totalAmount,
		Subtotal:      subtotal,
		Tax:           tax,
		ServiceCharge: serviceCharge,
		TotalDiscount: totalDiscount,
		PaidBy:        paidBy,
		SplitType:     "items",
		Items:         items,
	}
}
