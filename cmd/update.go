package cmd

import (
	"fmt"
	"os"

	"github.com/pbsladek/k8s-secret-manifest/internal/manifest"
	"github.com/pbsladek/k8s-secret-manifest/internal/validate"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing Secret manifest",
	Long: `Update data keys, labels, or annotations in an existing Kubernetes Secret manifest.

Values are plain text and will be base64-encoded automatically.
Existing keys not mentioned are left unchanged.

Examples:
  k8s-secret-manifest update --input secret.yaml --set API_KEY=newvalue

  k8s-secret-manifest update --input secret.yaml \
    --set-file CA_CERT=./ca.crt \
    --delete-key OLD_KEY \
    --label env=prod \
    --annotation last-rotated=2026-02-27`,
	RunE: runUpdate,
}

func init() {
	updateCmd.Flags().StringP("input", "i", "", "Input secret manifest file (required)")
	_ = updateCmd.MarkFlagRequired("input")

	updateCmd.Flags().StringP("output", "o", "",
		"Output file path (default: same as --input)")

	updateCmd.Flags().StringArrayP("set", "s", nil,
		"key=value to set or overwrite; repeatable (e.g. --set API_KEY=newval)")
	updateCmd.Flags().StringArrayP("set-file", "f", nil,
		"key=filepath; file content becomes the value; repeatable (e.g. --set-file CERT=./tls.crt)")
	updateCmd.Flags().StringArrayP("delete-key", "d", nil,
		"data key to remove; repeatable (e.g. --delete-key OLD_KEY)")

	updateCmd.Flags().StringArrayP("label", "l", nil,
		"Label to set or overwrite; repeatable (e.g. --label env=prod)")
	updateCmd.Flags().StringArrayP("annotation", "a", nil,
		"Annotation to set or overwrite; repeatable (e.g. --annotation managed-by=me)")
}

func runUpdate(cmd *cobra.Command, _ []string) error {
	inputPath, _ := cmd.Flags().GetString("input")
	outputPath, _ := cmd.Flags().GetString("output")
	sets, _ := cmd.Flags().GetStringArray("set")
	setFiles, _ := cmd.Flags().GetStringArray("set-file")
	deleteKeys, _ := cmd.Flags().GetStringArray("delete-key")
	labels, _ := cmd.Flags().GetStringArray("label")
	annotations, _ := cmd.Flags().GetStringArray("annotation")

	if outputPath == "" {
		outputPath = inputPath
	}

	safeInput, err := safePath("--input", inputPath)
	if err != nil {
		return err
	}

	return withExclusiveLock(outputPath, func() error {
		s, err := manifest.FromFile(safeInput)
		if err != nil {
			return fmt.Errorf("load secret: %w", err)
		}

		for _, kv := range sets {
			k, v, err := splitKeyValue(kv)
			if err != nil {
				return err
			}
			if err := validate.ValidateDataKey(k); err != nil {
				return fmt.Errorf("--set: %w", err)
			}
			manifest.SetPlainValue(s, k, v)
		}

		if err := applySetFiles(s, setFiles); err != nil {
			return err
		}

		for _, key := range deleteKeys {
			if _, ok := s.Data[key]; !ok {
				return fmt.Errorf("--delete-key %q: key not found in secret data", key)
			}
			delete(s.Data, key)
		}

		if len(labels) > 0 {
			if s.Labels == nil {
				s.Labels = make(map[string]string)
			}
			for _, l := range labels {
				k, v, err := splitKeyValue(l)
				if err != nil {
					return fmt.Errorf("--label: %w", err)
				}
				s.Labels[k] = v
			}
		}

		if len(annotations) > 0 {
			if s.Annotations == nil {
				s.Annotations = make(map[string]string)
			}
			for _, a := range annotations {
				k, v, err := splitKeyValue(a)
				if err != nil {
					return fmt.Errorf("--annotation: %w", err)
				}
				s.Annotations[k] = v
			}
		}

		if err := writeSecretTo(outputPath, s); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Updated %s\n", outputPath)
		return nil
	})
}
