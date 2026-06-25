# gitmap make-all-public-except-latest

Same as `gitmap make-all-public`, but **preserves the newest `-vN`
sibling** of every base group. Only repos whose name ends in
`-v<digits>` are eligible — the highest version per base stays on
its current visibility, every earlier version is flipped to public.

```
gitmap make-all-public-except-latest <owner-or-url> <patterns> \
    [-Y|--yes] [--verbose] [--parallel=N] [--cache-ttl=SECONDS]
```

## What "latest" means

Repo names are grouped by the literal prefix that precedes the final
`-v<digits>` segment. Within each group the highest integer wins.

| Repo name        | Base group  | Version |
|------------------|-------------|---------|
| `gitmap-v25`     | `gitmap`    | 25      |
| `gitmap-v26`     | `gitmap`    | 26 ✅   |
| `gitmap-v26-rc1` | (no match)  | —       |
| `tooling`        | (no match)  | —       |

In this example, `gitmap-v26` is preserved; `gitmap-v25` is flipped;
`tooling` and `gitmap-v26-rc1` are passed through to the normal flow.

## Flags

| Flag | Behavior |
|------|----------|
| `-Y`, `--yes` | Skip the interactive confirmation prompt. |
| `--verbose` | Echo every shell command to stderr before running it. |
| `--parallel=N` | Apply N repos concurrently (default 8, max 32). |
| `--cache-ttl=N` | Override the owner repo-list cache TTL (seconds; 0 disables). |

## Examples

```
gitmap make-all-public-except-latest alice "myapp-v*" -Y
gitmap make-all-public-except-latest alice "demo-v*" --parallel=16
gitmap MAPUBXL alice "demo-v*"
```

## See also

- `gitmap make-all-public` — flip every match, no preservation.
- `gitmap MAPUBXL` — uppercase shorthand for this command.
- `gitmap make-all-private-except-latest` — private counterpart.
