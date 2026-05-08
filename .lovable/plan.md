# Context-Menu E2E Test Plan (Windows / Linux / macOS)

Goal: end-to-end coverage of `gitmap install ctx` on all three platforms — driving the real install/uninstall code paths against a sandboxed filesystem (and, on Windows, a redirected registry root) and asserting the *exact* artifacts that ship to users: registry keys, `.desktop` files, Automator/Service bundles, plus the resolved argv for every menu entry, including `--explain` prefixing and `Extended` (Shift-click) gating.

Tests live next to the code in `gitmap/cmd/` and follow the existing `*_test.go` conventions (build-tagged where a platform syscall is unavoidable; otherwise pure-Go via injected writers so they run on any CI runner).

---

## Step 1 — Shared harness + argv contract (cross-platform)

Build the foundation every later step reuses.

- New `installctx_harness_test.go`: `tempCtxRoot(t)` returning a scratch dir, a fake-binary path, and a recorder that captures every `(platform, keyPath, exe, argv, mode, extended)` tuple emitted by the install pipeline.
- New `installctx_argv_e2e_test.go`: drives `runInstallCtx` end-to-end with `--explain` both off and on, asserts the announce prefix is present/absent in the rendered command string for every leaf, and asserts argv composition uses real constants (`CmdRelease`, `FlagBumpDash`, `BumpMinor`, `CmdPullAll`, `CmdFindNext`, raw-git `CtxGitHistoryArgs` etc.).
- Extends the existing `installctxentries_argv_test.go` table to feed the harness so there is one source of truth for expected argv.

## Step 2 — Windows registry E2E

Exercise the real Windows code path without touching the user's hive.

- Add `installctx_windows_e2e_test.go` (`//go:build windows`) that points the registry writer at a redirected `HKCU\Software\Classes\__gitmap_test_<pid>` root, runs install → uninstall → install (idempotency), and asserts:
  - Every `KeyName` from `ctxMenu()` exists with the right `MUIVerb`, `Icon`, and `command` default value.
  - `30_pull_all` carries the `Extended` `REG_SZ` (Shift-click gating).
  - `command` strings for Silent / Terminal / Prefill modes match the platform templates in `installctx.go`, including `--explain` `Write-Host` prefix when enabled.
  - Uninstall removes every key the install created and nothing else.
- Add a non-Windows stub test (`//go:build !windows`) that runs the same install pipeline against an in-memory `registryWriter` interface so the assertions execute on Linux CI too.

## Step 3 — Linux `.desktop` + Thunar E2E

- Add `installctx_linux_e2e_test.go` (`//go:build linux`, with a `!linux` parallel using a virtual FS) that runs `runInstallCtx` against a temp `XDG_DATA_HOME`/`XDG_CONFIG_HOME` and asserts:
  - One `.desktop` file per leaf entry under `applications/` with correct `Name=`, `Exec=`, `MimeType=inode/directory`, and `NoDisplay=true` for Extended entries.
  - Thunar custom-actions XML (`installctxlinuxthunar.go`) contains every entry with the right `<command>` and `<unique-id>`.
  - Confirmation guard (`zenity`/`kdialog`/`xmessage` chain) is prepended only for `Extended: true` entries.
  - `--explain` injects the `echo '> gitmap …'` prefix in `Exec=`.
  - Uninstall deletes exactly the files install created (golden file-set diff).

## Step 4 — macOS Automator/Service bundle E2E

- Add `installctx_darwin_e2e_test.go` (`//go:build darwin`, with cross-platform variant) that runs install against a temp `~/Library/Services` and asserts:
  - One `.workflow` bundle per leaf with valid `Contents/Info.plist` (`NSServices` array, `NSMenuItem`/default value, `NSSendTypes=NSFilenamesPboardType`) and `Contents/document.wflow` containing the expected `do shell script` payload.
  - Extended entries prepend the `osascript -e 'display dialog …'` confirm step.
  - `--explain` injects the `echo '> gitmap …'` prefix inside the shell script.
  - Bundle plists round-trip through `plutil -lint` when available; otherwise a pure-Go plist parse asserts structural correctness.
  - Uninstall removes every bundle install created.

## Step 5 — Cross-platform parity + CI wiring

- Add `installctx_parity_e2e_test.go` that runs the install pipeline three times in-process (Windows / Linux / macOS writer backends, all stubbed for cross-OS execution) and asserts the *same* set of `(KeyName, argv, mode, extended)` tuples is emitted on every platform — preventing per-platform drift.
- Add a regression test for the duplicate `90_terminal` / `91_docs` entries currently in `ctxMenu()` (either dedupe in code or pin the duplication intentionally — flag for decision in the PR).
- Wire the new test files into `.github/workflows/ci.yml`'s existing `go test ./...` matrix so the build-tagged platform tests actually execute on the matching runner; add a short `make test-ctx` shortcut for local iteration.
- Update `mem://features/install-ctx-menu` with the new E2E coverage surface and the harness location.

---

## Technical notes

- No new production code in steps 1–4; only a thin `registryWriter` / `fsWriter` interface seam in `installctx.go` so tests can inject fakes. Keeps each file under the 200-line cap.
- All argv assertions go through `reflect.DeepEqual` on `[]string` — never substring matching — so a stray space or quoting bug fails loudly.
- Golden file-set diffs use `testdata/installctx/<platform>/` directories; regenerate via `gitmap regoldens`.
- Build tags are used only where a real syscall is required; the cross-platform variant runs on any runner so PRs from contributors without Windows/macOS still get full signal.

Reply "next" (or with edits) to start Step 1.