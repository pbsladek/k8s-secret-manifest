package cmd

import (
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"

	"github.com/pbsladek/k8s-secret-manifest/internal/entrylist"
	"github.com/pbsladek/k8s-secret-manifest/internal/manifest"
	"github.com/spf13/cobra"
)

var addEntryCmd = &cobra.Command{
	Use:   "add-entry",
	Short: "Add an entry to a paired index-list secret",
	Long: `Add a key/value entry to a secret that stores two parallel
semicolon-separated lists in two data keys, matched by index position.

Append to the end (default):
  k8s-secret-manifest add-entry \
    --input secret.yaml \
    --entries-key  BACKEND_USERS \
    --entries-val  BACKEND_PASSWORDS \
    --key carol \
    --value newpass

Insert at a specific position (--index 1 inserts between existing index 0 and 1):
  k8s-secret-manifest add-entry \
    --input secret.yaml \
    --entries-key  BACKEND_USERS \
    --entries-val  BACKEND_PASSWORDS \
    --key carol \
    --value newpass \
    --index 1`,
	RunE: runAddEntry,
}

func init() {
	addEntryCmd.Flags().StringP("input", "i", "", "Input secret manifest file (required)")
	_ = addEntryCmd.MarkFlagRequired("input")

	addEntryCmd.Flags().StringP("output", "o", "",
		"Output file path (default: same as --input)")

	addEntryCmd.Flags().StringP("entries-key", "K", "",
		"Data key name holding the semicolon-separated identifier list (required)")
	_ = addEntryCmd.MarkFlagRequired("entries-key")

	addEntryCmd.Flags().StringP("entries-val", "V", "",
		"Data key name holding the semicolon-separated value list (required)")
	_ = addEntryCmd.MarkFlagRequired("entries-val")

	addEntryCmd.Flags().StringP("key", "k", "", "Identifier for the new entry (required)")
	_ = addEntryCmd.MarkFlagRequired("key")

	addEntryCmd.Flags().StringP("value", "v", "", "Value for the new entry (required)")
	_ = addEntryCmd.MarkFlagRequired("value")

	addEntryCmd.Flags().IntP("index", "x", -1,
		"Insert position (0 = first, default: append to end)")
	addEntryCmd.Flags().StringP("separator", "S", ";", "Separator used in the list values")
}

func runAddEntry(cmd *cobra.Command, _ []string) error {
	inputPath, _ := cmd.Flags().GetString("input")
	outputPath, _ := cmd.Flags().GetString("output")
	entriesKey, _ := cmd.Flags().GetString("entries-key")
	entriesVal, _ := cmd.Flags().GetString("entries-val")
	key, _ := cmd.Flags().GetString("key")
	value, _ := cmd.Flags().GetString("value")
	idx, _ := cmd.Flags().GetInt("index")
	sep, _ := cmd.Flags().GetString("separator")

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

	if idx >= 0 {
		entries, err = entrylist.Insert(entries, idx, key, value)
	} else {
		entries, err = entrylist.Add(entries, key, value)
	}
	if err != nil {
		return err
	}

	storeEntries(s, entriesKey, entriesVal, sep, entries)

	if err := writeSecretTo(outputPath, s); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Added entry %q to %s\n", key, outputPath)
	return nil
}

// loadEntries decodes the two list keys from the secret and parses them.
// A missing key is treated as an empty list so the first entry can be added freely.
func loadEntries(s *corev1.Secret, entriesKey, entriesVal, sep string) ([]entrylist.Entry, error) {
	keysPlain := string(s.Data[entriesKey])
	valsPlain := string(s.Data[entriesVal])

	if keysPlain == "" && valsPlain == "" {
		return []entrylist.Entry{}, nil
	}

	return entrylist.Parse(keysPlain, valsPlain, sep)
}

// storeEntries serialises entries and writes them back into the secret.
func storeEntries(s *corev1.Secret, entriesKey, entriesVal, sep string, entries []entrylist.Entry) {
	keysVal, valsVal := entrylist.Serialize(entries, sep)
	manifest.SetPlainValue(s, entriesKey, keysVal)
	manifest.SetPlainValue(s, entriesVal, valsVal)
}

// writeSecretTo serialises a secret and writes it to a file or stdout.
func writeSecretTo(path string, s *corev1.Secret) error {
	data, err := manifest.ToYAML(s)
	if err != nil {
		return err
	}
	return writeOutput(path, data)
}
