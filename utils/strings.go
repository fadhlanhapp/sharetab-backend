package utils

import "strings"

// NormalizeName converts a name to lowercase for storage consistency
func NormalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// FormatNameForDisplay converts a normalized name to title case for display
func FormatNameForDisplay(name string) string {
	return strings.Title(strings.ToLower(strings.TrimSpace(name)))
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