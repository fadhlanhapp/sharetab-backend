package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// Models
type ClaudeResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

type ProcessedReceipt struct {
	Merchant    string        `json:"merchant"`
	Date        string        `json:"date"`
	Items       []ReceiptItem `json:"items"`
	Subtotal    float64       `json:"subtotal"`
	Tax         float64       `json:"tax"`
	Service     float64       `json:"service"`
	Discount    float64       `json:"discount"`
	Total       float64       `json:"total"`
	ImagePath   string        `json:"image_path,omitempty"`
}

type ReceiptItem struct {
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
	Discount float64 `json:"discount"`
}

// API Handlers
func setupRoutes(router *gin.Engine) {
	// Create uploads directory if not exists
	os.MkdirAll("uploads", os.ModePerm)

	// API routes
	api := router.Group("/api")
	{
		// Receipt processing route
		api.POST("/process-receipt", handleProcessReceipt)
	}
}

func handleProcessReceipt(c *gin.Context) {
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
	processedReceipt, err := processReceiptWithClaude(fileBytes, ext[1:], filePath)
	if err != nil {
		log.Printf("Error processing receipt with Claude: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to process receipt: %v", err),
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

// Process receipt image with Claude API
func processReceiptWithClaude(imageBytes []byte, format string, filePath string) (*ProcessedReceipt, error) {
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
		"model": "claude-3-7-sonnet-20250219",
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
							"type": "base64",
							"media_type": "image/" + format,
							"data": base64Image,
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
	var claudeResp ClaudeResponse
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

	// Parse the JSON into our structure
	var processedReceipt ProcessedReceipt
	if err := json.Unmarshal([]byte(jsonResponse), &processedReceipt); err != nil {
		return nil, fmt.Errorf("failed to parse Claude's JSON output: %v", err)
	}

	// Add the image path to the response
	processedReceipt.ImagePath = filePath

	return &processedReceipt, nil
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}
	
	// Create uploads directory if it doesn't exist
	if err := os.MkdirAll("uploads", 0755); err != nil {
		log.Fatalf("Failed to create uploads directory: %v", err)
	}
	
	// Set up Gin router
	router := gin.Default()
	
	// Configure CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Change to your frontend URL in production
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	
	// Set up routes
	setupRoutes(router)
	
	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	// Start server
	log.Printf("Server starting on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}