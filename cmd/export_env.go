package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/pbsladek/k8s-secret-manifest/internal/manifest"
	"github.com/spf13/cobra"
)

var exportEnvCmd = &cobra.Command{
	Use:   "export-env",
	Short: "Export a Secret manifest as a .env file",
	Long: `Decode a Kubernetes Secret manifest and write it as a .env file.

Keys are sorted alphabetically. Values that contain spaces, quotes, or other
shell-significant characters are automatically wrapped in double quotes.

Example:
  k8s-secret-manifest export-env --input secret.yaml --output .env

Print to stdout (e.g. for piping into another tool):
  k8s-secret-manifest export-env --input secret.yaml`,
	RunE: runExportEnv,
}

func init() {
	exportEnvCmd.Flags().StringP("input", "i", "", "Input secret manifest file (required)")
	_ = exportEnvCmd.MarkFlagRequired("input")

	exportEnvCmd.Flags().StringP("output", "o", "", "Output .env file path (default: stdout)")
}

func runExportEnv(cmd *cobra.Command, _ []string) error {
	inputPath, _ := cmd.Flags().GetString("input")
	outputPath, _ := cmd.Flags().GetString("output")

	s, err := manifest.FromFile(inputPath)
	if err != nil {
		return fmt.Errorf("load secret: %w", err)
	}

	keys := make([]string, 0, len(s.Data))
	for k := range s.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&sb, "%s=%s\n", k, quoteEnvValue(string(s.Data[k])))
	}

	if outputPath == "" {
		_, err := fmt.Print(sb.String())
		return err
	}
	return os.WriteFile(outputPath, []byte(sb.String()), 0600)
}

// quoteEnvValue wraps val in double quotes when it contains characters that
// would confuse .env parsers. Double quotes and backslashes inside are escaped.
func quoteEnvValue(val string) string {
	if !needsEnvQuoting(val) {
		return val
	}
	escaped := strings.ReplaceAll(val, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}

func needsEnvQuoting(val string) bool {
	return strings.ContainsAny(val, " \t\n\r\"'#$\\=;,")
}
