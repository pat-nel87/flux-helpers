package main

import (
	"encoding/json"
	"testing"
    "os"
	"sigs.k8s.io/yaml"
)

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
