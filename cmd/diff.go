package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/pbsladek/k8s-secret-manifest/internal/manifest"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Diff two Secret manifests (decoded)",
	Long: `Decode and diff two Kubernetes Secret manifests.

Keys present only in the first file are shown with -.
Keys present only in the second file are shown with +.
Keys present in both with different values are shown with - and +.
Unchanged keys are hidden by default (use --unchanged to show them).

Color output is enabled by default; set NO_COLOR=1 to disable.

Example:
  k8s-secret-manifest diff --from secret-v1.yaml --to secret-v2.yaml
  k8s-secret-manifest diff --from secret-v1.yaml --to secret-v2.yaml --unchanged`,
	RunE: runDiff,
}

func init() {
	diffCmd.Flags().StringP("from", "A", "", "Base secret file (required)")
	_ = diffCmd.MarkFlagRequired("from")

	diffCmd.Flags().StringP("to", "B", "", "New secret file (required)")
	_ = diffCmd.MarkFlagRequired("to")

	diffCmd.Flags().Bool("unchanged", false, "Also show unchanged keys")
}

func runDiff(cmd *cobra.Command, _ []string) error {
	fromPath, _ := cmd.Flags().GetString("from")
	toPath, _ := cmd.Flags().GetString("to")
	showUnchanged, _ := cmd.Flags().GetBool("unchanged")

	a, err := manifest.FromFile(fromPath)
	if err != nil {
		return fmt.Errorf("load --from: %w", err)
	}
	b, err := manifest.FromFile(toPath)
	if err != nil {
		return fmt.Errorf("load --to: %w", err)
	}

	color := os.Getenv("NO_COLOR") == ""

	red := func(s string) string {
		if color {
			return "\033[31m" + s + "\033[0m"
		}
		return s
	}
	green := func(s string) string {
		if color {
			return "\033[32m" + s + "\033[0m"
		}
		return s
	}
	yellow := func(s string) string {
		if color {
			return "\033[33m" + s + "\033[0m"
		}
		return s
	}

	// Header
	fmt.Printf("--- %s (%s/%s  type: %s)\n", fromPath, a.Namespace, a.Name, a.Type)
	fmt.Printf("+++ %s (%s/%s  type: %s)\n", toPath, b.Namespace, b.Name, b.Type)

	// Metadata differences
	if a.Name != b.Name {
		fmt.Println(red(fmt.Sprintf("~ name: %s → %s", a.Name, b.Name)))
	}
	if a.Namespace != b.Namespace {
		fmt.Println(yellow(fmt.Sprintf("~ namespace: %s → %s", a.Namespace, b.Namespace)))
	}
	if a.Type != b.Type {
		fmt.Println(yellow(fmt.Sprintf("~ type: %s → %s", a.Type, b.Type)))
	}

	// Collect all keys
	keySet := make(map[string]struct{})
	for k := range a.Data {
		keySet[k] = struct{}{}
	}
	for k := range b.Data {
		keySet[k] = struct{}{}
	}
	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Diff data
	changed := 0
	for _, k := range keys {
		_, inA := a.Data[k]
		_, inB := b.Data[k]
		aVal := string(a.Data[k])
		bVal := string(b.Data[k])

		switch {
		case inA && !inB:
			fmt.Println(red(fmt.Sprintf("- %s=%s", k, aVal)))
			changed++
		case !inA && inB:
			fmt.Println(green(fmt.Sprintf("+ %s=%s", k, bVal)))
			changed++
		case aVal != bVal:
			fmt.Println(red(fmt.Sprintf("- %s=%s", k, aVal)))
			fmt.Println(green(fmt.Sprintf("+ %s=%s", k, bVal)))
			changed++
		default:
			if showUnchanged {
				fmt.Printf("  %s=%s\n", k, aVal)
			}
		}
	}

	if changed == 0 {
		fmt.Println("(no differences)")
	}
	return nil
}
