package main

import (
	"encoding/json"
	"fmt"
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"os"
	"regexp"
	"sigs.k8s.io/yaml"
	"strings"
)

// isValidSemver validates whether a given string conforms to the semantic versioning (SemVer) format.
// The function uses a regular expression to check for the following structure:
// - An optional "v" prefix (e.g., "v1.2.3" or "1.2.3").
// - A version in the format MAJOR.MINOR.PATCH (e.g., "1.2.3").
// - An optional pre-release identifier (e.g., "-alpha", "-rc.1").
// - An optional build metadata identifier (e.g., "+build123").
//
// Parameters:
// - tag: The version string to validate.
//
// Returns:
// - true if the string matches the SemVer format, false otherwise.
func isValidSemver(tag string) bool {
	var semverRegex = regexp.MustCompile(`^v?\d+\.\d+\.\d+(-[a-zA-Z0-9.-]+)?(\+[a-zA-Z0-9.-]+)?$`)
	return semverRegex.MatchString(tag)
}

// sanitizeHelmRelease removes specific fields from a Kubernetes Helm release object
// represented as a map. It performs the following sanitizations:
// 1. Removes the "creationTimestamp" field from the "metadata" section, if it exists.
// 2. Removes the "status" field if it exists and is an empty map.
//
// Parameters:
//   - obj: A map[string]interface{} representing the Helm release object to sanitize.
func sanitizeHelmRelease(obj map[string]interface{}) {
	// Remove .metadata.creationTimestamp
	if metadata, ok := obj["metadata"].(map[string]interface{}); ok {
		delete(metadata, "creationTimestamp")
	}

	// Remove empty .status
	if status, ok := obj["status"]; ok {
		if m, isMap := status.(map[string]interface{}); isMap && len(m) == 0 {
			delete(obj, "status")
		}
	}
}

// findImageBlocksUniversal searches through a nested map structure to find blocks
// that match a specific image name. It supports both structured blocks with a
// "repository" key and Aspire-style strings in the format "image:tag".
//
// Parameters:
//   - values: A map[string]interface{} representing the nested structure to search.
//   - imageName: A string representing the image name to match.
//
// Returns:
//   - A slice of map[string]interface{} containing the matched blocks. Each match
//     can either be a structured block or a map containing the key, value, and
//     parent path for Aspire-style matches.
//
// The function recursively traverses the input structure, handling both maps and
// slices, and collects matches based on the specified criteria.
func findImageBlocksUniversal(values map[string]interface{}, imageName string) []map[string]interface{} {
	var matches []map[string]interface{}

	var walk func(interface{})
	walk = func(node interface{}) {
		switch typed := node.(type) {

		case map[string]interface{}:
			// Match structured block
			if repo, ok := typed["repository"].(string); ok && repo == imageName {
				matches = append(matches, typed)
			}

			// Check each key/value recursively
			for key, val := range typed {
				// Also match Aspire-style string: "image:tag"
				if strVal, ok := val.(string); ok && strings.HasPrefix(strVal, imageName+":") {
					matches = append(matches, map[string]interface{}{
						"key":   key,
						"value": strVal,
						"path":  typed, // parent map so we can update it later
					})
				} else {
					walk(val)
				}
			}

		case []interface{}:
			for _, item := range typed {
				walk(item)
			}
		}
	}

	walk(values)
	return matches
}

// BumpTagInValuesUniversal updates the version tag of a specified image in a given values map.
// It supports two types of image blocks: structured blocks with "repository" and "tag" fields,
// and Aspire-style string entries with "key", "value", and "path" fields.
//
// Parameters:
//   - values: A map representing the values file where image blocks are defined.
//   - imageName: The name of the image to update.
//   - newVersion: The new version tag to set for the image.
//   - dryRun: If true, no actual changes are made; instead, the function logs what would be changed.
//
// Returns:
//   - A boolean indicating whether any updates were made.
//   - An error if any issues occur during processing.
//
// Behavior:
//   - If no matching image blocks are found, the function logs a warning and returns false.
//   - For structured image blocks, it checks if the "repository" matches the imageName and updates the "tag".
//   - For Aspire-style entries, it checks if the "value" starts with the imageName and updates the tag in the "path".
//   - If the newVersion is not a valid semantic version, the function skips the update and logs a warning.
//   - In dry-run mode, the function logs the intended changes without modifying the values map.
//
// Example Usage:
//
//	updated, err := BumpTagInValuesUniversal(values, "my-image", "1.2.3", false)
//	if err != nil {
//	    log.Fatalf("Error updating image tag: %v", err)
//	}
//	if updated {
//	    fmt.Println("Image tags updated successfully.")
//	} else {
//	    fmt.Println("No updates were necessary.")
func BumpTagInValuesUniversal(values map[string]interface{}, imageName, newVersion string, dryRun bool) (bool, error) {
	matches := findImageBlocksUniversal(values, imageName)
	if len(matches) == 0 {
		fmt.Printf("⚠️ No image block found for %s\n", imageName)
		return false, nil
	}

	updated := false

	for _, image := range matches {
		// Case 1: Structured image block (repository + tag)
		if repo, ok := image["repository"].(string); ok && repo == imageName {
			oldTag, _ := image["tag"].(string)

			if oldTag == newVersion {
				fmt.Printf("✅ %s already at %s, skipping\n", repo, newVersion)
				continue
			}
			if !isValidSemver(newVersion) {
				fmt.Printf("⚠️ Invalid version: %s (skipping %s)\n", newVersion, repo)
				continue
			}

			if dryRun {
				fmt.Printf("[dry-run] Would bump %s:%s → %s\n", repo, oldTag, newVersion)
			} else {
				image["tag"] = newVersion
				fmt.Printf("🔁 Bumped %s:%s → %s\n", repo, oldTag, newVersion)
			}
			updated = true
			continue
		}

		// Case 2: Aspire-style string entry
		key, kOk := image["key"].(string)
		val, vOk := image["value"].(string)
		path, pOk := image["path"].(map[string]interface{})

		if kOk && vOk && pOk && strings.HasPrefix(val, imageName+":") {
			parts := strings.Split(val, ":")
			oldTag := parts[len(parts)-1]
			if oldTag == newVersion {
				fmt.Printf("✅ %s already at %s, skipping\n", imageName, newVersion)
				continue
			}
			if !isValidSemver(newVersion) {
				fmt.Printf("⚠️ Invalid version: %s (skipping %s)\n", newVersion, imageName)
				continue
			}

			newImage := fmt.Sprintf("%s:%s", imageName, newVersion)

			if dryRun {
				fmt.Printf("[dry-run] Would bump %s → %s\n", val, newImage)
			} else {
				path[key] = newImage
				fmt.Printf("🔁 Bumped %s → %s\n", val, newImage)
			}
			updated = true
		}
	}

	return updated, nil
}

// BumpMultipleTagsUniversalAndSanitize updates the image tags in a HelmRelease YAML file
// based on the provided updates map, optionally performing a dry-run.
//
// This function reads a HelmRelease YAML file, parses its .spec.values field, and updates
// the image tags specified in the `updates` map. It supports a dry-run mode to preview
// changes without modifying the file. After updating, the function sanitizes the HelmRelease
// before writing it back to the file.
//
// Parameters:
//   - filePath: The path to the HelmRelease YAML file to be updated.
//   - updates: A map where the keys are image names and the values are the new tags to apply.
//   - dryRun: A boolean flag indicating whether to perform a dry-run (true) or apply changes (false).
//
// Returns:
//   - error: An error if any issues occur during file reading, parsing, updating, or writing.
//
// Behavior:
//   - Reads the HelmRelease YAML file specified by `filePath`.
//   - Parses the .spec.values field into a generic map.
//   - Iterates over the `updates` map to update image tags using the BumpTagInValuesUniversal function.
//   - If dryRun is true, prints the number of potential updates and exits without modifying the file.
//   - If updates are made, marshals the updated values back into the HelmRelease structure.
//   - Sanitizes the HelmRelease to ensure compatibility and correctness.
//   - Writes the updated and sanitized YAML back to the original file.
//
// Example Usage:
//
//	updates := map[string]string{
//	    "nginx": "1.21.0",
//	    "redis": "6.2.5",
//	}
//	err := BumpMultipleTagsUniversalAndSanitize("/path/to/helmrelease.yaml", updates, false)
//	if err != nil {
//	    log.Fatalf("Error updating tags: %v", err)
//	}
func BumpMultipleTagsUniversalAndSanitize(filePath string, updates map[string]string, dryRun bool) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var hr helmv2.HelmRelease
	if err := yaml.Unmarshal(data, &hr); err != nil {
		return fmt.Errorf("failed to unmarshal HelmRelease: %w", err)
	}

	var values map[string]interface{}
	if err := json.Unmarshal(hr.Spec.Values.Raw, &values); err != nil {
		return fmt.Errorf("failed to parse .spec.values: %w", err)
	}

	updatedCount := 0
	for imageName, newVersion := range updates {
		updated, err := BumpTagInValuesUniversal(values, imageName, newVersion, dryRun)
		if err != nil {
			return fmt.Errorf("error updating image %s: %w", imageName, err)
		}
		if updated {
			updatedCount++
		}
	}

	if dryRun {
		fmt.Printf("🧪 Dry-run complete. %d potential updates found.\n", updatedCount)
		return nil
	}

	if updatedCount == 0 {
		fmt.Println("ℹ️ No image tags were updated.")
		return nil
	}

	// Update .spec.values
	raw, _ := json.Marshal(values)
	hr.Spec.Values = &apiextv1.JSON{Raw: raw}

	// Marshal to YAML, then sanitize before final write
	yamlBytes, err := yaml.Marshal(&hr)
	if err != nil {
		return fmt.Errorf("failed to marshal updated HelmRelease: %w", err)
	}

	// Re-unmarshal to generic map to sanitize
	var hrMap map[string]interface{}
	if err := yaml.Unmarshal(yamlBytes, &hrMap); err != nil {
		return fmt.Errorf("failed to unmarshal for sanitization: %w", err)
	}

	sanitizeHelmRelease(hrMap)

	newYAML, err := yaml.Marshal(&hrMap)
	if err != nil {
		return fmt.Errorf("failed to marshal sanitized HelmRelease: %w", err)
	}

	if err := os.WriteFile(filePath, newYAML, 0644); err != nil {
		return fmt.Errorf("failed to write updated file: %w", err)
	}

	fmt.Printf("✅ Updated %d image(s) in %s\n", updatedCount, filePath)
	return nil
}
