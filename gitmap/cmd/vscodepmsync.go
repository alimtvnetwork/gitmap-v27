// Package cmd — vscodepmsync.go: implements the top-level
// `gitmap vscode-pm-sync` (alias `vpm`) command.
//
// What it does:
//
//	1. Resolves the alefragnani.project-manager projects.json path
//	   via vscodepm.ProjectsJSONPath. Soft-fails (exit 0) when the
//	   user-data root or extension dir is missing — same policy as
//	   the post-clone sync helper, so CI / headless boxes never
//	   break on this command.
//	2. Reads every entry currently in projects.json.
//	3. For each entry whose rootPath still exists on disk, builds a
//	   vscodepm.Pair carrying (rootPath, name, vscodepm.DetectTagsCustom).
//	   Entries pointing at deleted folders are skipped — their tags
//	   are left untouched on disk.
//	4. Calls vscodepm.Sync once with the full pair set. Sync's
//	   mergePairs UNIONs detected tags with whatever is already on
//	   disk, so user-added tags are preserved.
//
// Spec: spec/01-vscode-project-manager-sync/04-tag-resync.md
// Memory: see the "VS Code PM Sync" entry referenced from
// mem://features (added in v4.36.0).
package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/alimtvnetwork/gitmap-v18/gitmap/constants"
	"github.com/alimtvnetwork/gitmap-v18/gitmap/vscodepm"
)

// runVSCodePMSync is the entry point wired into the dispatcher.
func runVSCodePMSync(args []string) {
	checkHelp(constants.CmdVSCodePMSync, args)

	dryRun, mode, err := parseVSCodePMSyncFlags(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	path, entries, ok := loadVSCodePMEntries()
	if !ok {
		return
	}

	if len(entries) == 0 {
		fmt.Print(constants.MsgVSCodePMSyncEmptyFile)
		return
	}

	fmt.Printf(constants.MsgVSCodePMSyncStart, path)

	pairs, skipped := buildVSCodePMSyncPairs(entries)

	if dryRun {
		emitVSCodePMSyncDryRunReport(entries, pairs, skipped)
		return
	}

	commitVSCodePMSync(pairs, skipped, mode)
}

// parseVSCodePMSyncFlags parses --dry-run and --mode. Returns the
// dry-run bool, the resolved MergeMode, and a validation error from
// ParseMergeMode (unknown --mode values fail loud per the
// zero-swallow rule rather than silently defaulting to union).
func parseVSCodePMSyncFlags(args []string) (bool, vscodepm.MergeMode, error) {
	fs := flag.NewFlagSet(constants.CmdVSCodePMSync, flag.ExitOnError)

	dryRun := fs.Bool(
		constants.FlagVSCodePMSyncDryRun, false,
		constants.FlagDescVSCodePMSyncDryRun,
	)
	modeRaw := fs.String(
		constants.FlagVSCodePMSyncMode, constants.VSCodePMSyncModeUnion,
		constants.FlagDescVSCodePMSyncMode,
	)

	_ = fs.Parse(args)

	mode, err := vscodepm.ParseMergeMode(*modeRaw)
	if err != nil {
		return false, vscodepm.MergeModeUnion, err
	}

	return *dryRun, mode, nil
}

// loadVSCodePMEntries reads projects.json and returns the parsed
// entries plus the resolved file path. Soft-skips (returns ok=false)
// when the user-data root or extension dir is missing — the command
// must NEVER fail-loud on a headless / no-VS-Code box.
func loadVSCodePMEntries() (string, []vscodepm.Entry, bool) {
	path, pathErr := vscodepm.ProjectsJSONPath()
	entries, listErr := vscodepm.ListEntries()

	if pathErr != nil || listErr != nil {
		reportVSCodePMSoftError(firstNonNil(pathErr, listErr))
		return path, nil, false
	}

	return path, entries, true
}

// firstNonNil returns the first non-nil error, or nil.
func firstNonNil(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}

	return nil
}

// buildVSCodePMSyncPairs converts every on-disk entry into a Pair
// carrying freshly-detected tags. Entries whose rootPath is missing
// on disk are skipped (count returned as the second value) so the
// re-sync never inadvertently strips tags from intentionally-offline
// removable-drive projects — Sync only touches entries it sees.
func buildVSCodePMSyncPairs(entries []vscodepm.Entry) ([]vscodepm.Pair, int) {
	pairs := make([]vscodepm.Pair, 0, len(entries))
	skipped := 0

	for _, e := range entries {
		if !rootPathExists(e.RootPath) {
			skipped++
			continue
		}

		pairs = append(pairs, vscodepm.Pair{
			RootPath: e.RootPath,
			Name:     e.Name,
			Paths:    e.Paths,
			Tags:     vscodepm.DetectTagsCustom(e.RootPath),
		})
	}

	return pairs, skipped
}

// rootPathExists reports whether the entry's rootPath is a directory
// that currently exists on disk.
func rootPathExists(rootPath string) bool {
	if rootPath == "" {
		return false
	}

	info, err := os.Stat(rootPath)
	if err != nil {
		return false
	}

	return info.IsDir()
}

// emitVSCodePMSyncDryRunReport prints what would change without
// touching the file. We approximate "would change" as len(pairs)
// since Sync will at minimum re-evaluate tags for each one — exact
// per-entry diffing is intentionally deferred to the real run so
// the dry-run cost stays predictable on huge projects.json files.
func emitVSCodePMSyncDryRunReport(entries []vscodepm.Entry, pairs []vscodepm.Pair, skipped int) {
	_ = entries

	fmt.Printf(constants.MsgVSCodePMSyncDryRun, len(pairs)+skipped, len(pairs))
}

// commitVSCodePMSync runs vscodepm.SyncMode and prints the standard
// summary line, then a vscode-pm-sync-specific tally that includes
// the count of skipped (missing-on-disk) entries. The MergeMode is
// threaded through from the CLI flag so the merge engine knows
// whether to union / replace / intersect tags.
func commitVSCodePMSync(pairs []vscodepm.Pair, skipped int, mode vscodepm.MergeMode) {
	summary, err := vscodepm.SyncMode(pairs, mode)
	if err != nil {
		reportVSCodePMSoftError(err)
		return
	}

	fmt.Printf(constants.MsgVSCodePMSyncSummary,
		summary.Added, summary.Updated, summary.Unchanged, summary.Total)
	fmt.Printf(constants.MsgVSCodePMSyncEntryStat, len(pairs), skipped)
}
