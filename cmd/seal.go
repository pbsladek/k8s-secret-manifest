package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var sealCmd = &cobra.Command{
	Use:   "seal",
	Short: "Seal a Secret manifest using kubeseal",
	Long: `Seal a plain Kubernetes Secret manifest into a SealedSecret using kubeseal.

The plain secret YAML is piped through kubeseal via stdin/stdout.
The original plain-text file is left unchanged.

Online sealing (requires cluster access):
  k8s-secret-manifest seal \
    --input secret.yaml \
    --output sealed-secret.yaml \
    --controller-name sealed-secrets-controller \
    --controller-namespace kube-system

Offline sealing (using a fetched public cert):
  k8s-secret-manifest seal \
    --input secret.yaml \
    --output sealed-secret.yaml \
    --cert pub-cert.pem`,
	RunE: runSeal,
}

func init() {
	sealCmd.Flags().StringP("input", "i", "", "Input plain secret manifest file (required)")
	_ = sealCmd.MarkFlagRequired("input")

	sealCmd.Flags().StringP("output", "o", "", "Output sealed secret file (default: stdout)")

	sealCmd.Flags().StringP("controller-name", "c", "sealed-secrets-controller",
		"kubeseal --controller-name")
	sealCmd.Flags().StringP("controller-namespace", "C", "kube-system",
		"kubeseal --controller-namespace")
	sealCmd.Flags().StringP("cert", "r", "",
		"Path to public certificate for offline sealing (kubeseal --cert)")
	sealCmd.Flags().StringP("scope", "s", "",
		"Sealing scope: strict (default), namespace-wide, or cluster-wide")
}

func runSeal(cmd *cobra.Command, _ []string) error {
	inputPath, _ := cmd.Flags().GetString("input")
	outputPath, _ := cmd.Flags().GetString("output")
	controllerName, _ := cmd.Flags().GetString("controller-name")
	controllerNamespace, _ := cmd.Flags().GetString("controller-namespace")
	certPath, _ := cmd.Flags().GetString("cert")
	scope, _ := cmd.Flags().GetString("scope")
	kubesealPath, _ := cmd.Root().PersistentFlags().GetString("kubeseal-path")

	secretYAML, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read input file %q: %w", inputPath, err)
	}

	sealed, err := sealSecret(secretYAML, sealOptions{
		kubesealPath:        kubesealPath,
		controllerName:      controllerName,
		controllerNamespace: controllerNamespace,
		certPath:            certPath,
		scope:               scope,
	})
	if err != nil {
		return err
	}

	return writeOutput(outputPath, sealed)
}

type sealOptions struct {
	kubesealPath        string
	controllerName      string
	controllerNamespace string
	certPath            string
	scope               string
}

// sealSecret pipes secretYAML through kubeseal and returns the SealedSecret YAML.
func sealSecret(secretYAML []byte, opts sealOptions) ([]byte, error) {
	args := []string{
		"--format", "yaml",
		"--controller-name", opts.controllerName,
		"--controller-namespace", opts.controllerNamespace,
	}
	if opts.scope != "" {
		args = append(args, "--scope", opts.scope)
	}
	if opts.certPath != "" {
		args = append(args, "--cert", opts.certPath)
	}

	cmd := exec.Command(opts.kubesealPath, args...)
	cmd.Stdin = bytes.NewReader(secretYAML)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		var execErr *exec.Error
		if errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
			return nil, fmt.Errorf(
				"kubeseal not found at %q; install it or set --kubeseal-path\n"+
					"  https://github.com/bitnami-labs/sealed-secrets#installation",
				opts.kubesealPath,
			)
		}
		msg := stderr.String()
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("kubeseal failed: %s", msg)
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("kubeseal produced no output")
	}

	// Forward any warnings kubeseal printed to stderr even on success.
	if msg := stderr.String(); msg != "" {
		fmt.Fprint(os.Stderr, msg)
	}
	fmt.Fprintf(os.Stderr, "Sealed successfully\n")
	return out, nil
}
