package utils

import (
	"math/rand"
	"time"
)

// GenerateID generates a random ID for entities
func GenerateID() string {
	return generateRandomString(IDCharset, IDLength)
}

// GenerateCode generates a random trip code
func GenerateCode() string {
	return generateRandomString(CodeCharset, CodeLength)
}

// generateRandomString generates a random string with given charset and length
func generateRandomString(charset string, length int) string {
	rand.Seed(time.Now().UnixNano())
	
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}