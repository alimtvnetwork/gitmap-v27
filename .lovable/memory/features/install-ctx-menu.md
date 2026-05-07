---
name: Install Ctx Menu
description: Windows-only v1 right-click context menu (`gitmap install ctx`) — nested HKCU cascade, mixed Silent/Terminal/Prefill exec, table-driven from []ctxEntry; macOS+Linux deferred
type: feature
---

# `install ctx` — Windows right-click context menu

**Spec:** [spec/04-generic-cli/30-install-ctx.md](../../../spec/04-generic-cli/30-install-ctx.md)

## v1 scope

- **Windows only** (`HKCU\Software\Classes\Directory\{,Background\}shell\gitmap`).
  Same registry-only pattern as the existing `install vscode-ctx` /
  `install pwsh-ctx` in `gitmap/cmd/installctxmenu.go`. macOS
  `.workflow` bundles + Linux Nautilus/Dolphin servicemenus are
  specced as future work; not implemented.
- **Nested cascade** via legacy `MUIVerb` + `SubCommands` (no COM
  handler). Categories: Scan, Clone, Release, Repos, Visibility,
  Tools + separator + "Open terminal here" + Docs.
- **Mixed exec model:**
  - `Silent` (read-only: `find-next`, `list-releases`, `list-versions`,
    `*-repos`, `docs`, `release-pending`) → `pwsh -WindowStyle Hidden`
    + notifier (BurntToast → msg.exe → temp-log fallback chain
    detected **at install time**, baked into registry).
  - `Terminal` (mutating: `scan`, `rescan`, `clone-next`, `pull*`,
    `release*`, `fix-repo`, `visibility *`, `update`, `diff`, `logs`,
    `history`) → `pwsh -NoExit -Command "Set-Location '%V'; gitmap <args>"`.
  - `Prefill` (special: "Open terminal here") → opens pwsh at folder
    with literal `gitmap ` prompt waiting for input.
- **`install all` excludes `ctx`** — Explorer chrome is opt-in.

## Implementation invariants

- Single `[]ctxEntry` slice in `installctxentries.go` is the source of
  truth for install + uninstall + a unit test asserting every
  `Args[0]` is a real `Cmd*` constant from `constants_cli.go`.
- All literals in `constants_installctx.go` (registry paths, MUIVerbs,
  command templates, error strings).
- HKCU only — never HKLM. Uninstall scopes `reg delete /f` to the
  `gitmap` subtree; never wildcards the parent `shell` key.
- Errors via `cliexit.Reportf` (not bare `fmt.Fprintf`) — required by
  `check-bare-stderr-err.sh` CI gate.
- Files ≤200 lines, functions ≤15 lines (project-wide rule).

## Why nested submenus over flat

20+ top-level entries pollute the root context menu. User explicitly
chose nested categories. Legacy `MUIVerb`+`SubCommands` is sufficient
(no IExplorerCommand COM handler needed) and matches the existing
`vscode-ctx` registry style.

## Why mixed exec over always-terminal

Read-only commands like `list-versions` shouldn't open a terminal
window the user has to dismiss. Mutating/interactive commands must
keep their window so the user sees output + any prompts.
