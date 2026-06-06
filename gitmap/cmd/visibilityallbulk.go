// Package cmd — visibilityallbulk.go: top-level handler for the four
// bulk wildcard visibility commands (make-all-public / make-all-private
// / MAPUB / MAPRI). Owns dispatch, flag parsing, owner resolution,
// repo enumeration, pattern matching, the optional interactive prompt
// (-Y skips it), the per-repo apply loop, and exit-code aggregation.
//
// Heavy lifting is delegated:
//   - ResolveOwnerOnly        → visibilityresolveowner.go
//   - listOwnerRepos          → visibilityownerlist.go
//   - visibility.ParsePatternList / MatchOwnerRepos → gitmap/visibility
//   - renderMatchedTable / promptConfirmOrExclude   → visibilitybulkprompt.go
//   - mustEnsureProviderCLI / read+apply+verify     → visibilityapply.go
//
// Spec: spec/01-app/116-bulk-visibility-mapub-mapri.md §plan steps 13-14.
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/alimtvnetwork/gitmap-v25/gitmap/constants"
	"github.com/alimtvnetwork/gitmap-v25/gitmap/visibility"
)

// bulkFlags holds the parsed CLI flags for a bulk visibility run.
type bulkFlags struct {
	Yes     bool
	Verbose bool
}

// runMakeAllPublic / runMakeAllPrivate are the dispatcher entry points.
func runMakeAllPublic(args []string) {
	runMakeAllVisibility(constants.VisibilityPublic, constants.CmdMakeAllPublic, args)
}

func runMakeAllPrivate(args []string) {
	runMakeAllVisibility(constants.VisibilityPrivate, constants.CmdMakeAllPrivate, args)
}

// runMakeAllVisibility orchestrates the full bulk run. Exits with the
// most specific code per spec: ExitVisOK on full success, ExitVisAuthFailed
// when every repo failed, ExitVisBulkPartial on mixed outcomes.
func runMakeAllVisibility(target, cmdName string, args []string) {
	checkHelp(cmdName, args)
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, constants.ErrMakeAllMissingArgFmt, cmdName)
		os.Exit(constants.ExitVisBadFlag)
	}

	ownerArg, patternsRaw, flags := parseBulkArgs(args)
	ctx := resolveOwnerOrExit(ownerArg)
	mustEnsureProviderCLI(ctx.Provider, flags.Verbose)
	mustEnsureProviderAuth(ctx.Provider, flags.Verbose)

	patterns, err := visibility.ParsePatternList(patternsRaw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "make-all-*: %v\n", err)
		os.Exit(constants.ExitVisBadFlag)
	}

	matches, ownerTotal := matchOrExitEmpty(ctx, patterns, flags.Verbose)
	audit := beginRunAudit(ctx, target, cmdName, patternsRaw, flags, ownerTotal, matches)

	final := confirmOrAbort(matches, flags.Yes)
	if len(final) == 0 {
		excluded := audit.markExcluded(matches, nil)
		audit.finalize(excluded, 0, 0, 0, constants.ExitVisConfirmReq)
		fmt.Fprint(os.Stderr, constants.MsgBulkAborted)
		os.Exit(constants.ExitVisConfirmReq)
	}
	excludedCount := audit.markExcluded(matches, final)

	fmt.Fprintf(os.Stdout, constants.MsgBulkApplyHeaderFmt, target, len(final), ctx.Owner)
	changed, skipped, failed := applyBulkLoop(ctx, target, final, flags.Verbose, audit)
	fmt.Fprintf(os.Stdout, constants.MsgBulkSummaryFmt, changed, skipped, failed, len(final))
	exit := bulkExitCode(changed, failed)
	audit.finalize(excludedCount, changed, skipped, failed, exit)
	os.Exit(exit)
}

// parseBulkArgs splits owner / pattern-list / flags. Accepts -Y, -y,
// --yes, --verbose anywhere after the first two positional args.
func parseBulkArgs(args []string) (string, string, bulkFlags) {
	flags := bulkFlags{}
	for _, a := range args[2:] {
		switch a {
		case "-Y", "-y", "--yes":
			flags.Yes = true
		case "--verbose":
			flags.Verbose = true
		}
	}

	return args[0], args[1], flags
}

// resolveOwnerOrExit wraps ResolveOwnerOnly with Code Red exit handling.
func resolveOwnerOrExit(arg string) ownerContext {
	ctx, err := ResolveOwnerOnly(arg)
	if err != nil {
		fmt.Fprintf(os.Stderr, constants.ErrMakeAllResolveFmt, err)
		os.Exit(constants.ExitVisBadProvider)
	}

	return ctx
}

// matchOrExitEmpty lists owner repos, matches patterns, exits 0 with
// a friendly message when nothing matched. Returns the matched subset
// AND the owner-wide total so the audit layer can persist OwnerRepoTotal.
func matchOrExitEmpty(ctx ownerContext, patterns []visibility.Pattern, verbose bool) ([]visibility.MatchedRepo, int) {
	names, err := listOwnerRepos(ctx.Provider, ctx.Owner, verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "make-all-*: %v\n", err)
		os.Exit(constants.ExitVisAuthFailed)
	}

	matches := visibility.MatchOwnerRepos(names, patterns)
	fmt.Fprint(os.Stdout, renderMatchedTable(ctx.Owner, len(names), matches))
	if len(matches) == 0 {
		fmt.Fprint(os.Stderr, constants.MsgBulkNoMatches)
		os.Exit(constants.ExitVisOK)
	}

	return matches, len(names)
}

// confirmOrAbort runs the interactive prompt unless -Y was passed.
func confirmOrAbort(matches []visibility.MatchedRepo, yes bool) []visibility.MatchedRepo {
	if yes {
		return matches
	}
	final, proceed := promptConfirmOrExclude(os.Stdin, os.Stdout, matches)
	if !proceed {
		return nil
	}

	return final
}

// applyBulkLoop walks the matched set, applying target visibility +
// streaming results to the audit layer (timed per repo). Continues
// past per-repo failures and returns (changed, skipped, failed).
func applyBulkLoop(ctx ownerContext, target string, matches []visibility.MatchedRepo, verbose bool, audit *runAudit) (int, int, int) {
	changed, skipped, failed := 0, 0, 0
	total := len(matches)
	for i, m := range matches {
		fmt.Fprintf(os.Stdout, constants.MsgBulkApplyItemFmt, i+1, total, m.RepoName)
		start := time.Now()
		status := applyOneRepo(ctx, m.RepoName, target, verbose)
		audit.updateResult(m.RepoName, status, status.prev, status.next, start)
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

// bulkExitCode collapses the tallies into the spec's exit-code matrix.
func bulkExitCode(changed, failed int) int {
	if failed == 0 {
		return constants.ExitVisOK
	}
	if changed == 0 {
		return constants.ExitVisAuthFailed
	}

	return constants.ExitVisBulkPartial
}
