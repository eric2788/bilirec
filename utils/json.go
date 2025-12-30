package utils

import (
	"encoding/json"
	"fmt"
)

func PrettyPrintJSON(v interface{}) string {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(bytes)
}
