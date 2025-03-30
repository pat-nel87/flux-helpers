package main

import (
	"encoding/json"
	"fmt"
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"os"
	"regexp"
	"sigs.k8s.io/yaml"
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

// findImageBlocks searches through a nested map structure to find all "image blocks"
// that match a specified image repository name. It recursively traverses the structure,
// looking for maps that contain a "repository" key with a value equal to the provided imageName.
// If a match is found, the map is added to the result slice. The function supports traversing
// both maps and slices, ensuring it can handle deeply nested configurations.
//
// Parameters:
// - values: The root of the nested map structure to search.
// - imageName: The name of the image repository to match.
//
// Returns:
// - A slice of maps, where each map represents an "image block" that matches the imageName.
func findImageBlocks(values map[string]interface{}, imageName string) []map[string]interface{} {
	var matches []map[string]interface{}

	var walk func(interface{})
	walk = func(node interface{}) {
		switch typed := node.(type) {
		case map[string]interface{}:
			// Check if it's an image block
			if repo, ok := typed["repository"].(string); ok && repo == imageName {
				matches = append(matches, typed)
			}
			// Continue walking children
			for _, v := range typed {
				walk(v)
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

// BumpTagInValues updates the image tag for a specific container image within a nested map structure.
// It searches for all image blocks matching the specified imageName and updates their "tag" field to the newVersion.
// The function supports a dry-run mode, which logs the changes without modifying the data.
//
// Parameters:
// - values: The nested map structure containing image configuration.
// - imageName: The name of the image repository to update.
// - newVersion: The new semantic version to set as the image tag.
// - dryRun: If true, the function logs the changes without applying them.
//
// Returns:
// - A boolean indicating whether any updates were made.
// - An error if any issues occur during the process, such as invalid semantic versioning.
func BumpTagInValues(values map[string]interface{}, imageName, newVersion string, dryRun bool) (bool, error) {
	matches := findImageBlocks(values, imageName)
	if len(matches) == 0 {
		fmt.Printf("⚠️ No image block found for %s\n", imageName)
		return false, nil
	}

	for _, image := range matches {
		repo, ok := image["repository"].(string)
		if !ok || repo != imageName {
			continue
		}

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
	}

	return true, nil
}

// BumpMultipleTags updates the image tags for multiple container images in a HelmRelease YAML file.
// It reads the file, parses the HelmRelease resource, and iterates through the provided updates map,
// where each key is an image name and each value is the new tag to apply. The function supports a dry-run mode,
// allowing users to preview changes without modifying the file.
//
// Parameters:
// - filePath: The path to the HelmRelease YAML file to modify.
// - updates: A map where the keys are image repository names and the values are the new semantic versions to set as tags.
// - dryRun: If true, the function logs the changes without applying them.
//
// Returns:
// - An error if any step fails, such as reading the file, parsing the YAML, updating an image tag, or writing the updated file.
func BumpMultipleTags(filePath string, updates map[string]string, dryRun bool) error {
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
		updated, err := BumpTagInValues(values, imageName, newVersion, dryRun)
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

	// Update and write file
	raw, _ := json.Marshal(values)
	hr.Spec.Values = &apiextv1.JSON{Raw: raw}

	newYAML, err := yaml.Marshal(&hr)
	if err != nil {
		return fmt.Errorf("failed to marshal updated HelmRelease: %w", err)
	}

	if err := os.WriteFile(filePath, newYAML, 0644); err != nil {
		return fmt.Errorf("failed to write updated file: %w", err)
	}

	fmt.Printf("✅ Updated %d image(s) in %s\n", updatedCount, filePath)
	return nil
}

// BumpSingleTag updates the image tag for a single container image in a HelmRelease YAML file.
// This function is a wrapper around BumpMultipleTags, simplifying the interface for single-image updates.
// It constructs a map with a single entry for the specified imageName and newVersion, and delegates the update logic
// to BumpMultipleTags. The function supports a dry-run mode, allowing users to preview changes without modifying the file.
//
// Parameters:
// - filePath: The path to the HelmRelease YAML file to modify.
// - imageName: The name of the image repository to update.
// - newVersion: The new semantic version to set as the image tag.
// - dryRun: If true, the function logs the changes without applying them.
//
// Returns:
// - An error if any step fails, such as reading the file, parsing the YAML, updating the image tag, or writing the updated file.
func BumpSingleTag(filePath, imageName, newVersion string, dryRun bool) error {
    return BumpMultipleTags(filePath, map[string]string{imageName: newVersion}, dryRun)
}


