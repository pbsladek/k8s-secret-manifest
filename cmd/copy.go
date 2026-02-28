package cmd

import (
	"fmt"
	"os"

	"github.com/pbsladek/k8s-secret-manifest/internal/manifest"
	"github.com/spf13/cobra"
)

var copyCmd = &cobra.Command{
	Use:   "copy",
	Short: "Copy a Secret manifest with a new name and/or namespace",
	Long: `Clone a Secret manifest, assigning it a new name and optionally a new namespace.

All data keys, labels, annotations, type, and immutable flag are preserved.
The global --namespace flag controls the target namespace (default: default).

Example — rename within the same namespace:
  k8s-secret-manifest copy --input secret.yaml --name new-secret --output new-secret.yaml

Example — promote to a different namespace:
  k8s-secret-manifest copy --input secret.yaml --name prod-secret \
    --namespace production --output prod-secret.yaml`,
	RunE: runCopy,
}

func init() {
	copyCmd.Flags().StringP("input", "i", "", "Input secret manifest file (required)")
	_ = copyCmd.MarkFlagRequired("input")

	copyCmd.Flags().StringP("name", "N", "", "New secret name (required)")
	_ = copyCmd.MarkFlagRequired("name")

	copyCmd.Flags().StringP("output", "o", "", "Output file path (default: stdout)")
}

func runCopy(cmd *cobra.Command, _ []string) error {
	inputPath, _ := cmd.Flags().GetString("input")
	outputPath, _ := cmd.Flags().GetString("output")
	name, _ := cmd.Flags().GetString("name")
	namespace, _ := cmd.Root().PersistentFlags().GetString("namespace")

	s, err := manifest.FromFile(inputPath)
	if err != nil {
		return fmt.Errorf("load secret: %w", err)
	}

	s.Name = name
	s.Namespace = namespace

	if err := writeSecretTo(outputPath, s); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Copied to %s/%s\n", namespace, name)
	return nil
}
