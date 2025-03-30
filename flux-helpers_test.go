package main

import (
	"os"
	"testing"
	"encoding/json"

	"sigs.k8s.io/yaml"
)

func extractRawValues(data []byte) []byte {
	var hr map[string]interface{}
	_ = yaml.Unmarshal(data, &hr)
	spec := hr["spec"].(map[string]interface{})
	values := spec["values"]
	raw, _ := json.Marshal(values)
	return raw
}

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
