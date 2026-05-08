# Commit-In Tag Mirroring + Migration 006

Implements spec/03-commit-in/08-tag-mirroring-and-release-branches.md per the existing memory rule (`COMMIT-IN TAG MIRRORING`). Inline per-commit mirror inside stage 14, persisted into two new `RewrittenCommit` columns. Idempotent migration 006. CLI + profile JSON wired through the existing precedence chain.

## Scope check (already in tree, do NOT recreate)

- `gitmap/constants/constants_commitin_tagreplay.go` — `VersionTagPattern` regex constant exists.
- `gitmap/constants/constants_commitin_tagreplay_sql.go` — likely holds the column/index DDL strings; confirm before duplicating.
- `gitmap/cmd/commitin/runlog/tagreplay.go` — runlog surface for "would mirror" messages already present.
- `gitmap/store/migrate_commitin_replaymap_*` — sibling migration tests as the pattern reference.
- `constants.ReleaseBranchPrefix = "release/"` — reuse, never copy.

If any of the above already implements part of this spec, the plan step that touches it becomes a verification step, not a fresh write.

## Step 1 — Migration 006 (schema + enum seed)

- New file `gitmap/store/migrations/006_commit_in_tag_mirroring.sql` (or whatever the existing migration directory convention is — scan first).
- DDL exactly per §8.5: idempotent `ADD COLUMN MirroredTagName TEXT NULL`, `ADD COLUMN MirroredReleaseBranch TEXT NULL`, two `CREATE INDEX IF NOT EXISTS` lines.
- Seed `TagsMode (TagsModeId, Name UNIQUE)` enum table with `Annotated`, `All`, `None` (per §8.6). Use `INSERT OR IGNORE` so re-runs are no-ops.
- Register migration in the migration runner (follow whatever 005 does — likely a `migrations` slice in `gitmap/store/migrate*.go`).
- Add a `migrate_commitin_tagmirror_test.go` covering: fresh DB applies cleanly, second apply is a no-op, columns exist as TEXT NULL, both indexes present, enum table seeded with exactly 3 rows.

## Step 2 — CLI + profile plumbing

- Extend `gitmap/cmd/commitin/parse_flags.go` (or sibling) with `--tags`, `--no-release-branch`, `--release-branch-prefix`. Defaults per §8.2.
- Validation rules in `parse_validate.go`:
  - `--tags=None` + any other tag-mirroring flag → exit `BadArgs` (2) with the literal spec message.
  - `--release-branch-prefix` not ending in `/` → exit `BadArgs` (2).
- Profile JSON: extend the profile struct (in `gitmap/cmd/commitin/profile/`) with `TagsMode`, `CreateReleaseBranch`, `ReleaseBranchPrefix`. Apply the existing CLI > profile > default precedence (mirror what other commit-in flags already do — do NOT introduce a new merge mechanism).
- Add table-driven parse + validate tests for each rejection path AND the default-resolution path.
- Add the marker comment for completion generator opt-in (`// gitmap:cmd ...`) per the v3.0.0 mechanism.

## Step 3 — gitutil tag/branch primitives

- New file `gitmap/gitutil/tags.go` (only if not already present — grep first):
  - `AnnotatedTagsAt(repoDir, sha string, mode TagsMode) ([]TagRef, error)` — shells out to `git for-each-ref refs/tags --points-at <sha> --format=...` filtered by `objecttype=tag` for Annotated mode. Returns `Name`, `TaggerName`, `TaggerEmail`, `TaggerDate`, `Message`.
  - `CreateAnnotatedTag(repoDir, name, sha, taggerIdent, taggerDate, message string) error` — uses `GIT_COMMITTER_NAME/EMAIL/DATE` env injection + `git tag -a -m <msg>` so the tag's tagger ident/date are byte-faithful to the source.
  - `CreateBranchAt(repoDir, name, sha string) error` — `git branch <name> <sha>`. If branch exists with same SHA, no-op; if exists at different SHA, return a typed error so the caller can classify as `PartiallyFailed`.
- Unit tests use the existing test-repo helpers (look at how `replay_test.go` builds throwaway repos).

## Step 4 — Stage 14 wiring + persistence

- Touch `gitmap/cmd/commitin/orchestrator/commit.go` (the per-commit driver). Inline expansion of stage 14 per §8.4:
  - After `replay.ApplyCommit` + the existing `RewrittenCommit` insert, call `gitutil.AnnotatedTagsAt` filtered by the resolved `TagsMode`.
  - For each tag: mirror it; if version-tag and `CreateReleaseBranch` and not `--dry-run`, create the branch.
  - Persist into the FIRST tag's existing `RewrittenCommit` row via a new store method `store.UpdateMirroredTagAndBranch(rewId, tagName, branchName)` — single transaction with the InsertRewrittenCommit if possible.
  - Additional tags: insert sibling `Skipped/AdditionalTagAlias` rows. This requires adding `AdditionalTagAlias` to the `SkipReason` enum (its own seed in migration 006).
- `--dry-run`: tag detection still runs; mutations skipped; both columns remain NULL; runlog records the would-mirror lines.
- Failures during tag/branch creation → run is `PartiallyFailed` (exit 1), but the replay commit row is still `Created`. Use the existing zero-swallow error pattern (log to `os.Stderr` with the standardized format).

## Step 5 — End-to-end tests + memory refresh

- New `gitmap/cmd/commitin/e2e/tag_mirror_test.go` covering the §8.7 acceptance matrix T1–T7 verbatim. Reuse `e2e/repo.go` builders. Each row gets one subtest.
- Extend `runlog_test.go` for the dry-run "would mirror" surface.
- Update `mem://features/commit-in-tag-mirroring` to remove "(spec §08, not yet implemented)" and add the migration number, the file paths touched, and the SkipReason enum addition.
- Bump `Current version` core memory line + add a one-line entry to changelog if the project keeps one.

## Out of scope (explicit non-goals)

- No retroactive tag mirroring for already-rewritten history (would need a separate `gitmap commit-in retag` command — not in this spec).
- No lightweight-tag round-trip for `--tags=Annotated` (spec frozen on this).
- No edits to migrations 001–005 (project rule).

## Verification gates before claiming done

1. `go test ./gitmap/store/...` migration tests green on a fresh DB AND on an upgraded one.
2. `go test ./gitmap/cmd/commitin/...` covers all 7 acceptance rows.
3. `gofmt -l gitmap/` clean (auto-handled by fix-repo gofmt rule, but manually verify on touched files).
4. `--tags=None + --no-release-branch` exits 2 with the literal spec message.
5. Inspect a real test SQLite after a T1 run: `SELECT MirroredTagName, MirroredReleaseBranch FROM RewrittenCommit` returns `('v1.2.3', 'release/v1.2.3')`.

Reply `proceed` to start with Step 1.
