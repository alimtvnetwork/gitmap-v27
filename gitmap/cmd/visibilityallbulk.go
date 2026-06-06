// Package cmd — visibilityallbulk.go: top-level handler for the four
// bulk wildcard visibility commands. This file owns ONLY the dispatch
// stub + arg validation + owner resolution; pattern matching, repo
// listing, interactive prompt, and DB persistence land in sibling files
// across the rest of plan step 7-22.
//
// Stub status: steps 5-6. The handler resolves the owner and prints a
// "not yet wired" message; later steps replace the body. Keeping a
// compiling stub now lets the dispatcher (rootcore.go) reference the
// symbols without a TODO comment.
//
// Spec: spec/01-app/116-bulk-visibility-mapub-mapri.md.
package cmd

import (
	"fmt"
	"os"

	"github.com/alimtvnetwork/gitmap-v25/gitmap/constants"
)

// runMakeAllPublic / runMakeAllPrivate are the two dispatcher entry
// points. They differ only in the target visibility; everything else
// flows through runMakeAllVisibility.
func runMakeAllPublic(args []string) {
	runMakeAllVisibility(constants.VisibilityPublic, constants.CmdMakeAllPublic, args)
}

func runMakeAllPrivate(args []string) {
	runMakeAllVisibility(constants.VisibilityPrivate, constants.CmdMakeAllPrivate, args)
}

// runMakeAllVisibility is the shared orchestrator. Steps 7-22 will
// replace the body; for now it validates args, resolves the owner via
// the new ResolveOwnerOnly resolver, and reports "not yet wired" with
// a non-zero exit so callers see a loud signal instead of a silent
// no-op (zero-swallow rule).
func runMakeAllVisibility(target string, cmdName string, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, constants.ErrMakeAllMissingArgFmt, cmdName)
		os.Exit(constants.ExitVisBadFlag)
	}

	ctx, err := ResolveOwnerOnly(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, constants.ErrMakeAllResolveFmt, err)
		os.Exit(constants.ExitVisBadProvider)
	}

	fmt.Fprintf(os.Stderr,
		"make-all-*: resolved provider=%s owner=%s target=%s patterns=%q\n",
		ctx.Provider, ctx.Owner, target, args[1])
	fmt.Fprint(os.Stderr, constants.MsgMakeAllNotImpl)
	os.Exit(constants.ExitVisBadFlag)
}
