package main

import (
	"bytes"
	"fmt"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

func InjectImagePullSecrets(chartDir string) error {
	// Step 1: Load the chart
	ch, err := loader.Load(chartDir)
	if err != nil {
		return fmt.Errorf("failed to load chart at %s: %w", chartDir, err)
	}

	// Step 2: Inject conditional into deployment.yaml
	for _, tmpl := range ch.Templates {
		if strings.Contains(tmpl.Name, "deployment.yaml") {
			fmt.Printf("🔧 Injecting imagePullSecrets into %s\n", tmpl.Name)

			lines := strings.Split(string(tmpl.Data), "\n")
			var buf bytes.Buffer
			injected := false
			insideTemplate := false

			for _, line := range lines {
				buf.WriteString(line + "\n")
				trimmed := strings.TrimSpace(line)

				if strings.HasPrefix(trimmed, "template:") {
					insideTemplate = true
					continue
				}

				// Only inject after entering template and finding its `spec:`
				if insideTemplate && trimmed == "spec:" && !injected {
					buf.WriteString(`      {{- if .Values.image.imagePullSecret }}
      imagePullSecrets:
        - name: {{ .Values.image.imagePullSecret }}
      {{- end }}
`)
					injected = true
				}
			}

			tmpl.Data = buf.Bytes()
			outPath := filepath.Join(chartDir, tmpl.Name)
			if err := os.WriteFile(outPath, tmpl.Data, 0644); err != nil {
				return fmt.Errorf("failed to write updated deployment.yaml: %w", err)
			}
			fmt.Printf("💾 Wrote updated deployment.yaml to %s\n", outPath)
		}
	}

	// Step 3: Ensure image.imagePullSecret in values.yaml
	valuesPath := filepath.Join(chartDir, "values.yaml")
	rawVals, err := os.ReadFile(valuesPath)
	if err != nil {
		return fmt.Errorf("failed to read values.yaml: %w", err)
	}

	var values map[string]interface{}
	if err := yaml.Unmarshal(rawVals, &values); err != nil {
		return fmt.Errorf("invalid YAML in values.yaml: %w", err)
	}

	imageBlock, ok := values["image"].(map[string]interface{})
	if !ok {
		imageBlock = make(map[string]interface{})
	}

	if _, exists := imageBlock["imagePullSecret"]; !exists {
		fmt.Println("🔧 Adding image.imagePullSecret to values.yaml")
		imageBlock["imagePullSecret"] = ""
		values["image"] = imageBlock

		updated, err := yaml.Marshal(values)
		if err != nil {
			return fmt.Errorf("failed to marshal updated values.yaml: %w", err)
		}

		if err := os.WriteFile(valuesPath, updated, 0644); err != nil {
			return fmt.Errorf("failed to write values.yaml: %w", err)
		}
	} else {
		fmt.Println("✅ image.imagePullSecret already exists in values.yaml")
	}

	// Step 4: Render chart with values for preview
	valsMerged, err := chartutil.ToRenderValues(ch, values, chartutil.ReleaseOptions{
		Name:      "test-release",
		Namespace: "default",
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to prepare render values: %w", err)
	}

	rendered, err := engine.Render(ch, valsMerged)
	if err != nil {
		return fmt.Errorf("failed to render chart: %w", err)
	}

	fmt.Println("\n🖨️ Rendered Manifest (excerpt):")
	for name, content := range rendered {
		if strings.Contains(name, "deployment.yaml") {
			fmt.Printf("\n--- %s ---\n%s\n", name, content)
		}
	}

	fmt.Println("✅ Injection complete.")
	return nil
}

