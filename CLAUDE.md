# k8s-secret-manifest — CLAUDE.md

## Project

Go CLI tool for generating, editing, and sealing Kubernetes Secret manifests.

Module: `github.com/pbsladek/k8s-secret-manifest`

## Build & Test

```bash
go build ./...
go test ./...
```

## Structure

```
main.go
cmd/
  root.go          # cobra root; global flags --namespace, --kubeseal-path;
                   #   writeOutput (0600), splitKeyValue, safePath helpers
  generate.go      # generate subcommand; applySetFiles, applyTLS, applyDockerRegistry
  from_env.go      # from-env subcommand; parseEnvFile, unquote
  update.go        # update subcommand (RMW with file lock)
  rotate.go        # rotate subcommand (RMW with file lock; max length 4096)
  edit.go          # edit subcommand (RMW; resolveEditor via LookPath; MkdirTemp)
  add_user.go      # add-entry subcommand (RMW with file lock)
  remove_user.go   # remove-entry subcommand (RMW with file lock)
  copy.go          # copy subcommand (read-only; writes new file)
  export_env.go    # export-env subcommand
  seal.go          # seal subcommand (pipes through kubeseal; LookPath validated)
  show.go          # list + show subcommands
  diff.go          # diff subcommand
  validate.go      # validate subcommand
  filelock_unix.go   # withExclusiveLock via syscall.Flock (!windows build tag)
  filelock_windows.go # withExclusiveLock no-op (windows build tag)
internal/
  manifest/secret.go   # Secret struct helpers; FromFile, FromYAML, ToYAML
  entrylist/           # Parse/Serialize/Add/Insert/Remove for paired semicolon lists
  validate/validate.go # Secret/key/namespace validation; ValidateDataKey exported
```

## Key Design Decisions

- `data:` values are always base64-encoded (standard K8s secret format).
- YAML output is deterministic (fixed field order, data keys sorted alphabetically).
- Secrets written with 0600 permissions.
- `seal` pipes through kubeseal stdin/stdout; captures stderr for error messages.

## Security Invariants

The following security properties must be maintained in all future changes:

### Path Traversal Prevention
- **`safePath(flag, path string) (string, error)`** in `cmd/root.go` must be called on every user-supplied file path before any read or write operation.
- It rejects relative paths that traverse above the current directory (`..` components).
- `writeOutput` calls `safePath` internally for all non-stdout writes.

### Key Name Validation
- **`validate.ValidateDataKey(key string) error`** must be called on every data key that comes from user input (flags, .env files, edited temp files).
- The regex is `^[-._a-zA-Z0-9]+$`.
- Applied in: `applySetFiles`, `applyTLS` keys, `--set` loops in generate/update/from-env/edit, `--entries-key`/`--entries-val` in generate.

### Subprocess Execution
- **`exec.LookPath`** must be used to resolve any user-controlled binary path before calling `exec.Command`. Currently applied to `--kubeseal-path` (seal.go) and `$EDITOR` (edit.go).
- Never concatenate user input into a shell command string. Always pass arguments as separate elements to `exec.Command`.

### File Locking (Read-Modify-Write)
- All commands that read a file and write it back must wrap the entire operation in **`withExclusiveLock(outputPath, fn)`**.
- Commands with RMW: `update`, `rotate`, `add-entry`, `remove-entry`, `edit`.
- Uses `syscall.Flock` (advisory, Unix-only); Windows has a no-op stub.
- Lock is held on the output path (which defaults to the input path).

### Temp File Safety
- The `edit` command uses `os.MkdirTemp` (mode 0700) to create a private directory before writing decoded secret values. This prevents other local users from reading or swapping the temp file during editing.
- Never use `os.CreateTemp` directly in a shared temp directory for secret content.

### Rotate Length Cap
- `--length` in `rotate` is capped at `maxRotateLength = 4096` to prevent large memory allocations.

### File Permissions
- All secret file writes use mode **0600**. Do not change this.
