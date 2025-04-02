// Package provides a CLI tool called flux-helpers for automating
// the manipulation of Flux GitOps manifests, such as HelmRelease files.
// The tool includes functionality for safely and automatically updating
// image tags within these manifests.
//
// The main command is defined using the Cobra library and includes the
// following subcommands:
//
//   - bump: Allows users to update one or more image tags in a specified
//     HelmRelease YAML file. The command supports dry-run mode for previewing
//     changes without modifying the file.
//
// Flags for the `bump` command:
//   - --file (-f): Specifies the path to the HelmRelease YAML file.
//   - --set: Specifies image updates in the form "repo=version". This flag
//     can be repeated to update multiple images.
//   - --dry-run: Enables preview mode to display changes without applying them.
//
// The `splitArg` helper function is used to parse the "repo=version" format
// into its components, and the `BumpMultipleTags` function (not included in
// this code) is responsible for applying the updates to the YAML file.
//
// Usage example:
//
//	flux-helpers bump --file path/to/helmrelease.yaml --set repo1=version1 --set repo2=version2
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	filePath string
	tagArgs  []string
	dryRun   bool
)

var rootCmd = &cobra.Command{
	Use:   "flux-helpers",
	Short: "Flux YAML and HelmRelease automation tools",
	Long:  "flux-helpers is a CLI tool for manipulating Flux GitOps manifests such as HelmReleases, including safe and automated image tag updates.",
}

var bumpCmd = &cobra.Command{
	Use:   "bump",
	Short: "Bump one or more image tags in a HelmRelease file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if filePath == "" || len(tagArgs) == 0 {
			return fmt.Errorf("you must specify --file and at least one --set repo=version")
		}

		updates := map[string]string{}
		for _, set := range tagArgs {
			parts := splitArg(set)
			if parts == nil {
				return fmt.Errorf("invalid --set format: %s (expected repo=version)", set)
			}
			updates[parts[0]] = parts[1]
		}

		err := BumpMultipleTagsUniversalAndSanitize(filePath, updates, dryRun)
		if err != nil {
			return fmt.Errorf("failed to bump tags: %w", err)
		}

		return nil
	},
}

func init() {
	bumpCmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to HelmRelease YAML file")
	bumpCmd.Flags().StringArrayVar(&tagArgs, "set", nil, "Image update(s) in the form repo=version (repeatable)")
	bumpCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without modifying the file")

	rootCmd.AddCommand(bumpCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("❌", err)
		os.Exit(1)
	}
}

// splitArg splits "repo=version" into [repo, version]
func splitArg(s string) []string {
	parts := strings.SplitN(s, "=", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return nil
	}
	return parts
}
