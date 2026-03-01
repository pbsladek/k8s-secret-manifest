package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/pbsladek/k8s-secret-manifest/internal/manifest"
	"github.com/pbsladek/k8s-secret-manifest/internal/validate"
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

	safeInput, err := safePath("--input", inputPath)
	if err != nil {
		return err
	}

	editor, err := resolveEditor()
	if err != nil {
		return err
	}

	s, err := manifest.FromFile(safeInput)
	if err != nil {
		return fmt.Errorf("load secret: %w", err)
	}

	// Create a private temp directory (mode 0700) so other local users cannot
	// observe or tamper with the decoded secret while the editor is open.
	tmpDir, err := os.MkdirTemp("", "k8s-secret-edit-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tmpPath := filepath.Join(tmpDir, "secret.env")
	tmpFile, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

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
		if err := validate.ValidateDataKey(k); err != nil {
			return fmt.Errorf("edited file: %w", err)
		}
		manifest.SetPlainValue(s, k, v)
	}

	if err := writeSecretTo(outputPath, s); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Updated %s\n", outputPath)
	return nil
}

// resolveEditor looks up the user's preferred editor from $EDITOR and returns
// its absolute path. Falls back to "vi" if $EDITOR is unset.
func resolveEditor() (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	resolved, err := exec.LookPath(editor)
	if err != nil {
		return "", fmt.Errorf("editor %q not found in PATH: %w", editor, err)
	}
	return resolved, nil
}
