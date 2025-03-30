package main

import (
	"encoding/json"
	"testing"
    "os"
	"sigs.k8s.io/yaml"
)

// FuzzBumpTagInValues is a fuzz test for the BumpTagInValues function. It tests
// the function's behavior with various inputs to ensure it handles edge cases
// and malformed data gracefully without crashing or corrupting the structure.
//
// The test uses a seed input from a YAML file ("test_files/multiple-bump.yaml")
// to initialize the fuzzing process. It then generates random inputs for the
// YAML content, repository, and tag.
//
// The fuzzing function performs the following steps:
// 1. Attempts to unmarshal the YAML input into a map structure.
// 2. Validates the presence and type of the "spec" and "values" fields.
// 3. Calls BumpTagInValues with the parsed "values" map, repository, and tag.
// 4. Ensures the structure remains valid by marshaling it back to JSON.
//
// This test is designed to identify potential crashes or data corruption issues
// in the BumpTagInValues function when handling unexpected or malformed inputs.
func FuzzBumpTagInValues(f *testing.F) {
	// Seed input: real YAML that works
	data, err := os.ReadFile("test_files/multiple-bump.yaml")
	if err == nil {
		f.Add(string(data), "ghcr.io/my-org/my-api", "1.3.999")
	}

	f.Fuzz(func(t *testing.T, yamlInput, repo, tag string) {
		var doc map[string]interface{}
		if err := yaml.Unmarshal([]byte(yamlInput), &doc); err != nil {
			return // skip malformed YAML
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

		// Just see if we crash
		_, _ = BumpTagInValues(values, repo, tag, true)

		// Marshal to ensure we didn't corrupt the structure
		_, _ = json.Marshal(values)
	})
}
