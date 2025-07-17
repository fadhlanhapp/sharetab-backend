// services/receipt_service.go
package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/fadhlanhapp/sharetab-backend/models"
	"github.com/fadhlanhapp/sharetab-backend/utils"
)

// ProcessReceiptWithClaude processes a receipt image using Claude API
func ProcessReceiptWithClaude(imageBytes []byte, format string, filePath string) (*models.ProcessedReceipt, error) {
	// Encode image to base64
	base64Image := base64.StdEncoding.EncodeToString(imageBytes)

	// Get Claude API key from environment
	claudeAPIKey := os.Getenv("ANTHROPIC_API_KEY")
	if claudeAPIKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
	}

	// Create Claude API request
	claudeURL := "https://api.anthropic.com/v1/messages"

	// Detailed prompt for consistent JSON extraction
	prompt := `Extract receipt data in this JSON format:
{
  "merchant": "store name",
  "date": "YYYY-MM-DD",
  "items": [
    {
      "name": "item name",
      "price": number,
      "quantity": number,
      "discount": number
    }
  ],
  "subtotal": number,
  "tax": number,
  "service": number,
  "discount": number,
  "total": number
}
Return only valid JSON. No explanations or formatting.`

	// Construct Claude API request body
	requestBody := map[string]interface{}{
		"model":      "claude-sonnet-4-20250514",
		"max_tokens": 4000,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": prompt,
					},
					{
						"type": "image",
						"source": map[string]interface{}{
							"type":       "base64",
							"media_type": "image/" + format,
							"data":       base64Image,
						},
					},
				},
			},
		},
	}

	// Convert the request body to JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Claude API request: %v", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", claudeURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create Claude API request: %v", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", claudeAPIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Send the request
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to Claude API: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Claude API returned non-200 status: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse the response
	var claudeResp models.ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return nil, fmt.Errorf("failed to decode Claude API response: %v", err)
	}

	// Extract the JSON string from Claude's response
	var jsonResponse string
	for _, content := range claudeResp.Content {
		if content.Type == "text" {
			jsonResponse = content.Text
			break
		}
	}
	
	// Check if we found any text content
	if jsonResponse == "" {
		return nil, fmt.Errorf("no text content found in Claude's response")
	}

	// Parse the JSON into our structure
	var processedReceipt models.ProcessedReceipt
	if err := json.Unmarshal([]byte(jsonResponse), &processedReceipt); err != nil {
		return nil, fmt.Errorf("failed to parse Claude's JSON output: %v. Raw response: %s", err, jsonResponse)
	}

	// Add the image path to the response
	processedReceipt.ImagePath = filePath

	return &processedReceipt, nil
}

// CreateExpenseFromReceipt creates an expense from a processed receipt
func CreateExpenseFromReceipt(trip *models.Trip, receipt *models.ProcessedReceipt, paidBy string, splitType string,
	splitAmong, defaultConsumers []string, imagePath string) (*models.Expense, error) {

	// Generate expense ID
	expenseID := utils.GenerateID()

	// Set expense description
	expenseDescription := receipt.Merchant
	if expenseDescription == "" {
		expenseDescription = "Receipt " + time.Now().Format("2006-01-02")
	}

	if splitType == "equal" {
		// Normalize names
		normalizedPaidBy := utils.NormalizeName(paidBy)
		normalizedSplitAmong := utils.NormalizeNames(splitAmong)

		// Add participants if they don't exist
		for _, participant := range normalizedSplitAmong {
			err := AddParticipant(trip.ID, participant)
			if err != nil {
				return nil, fmt.Errorf("failed to add participant %s: %v", participant, err)
			}
		}

		// Create equal split expense
		expense := &models.Expense{
			ID:            expenseID,
			CreationTime:  time.Now().UnixMilli(),
			TripID:        trip.ID,
			Description:   expenseDescription,
			Amount:        utils.Round(receipt.Total),
			Subtotal:      utils.Round(receipt.Subtotal),
			Tax:           utils.Round(receipt.Tax),
			ServiceCharge: utils.Round(receipt.Service),
			TotalDiscount: utils.Round(receipt.Discount),
			PaidBy:        normalizedPaidBy,
			SplitType:     utils.SplitTypeEqual,
			SplitAmong:    normalizedSplitAmong,
			ReceiptImage:  imagePath,
		}

		// FIXED: Changed StoreExpense(trip.ID, expense) to StoreExpense(expense)
		err := StoreExpense(expense)
		if err != nil {
			return nil, fmt.Errorf("failed to store expense: %v", err)
		}

		return expense, nil
	} else {
		// Normalize names
		normalizedPaidBy := utils.NormalizeName(paidBy)
		normalizedDefaultConsumers := utils.NormalizeNames(defaultConsumers)

		// Create items-based expense
		expenseItems := make([]models.Item, 0, len(receipt.Items))

		for _, receiptItem := range receipt.Items {
			item := ConvertReceiptItemToExpenseItem(receiptItem, normalizedPaidBy, normalizedDefaultConsumers)
			expenseItems = append(expenseItems, item)

			// Add participants
			err := AddParticipant(trip.ID, normalizedPaidBy)
			if err != nil {
				return nil, fmt.Errorf("failed to add participant %s: %v", normalizedPaidBy, err)
			}

			for _, consumer := range normalizedDefaultConsumers {
				err := AddParticipant(trip.ID, consumer)
				if err != nil {
					return nil, fmt.Errorf("failed to add participant %s: %v", consumer, err)
				}
			}
		}

		expense := &models.Expense{
			ID:            expenseID,
			CreationTime:  time.Now().UnixMilli(),
			TripID:        trip.ID,
			Description:   expenseDescription,
			Amount:        utils.Round(receipt.Total),
			Subtotal:      utils.Round(receipt.Subtotal),
			Tax:           utils.Round(receipt.Tax),
			ServiceCharge: utils.Round(receipt.Service),
			TotalDiscount: utils.Round(receipt.Discount),
			PaidBy:        normalizedPaidBy,
			SplitType:     utils.SplitTypeItems,
			Items:         expenseItems,
			ReceiptImage:  imagePath,
		}

		// FIXED: Changed StoreExpense(trip.ID, expense) to StoreExpense(expense)
		err := StoreExpense(expense)
		if err != nil {
			return nil, fmt.Errorf("failed to store expense: %v", err)
		}

		return expense, nil
	}
}
