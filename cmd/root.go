package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "k8s-secret-manifest",
	Short: "Generate and seal Kubernetes Secret manifests",
	Long: `k8s-secret-manifest generates valid Kubernetes Secret YAML manifests,
handles base64 encoding of plain-text values, manages paired index-list keys,
and seals secrets using the kubeseal CLI.`,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringP("namespace", "n", "default", "Kubernetes namespace")
	rootCmd.PersistentFlags().StringP("kubeseal-path", "p", "kubeseal", "Path to kubeseal binary")

	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(fromEnvCmd)
	rootCmd.AddCommand(exportEnvCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(rotateCmd)
	rootCmd.AddCommand(copyCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(sealCmd)
	rootCmd.AddCommand(addEntryCmd)
	rootCmd.AddCommand(removeEntryCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(editCmd)
}

// writeOutput writes data to a file or stdout.
func writeOutput(path string, data []byte) error {
	if path == "" {
		_, err := os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// splitKeyValue parses "key=value", allowing "=" in the value portion.
func splitKeyValue(kv string) (string, string, error) {
	idx := strings.IndexByte(kv, '=')
	if idx < 0 {
		return "", "", fmt.Errorf("invalid key=value format: %q (missing '=')", kv)
	}
	return kv[:idx], kv[idx+1:], nil
}
