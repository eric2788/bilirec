package utils

import (
	"encoding/json"
	"fmt"
	"strings"
)

func PrettyPrintJSON(v interface{}) string {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(bytes)
}

// Helper to remove invalid filename characters
func SanitizeFilename(name string) string {
	// Replace invalid characters with underscore
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		".", "_",
	)
	return replacer.Replace(name)
}

func TruncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}
