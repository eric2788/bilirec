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

func TruncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen])
	}
	return s
}

func AsciiFallback(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r == '"' || r == '\\' || r < 0x20 {
			continue
		}
		if r > 127 {
			b.WriteByte('?')
		} else {
			b.WriteRune(r)
		}
	}
	s := strings.ReplaceAll(b.String(), "/", "-")
	if s == "" {
		s = "file"
	}
	return s
}
