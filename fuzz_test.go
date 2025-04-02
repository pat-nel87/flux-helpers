package main

import (
	"encoding/json"
	"os"
	"testing"

	"sigs.k8s.io/yaml"
)

// FuzzBumpTagInValuesUniversal is a fuzz test for the BumpTagInValuesUniversal function.
// It tests the function's ability to handle various YAML inputs and ensures that the
// structure of the input is not corrupted after processing.
//
// The test uses seed files containing YAML data to initialize the fuzzing process.
// Each seed file represents a different style of YAML structure, such as traditional
// container blocks or Aspire-style images maps.
//
// The fuzzing function performs the following steps:
//  1. Parses the YAML input into a map structure.
//  2. Extracts the "spec" and "values" fields from the parsed YAML.
//  3. Runs the BumpTagInValuesUniversal function in dry-run mode to simulate tag bumping.
//  4. Verifies that the structure of the "values" field remains valid by attempting to
//     marshal it back into JSON.
//
// If the YAML input is invalid or the required fields are missing, the test skips
// further processing for that input.
//
// This test ensures that the BumpTagInValuesUniversal function can handle a wide range
// of inputs without causing structural corruption or runtime errors.
func FuzzBumpTagInValuesUniversal(f *testing.F) {
	// Seed inputs
	seedFiles := []string{
		"test_files/structured-values.yaml", // traditional container blocks
		"test_files/aspire-values.yaml",     // Aspire-style images map
	}

	for _, file := range seedFiles {
		data, err := os.ReadFile(file)
		if err == nil {
			f.Add(string(data), "ghcr.io/my-org/my-api", "1.3.999")
		}
	}

	f.Fuzz(func(t *testing.T, yamlInput, repo, tag string) {
		var doc map[string]interface{}
		if err := yaml.Unmarshal([]byte(yamlInput), &doc); err != nil {
			return // Skip bad YAML
		}

		spec, ok := doc["spec"].(map[string]interface{})
		if !ok {
			return
		}

		valuesRaw, ok := spec["values"]
		if !ok {
			return
		}

		values, ok := valuesRaw.(map[string]interface{})
		if !ok {
			return
		}

		// Run the bump logic (dry run)
		_, _ = BumpTagInValuesUniversal(values, repo, tag, true)

		// Verify no structural corruption
		if _, err := json.Marshal(values); err != nil {
			t.Errorf("❌ Structure corrupted after bumping: %v", err)
		}
	})
}
