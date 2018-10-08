package models

import (
	"fmt"
	"testing"
)

func ValidateString(entry ConfigEntry) (bool, error) {
	if len(entry.Value) <= 3 {
		return false, fmt.Errorf("It should larger than 3 characters")
	}
	return true, nil

}
func TestUserConfigEntry(t *testing.T) {
	testItem := UserConfigItem{
		Name:      "test",
		Type:      "string",
		Validator: ValidateString,
	}

	result, err := testItem.Validate(
		ConfigEntry{
			Key:   "test",
			Value: "abcd",
		},
	)

	if err != nil {
		t.Errorf("Error occurred when : %v", err)
	}

	fmt.Printf("message need to print,%v\n", result)

}
