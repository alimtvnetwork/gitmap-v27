package runlog

import "github.com/alimtvnetwork/gitmap-v18/gitmap/constants"

// ResolveReleaseBranchName returns the destination release-branch
// name for a mirrored annotated tag, or "" when no branch should be
// created. Single decision point — every caller (§08 walker, dry-run
// planner, idempotency lookup) MUST go through here so the
// `--no-release-branch` semantics never drift.
//
// Rules (spec §08 §2.5 + §09 §9.4):
//
//   1. `isNoReleaseBranch=true`  → "" (suppressed regardless of tag).
//   2. tag name not a SemVer version tag → "" (only version tags get
//      auto branches; `nightly` etc. mirror the tag only).
//   3. `isDryRun=true`           → "" (dry-run never materialises a
//      branch; `MirroredReleaseBranch` is NULL per spec §9.4 + R6).
//   4. otherwise → `constants.ReleaseBranchPrefix + tagName`
//      (e.g. "release/v1.2.3"). The prefix is reused verbatim so a
//      future `--release-branch-prefix` flag has exactly one place
//      to override.
//
// Pure: no git, no DB, no clock. Trivially testable.
func ResolveReleaseBranchName(tagName string, isNoReleaseBranch, isDryRun bool) string {
	if isNoReleaseBranch {
		return ""
	}
	if !IsAnnotatedSemverVersionTag(tagName) {
		return ""
	}
	if isDryRun {
		return ""
	}
	return constants.ReleaseBranchPrefix + tagName
}
