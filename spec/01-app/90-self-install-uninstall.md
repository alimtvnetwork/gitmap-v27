# 90 — Self-Install / Self-Uninstall

> Spec for `gitmap-v26 self-install` and `gitmap-v26 self-uninstall` — manage
> the gitmap-v26 binary itself (NOT to be confused with `gitmap-v26 install` /
> `gitmap-v26 uninstall`, which manage third-party tools).

---

## Why two new commands?

`gitmap-v26 install` and `gitmap-v26 uninstall` were already taken by the
third-party tool installer (npp, vscode, dev tools). Users asked for a
way to wipe the gitmap-v26 binary itself and re-install it from the same
session. To avoid breaking the existing tool installer, we added two
new top-level verbs:

| Command              | Scope                                                     |
|----------------------|-----------------------------------------------------------|
| `gitmap-v26 install`     | Install a third-party tool (existing, unchanged)          |
| `gitmap-v26 uninstall`   | Uninstall a third-party tool (existing, unchanged)        |
| `gitmap-v26 self-install`   | Install / re-install the gitmap-v26 binary                |
| `gitmap-v26 self-uninstall` | Remove the gitmap-v26 binary, data dir, PATH snippet, completion |

---

## self-uninstall: removal scope

A single invocation removes:

1. **Binary + deploy artefacts** — anything under the directory that
   contains the running binary whose name matches `isGitmapArtifact`:
   `gitmap-v26`, `gitmap.exe`, `gitmap-handoff-*`, `*.old` backups,
   `gitmap-completion.*`.
2. **`.gitmap/` data dir** — SQLite DB, profiles, scan history. Skip
   with `--keep-data`.
3. **PATH snippet** — strips the `# gitmap-v26 shell wrapper v* - managed
   by *. Do not edit manually.` … `# gitmap-v26 shell wrapper v* end` block
   from the user's shell profile. Skip with `--keep-snippet`.
4. **Completion files** — `gitmap-completion.bash`, `.zsh`, `.fish` in
   the deploy dir.

### Windows self-deletion handoff

On Windows the running `gitmap.exe` is locked and cannot be deleted by
itself. When `shouldHandoffSelfUninstall()` detects that the running
binary lives inside the about-to-be-deleted deploy dir, it:

1. Copies itself to `%TEMP%\gitmap-handoff-<pid>.exe`.
2. Re-execs the hidden `self-uninstall-runner` verb with the same flags
   plus `--confirm`.
3. The temp copy performs the removal, then schedules its own deletion
   via `cmd.exe /C ping ... & del /F /Q <self>`.

On Unix we just `os.Remove(self)` — open files unlink cleanly.

### Confirmation flow

- Without `--confirm`: prints the target list and waits for typed `yes`.
- With `--confirm`: skips the prompt entirely (suitable for CI).

---

## self-install: source + path resolution

### Install directory

| Source       | Behaviour                                                  |
|--------------|------------------------------------------------------------|
| `--dir <p>`  | Used verbatim                                              |
| `--yes`      | Accept default without prompt                              |
| (default)    | Print prompt with default, accept enter for default        |

Defaults:

- **Windows**: `D:\gitmap-v26`
- **Unix**: `$HOME/.local/bin/gitmap-v26`

### Installer script source

The installer scripts (`install.ps1`, `install.sh`, `uninstall.ps1`)
are embedded into the binary via `go:embed` in
`gitmap-v26/scripts/embed.go`. `loadInstallScript()`:

1. Tries `scripts.ReadFile(name)` first (offline, instant).
2. Falls back to `https://raw.githubusercontent.com/alimtvnetwork/
   gitmap-v26/main/gitmap-v26/scripts/install.{ps1,sh}` if the embedded
   copy is missing or empty.

### Execution

The script is written to `os.TempDir()` (with UTF-8 BOM on PowerShell),
then invoked:

- **Windows**: `pwsh -ExecutionPolicy Bypass -NoProfile -NoLogo -File <tmp> -InstallDir <dir> [-Version <tag>]`
- **Unix**: `bash <tmp> --dir <dir> [--version <tag>]`

---

## File layout

| File                                       | Role                                                 |
|--------------------------------------------|------------------------------------------------------|
| `gitmap-v26/constants/constants_selfinstall.go` | Command IDs, messages, defaults, flag descriptions  |
| `gitmap-v26/scripts/embed.go`                  | `go:embed` of install.ps1, install.sh, uninstall.ps1 |
| `gitmap-v26/cmd/selfinstall.go`                | Entry, flag parsing, prompt, script loader, exec     |
| `gitmap-v26/cmd/selfuninstall.go`              | Entry, flag parsing, confirm, executeSelfUninstall   |
| `gitmap-v26/cmd/selfuninstallparts.go`         | Removers: deploy dir, completion, profile snippet    |
| `gitmap-v26/cmd/selfuninstallhandoff.go`       | Windows temp-copy handoff + self-delete scheduler    |
| `gitmap-v26/helptext/self-install.md`          | User-facing help                                     |
| `gitmap-v26/helptext/self-uninstall.md`        | User-facing help                                     |

---

## Memory rules respected

- All files <200 lines.
- All functions <15 lines.
- No magic strings — everything in `constants_selfinstall.go`.
- Errors written to `os.Stderr` with standardized format strings.
- PowerShell script written with UTF-8 BOM (per `mem://constraints/powershell-encoding`).

---

## See also

- [spec/04-generic-cli/21-post-install-shell-activation/04-idempotency.md](../04-generic-cli/21-post-install-shell-activation/04-idempotency.md)
  — Marker block conventions used by `stripMarkerBlock`.
- [spec/01-app/89-update-path-sync.md](89-update-path-sync.md) — Sister
  spec for keeping deployed and active PATH binaries in sync.
- `gitmap-v26/scripts/install.ps1`, `install.sh`, `uninstall.ps1` — the
  embedded scripts themselves.
