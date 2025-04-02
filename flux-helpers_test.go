package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sigs.k8s.io/yaml"
	"strings"
	"testing"
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

// TestBumpTagInValuesUniversal tests the BumpTagInValuesUniversal function to ensure
// it correctly updates image tags in a values map based on the provided parameters.
//
// The test reads a sample YAML file, unmarshals it into a map, and runs multiple
// test cases to validate the behavior of the function under different scenarios.
//
// Test cases include:
// - Updating an existing image tag with a valid new version.
// - Performing a dry-run update to ensure no changes are persisted.
// - Attempting to update a non-existent image tag.
// - Providing an invalid version string for an update.
//
// Each test case verifies:
// - Whether the function returns the expected update status (updated or not).
// - Whether the function returns an error when expected.
//
// The test uses table-driven testing to iterate through multiple scenarios,
// ensuring comprehensive coverage of the function's behavior.
func TestBumpTagInValuesUniversal(t *testing.T) {
	data, err := os.ReadFile("test_files/aspire-test.yaml")
	if err != nil {
		t.Fatalf("Failed to read test YAML: %v", err)
	}

	var values map[string]interface{}
	err = json.Unmarshal(extractRawValues(data), &values)
	if err != nil {
		t.Fatalf("Failed to unmarshal values: %v", err)
	}

	tests := []struct {
		imageName    string
		newVersion   string
		dryRun       bool
		expectUpdate bool
		expectError  bool
	}{
		{"ghcr.io/my-org/web-app", "1.3.99", false, true, false},
		{"ghcr.io/my-org/api-service", "1.3.99", true, true, false},
		{"ghcr.io/my-org/non-existent", "1.0.0", false, false, false},
		{"ghcr.io/my-org/my-api", "invalid-version", false, false, false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("imageName=%s,newVersion=%s,dryRun=%v", tt.imageName, tt.newVersion, tt.dryRun), func(t *testing.T) {
			updated, err := BumpTagInValuesUniversal(values, tt.imageName, tt.newVersion, tt.dryRun)
			if (err != nil) != tt.expectError {
				t.Errorf("Unexpected error: %v", err)
			}
			if updated != tt.expectUpdate {
				t.Errorf("Expected update: %v, got: %v", tt.expectUpdate, updated)
			}
		})
	}
}

// TestBumpTagInValuesUniversalTwo tests the functionality of the BumpTagInValuesUniversal
// function to ensure it correctly updates image tags in a nested map structure.
//
// The test performs the following steps:
// 1. Initializes a map structure `values` containing image information in two formats:
//   - A structured block with "repository" and "tag" keys.
//   - A flat Aspire-style format with image references as strings.
//     2. Calls BumpTagInValuesUniversal to update the tag of an Aspire-style image reference
//     ("ghcr.io/my-org/api") to a new version ("1.3.9").
//   - Verifies that the update occurred and the tag was correctly updated.
//     3. Calls BumpTagInValuesUniversal again to update the tag in the structured block
//     ("ghcr.io/my-org/web-app") to a new version ("1.2.4").
//   - Verifies that the update occurred and the tag in the structured block was correctly updated.
//
// The test ensures that both Aspire-style and structured image references are handled
// correctly by the BumpTagInValuesUniversal function.
func TestBumpTagInValuesUniversalTwo(t *testing.T) {
	values := map[string]interface{}{
		"image": map[string]interface{}{
			"repository": "ghcr.io/my-org/web-app",
			"tag":        "1.2.3",
		},
		"images": map[string]interface{}{
			"api":   "ghcr.io/my-org/api:1.3.8",
			"nginx": "nginx:1.25.0",
		},
	}

	// Run bump
	updated, err := BumpTagInValuesUniversal(values, "ghcr.io/my-org/api", "1.3.9", false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !updated {
		t.Errorf("Expected update to occur but it didn't")
	}

	// Verify Aspire-style value updated
	images := values["images"].(map[string]interface{})
	apiVal := images["api"].(string)
	if !strings.HasSuffix(apiVal, ":1.3.9") {
		t.Errorf("Expected API tag to be updated to 1.3.9, got: %s", apiVal)
	}

	// Run another bump on structured block
	updated, err = BumpTagInValuesUniversal(values, "ghcr.io/my-org/web-app", "1.2.4", false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !updated {
		t.Errorf("Expected update to occur on structured block")
	}

	// Check structured block
	image := values["image"].(map[string]interface{})
	tag := image["tag"].(string)
	if tag != "1.2.4" {
		t.Errorf("Expected tag to be updated to 1.2.4, got: %s", tag)
	}
}
