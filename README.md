# k8s-secret-manifest

A CLI tool for generating, managing, and sealing Kubernetes Secret manifests.

- Generate valid `Secret` YAML with automatic base64 encoding
- Import from / export to `.env` files
- Update, rotate, copy, inspect, diff, and validate existing secret files
- Edit Secret values interactively in `$EDITOR`
- Manage paired index-list keys (e.g. Bitnami pgpool-style semicolon-separated lists)
- Seal secrets with [Sealed Secrets](https://github.com/bitnami-labs/sealed-secrets) (`kubeseal`)

## Installation

### Homebrew / pre-built binary

Download the latest release from the [releases page](https://github.com/pbsladek/k8s-secret-manifest/releases) and place the binary on your `PATH`.

### From source

```bash
go install github.com/pbsladek/k8s-secret-manifest@latest
```

## Global flags

| Flag | Short | Default | Description |
|---|---|---|---|
| `--namespace` | `-n` | `default` | Kubernetes namespace |
| `--kubeseal-path` | `-p` | `kubeseal` | Path to the `kubeseal` binary |

---

## Commands

### `generate` — Create a new Secret manifest

```bash
k8s-secret-manifest generate --name my-secret \
  --set API_KEY=mysecret \
  --set DB_PASS=hunter2 \
  --output secret.yaml
```

**TLS secret** (type set automatically):

```bash
k8s-secret-manifest generate --name tls-secret \
  --tls-cert ./tls.crt --tls-key ./tls.key \
  --output tls-secret.yaml
```

**Docker registry pull secret** (type set automatically):

```bash
k8s-secret-manifest generate --name registry-secret \
  --docker-server ghcr.io \
  --docker-username myuser \
  --docker-password mytoken \
  --output registry-secret.yaml
```

**Paired index-list** (two data keys whose values are separator-matched by position):

```bash
k8s-secret-manifest generate --name pgpool-secret \
  --entries-key PGPOOL_BACKEND_PASSWORD_USERS \
  --entries-val PGPOOL_BACKEND_PASSWORD_PASSWORDS \
  --entry "alice:secretpass" \
  --entry "bob:otherpass" \
  --output pgpool-secret.yaml
```

| Flag | Short | Description |
|---|---|---|
| `--name` | `-N` | Secret name (required) |
| `--set` | `-s` | `key=value`; repeatable |
| `--set-file` | `-f` | `key=filepath`; file content becomes the value; repeatable |
| `--type` | `-t` | Secret type (default: `Opaque`) |
| `--label` | `-l` | Label to set; repeatable |
| `--annotation` | `-a` | Annotation to set; repeatable |
| `--immutable` | | Mark the secret as immutable |
| `--tls-cert` | | Path to TLS certificate file |
| `--tls-key` | | Path to TLS private key file |
| `--docker-server` | | Docker registry server |
| `--docker-username` | | Docker registry username |
| `--docker-password` | | Docker registry password or token |
| `--docker-email` | | Docker registry email (optional) |
| `--entries-key` | `-K` | Data key holding the delimiter-separated identifier list |
| `--entries-val` | `-V` | Data key holding the delimiter-separated value list |
| `--entry` | `-e` | `key:value` entry for the paired lists; repeatable |
| `--separator` | `-S` | Separator for list values (default: `;`) |
| `--output` | `-o` | Output file path (default: stdout) |

---

### `from-env` — Generate a Secret from a `.env` file

Blank lines and `#` comments are skipped. The `export ` prefix is stripped. Values surrounded by single or double quotes are unquoted. See `export-env` for the inverse operation.

```bash
k8s-secret-manifest from-env --name my-secret \
  --env-file .env \
  --output secret.yaml
```

Override or add keys on top of the file:

```bash
k8s-secret-manifest from-env --name my-secret \
  --env-file .env \
  --set EXTRA_KEY=extra \
  --output secret.yaml
```

| Flag | Short | Description |
|---|---|---|
| `--name` | `-N` | Secret name (required) |
| `--env-file` | `-e` | Path to `.env` file (required) |
| `--set` | `-s` | Additional `key=value` to set or overwrite; repeatable |
| `--type` | `-t` | Secret type (default: `Opaque`) |
| `--label` | `-l` | Label to set; repeatable |
| `--annotation` | `-a` | Annotation to set; repeatable |
| `--immutable` | | Mark the secret as immutable |
| `--output` | `-o` | Output file path (default: stdout) |

---

### `update` — Update an existing Secret manifest

Existing keys not mentioned are left unchanged. Outputs to the same file by default.

```bash
k8s-secret-manifest update --input secret.yaml \
  --set API_KEY=newvalue \
  --set-file CA_CERT=./ca.crt \
  --delete-key OLD_KEY \
  --label env=prod \
  --annotation last-rotated=2026-01-01
```

| Flag | Short | Description |
|---|---|---|
| `--input` | `-i` | Input secret manifest file (required) |
| `--output` | `-o` | Output file path (default: same as `--input`) |
| `--set` | `-s` | `key=value` to set or overwrite; repeatable |
| `--set-file` | `-f` | `key=filepath`; file content becomes the value; repeatable |
| `--delete-key` | `-d` | Data key to remove; repeatable |
| `--label` | `-l` | Label to set or overwrite; repeatable |
| `--annotation` | `-a` | Annotation to set or overwrite; repeatable |

---

### `rotate` — Rotate keys with new random values

Replaces one or more data keys with cryptographically random values and updates the file in place. The new plain-text values are printed to stderr so they can be recorded.

```bash
# Rotate a single key (32-char alphanumeric, default)
k8s-secret-manifest rotate --input secret.yaml --key API_KEY

# Rotate multiple keys as 64-char hex strings
k8s-secret-manifest rotate --input secret.yaml \
  --key DB_PASS --key JWT_SECRET \
  --length 64 --charset hex
```

| Flag | Short | Description |
|---|---|---|
| `--input` | `-i` | Input secret manifest file (required) |
| `--output` | `-o` | Output file path (default: same as `--input`) |
| `--key` | `-k` | Key to rotate; repeatable (required) |
| `--length` | `-l` | Length of generated value (default: `32`) |
| `--charset` | `-c` | `alphanumeric` (default), `hex`, or `base64url` |

---

### `export-env` — Export a Secret as a `.env` file

Decodes a Secret manifest and writes it as `KEY=value` lines. Values that contain spaces, quotes, or other shell-significant characters are automatically double-quoted. This is the inverse of `from-env`.

```bash
# Write to a .env file
k8s-secret-manifest export-env --input secret.yaml --output .env

# Print to stdout (e.g. for piping)
k8s-secret-manifest export-env --input secret.yaml
```

| Flag | Short | Description |
|---|---|---|
| `--input` | `-i` | Input secret manifest file (required) |
| `--output` | `-o` | Output `.env` file path (default: stdout) |

---

### `copy` — Clone a Secret with a new name and/or namespace

Copies all data keys, labels, annotations, type, and immutable flag to a new Secret. Uses the global `--namespace` flag for the target namespace.

```bash
# Rename within the same namespace
k8s-secret-manifest copy --input secret.yaml --name new-secret --output new-secret.yaml

# Promote to a different namespace
k8s-secret-manifest copy --input secret.yaml --name prod-secret \
  --namespace production --output prod-secret.yaml
```

| Flag | Short | Description |
|---|---|---|
| `--input` | `-i` | Input secret manifest file (required) |
| `--name` | `-N` | New secret name (required) |
| `--output` | `-o` | Output file path (default: stdout) |

---

### `show` — Decode and display a Secret manifest

```bash
# Show all keys and values (decoded)
k8s-secret-manifest show --input secret.yaml

# Show a single key (useful for scripting)
k8s-secret-manifest show --input secret.yaml --key API_KEY
```

| Flag | Short | Description |
|---|---|---|
| `--input` | `-i` | Input secret manifest file (required) |
| `--key` | `-k` | Show only this key (default: show all) |

---

### `list` — List key names in a Secret manifest

```bash
k8s-secret-manifest list --input secret.yaml
```

| Flag | Short | Description |
|---|---|---|
| `--input` | `-i` | Input secret manifest file (required) |

---

### `diff` — Diff two Secret manifests (decoded)

Keys only in the first file are shown with `-`. Keys only in the second file are shown with `+`. Changed keys show both lines. Color is enabled by default; set `NO_COLOR=1` to disable.

```bash
k8s-secret-manifest diff --from secret-v1.yaml --to secret-v2.yaml

# Also show unchanged keys
k8s-secret-manifest diff --from secret-v1.yaml --to secret-v2.yaml --unchanged
```

| Flag | Short | Description |
|---|---|---|
| `--from` | `-A` | Base secret file (required) |
| `--to` | `-B` | New secret file (required) |
| `--unchanged` | | Also show unchanged keys |

---

### `validate` — Validate a Secret manifest

Check a Secret manifest for spec violations and likely mistakes.

```bash
k8s-secret-manifest validate --input secret.yaml
```

Errors indicate spec violations (invalid name/namespace, missing required keys for the secret type).
Warnings indicate likely mistakes (empty data section, missing recommended keys).

Color output is enabled by default; set `NO_COLOR=1` to disable.

| Flag | Short | Description |
|---|---|---|
| `--input` | `-i` | Input secret manifest file (required) |

---

### `edit` — Edit Secret values interactively

Opens the decoded Secret values in `$EDITOR` as a `.env`-style file. On save and exit the values are re-encoded and the manifest is updated.

```bash
# Edit in $EDITOR (falls back to vi)
k8s-secret-manifest edit --input secret.yaml

# Use a specific editor and write to a different file
EDITOR=nano k8s-secret-manifest edit --input secret.yaml --output new-secret.yaml
```

| Flag | Short | Description |
|---|---|---|
| `--input` | `-i` | Input secret manifest file (required) |
| `--output` | `-o` | Output file path (default: same as `--input`) |

---

### `seal` — Seal a Secret using kubeseal

The plain secret file is left unchanged.

```bash
# Online sealing (requires cluster access)
k8s-secret-manifest seal --input secret.yaml --output sealed-secret.yaml

# Offline sealing (using a fetched public cert)
k8s-secret-manifest seal --input secret.yaml --output sealed-secret.yaml \
  --cert pub-cert.pem
```

| Flag | Short | Description |
|---|---|---|
| `--input` | `-i` | Input plain secret manifest file (required) |
| `--output` | `-o` | Output sealed secret file (default: stdout) |
| `--controller-name` | `-c` | kubeseal controller name (default: `sealed-secrets-controller`) |
| `--controller-namespace` | `-C` | kubeseal controller namespace (default: `kube-system`) |
| `--cert` | `-r` | Path to public certificate for offline sealing |
| `--scope` | `-s` | Sealing scope: `strict`, `namespace-wide`, or `cluster-wide` |

---

### `add-entry` — Add an entry to a paired index-list Secret

```bash
# Append to end (default)
k8s-secret-manifest add-entry --input secret.yaml \
  --entries-key BACKEND_USERS \
  --entries-val BACKEND_PASSWORDS \
  --key carol --value newpass

# Insert at a specific position (index 1 = between existing entries 0 and 1)
k8s-secret-manifest add-entry --input secret.yaml \
  --entries-key BACKEND_USERS \
  --entries-val BACKEND_PASSWORDS \
  --key carol --value newpass \
  --index 1
```

| Flag | Short | Description |
|---|---|---|
| `--input` | `-i` | Input secret manifest file (required) |
| `--output` | `-o` | Output file path (default: same as `--input`) |
| `--entries-key` | `-K` | Data key holding the identifier list (required) |
| `--entries-val` | `-V` | Data key holding the value list (required) |
| `--key` | `-k` | Identifier for the new entry (required) |
| `--value` | `-v` | Value for the new entry (required) |
| `--index` | `-x` | Insert position (default: append to end) |
| `--separator` | `-S` | Separator for list values (default: `;`) |

---

### `remove-entry` — Remove an entry from a paired index-list Secret

Specify the entry to remove by its key **or** its value — not both.

```bash
# Remove by key (removes the key and its paired value)
k8s-secret-manifest remove-entry --input secret.yaml \
  --entries-key BACKEND_USERS \
  --entries-val BACKEND_PASSWORDS \
  --key alice

# Remove by value (removes the value and its paired key)
k8s-secret-manifest remove-entry --input secret.yaml \
  --entries-key BACKEND_USERS \
  --entries-val BACKEND_PASSWORDS \
  --value pass1
```

| Flag | Short | Description |
|---|---|---|
| `--input` | `-i` | Input secret manifest file (required) |
| `--output` | `-o` | Output file path (default: same as `--input`) |
| `--entries-key` | `-K` | Data key holding the identifier list (required) |
| `--entries-val` | `-V` | Data key holding the value list (required) |
| `--key` | `-k` | Remove the entry with this key (mutually exclusive with `--value`) |
| `--value` | `-v` | Remove the entry with this value (mutually exclusive with `--key`) |
| `--separator` | `-S` | Separator for list values (default: `;`) |

---

## Paired index-list format

Some applications (e.g. Bitnami pgpool) store related values as two parallel delimiter-separated strings in two separate Secret data keys, matched by index position:

```yaml
data:
  PGPOOL_BACKEND_PASSWORD_USERS:     YWxpY2U7Ym9i       # base64("alice;bob")
  PGPOOL_BACKEND_PASSWORD_PASSWORDS: cGFzczE7cGFzczI=   # base64("pass1;pass2")
```

`alice ↔ pass1`, `bob ↔ pass2`. The `generate`, `add-entry`, and `remove-entry` commands all manage this format with the `--entries-key` / `--entries-val` / `--separator` flags.

---

## Development

```bash
make          # fmt + vet + test
make build    # compile binary
make test     # verbose test output
make cover    # test coverage report
make lint     # golangci-lint (falls back to go vet)
make release-dry-run  # goreleaser snapshot build
```

## License

[MIT](LICENSE)
