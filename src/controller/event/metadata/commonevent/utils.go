package commonevent

import (
	"encoding/json"
	"log"
)

// replacePassword recursively replaces "user_password" values with "***"
func replacePassword(data map[string]interface{}, maskAttributes []string) {
	for key, value := range data {
		if inAttributes(key, maskAttributes) {
			data[key] = "***"
		} else if nestedMap, ok := value.(map[string]interface{}); ok {
			replacePassword(nestedMap, maskAttributes)
		} else if nestedArray, ok := value.([]interface{}); ok {
			for _, item := range nestedArray {
				if itemMap, ok := item.(map[string]interface{}); ok {
					replacePassword(itemMap, maskAttributes)
				}
			}
		}
	}
}

func inAttributes(key string, maskAttributes []string) bool {
	for _, attribute := range maskAttributes {
		if key == attribute {
			return true
		}
	}
	return false
}

func RedactPayload(payload string, sensitiveAttributes []string) string {
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &jsonData); err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
		return ""
	}
	// Replace user_password values
	replacePassword(jsonData, sensitiveAttributes)
	// Convert the modified map back to JSON
	modifiedJSON, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		log.Fatalf("Error converting to JSON: %v", err)
		return ""
	}
	return string(modifiedJSON)
}
