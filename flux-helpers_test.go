package main

import (
	"os"
	"testing"
	"encoding/json"

	"sigs.k8s.io/yaml"
)

// extractRawValues extracts the "values" field from the "spec" section of a YAML document,
// marshals it into JSON format, and returns the resulting byte slice.
// 
// Parameters:
//   - data: A byte slice containing the YAML document.
//
// Returns:
//   - A byte slice containing the JSON representation of the "values" field.
//
// Note:
//   - The function assumes that the input YAML document has a "spec" field
//     which contains a "values" field. If these fields are missing or have
//     an unexpected structure, the behavior is undefined.
func extractRawValues(data []byte) []byte {
	var hr map[string]interface{}
	_ = yaml.Unmarshal(data, &hr)
	spec := hr["spec"].(map[string]interface{})
	values := spec["values"]
	raw, _ := json.Marshal(values)
	return raw
}

// TestBumpTagInValues tests the functionality of the BumpTagInValues function.
// It verifies that the function correctly updates the tag of a specified image
// in a given set of values. The test reads a YAML file containing test data,
// unmarshals it into a map, and then calls BumpTagInValues with the specified
// image and tag. The test ensures that the function returns no errors and
// that an update is performed as expected.
func TestBumpTagInValues(t *testing.T) {
	data, err := os.ReadFile("test_files/multiple-bump.yaml")
	if err != nil {
		t.Fatalf("Failed to read test YAML: %v", err)
	}

	var values map[string]interface{}
	err = json.Unmarshal(extractRawValues(data), &values)
	if err != nil {
		t.Fatalf("Failed to unmarshal values: %v", err)
	}

	updated, err := BumpTagInValues(values, "ghcr.io/my-org/my-api", "1.3.99", false)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !updated {
		t.Errorf("Expected update but got none")
	}
}
