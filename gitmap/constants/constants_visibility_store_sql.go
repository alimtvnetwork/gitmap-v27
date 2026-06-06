// Package constants — constants_visibility_store_sql.go: INSERT /
// UPDATE statements for the bulk wildcard visibility audit trail.
// Kept in a separate file from the CREATE TABLE schema to honor the
// ≤200-line per-file rule.
//
// Spec: spec/01-app/116-bulk-visibility-mapub-mapri.md §plan steps 17-18.
package constants

// SQLInsertMakeAllVisibilityRun — pre-prompt INSERT capturing the
// invocation parameters. Counts default to 0 and are flushed by
// SQLUpdateMakeAllVisibilityRunCounts at the end of the run.
const SQLInsertMakeAllVisibilityRun = `INSERT INTO MakeAllVisibilityRun
	(CommandKind, TargetVisibility, Provider, Owner, TargetRaw,
	 PatternList, YesFlag, VerboseFlag, OwnerRepoTotal, MatchedCount,
	 StartedAt)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

// SQLInsertMakeAllVisibilityResult — one per matched repo, written
// inside the pre-prompt transaction so a crash leaves an auditable
// 'Pending' trail.
const SQLInsertMakeAllVisibilityResult = `INSERT INTO MakeAllVisibilityResult
	(MakeAllVisibilityRunId, RepoName, MatchedPattern, Status, StartedAt)
	VALUES (?, ?, ?, ?, ?)`

// SQLUpdateMakeAllVisibilityResultExcluded — bulk mark-as-excluded
// after the user trims the matched set via the prompt's exclude grammar.
const SQLUpdateMakeAllVisibilityResultExcluded = `UPDATE MakeAllVisibilityResult
	SET Status = 'Excluded', FinishedAt = ?
	WHERE MakeAllVisibilityResultId = ?`

// SQLUpdateMakeAllVisibilityResult — per-repo terminal status write
// after the apply+verify pipeline finishes for a single repo.
const SQLUpdateMakeAllVisibilityResult = `UPDATE MakeAllVisibilityResult
	SET Status = ?, PrevVisibility = ?, NewVisibility = ?,
	    FailureMessage = ?, FinishedAt = ?, DurationMs = ?
	WHERE MakeAllVisibilityResultId = ?`

// SQLUpdateMakeAllVisibilityRunCounts — final tally flush + exit code.
const SQLUpdateMakeAllVisibilityRunCounts = `UPDATE MakeAllVisibilityRun
	SET ExcludedCount = ?, OkCount = ?, SkippedCount = ?,
	    FailedCount = ?, ExitCode = ?, FinishedAt = ?
	WHERE MakeAllVisibilityRunId = ?`

// Error format strings — Code Red standard (operation + reason).
const (
	ErrMakeAllRunInsertFmt    = "Error: insert MakeAllVisibilityRun failed: %v (operation: SQLInsertMakeAllVisibilityRun, reason: %s)"
	ErrMakeAllResultInsertFmt = "Error: insert MakeAllVisibilityResult failed: %v (operation: SQLInsertMakeAllVisibilityResult, reason: %s)"
	ErrMakeAllResultUpdateFmt = "Error: update MakeAllVisibilityResult failed: %v (operation: SQLUpdateMakeAllVisibilityResult, reason: %s)"
	ErrMakeAllRunFinalizeFmt  = "Error: finalize MakeAllVisibilityRun failed: %v (operation: SQLUpdateMakeAllVisibilityRunCounts, reason: %s)"
	ErrMakeAllResultExcludeFmt = "Error: exclude MakeAllVisibilityResult rows failed: %v (operation: SQLUpdateMakeAllVisibilityResultExcluded, reason: %s)"
)

// SQLSelectLatestUndoableRun — picks the most recent run that has at
// least one Ok result with a captured PrevVisibility. Used by
// `gitmap visibility-undo` when no explicit --run is supplied.
const SQLSelectLatestUndoableRun = `SELECT
	MakeAllVisibilityRunId, CommandKind, TargetVisibility, Provider,
	Owner, TargetRaw, OkCount
	FROM MakeAllVisibilityRun
	WHERE OkCount > 0
	ORDER BY MakeAllVisibilityRunId DESC
	LIMIT 1`

// SQLSelectUndoableResultsForRun — Ok results with non-empty Prev/New
// visibility that still need reversing, in deterministic ID order.
const SQLSelectUndoableResultsForRun = `SELECT
	MakeAllVisibilityResultId, RepoName, MatchedPattern,
	PrevVisibility, NewVisibility
	FROM MakeAllVisibilityResult
	WHERE MakeAllVisibilityRunId = ?
	  AND Status = 'Ok'
	  AND PrevVisibility != ''
	  AND PrevVisibility != NewVisibility
	ORDER BY MakeAllVisibilityResultId ASC`

// Error format strings for the select path.
const (
	ErrUndoSelectRunFmt     = "Error: select latest undoable run failed: %v (operation: SQLSelectLatestUndoableRun, reason: %s)"
	ErrUndoSelectResultsFmt = "Error: select undoable results failed: %v (operation: SQLSelectUndoableResultsForRun, reason: %s)"
	ErrUndoNoRunFound       = "Error: no undoable make-all-* run found (operation: visibility-undo, reason: MakeAllVisibilityRun has no row with OkCount>0)"
)
