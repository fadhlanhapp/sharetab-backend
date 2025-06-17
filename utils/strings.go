package utils

import (
	"regexp"
	"strings"
)

// NormalizeName converts a name to lowercase for storage consistency
func NormalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// FormatNameForDisplay converts a normalized name to title case for display
func FormatNameForDisplay(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	
	// Convert to lowercase and capitalize first letter
	name = strings.ToLower(name)
	return strings.ToUpper(string(name[0])) + name[1:]
}

// NormalizeNames converts a slice of names to lowercase
func NormalizeNames(names []string) []string {
	normalized := make([]string, len(names))
	for i, name := range names {
		normalized[i] = NormalizeName(name)
	}
	return normalized
}

// FormatNamesForDisplay converts a slice of names to title case
func FormatNamesForDisplay(names []string) []string {
	formatted := make([]string, len(names))
	for i, name := range names {
		formatted[i] = FormatNameForDisplay(name)
	}
	return formatted
}

// FormatNameMap converts a map with names as keys to display format
func FormatNameMapKeys[T any](input map[string]T) map[string]T {
	result := make(map[string]T)
	for name, value := range input {
		formattedName := FormatNameForDisplay(name)
		result[formattedName] = value
	}
	return result
}

// CleanFileName removes invalid characters from filename
func CleanFileName(filename string) string {
	// Replace invalid characters with underscore
	reg := regexp.MustCompile(`[<>:"/\\|?*]`)
	cleaned := reg.ReplaceAllString(filename, "_")
	
	// Remove extra spaces and trim
	cleaned = strings.TrimSpace(cleaned)
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, "_")
	
	return cleaned
}