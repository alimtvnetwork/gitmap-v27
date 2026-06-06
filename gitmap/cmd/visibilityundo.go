// Package cmd — visibilityundo.go: `gitmap visibility-undo` (`vu`)
// reverses the most recent successful bulk make-all-* run by reading
// the persisted MakeAllVisibilityResult rows and re-applying each
// repo's PrevVisibility. The undo itself is logged as a new run with
// CommandKind=VisibilityUndo, so a follow-up `vu` reverses the undo
// (this is also how `vr` / visibility-redo is wired in step 23).
//
// Accepts `--run <id>` to target a specific historical run instead
// of the latest one (step 24).
//
// Spec: spec/01-app/116-bulk-visibility-mapub-mapri.md §undo-redo.
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/alimtvnetwork/gitmap-v25/gitmap/constants"
	"github.com/alimtvnetwork/gitmap-v25/gitmap/model"
	"github.com/alimtvnetwork/gitmap-v25/gitmap/store"
	"github.com/alimtvnetwork/gitmap-v25/gitmap/visibility"
)

// undoFlags is parseBulkArgs's sibling for the undo/redo path.
// RunID == 0 means "pick latest". DryRun prints the plan without
// touching the provider. Force bypasses the drift guard (current
// visibility != persisted NewVisibility) so users can overwrite
// out-of-band manual changes explicitly.
type undoFlags struct {
	Verbose bool
	DryRun  bool
	Force   bool
	RunID   int64
}

// runVisibilityUndo is the dispatcher entry point.
func runVisibilityUndo(args []string) {
	flags := parseUndoArgs(args)
	run, results := loadReversible(flags.RunID, "", constants.ErrUndoNoRunFound)
	if flags.DryRun {
		printDryRun(constants.CmdVisibilityUndo, run, results)
		os.Exit(constants.ExitVisOK)
	}
	reverseRunAndExit(run, results, flags, constants.CmdVisibilityUndo)
}

// parseUndoArgs + mustParseRunID + printDryRun live in
// visibilityundoflags.go to keep this file under the 200-line cap.

// loadReversible resolves the target run for either undo or redo.
// When runID > 0 it loads that exact row; otherwise it picks the
// latest row matching `kind` (empty kind = any undoable run).
func loadReversible(runID int64, kind, notFoundMsg string) (model.MakeAllVisibilityRunRecord, []model.MakeAllVisibilityResultRecord) {
	db := openDBOrExit("visibility-undo/redo")
	run := pickReversibleRun(db, runID, kind, notFoundMsg)
	results, err := db.SelectUndoableResultsForRun(run.ID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(constants.ExitVisAuthFailed)
	}

	return run, results
}

// pickReversibleRun dispatches between the by-ID and by-latest paths.
func pickReversibleRun(db *store.DB, runID int64, kind, notFoundMsg string) model.MakeAllVisibilityRunRecord {
	if runID > 0 {
		return mustLoadRunByID(db, runID)
	}

	return mustLoadLatestRun(db, kind, notFoundMsg)
}

// mustLoadRunByID resolves --run <id>.
func mustLoadRunByID(db *store.DB, id int64) model.MakeAllVisibilityRunRecord {
	run, err := db.SelectMakeAllVisibilityRunByID(id)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(constants.ExitVisAuthFailed)
	}
	if run.ID == 0 {
		fmt.Fprintf(os.Stderr, constants.ErrUndoRunNotFoundFmt, id)
		os.Exit(constants.ExitVisConfirmReq)
	}

	return run
}

// mustLoadLatestRun resolves "latest" — optionally filtered by kind.
func mustLoadLatestRun(db *store.DB, kind, notFoundMsg string) model.MakeAllVisibilityRunRecord {
	var (
		run model.MakeAllVisibilityRunRecord
		err error
	)
	if kind == "" {
		run, err = db.SelectLatestUndoableMakeAllVisibilityRun()
	} else {
		run, err = db.SelectLatestMakeAllVisibilityRunByKind(kind)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(constants.ExitVisAuthFailed)
	}
	if run.ID == 0 {
		fmt.Fprintln(os.Stderr, notFoundMsg)
		os.Exit(constants.ExitVisConfirmReq)
	}

	return run
}

// openDBOrExit centralizes the audit-DB open failure path.
func openDBOrExit(cmdLabel string) *store.DB {
	db, err := openDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: audit DB open failed: %v\n", cmdLabel, err)
		os.Exit(constants.ExitVisAuthFailed)
	}

	return db
}

// reverseRunAndExit replays each result row's PrevVisibility through
// the existing read→apply→verify pipeline, logging the operation as
// a fresh run keyed by `cmdName` (CmdVisibilityUndo or *Redo).
func reverseRunAndExit(run model.MakeAllVisibilityRunRecord, results []model.MakeAllVisibilityResultRecord, flags undoFlags, cmdName string) {
	mustEnsureProviderCLI(run.Provider, flags.Verbose)
	mustEnsureProviderAuth(run.Provider, flags.Verbose)
	ctx := ownerContext{Provider: run.Provider, Owner: run.Owner, TargetRaw: run.TargetRaw}
	matches := matchesFromResults(results)
	audit := beginReverseAudit(ctx, flags, run.ID, matches, cmdName)

	fmt.Fprintf(os.Stdout, "%s: reversing run #%d (%s/%s) — %d repo(s)\n",
		cmdName, run.ID, run.Provider, run.Owner, len(results))
	changed, skipped, failed := applyUndoLoop(ctx, results, flags, audit)
	fmt.Fprintf(os.Stdout, constants.MsgBulkSummaryFmt, changed, skipped, failed, len(results))
	exit := bulkExitCode(changed, failed)
	audit.finalize(0, changed, skipped, failed, exit)
	os.Exit(exit)
}

// matchesFromResults synthesizes MatchedRepo entries so the existing
// audit wiring (which expects matches) can be reused unchanged.
func matchesFromResults(rs []model.MakeAllVisibilityResultRecord) []visibility.MatchedRepo {
	out := make([]visibility.MatchedRepo, 0, len(rs))
	for _, r := range rs {
		out = append(out, visibility.MatchedRepo{RepoName: r.RepoName, MatchedPattern: r.MatchedPattern})
	}

	return out
}

// beginReverseAudit writes a fresh MakeAllVisibilityRun row with the
// supplied cmdName (which `commandKindFor` maps to the right enum).
func beginReverseAudit(ctx ownerContext, flags undoFlags, sourceRunID int64, matches []visibility.MatchedRepo, cmdName string) *runAudit {
	patternsRaw := fmt.Sprintf("%s:source-run=%d", cmdName, sourceRunID)

	return beginRunAudit(ctx, "mixed", cmdName, patternsRaw, bulkFlags{Yes: true, Verbose: flags.Verbose}, len(matches), matches)
}

// applyUndoLoop walks the persisted result rows, calling
// applyOneRepo with target = the row's original PrevVisibility.
// When --force is NOT set, a per-repo drift check reads the *current*
// visibility first and skips the reversal if it no longer matches the
// persisted NewVisibility — i.e. someone (or something) changed it
// out-of-band after the original run; blindly re-applying PrevVisibility
// would silently destroy that intentional change.
func applyUndoLoop(ctx ownerContext, rs []model.MakeAllVisibilityResultRecord, flags undoFlags, audit *runAudit) (int, int, int) {
	changed, skipped, failed := 0, 0, 0
	total := len(rs)
	for i, r := range rs {
		fmt.Fprintf(os.Stdout, constants.MsgBulkApplyItemFmt, i+1, total, r.RepoName)
		start := time.Now()
		status := reverseOneRepo(ctx, r, flags)
		audit.updateResult(r.RepoName, status, status.prev, status.next, start)
		switch status.outcome {
		case "skip":
			skipped++
		case "ok":
			changed++
		default:
			failed++
		}
	}

	return changed, skipped, failed
}

// reverseOneRepo enforces the drift guard, then delegates to applyOneRepo.
// Drift = current visibility != the NewVisibility we persisted last time.
// Force overrides the guard with an audible log line.
func reverseOneRepo(ctx ownerContext, r model.MakeAllVisibilityResultRecord, flags undoFlags) applyStatus {
	if flags.Force {
		fmt.Fprintf(os.Stdout, constants.MsgUndoForceOverrideFmt, r.RepoName)

		return applyOneRepo(ctx, r.RepoName, r.PrevVisibility, flags.Verbose)
	}
	slug := ctx.Owner + "/" + r.RepoName
	current, readErr := readVisibilityNoExit(visibilityContext{Provider: ctx.Provider, Slug: slug}, flags.Verbose)
	if readErr != nil {
		fmt.Fprintf(os.Stdout, constants.MsgBulkApplyFailFmt, readErr)

		return applyStatus{outcome: "fail", err: readErr}
	}
	if current != r.NewVisibility {
		fmt.Fprintf(os.Stdout, constants.MsgUndoDriftSkipFmt, current, r.NewVisibility)

		return applyStatus{outcome: "skip", prev: current, next: current}
	}

	return applyOneRepo(ctx, r.RepoName, r.PrevVisibility, flags.Verbose)
}
