// Package cmd — visibilityallbulk.go: top-level handler for the four
// bulk wildcard visibility commands (make-all-public / make-all-private
// / MAPUB / MAPRI) plus their except-latest counterparts. Owns dispatch,
// flag parsing (-Y / --verbose / --parallel / --cache-ttl / --except-latest),
// owner resolution, repo enumeration (TTL-cached), pattern matching,
// optional except-latest filtering, the optional interactive prompt
// (-Y skips it), the parallel per-repo apply loop, and exit-code
// aggregation.
//
// Heavy lifting is delegated:
//   - ResolveOwnerOnly           → visibilityresolveowner.go
//   - listOwnerReposCached       → visibilityownerlistcache.go
//   - visibility.ParsePatternList / MatchOwnerRepos → gitmap/visibility
//   - filterExceptLatest         → visibilityexceptlatest.go
//   - renderMatchedTable / promptConfirmOrExclude → visibilitybulkprompt.go
//   - applyBulkLoopParallel      → visibilityparallel.go
//
// Spec: spec/01-app/116-bulk-visibility-mapub-mapri.md §plan + §parallel.
package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/alimtvnetwork/gitmap-v26/gitmap/constants"
	"github.com/alimtvnetwork/gitmap-v26/gitmap/visibility"
)

// bulkFlags holds the parsed CLI flags for a bulk visibility run.
type bulkFlags struct {
	Yes           bool
	Verbose       bool
	ExceptLatest  bool
	Parallel      int
	CacheTTLSecs  int
	CacheTTLSet   bool
}

// runMakeAllPublic / runMakeAllPrivate are the dispatcher entry points.
func runMakeAllPublic(args []string) {
	runMakeAllVisibility(constants.VisibilityPublic, constants.CmdMakeAllPublic, args, false)
}

func runMakeAllPrivate(args []string) {
	runMakeAllVisibility(constants.VisibilityPrivate, constants.CmdMakeAllPrivate, args, false)
}

// Except-latest entry points pre-set the filter and reuse the rest
// of the pipeline so behavior stays in lock-step with the base
// commands.
func runMakeAllPublicExceptLatest(args []string) {
	runMakeAllVisibility(constants.VisibilityPublic, constants.CmdMakeAllPublicExceptLatest, args, true)
}

func runMakeAllPrivateExceptLatest(args []string) {
	runMakeAllVisibility(constants.VisibilityPrivate, constants.CmdMakeAllPrivateExceptLatest, args, true)
}

// runMakeAllVisibility orchestrates the full bulk run.
func runMakeAllVisibility(target, cmdName string, args []string, exceptLatestDefault bool) {
	checkHelp(cmdName, args)
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, constants.ErrMakeAllMissingArgFmt, cmdName)
		os.Exit(constants.ExitVisBadFlag)
	}

	ownerArg, patternsRaw, flags := parseBulkArgs(args)
	if exceptLatestDefault {
		flags.ExceptLatest = true
	}
	ctx := resolveOwnerOrExit(ownerArg)
	mustEnsureProviderCLI(ctx.Provider, flags.Verbose)
	mustEnsureProviderAuth(ctx.Provider, flags.Verbose)

	patterns, err := visibility.ParsePatternList(patternsRaw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "make-all-*: %v\n", err)
		os.Exit(constants.ExitVisBadFlag)
	}

	matches, ownerTotal := matchOrExitEmpty(ctx, patterns, flags)
	if flags.ExceptLatest {
		fmt.Fprint(os.Stdout, constants.MsgBulkExceptLatest)
		matches = filterExceptLatest(matches, os.Stdout)
		if len(matches) == 0 {
			fmt.Fprint(os.Stderr, constants.MsgBulkNoMatches)
			os.Exit(constants.ExitVisOK)
		}
	}
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
	changed, skipped, failed := applyBulkLoopParallel(ctx, target, final, flags, audit)
	fmt.Fprintf(os.Stdout, constants.MsgBulkSummaryFmt, changed, skipped, failed, len(final))
	exit := bulkExitCode(changed, failed)
	audit.finalize(excludedCount, changed, skipped, failed, exit)
	os.Exit(exit)
}

// bulkAuditFlags downcasts our extended bulkFlags into the legacy
// audit-layer struct (only Yes/Verbose are persisted).
func bulkAuditFlags(f bulkFlags) bulkAuditFlagShape {
	return bulkAuditFlagShape{Yes: f.Yes, Verbose: f.Verbose}
}

// bulkAuditFlagShape mirrors the original bulkFlags fields the audit
// layer reads. Kept distinct from bulkFlags so future audit fields
// don't bleed into the CLI parser.
type bulkAuditFlagShape = struct {
	Yes     bool
	Verbose bool
}

// parseBulkArgs splits owner / pattern-list / flags. Accepts the
// legacy -Y/-y/--yes/--verbose plus the new --parallel=N, --cache-ttl=N,
// and --except-latest/-XL flags anywhere after the first two positional
// args. Unknown flags are ignored (forwards-compatible with future spec
// additions).
func parseBulkArgs(args []string) (string, string, bulkFlags) {
	flags := bulkFlags{Parallel: constants.DefaultBulkParallelism}
	for _, a := range args[2:] {
		switch {
		case a == "-Y" || a == "-y" || a == "--yes":
			flags.Yes = true
		case a == "--verbose":
			flags.Verbose = true
		case a == constants.FlagBulkExceptLatest || a == constants.FlagBulkExceptLatestShort:
			flags.ExceptLatest = true
		case strings.HasPrefix(a, constants.FlagBulkParallel+"="):
			if n, err := strconv.Atoi(strings.TrimPrefix(a, constants.FlagBulkParallel+"=")); err == nil && n > 0 {
				if n > constants.MaxBulkParallelism {
					n = constants.MaxBulkParallelism
				}
				flags.Parallel = n
			}
		case strings.HasPrefix(a, constants.FlagBulkCacheTTL+"="):
			if n, err := strconv.Atoi(strings.TrimPrefix(a, constants.FlagBulkCacheTTL+"=")); err == nil && n >= 0 {
				flags.CacheTTLSecs = n
				flags.CacheTTLSet = true
			}
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

// matchOrExitEmpty lists owner repos (via TTL cache), matches patterns,
// exits 0 with a friendly message when nothing matched. Returns the
// matched subset AND the owner-wide total so the audit layer can
// persist OwnerRepoTotal.
func matchOrExitEmpty(ctx ownerContext, patterns []visibility.Pattern, flags bulkFlags) ([]visibility.MatchedRepo, int) {
	names, err := listOwnerReposCached(ctx.Provider, ctx.Owner, flags)
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
