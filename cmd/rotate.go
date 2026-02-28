package cmd

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/pbsladek/k8s-secret-manifest/internal/manifest"
	"github.com/spf13/cobra"
)

const (
	charsetAlphanumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	charsetHex          = "0123456789abcdef"
	charsetBase64URL    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
)

var rotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Rotate keys with new cryptographically random values",
	Long: `Replace one or more data keys with new cryptographically random values.

The new plain-text values are printed to stderr so they can be recorded.
The secret file is updated in place (or to --output if specified).

Example — rotate a single key:
  k8s-secret-manifest rotate --input secret.yaml --key API_KEY

Example — rotate multiple keys with a hex value of length 64:
  k8s-secret-manifest rotate --input secret.yaml \
    --key DB_PASS --key JWT_SECRET \
    --length 64 --charset hex`,
	RunE: runRotate,
}

func init() {
	rotateCmd.Flags().StringP("input", "i", "", "Input secret manifest file (required)")
	_ = rotateCmd.MarkFlagRequired("input")

	rotateCmd.Flags().StringP("output", "o", "",
		"Output file path (default: same as --input)")

	rotateCmd.Flags().StringArrayP("key", "k", nil,
		"Key to rotate; repeatable (required)")
	_ = rotateCmd.MarkFlagRequired("key")

	rotateCmd.Flags().IntP("length", "l", 32,
		"Length of the generated value in characters")
	rotateCmd.Flags().StringP("charset", "c", "alphanumeric",
		"Character set for generated value: alphanumeric, hex, base64url")
}

func runRotate(cmd *cobra.Command, _ []string) error {
	inputPath, _ := cmd.Flags().GetString("input")
	outputPath, _ := cmd.Flags().GetString("output")
	keys, _ := cmd.Flags().GetStringArray("key")
	length, _ := cmd.Flags().GetInt("length")
	charsetName, _ := cmd.Flags().GetString("charset")

	if outputPath == "" {
		outputPath = inputPath
	}

	charset, err := resolveCharset(charsetName)
	if err != nil {
		return err
	}

	s, err := manifest.FromFile(inputPath)
	if err != nil {
		return fmt.Errorf("load secret: %w", err)
	}

	for _, key := range keys {
		if _, ok := s.Data[key]; !ok {
			return fmt.Errorf("key %q not found in secret data", key)
		}
		val, err := randomString(length, charset)
		if err != nil {
			return fmt.Errorf("generate value for %q: %w", key, err)
		}
		manifest.SetPlainValue(s, key, val)
		fmt.Fprintf(os.Stderr, "%s=%s\n", key, val)
	}

	if err := writeSecretTo(outputPath, s); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Rotated %d key(s) in %s\n", len(keys), outputPath)
	return nil
}

// resolveCharset returns the character set string for the given name.
func resolveCharset(name string) (string, error) {
	switch strings.ToLower(name) {
	case "alphanumeric":
		return charsetAlphanumeric, nil
	case "hex":
		return charsetHex, nil
	case "base64url":
		return charsetBase64URL, nil
	default:
		return "", fmt.Errorf("unknown charset %q: use alphanumeric, hex, or base64url", name)
	}
}

// randomString generates a cryptographically random string of the given length
// using characters drawn uniformly from charset.
func randomString(length int, charset string) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be positive")
	}
	n := big.NewInt(int64(len(charset)))
	result := make([]byte, length)
	for i := range result {
		idx, err := rand.Int(rand.Reader, n)
		if err != nil {
			return "", err
		}
		result[i] = charset[idx.Int64()]
	}
	return string(result), nil
}
