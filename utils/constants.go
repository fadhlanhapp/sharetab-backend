package utils

const (
	// Split types
	SplitTypeEqual = "equal"
	SplitTypeItems = "items"

	// ID and code generation
	IDCharset   = "abcdefghijklmnopqrstuvwxyz0123456789"
	CodeCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	IDLength    = 20
	CodeLength  = 6

	// HTTP status messages
	ErrInvalidRequest     = "Invalid request"
	ErrTripNotFound      = "Trip not found"
	ErrExpenseNotFound   = "Expense not found"
	ErrFailedToStore     = "Failed to store data"
	ErrFailedToRetrieve  = "Failed to retrieve data"
	ErrInvalidItemData   = "Invalid item price or quantity"
	ErrMissingPaidBy     = "Missing paidBy or consumers for item"
	ErrCodeRequired      = "Code is required"

	// Precision for monetary calculations
	MoneyPrecision = 100.0
)