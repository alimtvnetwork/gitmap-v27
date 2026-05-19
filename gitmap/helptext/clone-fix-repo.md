# gitmap clone-fix-repo

> 🚀 **One-shot**: `clone` → `cd` → `fix-repo --all`. Same
> URL semantics as `gitmap clone`, including transport coercion
> (`--ssh` / `--https`) and versioned-URL auto-flatten.

Replaces the manual three-step dance:

```
gitmap clone <url>
cd <folder>
gitmap fix-repo --all
```

## Aliases

- 🪄 `cfr` — short form

## Synopsis

```
gitmap clone-fix-repo <url> [folder] [flags]
gitmap cfr           <url> [folder] [flags]
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| 🔐 `--ssh` / `-ssh` / `--sh` | false | Force the URL into `git@host:owner/repo.git` SSH-shorthand form before clone runs. Auto-converts `https://…` and `ssh://git@…` URLs. Mutually exclusive with `--https` (`--ssh` wins with a one-line stderr warning). |
| 🌐 `--https` / `-https` / `--ht` | false | Force the URL into `https://host/owner/repo.git` form. Converts SSH-shorthand and `ssh://…` URLs. Useful in CI where the SSH agent isn't unlocked. |
| 🚫 `--no-vscode-sync` | false | Forwarded to the `clone` step — skips writing the resolved folder into VS Code Project Manager `projects.json`. The `fix-repo` step is unaffected. |
| 🔒 `--require-version` | false | Restore the strict (exit-4) failure mode: fail when the cloned repo identity has no `-vN` suffix instead of skipping the `fix-repo` step. |

Path canonicalization (Clean + EvalSymlinks for Windows 8.3 short
names and symlinks, with soft-fail to the cleaned absolute path on
resolver error) is inherited from the forwarded `clone` step. See
`gitmap clone --help` "Windows path canonicalization & EvalSymlinks
soft-fail" for the full rule set.

## Behavior

1. 📥 **Clone** — exactly like `gitmap clone <url>`. Versioned URLs auto-flatten (e.g. `myrepo-v13` → `myrepo/`). If `[folder]` is given, that name is used verbatim. `--ssh` / `--https` rewrite the URL before clone runs and print `↪ --ssh rewrite: <old> → <new>` to stdout.
2. 📂 **cd** — chdirs into the resolved folder.
3. 🔧 **fix-repo** — re-execs the same gitmap binary with `fix-repo --all` so every prior `{base}-vN` token in tracked text files is rewritten to the current version. Skipped (with a notice) when the repo identity has no `-vN` suffix, unless `--require-version` is set.

## Examples

```
# HTTPS clone + fix
gitmap clone-fix-repo https://github.com/acme/myrepo-v13.git

# 🔐 Same URL, force SSH transport before clone
gitmap cfr https://github.com/acme/myrepo-v13.git --ssh

# 🌐 SSH URL, coerce to HTTPS (CI without SSH agent)
gitmap cfr git@github.com:acme/myrepo-v13.git --https

# SSH clone with explicit folder name
gitmap cfr git@github.com:acme/myrepo-v13.git myrepo-fresh
```

## Exit codes

| Code | Meaning |
|------|---------|
| `0`  | ✅ ok |
| `6`  | ❌ bad-flag (missing URL) |
| `9`  | ❌ chdir failed |
| `10` | ❌ chained step failed (underlying `clone` or `fix-repo` exit code is propagated as-is) |

## See also

- `gitmap clone-fix-repo-pub` (`cfrp`) — same pipeline, plus `make-public --yes` at the end.
- `gitmap clone` — the underlying clone step (full `--ssh` / `--https` semantics live there).
- `gitmap fix-repo` — the underlying rewrite step.
