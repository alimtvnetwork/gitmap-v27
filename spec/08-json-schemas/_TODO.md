# JSON Schema migration TODO

The following CLI commands currently emit JSON via `json.MarshalIndent(struct, ...)`,
which means **field order is reflection-defined and NOT contractual**. Each one
needs:

1. Migration of the encoder to `gitmap-v23/stablejson` (so ordering becomes a
   compile-time decision instead of a reflection accident).
2. A hand-written `<command>.schema.json` next to this file.
3. A `<command>_jsonschema_contract_test.go` in `gitmap-v23/cmd/` pinning the schema
   against the actual encoder output.

Until a command appears in the table in `README.md`, downstream consumers should
treat its JSON output as **shape-stable but key-order-unstable**.

## Pending commands

(Discovered via `rg -n "json.NewEncoder|json.Marshal" gitmap-v23/cmd/`. Order is
roughly by perceived consumer impact — high-traffic / scripting-friendly first.)

| Priority | Command (file) | Notes |
|---|---|---|
| ✅ done | ~~`gitmap-v23 list-releases --json` (`listreleases.go`, `listreleasesallrepos.go`)~~ | Migrated to `gitmap-v23/stablejson` via `listreleasesrender.go`. Schemas: [`list-releases.schema.json`](list-releases.schema.json) (per-repo, lowerCamel) + [`list-releases-all-repos.schema.json`](list-releases-all-repos.schema.json) (joined --all-repos, PascalCase preserved from legacy `MarshalIndent`). Pinned by `gitmap-v23/cmd/listreleases_jsonschema_contract_test.go` (9 tests incl. byte-compat with legacy output). |
| high | `gitmap-v23 history --json` (`history.go`) | Migrated to stablejson via `historyrender.go` (v5.64.0). Schema: [`history.schema.json`](history.schema.json). Pinned by `gitmap/cmd/history_jsonschema_contract_test.go` + `historyjson_contract_test.go`. |
| high | `gitmap-v23 watch --json` (`watch.go`) | Migrated to stablejson via `watchrender.go` (v5.65.0). Nested repos + summary pre-rendered in compact mode. Schema: [`watch.schema.json`](watch.schema.json). Pinned by `gitmap/cmd/watch_jsonschema_contract_test.go` + `watchjson_contract_test.go`. |
| high | `gitmap-v23 probe-report` (`probereport.go`) | Migrated to stablejson via `proberender.go` (v5.66.0). Schema: [`probe-report.schema.json`](probe-report.schema.json). Pinned by `gitmap/cmd/proberepor_jsonschema_contract_test.go` + `probereporjson_contract_test.go`. |
| med | `gitmap-v23 amend list --json` (`amendlist.go`) | Migrated to stablejson via `amendlistrender.go` (v5.67.0). Schema: [`amend-list.schema.json`](amend-list.schema.json). Pinned by `gitmap/cmd/amendlist_jsonschema_contract_test.go` + `amendlistjson_contract_test.go`. |
| ✅ done | ~~`gitmap-v23 amend audit` (`amendaudit.go`)~~ | Migrated to stablejson via `amendauditrender.go` (v5.68.0). Schema: [`amend-audit.schema.json`](amend-audit.schema.json). Pinned by `gitmap/cmd/amendaudit_jsonschema_contract_test.go` + `amendauditjson_contract_test.go`. |
| ✅ done | ~~`gitmap-v23 diff-profiles --json` (`diffprofiles.go`)~~ | Migrated to stablejson via `diffprofilesrender.go` (v5.69.0). Nested arrays pre-rendered in compact mode. Schema: [`diff-profiles.schema.json`](diff-profiles.schema.json). Pinned by `gitmap/cmd/diffprofiles_jsonschema_contract_test.go` + `diffprofilesjson_contract_test.go`. |
| med | `gitmap-v23 bookmark list --json` (`bookmarklist.go`) | |
| med | `gitmap-v23 project repos --json` (`projectreposoutput.go`) | |
| med | `gitmap-v23 env-registry --json` (`envregistry.go`) | |
| med | `gitmap-v23 export` (`export.go`) | Backup/restore round-trip — strict ordering may matter |
| ✅ done | ~~`gitmap-v23 find-next --json` (`findnext.go`)~~ | Migrated to `gitmap-v23/stablejson` via `findnextrender.go` (v5.63.0). Schema: [`find-next.schema.json`](find-next.schema.json) — nested `repo` allows passthrough. Pinned by `gitmap-v23/cmd/findnext_jsonschema_contract_test.go` (top-level shape + encoder-keys ⊂ schema.properties) on top of the pre-existing tag-order golden in `findnextjson_contract_test.go`. |
| med | `gitmap-v23 rescan --json` (`rescan.go`) | |
| med | `gitmap-v23 latest-branch --json` (`latestbranchoutput.go`) | |
| med | `gitmap-v23 llm-docs` (`llmdocs.go`) | LLM-consumed; ordering helps determinism |
| med | `gitmap-v23 list-versions --json` (`listversionsutil.go`) | |
| med | `gitmap-v23 task list --json` (`taskops.go`) | |
| med | `gitmap-v23 seo write` (`seowritecreate.go`) | Sample/template output |
| low | `gitmap-v23 scan-project` (`scanprojectoutput.go`) | File output, not piped |

## Estimated effort

~30-60 min per command (encoder migration + schema + test). Total ~10-20 hours
of focused work. Recommend tackling in a single sprint to keep the
consumer-contract surface consistent rather than dribbling out one at a time.
