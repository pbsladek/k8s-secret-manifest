package cmd

import (
	"fmt"
	"sort"

	"github.com/pbsladek/k8s-secret-manifest/internal/manifest"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List key names in a Secret manifest",
	Long: `List the key names present in the data: field of a Secret manifest.
Values are not decoded or displayed.

Example:
  k8s-secret-manifest list --input secret.yaml`,
	RunE: runList,
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show decoded values from a Secret manifest",
	Long: `Decode and display metadata and data key/value pairs from a Secret manifest.

All data values are base64-decoded and printed as plain text.

Example:
  k8s-secret-manifest show --input secret.yaml
  k8s-secret-manifest show --input secret.yaml --key API_KEY`,
	RunE: runShow,
}

func init() {
	listCmd.Flags().StringP("input", "i", "", "Input secret manifest file (required)")
	_ = listCmd.MarkFlagRequired("input")

	showCmd.Flags().StringP("input", "i", "", "Input secret manifest file (required)")
	_ = showCmd.MarkFlagRequired("input")
	showCmd.Flags().StringP("key", "k", "", "Show only this key (default: show all)")
}

func runList(cmd *cobra.Command, _ []string) error {
	inputPath, _ := cmd.Flags().GetString("input")

	safeInput, err := safePath("--input", inputPath)
	if err != nil {
		return err
	}

	s, err := manifest.FromFile(safeInput)
	if err != nil {
		return fmt.Errorf("load secret: %w", err)
	}

	keys := make([]string, 0, len(s.Data))
	for k := range s.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Printf("Secret: %s/%s  type: %s  (%d key(s))\n",
		s.Namespace, s.Name, s.Type, len(keys))
	for _, k := range keys {
		fmt.Printf("  %s\n", k)
	}
	return nil
}

func runShow(cmd *cobra.Command, _ []string) error {
	inputPath, _ := cmd.Flags().GetString("input")
	onlyKey, _ := cmd.Flags().GetString("key")

	safeInput, err := safePath("--input", inputPath)
	if err != nil {
		return err
	}

	s, err := manifest.FromFile(safeInput)
	if err != nil {
		return fmt.Errorf("load secret: %w", err)
	}

	// Single-key mode: print just the value for scripting convenience.
	if onlyKey != "" {
		val, err := manifest.GetPlainValue(s, onlyKey)
		if err != nil {
			return err
		}
		fmt.Println(val)
		return nil
	}

	// Full display.
	fmt.Printf("Secret: %s/%s\n", s.Namespace, s.Name)
	fmt.Printf("  type: %s\n", s.Type)

	if s.Immutable != nil && *s.Immutable {
		fmt.Printf("  immutable: true\n")
	}

	if len(s.Labels) > 0 {
		fmt.Printf("  labels:\n")
		for _, k := range sortedStringKeys(s.Labels) {
			fmt.Printf("    %s: %s\n", k, s.Labels[k])
		}
	}

	if len(s.Annotations) > 0 {
		fmt.Printf("  annotations:\n")
		for _, k := range sortedStringKeys(s.Annotations) {
			fmt.Printf("    %s: %s\n", k, s.Annotations[k])
		}
	}

	keys := make([]string, 0, len(s.Data))
	for k := range s.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Printf("  data:\n")
	for _, k := range keys {
		val, err := manifest.GetPlainValue(s, k)
		if err != nil {
			return err
		}
		fmt.Printf("    %s: %s\n", k, val)
	}
	return nil
}

func sortedStringKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
