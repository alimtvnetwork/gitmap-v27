# gitmap make-all-private-except-latest

Same as `gitmap make-all-private`, but **preserves the newest `-vN`
sibling** of every base group. Only repos whose name ends in
`-v<digits>` are eligible — the highest version per base stays on
its current visibility, every earlier version is flipped to private.

```
gitmap make-all-private-except-latest <owner-or-url> <patterns> \
    [-Y|--yes] [--verbose] [--parallel=N] [--cache-ttl=SECONDS]
```

## Flags

| Flag | Behavior |
|------|----------|
| `-Y`, `--yes` | Skip the interactive confirmation prompt. |
| `--verbose` | Echo every shell command to stderr before running it. |
| `--parallel=N` | Apply N repos concurrently (default 8, max 32). |
| `--cache-ttl=N` | Override the owner repo-list cache TTL (seconds; 0 disables). |

## Examples

```
gitmap make-all-private-except-latest alice "myapp-v*" -Y
gitmap MAPRIXL alice "demo-v*" --parallel=16
```

## See also

- `gitmap make-all-private` — flip every match, no preservation.
- `gitmap MAPRIXL` — uppercase shorthand for this command.
- `gitmap make-all-public-except-latest` — public counterpart.
- `gitmap visibility-undo` (`vu`) — reverse a completed run.
