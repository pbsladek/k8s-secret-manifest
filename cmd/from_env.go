package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/pbsladek/k8s-secret-manifest/internal/manifest"
	"github.com/spf13/cobra"
)

var fromEnvCmd = &cobra.Command{
	Use:   "from-env",
	Short: "Generate a Secret manifest from a .env file",
	Long: `Generate a Kubernetes Secret manifest by reading key=value pairs from a .env file.

Blank lines and lines starting with # are ignored.
The "export " prefix is stripped if present.
Values surrounded by single or double quotes are unquoted.

Example:
  k8s-secret-manifest from-env \
    --name my-secret \
    --env-file .env \
    --output secret.yaml

Override or add keys on top of the .env file:
  k8s-secret-manifest from-env \
    --name my-secret \
    --env-file .env \
    --set EXTRA_KEY=extra`,
	RunE: runFromEnv,
}

func init() {
	fromEnvCmd.Flags().StringP("name", "N", "", "Secret name (required)")
	_ = fromEnvCmd.MarkFlagRequired("name")

	fromEnvCmd.Flags().StringP("env-file", "e", "", "Path to .env file (required)")
	_ = fromEnvCmd.MarkFlagRequired("env-file")

	fromEnvCmd.Flags().StringP("output", "o", "", "Output file path (default: stdout)")

	fromEnvCmd.Flags().StringP("type", "t", "",
		"Secret type (default: Opaque)")
	fromEnvCmd.Flags().StringArrayP("label", "l", nil,
		"Label to set; repeatable (e.g. --label app=myapp)")
	fromEnvCmd.Flags().StringArrayP("annotation", "a", nil,
		"Annotation to set; repeatable (e.g. --annotation managed-by=me)")
	fromEnvCmd.Flags().Bool("immutable", false, "Mark the secret as immutable")

	fromEnvCmd.Flags().StringArrayP("set", "s", nil,
		"Additional key=value to set or overwrite; repeatable")
}

func runFromEnv(cmd *cobra.Command, _ []string) error {
	name, _ := cmd.Flags().GetString("name")
	namespace, _ := cmd.Root().PersistentFlags().GetString("namespace")
	envFile, _ := cmd.Flags().GetString("env-file")
	outputPath, _ := cmd.Flags().GetString("output")
	secretType, _ := cmd.Flags().GetString("type")
	labels, _ := cmd.Flags().GetStringArray("label")
	annotations, _ := cmd.Flags().GetStringArray("annotation")
	immutable, _ := cmd.Flags().GetBool("immutable")
	sets, _ := cmd.Flags().GetStringArray("set")

	pairs, err := parseEnvFile(envFile)
	if err != nil {
		return fmt.Errorf("parse env file: %w", err)
	}

	s := manifest.NewSecret(name, namespace)

	if secretType != "" {
		s.Type = corev1.SecretType(secretType)
	}

	if len(labels) > 0 {
		lmap, err := parseKeyValuePairs(labels, "--label")
		if err != nil {
			return err
		}
		s.Labels = lmap
	}

	if len(annotations) > 0 {
		amap, err := parseKeyValuePairs(annotations, "--annotation")
		if err != nil {
			return err
		}
		s.Annotations = amap
	}

	if immutable {
		t := true
		s.Immutable = &t
	}

	for k, v := range pairs {
		manifest.SetPlainValue(s, k, v)
	}

	// --set overrides env file values
	for _, kv := range sets {
		k, v, err := splitKeyValue(kv)
		if err != nil {
			return err
		}
		manifest.SetPlainValue(s, k, v)
	}

	yamlBytes, err := manifest.ToYAML(s)
	if err != nil {
		return err
	}

	return writeOutput(outputPath, yamlBytes)
}

// parseEnvFile reads a .env file and returns key=value pairs.
// Blank lines and # comments are skipped. "export " prefix is stripped.
// Values surrounded by matching single or double quotes are unquoted.
func parseEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.TrimPrefix(line, "export ")

		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			return nil, fmt.Errorf("line %d: expected KEY=value, got %q", lineNum, line)
		}

		key := strings.TrimSpace(line[:idx])
		if key == "" {
			return nil, fmt.Errorf("line %d: empty key", lineNum)
		}

		val := unquote(line[idx+1:])
		result[key] = val
	}

	return result, scanner.Err()
}

// unquote strips a matching pair of surrounding single or double quotes.
func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
