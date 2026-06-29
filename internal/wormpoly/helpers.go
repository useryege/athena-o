package wormpoly

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func boolValue(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func float64Value(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func parseHotMarketStringList(raw *string) []string {
	value := strings.TrimSpace(stringValue(raw))
	if value == "" {
		return []string{}
	}

	var stringValues []string
	if err := json.Unmarshal([]byte(value), &stringValues); err == nil {
		return cleanStringList(stringValues)
	}

	var mixedValues []any
	if err := json.Unmarshal([]byte(value), &mixedValues); err == nil {
		out := make([]string, 0, len(mixedValues))
		for i := range mixedValues {
			switch typed := mixedValues[i].(type) {
			case string:
				out = append(out, typed)
			case float64:
				out = append(out, strconv.FormatFloat(typed, 'f', -1, 64))
			case bool:
				out = append(out, strconv.FormatBool(typed))
			default:
				encoded, err := json.Marshal(typed)
				if err == nil {
					out = append(out, string(encoded))
				}
			}
		}
		return cleanStringList(out)
	}

	if strings.Contains(value, ",") {
		return cleanStringList(strings.Split(value, ","))
	}
	return cleanStringList([]string{value})
}

func cleanStringList(values []string) []string {
	out := make([]string, 0, len(values))
	for i := range values {
		value := strings.TrimSpace(values[i])
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func parseFloatOrZero(value string) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0
	}
	return parsed
}

func formatTimeRFC3339(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
