// Package cmd — visibilityundo.go: `gitmap visibility-undo` (`vu`)
// reverses the most recent successful bulk make-all-* run by reading
// the persisted MakeAllVisibilityResult rows and re-applying each
// repo's PrevVisibility. The undo itself is logged as a new run with
// CommandKind=VisibilityUndo, so a follow-up `vu` reverses the undo
// (this is also how redo will be wired in step 23).
//
// Spec: spec/01-app/116-bulk-visibility-mapub-mapri.md §undo-redo.
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/alimtvnetwork/gitmap-v25/gitmap/constants"
	"github.com/alimtvnetwork/gitmap-v25/gitmap/model"
	"github.com/alimtvnetwork/gitmap-v25/gitmap/visibility"
)

// runVisibilityUndo is the dispatcher entry point.
func runVisibilityUndo(args []string) {
	flags := parseUndoFlags(args)
	run, results := loadUndoTargets()
	mustEnsureProviderCLI(run.Provider, flags.Verbose)

	ctx := ownerContext{Provider: run.Provider, Owner: run.Owner, TargetRaw: run.TargetRaw}
	matches := matchesFromResults(results)
	audit := beginUndoAudit(ctx, flags, run.ID, matches)

	fmt.Fprintf(os.Stdout, "visibility-undo: reversing run #%d (%s/%s) — %d repo(s)\n",
		run.ID, run.Provider, run.Owner, len(results))
	changed, skipped, failed := applyUndoLoop(ctx, results, flags.Verbose, audit)
	fmt.Fprintf(os.Stdout, constants.MsgBulkSummaryFmt, changed, skipped, failed, len(results))
	exit := bulkExitCode(changed, failed)
	audit.finalize(0, changed, skipped, failed, exit)
	os.Exit(exit)
}

// parseUndoFlags accepts --verbose anywhere; --yes is accepted but a
// no-op (undo always runs without re-prompting — the original prompt
// already gated the data being reversed).
func parseUndoFlags(args []string) bulkFlags {
	flags := bulkFlags{Yes: true}
	for _, a := range args {
		if a == "--verbose" {
			flags.Verbose = true
		}
	}

	return flags
}

// loadUndoTargets opens the audit DB, picks the latest undoable run,
// and returns its result rows. Exits with a clean message when no
// undoable run exists (zero-swallow — never silently no-op).
func loadUndoTargets() (model.MakeAllVisibilityRunRecord, []model.MakeAllVisibilityResultRecord) {
	db, err := openDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "visibility-undo: audit DB open failed: %v\n", err)
		os.Exit(constants.ExitVisAuthFailed)
	}
	run, err := db.SelectLatestUndoableMakeAllVisibilityRun()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(constants.ExitVisAuthFailed)
	}
	if run.ID == 0 {
		fmt.Fprintln(os.Stderr, constants.ErrUndoNoRunFound)
		os.Exit(constants.ExitVisConfirmReq)
	}
	results, err := db.SelectUndoableResultsForRun(run.ID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(constants.ExitVisAuthFailed)
	}

	return run, results
}

// matchesFromResults synthesizes MatchedRepo entries from the persisted
// result rows so the existing audit wiring (which expects matches) can
// be reused without modification.
func matchesFromResults(rs []model.MakeAllVisibilityResultRecord) []visibility.MatchedRepo {
	out := make([]visibility.MatchedRepo, 0, len(rs))
	for _, r := range rs {
		out = append(out, visibility.MatchedRepo{RepoName: r.RepoName, MatchedPattern: r.MatchedPattern})
	}

	return out
}

// beginUndoAudit writes a fresh MakeAllVisibilityRun row with
// CommandKind=VisibilityUndo + one Pending result per reversed repo.
// PatternList encodes the source run ID for traceability.
func beginUndoAudit(ctx ownerContext, flags bulkFlags, sourceRunID int64, matches []visibility.MatchedRepo) *runAudit {
	patternsRaw := fmt.Sprintf("undo:source-run=%d", sourceRunID)
	cmdName := constants.CmdVisibilityUndo

	return beginRunAudit(ctx, "mixed", cmdName, patternsRaw, flags, len(matches), matches)
}

// applyUndoLoop walks the persisted result rows, calling
// applyOneRepo with target = the row's original PrevVisibility.
// Mirrors applyBulkLoop's timing + audit contract.
func applyUndoLoop(ctx ownerContext, rs []model.MakeAllVisibilityResultRecord, verbose bool, audit *runAudit) (int, int, int) {
	changed, skipped, failed := 0, 0, 0
	total := len(rs)
	for i, r := range rs {
		fmt.Fprintf(os.Stdout, constants.MsgBulkApplyItemFmt, i+1, total, r.RepoName)
		start := time.Now()
		status := applyOneRepo(ctx, r.RepoName, r.PrevVisibility, verbose)
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
