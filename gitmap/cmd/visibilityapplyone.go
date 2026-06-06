// Package cmd — visibilityapplyone.go: non-exiting per-repo apply
// helper for the bulk wildcard visibility commands. Mirrors the
// read→skip-if-same→apply→verify pipeline of the single-repo path
// in visibilityapply.go, but returns a structured status instead of
// calling os.Exit so the outer loop can continue past per-repo
// failures and tally a summary.
//
// Spec: spec/01-app/116-bulk-visibility-mapub-mapri.md §plan step 14.
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/alimtvnetwork/gitmap-v25/gitmap/constants"
)

// applyOutcome is one of: "ok" | "skip" | "fail". prev/next carry the
// visibility tokens observed pre/post-apply so the audit layer can
// persist them on the result row.
type applyStatus struct {
	outcome string
	err     error
	prev    string
	next    string
}

// applyOneRepo runs read → (skip|apply → verify) for a single repo.
// Prints the per-line result token to stdout so the caller's
// MsgBulkApplyItemFmt prefix flows into a one-line-per-repo log.
func applyOneRepo(owner ownerContext, repoName, target string, verbose bool) applyStatus {
	slug := owner.Owner + "/" + repoName
	repoCtx := visibilityContext{Provider: owner.Provider, Slug: slug}

	current, readErr := readVisibilityNoExit(repoCtx, verbose)
	if readErr != nil {
		fmt.Fprintf(os.Stdout, constants.MsgBulkApplyFailFmt, readErr)

		return applyStatus{outcome: "fail", err: readErr}
	}

	if current == target {
		fmt.Fprintf(os.Stdout, constants.MsgBulkApplySkipFmt, current)

		return applyStatus{outcome: "skip", prev: current, next: current}
	}

	if applyErr := applyVisibilityNoExit(repoCtx, target, verbose); applyErr != nil {
		fmt.Fprintf(os.Stdout, constants.MsgBulkApplyFailFmt, applyErr)

		return applyStatus{outcome: "fail", err: applyErr, prev: current}
	}

	verified, verifyErr := readVisibilityNoExit(repoCtx, verbose)
	if verifyErr != nil || verified != target {
		err := fmt.Errorf("verify failed: got=%q want=%q (%v)", verified, target, verifyErr)
		fmt.Fprintf(os.Stdout, constants.MsgBulkApplyFailFmt, err)

		return applyStatus{outcome: "fail", err: err, prev: current, next: verified}
	}

	fmt.Fprintf(os.Stdout, constants.MsgBulkApplyOKFmt, current, verified)

	return applyStatus{outcome: "ok", prev: current, next: verified}
}

// readVisibilityNoExit mirrors mustReadCurrentVisibility but returns
// an error instead of exiting. Wraps the provider error with Code Red
// context so the summary line is actionable.
func readVisibilityNoExit(ctx visibilityContext, verbose bool) (string, error) {
	args := readVisibilityArgs(ctx.Provider, ctx.Slug)
	out, err := runProviderCLI(ctx.Provider, args, verbose)
	if err != nil {
		return "", fmt.Errorf("Error: read visibility failed for %s: %v (operation: %s repo view, reason: %s)",
			ctx.Slug, err, providerCLI(ctx.Provider), err.Error())
	}

	return parseVisibilityOutput(ctx.Provider, out), nil
}

// applyVisibilityNoExit mirrors applyVisibilityOrExit but returns
// an error including captured stderr instead of exiting.
func applyVisibilityNoExit(ctx visibilityContext, target string, verbose bool) error {
	args := applyVisibilityArgs(ctx.Provider, ctx.Slug, target)
	stderr, err := runProviderCLICapturingStderr(ctx.Provider, args, verbose)
	if err == nil {
		return nil
	}

	return fmt.Errorf("Error: apply visibility failed for %s: %v (operation: %s repo edit, reason: %s)",
		ctx.Slug, err, providerCLI(ctx.Provider), strings.TrimSpace(stderr))
}
