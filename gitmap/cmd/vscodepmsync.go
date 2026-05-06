// Package cmd — vscodepmsync.go: implements the top-level
// `gitmap vscode-pm-sync` (alias `vpm`) command.
//
// What it does:
//
//	1. Resolves the alefragnani.project-manager projects.json path
//	   via vscodepm.ProjectsJSONPath — OR uses --projects-json when
//	   the user supplied an absolute override. Soft-fails (exit 0)
//	   when discovery returns a missing user-data root / extension
//	   dir, so CI / headless boxes never break on this command.
//	2. Reads every entry currently in projects.json.
//	3. For each entry whose rootPath still exists on disk, builds a
//	   vscodepm.Pair carrying (rootPath, name, tags). Tags come from
//	   vscodepm.DetectTagsCustom by default; --tag <name> (repeat /
//	   comma-list) replaces the per-pair detected set with the
//	   user-supplied list verbatim — the brand tag is NOT
//	   auto-prepended in that mode.
//	4. Calls vscodepm.SyncMode (or SyncAtMode when --projects-json
//	   is set) once with the full pair set. The MergeMode picked by
//	   --mode (union | replace | intersection) governs how the
//	   detected/override tag set is reconciled with whatever is
//	   already on disk.
//
// Spec: spec/01-vscode-project-manager-sync/04-tag-resync.md
// Memory: see the "VS Code PM Sync" entry referenced from
// mem://features (added in v4.36.0; --projects-json + --tag wired
// in v4.37.0 alongside --mode).
package cmd

import (
	"fmt"
	"os"

	"github.com/alimtvnetwork/gitmap-v18/gitmap/constants"
	"github.com/alimtvnetwork/gitmap-v18/gitmap/vscodepm"
)

// runVSCodePMSync is the entry point wired into the dispatcher.
func runVSCodePMSync(args []string) {
	checkHelp(constants.CmdVSCodePMSync, args)

	opts, err := parseVSCodePMSyncFlags(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	path, entries, ok := loadVSCodePMEntries(opts)
	if !ok {
		return
	}

	if len(entries) == 0 {
		fmt.Print(constants.MsgVSCodePMSyncEmptyFile)
		return
	}

	fmt.Printf(constants.MsgVSCodePMSyncStart, path)

	pairs, skipped := buildVSCodePMSyncPairs(entries, opts)

	if opts.DryRun {
		emitVSCodePMSyncDryRunReport(entries, pairs, skipped)
		return
	}

	commitVSCodePMSync(pairs, skipped, opts)
}

// loadVSCodePMEntries reads projects.json and returns the parsed
// entries plus the resolved file path. When opts.ProjectsJSON is set
// the resolver is bypassed and the override path is used verbatim
// (vscodepm.ListEntriesAt treats a missing file as an empty list, so
// the override path can also be used to bootstrap a fresh file).
// Otherwise the same soft-skip semantics as v4.36.0 apply: a missing
// user-data root or extension dir returns ok=false so headless boxes
// never fail-loud.
func loadVSCodePMEntries(opts vscodePMSyncOpts) (string, []vscodepm.Entry, bool) {
	if opts.ProjectsJSON != "" {
		entries, err := vscodepm.ListEntriesAt(opts.ProjectsJSON)
		if err != nil {
			reportVSCodePMSoftError(err)
			return opts.ProjectsJSON, nil, false
		}

		return opts.ProjectsJSON, entries, true
	}

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
// carrying tags. Tag source:
//
//   - opts.HasTagOverride => opts.TagOverride verbatim (the
//     `gitmap` brand tag is NOT auto-prepended; the user owns the
//     full set).
//   - otherwise           => vscodepm.DetectTagsCustom(rootPath).
//
// Entries whose rootPath is missing on disk are skipped (count
// returned as the second value) so the re-sync never inadvertently
// strips tags from intentionally-offline removable-drive projects.
func buildVSCodePMSyncPairs(entries []vscodepm.Entry, opts vscodePMSyncOpts) ([]vscodepm.Pair, int) {
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
			Tags:     resolveVSCodePMSyncTags(e.RootPath, opts),
		})
	}

	return pairs, skipped
}

// resolveVSCodePMSyncTags returns the override list when --tag was
// passed, else falls back to the detector. Centralised so the
// "brand-tag NOT auto-prepended in override mode" contract is in
// one place.
func resolveVSCodePMSyncTags(rootPath string, opts vscodePMSyncOpts) []string {
	if opts.HasTagOverride {
		return append([]string(nil), opts.TagOverride...)
	}

	return vscodepm.DetectTagsCustom(rootPath)
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

// commitVSCodePMSync runs the appropriate sync entry point and
// prints the standard summary line plus a vscode-pm-sync-specific
// tally that includes the count of skipped (missing-on-disk)
// entries. SyncAtMode is used when --projects-json bypassed the
// resolver; otherwise the default-discovery SyncMode runs.
func commitVSCodePMSync(pairs []vscodepm.Pair, skipped int, opts vscodePMSyncOpts) {
	summary, err := runVSCodePMSyncWriter(pairs, opts)
	if err != nil {
		reportVSCodePMSoftError(err)
		return
	}

	fmt.Printf(constants.MsgVSCodePMSyncSummary,
		summary.Added, summary.Updated, summary.Unchanged, summary.Total)
	fmt.Printf(constants.MsgVSCodePMSyncEntryStat, len(pairs), skipped)
}

// runVSCodePMSyncWriter dispatches to SyncAtMode (override path) or
// SyncMode (default discovery). Split out so commitVSCodePMSync
// stays under the 15-line function budget.
func runVSCodePMSyncWriter(pairs []vscodepm.Pair, opts vscodePMSyncOpts) (vscodepm.SyncSummary, error) {
	if opts.ProjectsJSON != "" {
		return vscodepm.SyncAtMode(opts.ProjectsJSON, pairs, opts.Mode)
	}

	return vscodepm.SyncMode(pairs, opts.Mode)
}
