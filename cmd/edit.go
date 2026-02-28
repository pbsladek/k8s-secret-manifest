package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sort"

	"github.com/pbsladek/k8s-secret-manifest/internal/manifest"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit a Secret manifest's values interactively",
	Long: `Open the decoded Secret values in $EDITOR as a .env-style file.

On save and exit, the updated values are re-encoded and the Secret manifest
is written back. The EDITOR environment variable is used; falls back to "vi".

Note: data keys whose values contain newlines (e.g. PEM certificates) are
written as-is and must remain intact in the editor. For cert-style values
consider using --set-file in the update command instead.

Example:
  k8s-secret-manifest edit --input secret.yaml
  EDITOR=nano k8s-secret-manifest edit --input secret.yaml --output new.yaml`,
	RunE: runEdit,
}

func init() {
	editCmd.Flags().StringP("input", "i", "", "Input secret manifest file (required)")
	_ = editCmd.MarkFlagRequired("input")

	editCmd.Flags().StringP("output", "o", "",
		"Output file path (default: same as --input)")
}

func runEdit(cmd *cobra.Command, _ []string) error {
	inputPath, _ := cmd.Flags().GetString("input")
	outputPath, _ := cmd.Flags().GetString("output")
	if outputPath == "" {
		outputPath = inputPath
	}

	s, err := manifest.FromFile(inputPath)
	if err != nil {
		return fmt.Errorf("load secret: %w", err)
	}

	// Write decoded values to a temp .env file.
	tmpFile, err := os.CreateTemp("", "k8s-secret-edit-*.env")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	keys := make([]string, 0, len(s.Data))
	for k := range s.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if _, err := fmt.Fprintf(tmpFile, "%s=%s\n", k, string(s.Data[k])); err != nil {
			_ = tmpFile.Close()
			return fmt.Errorf("write temp file: %w", err)
		}
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// Launch editor.
	editor := resolveEditor()
	editorCmd := exec.Command(editor, tmpPath) //nolint:gosec
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr
	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("editor %q: %w", editor, err)
	}

	// Re-read and re-encode.
	edited, err := parseEnvFile(tmpPath)
	if err != nil {
		return fmt.Errorf("parse edited file: %w", err)
	}

	s.Data = make(map[string][]byte)
	for k, v := range edited {
		manifest.SetPlainValue(s, k, v)
	}

	if err := writeSecretTo(outputPath, s); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Updated %s\n", outputPath)
	return nil
}

// resolveEditor returns the user's preferred editor from $EDITOR, defaulting to vi.
func resolveEditor() string {
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	return "vi"
}
