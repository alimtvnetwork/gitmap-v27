# `install ctx` — Windows Right-Click Context Menu (v1)

> Status: v1 ships **Windows-only** (registry-based, mirrors the existing
> `install vscode-ctx` / `install pwsh-ctx` pattern in
> `gitmap/cmd/installctxmenu.go`). macOS Services and Linux file-manager
> integrations are specced in §7 as **deferred** future work.

## 1. Purpose

Add a single `gitmap ▸` entry to the Windows Explorer right-click menu
on **folder backgrounds** (clicking inside a folder) and on **folder
items** (right-clicking a folder). The entry expands into nested
category submenus that invoke `gitmap` subcommands against the clicked
folder (`%V`).

Installed via:

```
gitmap install ctx          # add the menu (HKCU only — no admin)
gitmap uninstall ctx        # remove the menu
```

`ctx` is added to the existing install-tool table alongside `vscode-ctx`
and `pwsh-ctx`; this spec is **strictly additive** — neither of the
existing two commands is altered.

## 2. Menu Structure

One nested layout under a top-level `gitmap` cascade. Categories mirror
the CLI command groups so users discover commands the same way they do
on the terminal.

```
gitmap ▸
├─ Scan ▸
│   ├─ Scan here                       (gitmap scan)
│   ├─ Rescan                          (gitmap rescan)
│   └─ Find next                       (gitmap find-next)
├─ Clone ▸
│   ├─ Clone-next here                 (gitmap clone-next)
│   ├─ Pull                            (gitmap pull)
│   └─ Pull all                        (gitmap pull-all)
├─ Release ▸
│   ├─ Release current                 (gitmap release)
│   ├─ Release next (bump minor)       (gitmap release --bump minor)
│   ├─ Release pull                    (gitmap release-pull)
│   ├─ Release pending          [N]    (gitmap release-pending)
│   ├─ List releases                   (gitmap list-releases)
│   └─ List versions                   (gitmap list-versions)
├─ Repos ▸
│   ├─ Go projects                     (gitmap go-repos)
│   ├─ Node projects                   (gitmap node-repos)
│   ├─ React projects                  (gitmap react-repos)
│   ├─ C++ projects                    (gitmap cpp-repos)
│   ├─ C# projects                     (gitmap csharp-repos)
│   ├─ Rust projects        [future]   (gitmap rust-repos)
│   └─ PHP projects         [future]   (gitmap php-repos)
├─ Visibility ▸
│   ├─ Make public                     (gitmap visibility public)
│   └─ Make private                    (gitmap visibility private)
├─ Tools ▸
│   ├─ Fix repo                        (gitmap fix-repo)
│   ├─ Diff                            (gitmap diff)
│   ├─ Logs                            (gitmap logs)
│   ├─ History                         (gitmap history)
│   └─ Update                          (gitmap update)
├─ ─────────────                       (separator)
├─ Open terminal here                  (open pwsh, prefill `gitmap `)
└─ Docs                                (gitmap docs)
```

Rust/PHP entries are stubbed in the menu only when the underlying
commands ship (gated by `constants.HasRustRepos` / `HasPhpRepos`
build-time flags). Until then the rows are omitted.

### 2.1 Windows registry layout

Use the legacy `MUIVerb` + `SubCommands` cascade (no COM handler). All
keys live under **HKCU** so install requires no elevation:

```
HKCU\Software\Classes\Directory\Background\shell\gitmap
    (Default)        = (empty)
    MUIVerb          = "gitmap"
    SubCommands      = ""               ; empty => use ExtendedSubCommandsKey
    Icon             = "<gitmap.exe path>,0"

HKCU\Software\Classes\Directory\Background\shell\gitmap\shell\01_scan
    MUIVerb          = "Scan"
    SubCommands      = ""
    HKCU\...\01_scan\shell\01_scan_here
        MUIVerb      = "Scan here"
        \command (Default) = "<exec template>"
    ...
```

Mirror the same tree under `Directory\shell\gitmap` so right-clicking
the folder **item** (not just background) also works. Generation is
table-driven from a single `[]ctxEntry` slice — see §4.

## 3. Execution Model (mixed)

Each entry declares `Mode` ∈ {`Silent`, `Terminal`}.

| Mode       | Used for                                                 | Command template                                                                                                                                                              |
| ---------- | -------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Silent`   | Read-only / fast queries: `find-next`, `list-releases`, `list-versions`, `*-repos`, `docs`, `release-pending` | `pwsh -NoProfile -WindowStyle Hidden -Command "Set-Location '%V'; $o = gitmap <args> 2>&1 \| Out-String; New-BurntToastNotification -Text 'gitmap <label>', $o"` (BurntToast optional; falls back to `msg.exe %username% "<first 200 chars>"`) |
| `Terminal` | Mutating / interactive / long: `scan`, `rescan`, `clone-next`, `pull`, `pull-all`, `release*`, `fix-repo`, `visibility *`, `update`, `diff`, `logs`, `history` | `pwsh -NoExit -NoProfile -Command "Set-Location '%V'; gitmap <args>"`                                                                                                         |
| `Prefill`  | Special: **Open terminal here**                          | `pwsh -NoExit -NoProfile -Command "Set-Location '%V'; Write-Host -NoNewline 'gitmap '"` — leaves a `gitmap ` prompt for the user to complete                                  |

### 3.1 Notification fallback chain

1. If `BurntToast` PowerShell module is available → toast.
2. Else if `wsl` not running and `msg.exe` exists → modal popup.
3. Else write to `%TEMP%\gitmap-ctx-<unix>.log` and toast a single
   "Output saved to %TEMP%\…" line via `[System.Windows.Forms]`.

Detection happens **at install time once**, and the chosen template is
baked into the `(Default)` of each `\command` key. No runtime probing.

## 4. Implementation Layout

```
gitmap/cmd/installctx.go            // entry point — runInstallCtx / runUninstallCtx
gitmap/cmd/installctxentries.go     // []ctxEntry table (single source of truth)
gitmap/cmd/installctxregistry.go    // reg add/delete helpers (table-driven)
gitmap/cmd/installctxnotify.go      // probe BurntToast/msg.exe at install time
gitmap/constants/constants_installctx.go  // all literals (tool name, key paths, MUIVerbs, flag names)
```

`ctxEntry` shape:

```go
type ctxEntry struct {
    KeyName  string   // "10_release_next" — numeric prefix preserves order
    MUIVerb  string   // "Release next (bump minor)"
    Args     []string // {"release", "--bump", "minor"}
    Mode     ctxMode  // Silent | Terminal | Prefill
    Category string   // "Release" — empty = top-level under gitmap
}
```

The same slice drives:
- install (write keys),
- uninstall (delete the `gitmap` subtree only, never neighbors),
- a unit test that asserts every entry references a real `Cmd*`
  constant from `constants_cli.go` (catches drift when commands are
  renamed).

### 4.1 Wire-up to the existing install dispatcher

`gitmap/constants/constants_installctx.go`:

```go
const ToolCtx = "ctx"
```

`gitmap/cmd/install.go::specialInstallHandler`:

```go
case constants.ToolCtx:
    return func(installOptions) { runInstallCtx() }
```

`gitmap/cmd/uninstall.go` mirrors the `vscode-ctx` / `pwsh-ctx` branches.

### 4.2 Tool-table entry

Append to `constants_install.go`:

| Field           | Value                                                |
| --------------- | ---------------------------------------------------- |
| `ToolCtx`       | `"ctx"`                                              |
| description     | `"Add gitmap to Windows right-click context menu"`   |
| `allInstallable`| **omit** — `install all` should NOT install `ctx` (it changes Explorer chrome; users opt in explicitly). |

## 5. Acceptance Criteria

1. `gitmap install ctx` on Windows writes the full key tree under
   `HKCU\Software\Classes\Directory\{Background,}\shell\gitmap` and
   prints `✓ gitmap context menu installed (X/X registry keys).`.
2. `gitmap uninstall ctx` deletes **only** the `gitmap` subtree from
   both locations and prints a parallel summary. `vscode-ctx` /
   `pwsh-ctx` keys are untouched.
3. Right-clicking a folder background shows `gitmap ▸` with all five
   category submenus + the separator + Open-terminal + Docs entries.
4. Each `Terminal`-mode entry opens a non-closing `pwsh` window at the
   clicked folder and runs `gitmap <args>`.
5. Each `Silent`-mode entry surfaces output via the
   install-time-detected notifier (BurntToast → msg.exe → temp-log).
6. **Open terminal here** opens `pwsh` at the folder with a literal
   `gitmap ` prompt waiting for input (no command executed yet).
7. On non-Windows, both commands print the same OS-not-supported error
   the existing `vscode-ctx` handler prints, then exit 0 (parity).
8. A unit test (`installctxentries_test.go`) asserts every `ctxEntry.Args[0]`
   is one of the `Cmd*` constants in `constants_cli.go`.

## 6. Constraints

- All literals (registry paths, MUIVerbs, command templates, error
  strings) live in `constants_installctx.go` — no string literals in
  `installctx*.go`.
- Functions ≤15 lines; files ≤200 lines (split into the four files
  above).
- HKCU only — never write to `HKLM` (would require admin and affect
  other users).
- Uninstall must use `reg delete /f` scoped to the `gitmap` key only —
  never wildcard the parent `shell` key.
- Use `cliexit.Reportf` for any error print (not bare `fmt.Fprintf`),
  per the `check-bare-stderr-err.sh` CI gate.

## 7. Deferred — macOS / Linux (future)

### 7.1 macOS — `~/Library/Services/*.workflow`

Each menu entry becomes one Automator-Service `.workflow` bundle
generated from a Go-embedded plist template. Bundles are dropped in
`~/Library/Services/` and appear under **Finder ▸ Services** and the
right-click contextual menu after `pkill -kill -u $USER cfprefsd`. No
code-signing required for user-installed Services. **Deferred**: the
template + 25-bundle generator is non-trivial and the cascade UX
differs from Windows (Services list is flat).

### 7.2 Linux — Nautilus + Dolphin (GNOME + KDE only)

- **Nautilus**: one shell script per entry in
  `~/.local/share/nautilus/scripts/gitmap/`. Cascade is implicit via
  the `gitmap/` subdirectory.
- **Dolphin**: one `.desktop` file per category in
  `~/.local/share/kio/servicemenus/` with `Actions=` listing the
  entries. Single `gitmap-ctx.desktop` produces the cascade.

Thunar / Nemo / Caja are **out of scope** for v2 — would need per-WM
config-file generators with no shared format. v2 ships if and only if
both Nautilus and Dolphin can be covered by the same `[]ctxEntry`
table that v1 uses (no per-platform action drift).

## 8. Cross-References

- Existing pattern: `gitmap/cmd/installctxmenu.go` (`vscode-ctx`,
  `pwsh-ctx`) — copy the `runRegistryCommands` reporting style.
- Memory: `mem://features/install-ctx-menu`.
- CI gate: `.github/scripts/check-bare-stderr-err.sh` — must pass.
