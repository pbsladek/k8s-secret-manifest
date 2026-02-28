package cmd

import (
	"fmt"
	"os"

	"github.com/pbsladek/k8s-secret-manifest/internal/entrylist"
	"github.com/pbsladek/k8s-secret-manifest/internal/manifest"
	"github.com/spf13/cobra"
)

var removeEntryCmd = &cobra.Command{
	Use:   "remove-entry",
	Short: "Remove an entry from a paired index-list secret",
	Long: `Remove a key/value entry from a secret that stores two parallel
semicolon-separated lists in two data keys, matched by index position.

Specify the entry to remove by its key OR by its value — not both.

Remove by key (removes alice and its paired value):
  k8s-secret-manifest remove-entry \
    --input secret.yaml \
    --entries-key  BACKEND_USERS \
    --entries-val  BACKEND_PASSWORDS \
    --key alice

Remove by value (removes pass1 and its paired key):
  k8s-secret-manifest remove-entry \
    --input secret.yaml \
    --entries-key  BACKEND_USERS \
    --entries-val  BACKEND_PASSWORDS \
    --value pass1`,
	RunE: runRemoveEntry,
}

func init() {
	removeEntryCmd.Flags().StringP("input", "i", "", "Input secret manifest file (required)")
	_ = removeEntryCmd.MarkFlagRequired("input")

	removeEntryCmd.Flags().StringP("output", "o", "",
		"Output file path (default: same as --input)")

	removeEntryCmd.Flags().StringP("entries-key", "K", "",
		"Data key name holding the semicolon-separated identifier list (required)")
	_ = removeEntryCmd.MarkFlagRequired("entries-key")

	removeEntryCmd.Flags().StringP("entries-val", "V", "",
		"Data key name holding the semicolon-separated value list (required)")
	_ = removeEntryCmd.MarkFlagRequired("entries-val")

	removeEntryCmd.Flags().StringP("key", "k", "", "Remove the entry with this key (mutually exclusive with --value)")
	removeEntryCmd.Flags().StringP("value", "v", "", "Remove the entry with this value (mutually exclusive with --key)")

	removeEntryCmd.Flags().StringP("separator", "S", ";", "Separator used in the list values")
}

func runRemoveEntry(cmd *cobra.Command, _ []string) error {
	inputPath, _ := cmd.Flags().GetString("input")
	outputPath, _ := cmd.Flags().GetString("output")
	entriesKey, _ := cmd.Flags().GetString("entries-key")
	entriesVal, _ := cmd.Flags().GetString("entries-val")
	key, _ := cmd.Flags().GetString("key")
	value, _ := cmd.Flags().GetString("value")
	sep, _ := cmd.Flags().GetString("separator")

	if key == "" && value == "" {
		return fmt.Errorf("one of --key or --value is required")
	}
	if key != "" && value != "" {
		return fmt.Errorf("--key and --value are mutually exclusive")
	}

	if outputPath == "" {
		outputPath = inputPath
	}

	s, err := manifest.FromFile(inputPath)
	if err != nil {
		return fmt.Errorf("load secret: %w", err)
	}

	entries, err := loadEntries(s, entriesKey, entriesVal, sep)
	if err != nil {
		return err
	}

	var removed string
	if key != "" {
		entries, err = entrylist.Remove(entries, key)
		removed = key
	} else {
		entries, err = entrylist.RemoveByValue(entries, value)
		removed = value
	}
	if err != nil {
		return err
	}

	storeEntries(s, entriesKey, entriesVal, sep, entries)

	if err := writeSecretTo(outputPath, s); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Removed entry %q from %s\n", removed, outputPath)
	return nil
}
