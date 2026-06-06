// Package cmd — visibilityhistory.go: `gitmap visibility-history`
// (`vh`) prints the most recent make-all-* / VisibilityUndo /
// VisibilityRedo runs newest-first so users can select a `--run <id>`
// for `vu` / `vr`.
//
// Spec: spec/01-app/116-bulk-visibility-mapub-mapri.md §history.
package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/alimtvnetwork/gitmap-v25/gitmap/constants"
	"github.com/alimtvnetwork/gitmap-v25/gitmap/model"
)

// runVisibilityHistory is the dispatcher entry point.
func runVisibilityHistory(args []string) {
	limit := parseHistoryLimit(args)
	db := openDBOrExit(constants.CmdVisibilityHistory)
	runs, err := db.SelectRecentMakeAllVisibilityRuns(limit)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(constants.ExitVisAuthFailed)
	}
	if len(runs) == 0 {
		fmt.Fprint(os.Stderr, constants.MsgHistoryEmpty)
		os.Exit(constants.ExitVisOK)
	}
	printHistory(runs)
	os.Exit(constants.ExitVisOK)
}

// parseHistoryLimit accepts `--limit N` (default HistoryDefaultLimit).
// Bad values exit ExitVisBadFlag (zero-swallow).
func parseHistoryLimit(args []string) int {
	for i := 0; i+1 < len(args); i++ {
		if args[i] != "--limit" {
			continue
		}
		n, err := strconv.Atoi(args[i+1])
		if err != nil || n <= 0 {
			fmt.Fprintf(os.Stderr, constants.ErrUndoBadRunFlagFmt, args[i+1], err, "--limit must be positive integer")
			os.Exit(constants.ExitVisBadFlag)
		}

		return n
	}

	return constants.HistoryDefaultLimit
}

// printHistory writes the table to stdout.
func printHistory(runs []model.MakeAllVisibilityRunRecord) {
	fmt.Fprint(os.Stdout, constants.MsgHistoryHeader)
	for _, r := range runs {
		owner := truncate(r.Provider+"/"+r.Owner, 21)
		fmt.Fprintf(os.Stdout, constants.MsgHistoryRowFmt,
			r.ID, truncate(r.CommandKind, 16), owner,
			r.MatchedCount, r.OkCount, r.SkippedCount, r.FailedCount,
			r.ExcludedCount, r.ExitCode, r.StartedAt)
	}
}

// truncate clips overflowing cells to keep the table aligned.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}

	return s[:n-1] + "…"
}
