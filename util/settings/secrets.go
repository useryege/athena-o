package settings

import (
	"strings"

	log "github.com/sirupsen/logrus"
)

// ReplaceMapSecrets takes a json object and recursively looks for any secret key references in the
// object and replaces the value with the secret value
func ReplaceMapSecrets(obj map[string]any, secretValues map[string]string) map[string]any {
	newObj := make(map[string]any)
	for k, v := range obj {
		switch val := v.(type) {
		case map[string]any:
			newObj[k] = ReplaceMapSecrets(val, secretValues)
		case []any:
			newObj[k] = replaceListSecrets(val, secretValues)
		case string:
			newObj[k] = ReplaceStringSecret(val, secretValues)
		default:
			newObj[k] = val
		}
	}

	return newObj
}

func replaceListSecrets(obj []any, secretValues map[string]string) []any {
	newObj := make([]any, len(obj))
	for i, v := range obj {
		switch val := v.(type) {
		case map[string]any:
			newObj[i] = ReplaceMapSecrets(val, secretValues)
		case []any:
			newObj[i] = replaceListSecrets(val, secretValues)
		case string:
			newObj[i] = ReplaceStringSecret(val, secretValues)
		default:
			newObj[i] = val
		}
	}

	return newObj
}

// ReplaceStringSecret checks if given string is a secret key reference ( starts with $ ) and returns corresponding value from provided map
func ReplaceStringSecret(val string, secretValues map[string]string) string {
	if val == "" || !strings.HasPrefix(val, "$") {
		return val
	}

	secretKey := val[1:]
	secretVal, ok := secretValues[secretKey]
	if !ok {
		log.Warnf("config referenced '%s', but key does not exist in secret", val)
		return val
	}

	return strings.TrimSpace(secretVal)
}
