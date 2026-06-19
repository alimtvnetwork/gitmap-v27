# gitmap rm

Remove one or more repositories from the gitmap database.

## Usage

```
gitmap rm <name-or-path> [<name-or-path> ...]
gitmap remove <name-or-path> [<name-or-path> ...]
```

Alias: `remove`.

## What it does

For each target, `gitmap rm` first resolves it as an absolute filesystem
path (`filepath.Abs`) and deletes the matching row from the gitmap DB.
If no row matches the path, it falls back to treating the target as a
repo slug/name and deletes every row with that slug.

**On-disk files are NOT touched.** This command only untracks the repo
in the gitmap database — `git`, `node_modules`, working trees, and
remotes are all left alone. To delete the folder too, use your shell
(`rm -rf <path>`) after `gitmap rm`.

## Examples

```
$ gitmap rm my-repo
removed 1 repo(s) by name: my-repo

$ gitmap rm ./projects/foo ../bar
removed 1 repo by path: /home/me/projects/foo
removed 1 repo by path: /home/me/bar

$ gitmap rm repo-a repo-b /abs/path/repo-c
removed 1 repo(s) by name: repo-a
removed 1 repo(s) by name: repo-b
removed 1 repo by path: /abs/path/repo-c
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0    | Every target matched and was removed |
| 1    | At least one target did not match (warning per missing target on stderr) |

## See also

- `gitmap list` (`gitmap ls`) — show every tracked repo
- `gitmap rescan` — re-discover repos under a scan root
