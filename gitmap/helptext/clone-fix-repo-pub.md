# gitmap clone-fix-repo-pub

> 🚀 **One-shot**: `clone` → `cd` → `fix-repo --all` → `make-public --yes`.
> Same URL semantics as `gitmap clone`, including transport coercion
> (`--ssh` / `--https`) and versioned-URL auto-flatten.

Replaces the manual four-step dance:

```
gitmap clone <url>
cd <folder>
gitmap fix-repo --all
gitmap make-public --yes
```

## Aliases

- 🪄 `cfrp` — short form

## Synopsis

```
gitmap clone-fix-repo-pub <url> [folder] [flags]
gitmap cfrp               <url> [folder] [flags]
```

## Requirements

- `gh` or `glab` installed and authenticated (`gh auth login` /
  `glab auth login`). The `make-public` step wraps these CLIs.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| 🔐 `--ssh` / `-ssh` / `--sh` | false | Force the URL into `git@host:owner/repo.git` SSH-shorthand form before clone runs. Auto-converts `https://…` and `ssh://git@…` URLs. Mutually exclusive with `--https` (`--ssh` wins with a one-line stderr warning). |
| 🌐 `--https` / `-https` / `--ht` | false | Force the URL into `https://host/owner/repo.git` form. Converts SSH-shorthand and `ssh://…` URLs. Useful in CI where the SSH agent isn't unlocked. |
| 🚫 `--no-vscode-sync` | false | Forwarded to the `clone` step — skips writing the resolved folder into VS Code Project Manager `projects.json`. The `fix-repo` and `make-public` steps are unaffected. |
| 🤐 `--yes` / `-y` | false | Non-interactive: skip the prior-version privatize prompt (see §Behavior step 5) and auto-confirm any chained `make-public` confirmation. |
| 🔒 `--require-version` | false | Strict mode: fail (exit 4) when the cloned repo identity has no `-vN` suffix instead of skipping the `fix-repo` step. |
| 👁️ `--dry-run` / `-n` | false | Preview only — prints the exact `git clone <url> <dest>` command, the absolute target path, and the chained `fix-repo --all → make-public --yes` sequence that *would* run, without invoking git, chdir-ing, or touching remote visibility. (v6.49.0+) |

Path canonicalization (Clean + EvalSymlinks for Windows 8.3 short
names and symlinks, with soft-fail to the cleaned absolute path on
resolver error) is inherited from the forwarded `clone` step. See
`gitmap clone --help` "Windows path canonicalization & EvalSymlinks
soft-fail" for the full rule set.

## Behavior

1. 📥 **Clone** — versioned URLs auto-flatten. `--ssh` / `--https` rewrite the URL before clone runs and print `↪ --ssh rewrite: <old> → <new>` to stdout.
2. 📂 **cd** — chdirs into the resolved folder.
3. 🔧 **fix-repo** — re-execs `fix-repo --all`. Skipped (with a notice) when the repo identity has no `-vN` suffix, unless `--require-version` is set.
4. 🌍 **make-public** — re-execs `make-public --yes` (non-interactive — no confirmation prompt, since the intent is explicit in the command name).

> **v6.50.0+** — `cfrp` no longer scans sibling `-vN` repos nor prompts to privatize prior versions after `make-public`. Run `gitmap mapri <repo>` explicitly when you want bulk-privatize behavior.

Also (v5.61.0+) — if the user's shell cwd is already inside the
target folder, `cfrp` chdir's to the parent before re-cloning so the
Windows file-handle lock never blocks the remove step.

Each step's exit code is propagated as-is; the pipeline halts on
the first non-zero exit.

## Examples

```
# Clone, fix tokens, expose publicly
gitmap clone-fix-repo-pub https://github.com/acme/myrepo-v13.git

# 🔐 Coerce HTTPS URL to SSH transport, then fix + publish
gitmap cfrp https://github.com/acme/myrepo-v13.git --ssh

# 🌐 Coerce SSH URL to HTTPS (CI without SSH agent)
gitmap cfrp git@github.com:acme/myrepo-v13.git --https

# Explicit destination folder
gitmap cfrp git@github.com:acme/myrepo-v13.git myrepo-fresh
```

## Exit codes

| Code | Meaning |
|------|---------|
| `0`  | ✅ ok |
| `6`  | ❌ bad-flag (missing URL) |
| `9`  | ❌ chdir failed |
| `10` | ❌ chained step failed (forwards underlying `clone`, `fix-repo`, or `make-public` exit code) |

## See also

- `gitmap clone-fix-repo` (`cfr`) — same pipeline, without the visibility flip.
- `gitmap clone` — the underlying clone step (full `--ssh` / `--https` semantics live there).
- `gitmap make-public` — the visibility step on its own.
- `gitmap fix-repo` — the rewrite step on its own.

## Scripting (JSON)

Discover this command from a script using the machine-readable help payload:

```bash
gitmap help --json --filter clone-fix-repo-pub
```

The JSON schema is published at `spec/08-json-schemas/help-json.schema.json` (v5.43.0+).
