package utils

import "math"

// Round rounds a number to 2 decimal places for monetary calculations
func Round(num float64) float64 {
	return math.Round(num*MoneyPrecision) / MoneyPrecision
}

// Min returns the minimum of two float64 values
func Min(a, b float64) float64 {
	return math.Min(a, b)
}

// CalculateSubtotal calculates the total amount from a slice of items
func CalculateSubtotal(items []ItemAmount) float64 {
	var subtotal float64
	for _, item := range items {
		itemAmount := item.UnitPrice*float64(item.Quantity) - item.ItemDiscount
		subtotal += itemAmount
	}
	return Round(subtotal)
}

// ItemAmount represents the basic structure for items with amounts
type ItemAmount struct {
	UnitPrice    float64
	Quantity     int
	ItemDiscount float64
}