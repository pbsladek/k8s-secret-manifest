package cmd

import (
	"fmt"
	"os"

	"github.com/pbsladek/k8s-secret-manifest/internal/manifest"
	"github.com/pbsladek/k8s-secret-manifest/internal/validate"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a Secret manifest for correctness",
	Long: `Check a Kubernetes Secret manifest for spec violations and likely mistakes.

Errors indicate actual spec violations (invalid name/namespace format,
missing required data keys for the secret type, etc.).

Warnings indicate likely mistakes (empty data section, missing recommended
keys for the secret type, etc.).

Exit codes:
  0  no issues found
  1  one or more errors found (or warnings with no errors)

Example:
  k8s-secret-manifest validate --input secret.yaml`,
	RunE: runValidate,
}

func init() {
	validateCmd.Flags().StringP("input", "i", "", "Input secret manifest file (required)")
	_ = validateCmd.MarkFlagRequired("input")
}

func runValidate(cmd *cobra.Command, _ []string) error {
	inputPath, _ := cmd.Flags().GetString("input")

	s, err := manifest.FromFile(inputPath)
	if err != nil {
		return fmt.Errorf("load secret: %w", err)
	}

	issues := validate.Secret(s)

	useColor := os.Getenv("NO_COLOR") == ""
	colorRed := "\033[31m"
	colorYellow := "\033[33m"
	colorReset := "\033[0m"

	hasErrors := false
	for _, issue := range issues {
		if issue.IsError() {
			hasErrors = true
			if useColor {
				fmt.Fprintf(os.Stderr, "%serror:%s %s\n", colorRed, colorReset, issue.Message)
			} else {
				fmt.Fprintf(os.Stderr, "error: %s\n", issue.Message)
			}
		} else {
			if useColor {
				fmt.Fprintf(os.Stderr, "%swarning:%s %s\n", colorYellow, colorReset, issue.Message)
			} else {
				fmt.Fprintf(os.Stderr, "warning: %s\n", issue.Message)
			}
		}
	}

	if hasErrors {
		return fmt.Errorf("validation failed with %d error(s)", countErrors(issues))
	}
	if len(issues) > 0 {
		fmt.Fprintf(os.Stderr, "validation passed with %d warning(s)\n", len(issues))
		return nil
	}
	fmt.Fprintf(os.Stderr, "validation passed\n")
	return nil
}

func countErrors(issues []validate.Issue) int {
	n := 0
	for _, i := range issues {
		if i.IsError() {
			n++
		}
	}
	return n
}
