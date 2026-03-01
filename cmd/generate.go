package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/pbsladek/k8s-secret-manifest/internal/entrylist"
	"github.com/pbsladek/k8s-secret-manifest/internal/manifest"
	"github.com/pbsladek/k8s-secret-manifest/internal/validate"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a Kubernetes Secret manifest",
	Long: `Generate a valid Kubernetes Secret YAML manifest.

Plain-text values are automatically base64-encoded.

Generic key/value:
  k8s-secret-manifest generate --name my-secret \
    --set API_KEY=mysecret \
    --set-file CA_CERT=./ca.crt

TLS secret (type set automatically):
  k8s-secret-manifest generate --name tls-secret \
    --tls-cert ./tls.crt --tls-key ./tls.key

Docker registry pull secret (type set automatically):
  k8s-secret-manifest generate --name registry-secret \
    --docker-server ghcr.io \
    --docker-username myuser \
    --docker-password mytoken

Paired index-list (two data keys whose values are semicolon-separated and index-matched):
  k8s-secret-manifest generate --name pgpool-secret \
    --entries-key  PGPOOL_BACKEND_PASSWORD_USERS \
    --entries-val  PGPOOL_BACKEND_PASSWORD_PASSWORDS \
    --entry "alice:secretpass" \
    --entry "bob:otherpass"`,
	RunE: runGenerate,
}

func init() {
	generateCmd.Flags().StringP("name", "N", "", "Secret name (required)")
	_ = generateCmd.MarkFlagRequired("name")

	generateCmd.Flags().StringArrayP("set", "s", nil,
		"key=value pair; repeatable (e.g. --set API_KEY=abc)")
	generateCmd.Flags().StringArrayP("set-file", "f", nil,
		"key=filepath pair; file content becomes the value; repeatable (e.g. --set-file CERT=./tls.crt)")

	generateCmd.Flags().StringP("type", "t", "",
		`Secret type (default: Opaque). Common values:
  Opaque
  kubernetes.io/tls
  kubernetes.io/basic-auth
  kubernetes.io/ssh-auth
  kubernetes.io/dockerconfigjson`)

	generateCmd.Flags().StringArrayP("label", "l", nil,
		"Label to set; repeatable (e.g. --label app=myapp)")
	generateCmd.Flags().StringArrayP("annotation", "a", nil,
		"Annotation to set; repeatable (e.g. --annotation managed-by=me)")
	generateCmd.Flags().Bool("immutable", false,
		"Mark the secret as immutable")

	// TLS helper
	generateCmd.Flags().String("tls-cert", "",
		"Path to TLS certificate file; sets type=kubernetes.io/tls and key tls.crt")
	generateCmd.Flags().String("tls-key", "",
		"Path to TLS private key file; sets type=kubernetes.io/tls and key tls.key")

	// Docker registry helper
	generateCmd.Flags().String("docker-server", "",
		"Docker registry server (e.g. ghcr.io); sets type=kubernetes.io/dockerconfigjson")
	generateCmd.Flags().String("docker-username", "", "Docker registry username")
	generateCmd.Flags().String("docker-password", "", "Docker registry password or token")
	generateCmd.Flags().String("docker-email", "", "Docker registry email (optional)")

	// paired index-list
	generateCmd.Flags().StringP("entries-key", "K", "",
		"Data key name holding the delimiter-separated identifier list")
	generateCmd.Flags().StringP("entries-val", "V", "",
		"Data key name holding the delimiter-separated value list")
	generateCmd.Flags().StringArrayP("entry", "e", nil,
		"key:value entry for the paired lists; repeatable (e.g. --entry alice:pass)")
	generateCmd.Flags().StringP("separator", "S", ";",
		"Separator used between entries in the list values (default: \";\")")

	generateCmd.Flags().StringP("output", "o", "", "Output file path (default: stdout)")
}

func runGenerate(cmd *cobra.Command, _ []string) error {
	name, _ := cmd.Flags().GetString("name")
	namespace, _ := cmd.Root().PersistentFlags().GetString("namespace")
	sets, _ := cmd.Flags().GetStringArray("set")
	setFiles, _ := cmd.Flags().GetStringArray("set-file")
	secretType, _ := cmd.Flags().GetString("type")
	labels, _ := cmd.Flags().GetStringArray("label")
	annotations, _ := cmd.Flags().GetStringArray("annotation")
	immutable, _ := cmd.Flags().GetBool("immutable")
	tlsCert, _ := cmd.Flags().GetString("tls-cert")
	tlsKey, _ := cmd.Flags().GetString("tls-key")
	dockerServer, _ := cmd.Flags().GetString("docker-server")
	dockerUsername, _ := cmd.Flags().GetString("docker-username")
	dockerPassword, _ := cmd.Flags().GetString("docker-password")
	dockerEmail, _ := cmd.Flags().GetString("docker-email")
	entriesKey, _ := cmd.Flags().GetString("entries-key")
	entriesVal, _ := cmd.Flags().GetString("entries-val")
	entryFlags, _ := cmd.Flags().GetStringArray("entry")
	sep, _ := cmd.Flags().GetString("separator")
	outputPath, _ := cmd.Flags().GetString("output")

	s := manifest.NewSecret(name, namespace)

	// Explicit type override (applies before helpers so helpers can still set a default)
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

	// Generic key=value pairs
	for _, kv := range sets {
		k, v, err := splitKeyValue(kv)
		if err != nil {
			return err
		}
		if err := validate.ValidateDataKey(k); err != nil {
			return fmt.Errorf("--set: %w", err)
		}
		manifest.SetPlainValue(s, k, v)
	}

	// File-sourced values
	if err := applySetFiles(s, setFiles); err != nil {
		return err
	}

	// TLS helper
	if tlsCert != "" || tlsKey != "" {
		if tlsCert == "" || tlsKey == "" {
			return fmt.Errorf("--tls-cert and --tls-key must both be provided")
		}
		if err := applyTLS(s, tlsCert, tlsKey, secretType); err != nil {
			return err
		}
	}

	// Docker registry helper
	if dockerServer != "" || dockerUsername != "" || dockerPassword != "" {
		if dockerServer == "" || dockerUsername == "" || dockerPassword == "" {
			return fmt.Errorf("--docker-server, --docker-username, and --docker-password are all required")
		}
		if err := applyDockerRegistry(s, dockerServer, dockerUsername, dockerPassword, dockerEmail, secretType); err != nil {
			return err
		}
	}

	// paired index-list
	if entriesKey != "" || entriesVal != "" || len(entryFlags) > 0 {
		if entriesKey == "" || entriesVal == "" {
			return fmt.Errorf("--entries-key and --entries-val are both required when using --entry flags")
		}
		if err := validate.ValidateDataKey(entriesKey); err != nil {
			return fmt.Errorf("--entries-key: %w", err)
		}
		if err := validate.ValidateDataKey(entriesVal); err != nil {
			return fmt.Errorf("--entries-val: %w", err)
		}
		entries, err := parseEntryFlags(entryFlags)
		if err != nil {
			return err
		}
		keysVal, valsVal := entrylist.Serialize(entries, sep)
		manifest.SetPlainValue(s, entriesKey, keysVal)
		manifest.SetPlainValue(s, entriesVal, valsVal)
	}

	yamlBytes, err := manifest.ToYAML(s)
	if err != nil {
		return err
	}

	return writeOutput(outputPath, yamlBytes)
}

// applySetFiles reads key=filepath pairs and stores the file contents as values.
func applySetFiles(s *corev1.Secret, setFiles []string) error {
	for _, kf := range setFiles {
		k, path, err := splitKeyValue(kf)
		if err != nil {
			return fmt.Errorf("--set-file: %w", err)
		}
		if err := validate.ValidateDataKey(k); err != nil {
			return fmt.Errorf("--set-file: %w", err)
		}
		path, err = safePath("--set-file", path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("--set-file %s: %w", k, err)
		}
		if s.Data == nil {
			s.Data = make(map[string][]byte)
		}
		s.Data[k] = data
	}
	return nil
}

// applyTLS reads cert and key files and configures the secret as kubernetes.io/tls.
// The explicit --type flag takes precedence if the user set it.
func applyTLS(s *corev1.Secret, certPath, keyPath, explicitType string) error {
	cleanCert, err := safePath("--tls-cert", certPath)
	if err != nil {
		return err
	}
	cleanKey, err := safePath("--tls-key", keyPath)
	if err != nil {
		return err
	}

	cert, err := os.ReadFile(cleanCert)
	if err != nil {
		return fmt.Errorf("--tls-cert: %w", err)
	}
	key, err := os.ReadFile(cleanKey)
	if err != nil {
		return fmt.Errorf("--tls-key: %w", err)
	}

	if explicitType == "" {
		s.Type = corev1.SecretTypeTLS
	}
	s.Data["tls.crt"] = cert
	s.Data["tls.key"] = key
	return nil
}

// dockerConfigJSON is the structure expected by kubernetes.io/dockerconfigjson.
type dockerConfigJSON struct {
	Auths map[string]dockerAuth `json:"auths"`
}

type dockerAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email,omitempty"`
	Auth     string `json:"auth"` // base64(username:password)
}

// applyDockerRegistry builds the .dockerconfigjson blob and stores it in the secret.
func applyDockerRegistry(s *corev1.Secret, server, username, password, email, explicitType string) error {
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	cfg := dockerConfigJSON{
		Auths: map[string]dockerAuth{
			server: {
				Username: username,
				Password: password,
				Email:    email,
				Auth:     auth,
			},
		},
	}
	blob, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("build dockerconfigjson: %w", err)
	}

	if explicitType == "" {
		s.Type = corev1.SecretTypeDockerConfigJson
	}
	s.Data[corev1.DockerConfigJsonKey] = blob
	return nil
}

// parseEntryFlags parses --entry "key:value" flags.
// The first ":" is the delimiter; values may contain colons.
func parseEntryFlags(flags []string) ([]entrylist.Entry, error) {
	entries := make([]entrylist.Entry, 0, len(flags))
	seen := make(map[string]bool)

	for _, f := range flags {
		idx := strings.IndexByte(f, ':')
		if idx < 0 {
			return nil, fmt.Errorf("invalid --entry %q: expected format key:value", f)
		}
		key := f[:idx]
		value := f[idx+1:]
		if key == "" {
			return nil, fmt.Errorf("invalid --entry %q: key must not be empty", f)
		}
		if seen[key] {
			return nil, fmt.Errorf("duplicate --entry key %q", key)
		}
		seen[key] = true
		entries = append(entries, entrylist.Entry{Key: key, Value: value})
	}
	return entries, nil
}

// parseKeyValuePairs parses a slice of "key=value" strings into a map.
func parseKeyValuePairs(pairs []string, flagName string) (map[string]string, error) {
	m := make(map[string]string, len(pairs))
	for _, p := range pairs {
		k, v, err := splitKeyValue(p)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", flagName, err)
		}
		m[k] = v
	}
	return m, nil
}
