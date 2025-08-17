// handlers/receipt_handlers.go
package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fadhlanhapp/sharetab-backend/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HandleProcessReceiptV1 processes a receipt image using Claude (v1 API)
func HandleProcessReceiptV1(c *gin.Context) {
	handleProcessReceiptImpl(c)
}


// handleProcessReceiptImpl implements the receipt processing logic
func handleProcessReceiptImpl(c *gin.Context) {
	// 1. Receive the image file
	file, header, err := c.Request.FormFile("receipt")
	if err != nil {
		log.Printf("Error receiving file: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("No file uploaded or invalid form: %v", err)})
		return
	}
	defer file.Close()

	// Log file info for debugging
	log.Printf("Received file: %s, Size: %d, Content-Type: %s",
		header.Filename, header.Size, header.Header.Get("Content-Type"))

	// Check file type
	ext := filepath.Ext(header.Filename)
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		log.Printf("Invalid file type: %s", ext)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only JPG, JPEG, and PNG files are supported"})
		return
	}

	// Generate unique filename
	filename := uuid.New().String() + ext
	filePath := filepath.Join("uploads", filename)
	log.Printf("Saving file to: %s", filePath)

	// Create the file
	out, err := os.Create(filePath)
	if err != nil {
		log.Printf("Error creating file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save file: %v", err)})
		return
	}
	defer out.Close()

	// Copy the uploaded file to the created file
	bytesWritten, err := io.Copy(out, file)
	if err != nil {
		log.Printf("Error copying file data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save file: %v", err)})
		return
	}
	log.Printf("Successfully saved %d bytes to %s", bytesWritten, filePath)

	// Read the file again for base64 encoding
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Error reading saved file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read saved file: %v", err)})
		return
	}
	log.Printf("Successfully read %d bytes from file for processing", len(fileBytes))

	// 2. Process the image using Claude API
	log.Printf("Calling Claude API to process receipt...")
	processedReceipt, err := services.ProcessReceiptWithClaude(fileBytes, ext[1:], filePath)
	if err != nil {
		log.Printf("Error processing receipt with Claude: %v", err)
		
		// Parse error type for better user messaging
		errorMsg := err.Error()
		statusCode := http.StatusInternalServerError
		userFriendlyMsg := "Failed to process receipt"
		
		if strings.HasPrefix(errorMsg, "receipt_processing_failed:") {
			userFriendlyMsg = "Unable to read the receipt. Please ensure the image is clear and shows a complete receipt."
			statusCode = http.StatusBadRequest
		} else if strings.HasPrefix(errorMsg, "invalid_receipt:") {
			userFriendlyMsg = "The uploaded image does not appear to be a valid receipt. Please upload a photo of a receipt."
			statusCode = http.StatusBadRequest
		} else if strings.HasPrefix(errorMsg, "receipt_format_error:") {
			userFriendlyMsg = "Cannot read the receipt format. Please ensure the image is clear and not blurry."
			statusCode = http.StatusBadRequest
		} else if strings.HasPrefix(errorMsg, "invalid_receipt_data:") {
			userFriendlyMsg = "No items or amounts found in the receipt. Please ensure the entire receipt is visible."
			statusCode = http.StatusBadRequest
		}
		
		c.JSON(statusCode, gin.H{
			"error": userFriendlyMsg,
			"details": errorMsg, // Include full error for debugging
		})
		
		// Clean up the image even on error
		if err := os.Remove(filePath); err != nil {
			log.Printf("Failed to delete image file after error: %v", err)
		} else {
			log.Printf("Deleted image file after processing error: %s", filePath)
		}
		return
	}

	log.Printf("Successfully processed receipt. Merchant: %s, Total: %.2f",
		processedReceipt.Merchant, processedReceipt.Total)

	// Delete the image file after successful processing
	if err := os.Remove(filePath); err != nil {
		log.Printf("Warning: Failed to delete image file after processing: %v", err)
	} else {
		log.Printf("Successfully deleted image file after processing: %s", filePath)
		// Since the file is deleted, remove the path from the response
		processedReceipt.ImagePath = ""
	}

	// 3. Return the processed data
	c.JSON(http.StatusOK, processedReceipt)
}

// AddExpenseFromReceiptV1 creates an expense from a receipt image (v1 API)
func AddExpenseFromReceiptV1(c *gin.Context) {
	addExpenseFromReceiptImpl(c)
}


// addExpenseFromReceiptImpl implements the expense from receipt logic
func addExpenseFromReceiptImpl(c *gin.Context) {
	// Parse multipart form
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to parse form: %v", err)})
		return
	}

	// Get trip code
	tripCode := c.Request.FormValue("code")
	if tripCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing trip code"})
		return
	}

	// Get paidBy
	paidBy := c.Request.FormValue("paidBy")
	if paidBy == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing paidBy field"})
		return
	}

	// Get splitType
	splitType := c.Request.FormValue("splitType")
	if splitType != "equal" && splitType != "items" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid split type. Must be 'equal' or 'items'"})
		return
	}

	// Get splitAmong (for equal split) or defaultConsumers (for items split)
	var splitAmong []string
	var defaultConsumers []string

	if splitType == "equal" {
		splitAmongStr := c.Request.FormValue("splitAmong")
		if splitAmongStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing splitAmong field for equal split"})
			return
		}
		splitAmong = strings.Split(splitAmongStr, ",")
	} else {
		defaultConsumersStr := c.Request.FormValue("defaultConsumers")
		if defaultConsumersStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing defaultConsumers field for items split"})
			return
		}
		defaultConsumers = strings.Split(defaultConsumersStr, ",")
	}

	// Receive the image file
	file, header, err := c.Request.FormFile("receipt")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("No file uploaded or invalid form: %v", err)})
		return
	}
	defer file.Close()

	// Check file type
	ext := filepath.Ext(header.Filename)
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only JPG, JPEG, and PNG files are supported"})
		return
	}

	// Generate unique filename
	filename := uuid.New().String() + ext
	filePath := filepath.Join("uploads", filename)

	// Create the file
	out, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save file: %v", err)})
		return
	}
	defer out.Close()

	// Copy the uploaded file to the created file
	if _, err := io.Copy(out, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save file: %v", err)})
		return
	}

	// Read the file again for base64 encoding
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read saved file: %v", err)})
		return
	}

	// Process the image using Claude API
	processedReceipt, err := services.ProcessReceiptWithClaude(fileBytes, ext[1:], filePath)
	if err != nil {
		log.Printf("Error processing receipt with Claude: %v", err)
		
		// Parse error type for better user messaging
		errorMsg := err.Error()
		statusCode := http.StatusInternalServerError
		userFriendlyMsg := "Failed to process receipt"
		
		if strings.HasPrefix(errorMsg, "receipt_processing_failed:") {
			userFriendlyMsg = "Unable to read the receipt. Please ensure the image is clear and shows a complete receipt."
			statusCode = http.StatusBadRequest
		} else if strings.HasPrefix(errorMsg, "invalid_receipt:") {
			userFriendlyMsg = "The uploaded image does not appear to be a valid receipt. Please upload a photo of a receipt."
			statusCode = http.StatusBadRequest
		} else if strings.HasPrefix(errorMsg, "receipt_format_error:") {
			userFriendlyMsg = "Cannot read the receipt format. Please ensure the image is clear and not blurry."
			statusCode = http.StatusBadRequest
		} else if strings.HasPrefix(errorMsg, "invalid_receipt_data:") {
			userFriendlyMsg = "No items or amounts found in the receipt. Please ensure the entire receipt is visible."
			statusCode = http.StatusBadRequest
		}
		
		c.JSON(statusCode, gin.H{
			"error": userFriendlyMsg,
			"details": errorMsg, // Include full error for debugging
		})
		
		// Clean up the image on error
		os.Remove(filePath)
		return
	}

	// Get services
	tripService := services.NewTripService()
	
	// Get trip by code
	trip, err := tripService.GetTripByCode(tripCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		// Clean up the image on error
		os.Remove(filePath)
		return
	}

	// Create expense from receipt
	expense, err := services.CreateExpenseFromReceipt(trip, processedReceipt, paidBy, splitType, splitAmong, defaultConsumers, filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create expense: %v", err)})
		// Clean up the image on error
		os.Remove(filePath)
		return
	}

	c.JSON(http.StatusOK, expense)
}
