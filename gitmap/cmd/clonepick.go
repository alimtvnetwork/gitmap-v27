package cmd

// CLI entry point for `gitmap clone-pick <repo-url> <paths>` (spec
// 100, v3.153.0+). Sparse-checkout a subset of a git repo into the
// current working directory (or --dest), and auto-save the selection
// to the CloneInteractiveSelection table.
//
// Exit codes:
//
//   0   -- dry-run rendered OR clone succeeded
//   1   -- runtime failure (git, fs, db)
//   2   -- bad CLI usage (missing args, invalid flag value)
//   130 -- user canceled the picker (reserved for --ask v2)
//
// The picker (--ask) and --replay are scaffolded as stubs in v1: the
// flag is accepted and the value flows to the Plan, but the picker
// UI lands in a follow-up patch (tracked in .lovable/plan.md).

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/alimtvnetwork/gitmap-v16/gitmap/cliexit"
	"github.com/alimtvnetwork/gitmap-v16/gitmap/clonepick"
	"github.com/alimtvnetwork/gitmap-v16/gitmap/constants"
	"github.com/alimtvnetwork/gitmap-v16/gitmap/store"
)

// runClonePick is the dispatcher entry registered in rootcore.go.
func runClonePick(args []string) {
	checkHelp("clone-pick", args)

	parsed := parseClonePickFlags(args)
	setCmdFaithfulVerify(parsed.VerifyCmdFaithful)
	setCmdFaithfulExitOnMismatch(parsed.VerifyCmdFaithfulExitOnMismatch)
	setCmdPrintArgv(parsed.PrintCloneArgv)

	plan, replayId, err := buildClonePickPlan(parsed)
	if err != nil {
		cliexit.Fail(constants.CmdClonePick, "parse-args", parsed.RawURL, err, 2)
	}

	if plan.DryRun {
		// `--output terminal`: emit the standardized block instead
		// of the legacy clonepick.Render output. Keeps the per-repo
		// summary shape consistent across every clone command.
		if parsed.Output == constants.OutputTerminal {
			printClonePickTermBlock(plan)
			maybeExitOnCmdFaithfulMismatch()

			return
		}
		if err := clonepick.Render(os.Stdout, plan); err != nil {
			cliexit.Fail(constants.CmdClonePick, "render-dry-run", parsed.RawURL, err, 1)
		}
		maybeExitOnCmdFaithfulMismatch()

		return
	}

	if parsed.Output == constants.OutputTerminal {
		printClonePickTermBlock(plan)
	}
	runClonePickExecute(plan, parsed.NoVSCodeSync, replayId)
}

// buildClonePickPlan picks between the parse path (positional args)
// and the replay path (--replay <id|name> hits the DB). Returns the
// Plan plus the replayed SelectionId (0 for fresh runs) so the
// executor can bump CreatedAt without re-deriving the id.
func buildClonePickPlan(parsed clonePickParsed) (clonepick.Plan, int64, error) {
	if len(parsed.Flags.Replay) == 0 {
		plan, err := clonepick.ParseArgs(parsed.RawURL, parsed.RawPaths, parsed.Flags)

		return plan, 0, err
	}
	loader, err := openDB()
	if err != nil {
		return clonepick.Plan{}, 0, err
	}
	plan, replayId, loadErr := clonepick.LoadFromDB(loader, parsed.Flags.Replay)
	if loadErr != nil {
		return clonepick.Plan{}, 0, loadErr
	}
	// Runtime-only flags from THIS invocation override the
	// persisted Plan -- spec rule: replay reproduces the
	// selection, not the verbosity choice.
	plan.DryRun = parsed.Flags.DryRun
	plan.Quiet = parsed.Flags.Quiet
	plan.Force = parsed.Flags.Force
	if len(parsed.Flags.Dest) > 0 && parsed.Flags.Dest != "." {
		plan.DestDir = parsed.Flags.Dest
	}

	return plan, replayId, nil
}

// clonePickParsed bundles every output of parseClonePickFlags so a
// new audit/debug toggle can be added without churning the call
// site signature each time. Fields are exported because the struct
// itself stays unexported (cmd-package-internal).
type clonePickParsed struct {
	RawURL                          string
	RawPaths                        string
	Flags                           clonepick.Flags
	Output                          string
	VerifyCmdFaithful               bool
	VerifyCmdFaithfulExitOnMismatch bool
	PrintCloneArgv                  bool
	// NoVSCodeSync suppresses the post-clone update of the
	// alefragnani.project-manager projects.json file. Mirrors
	// `gitmap scan --no-vscode-sync`. Default false. See
	// spec/01-vscode-project-manager-sync/02-clone-sync.md.
	NoVSCodeSync bool
}

// parseClonePickFlags binds every clone-pick flag and extracts the
// two positional args. Validation that needs cross-flag knowledge
// happens in clonepick.ParseArgs so this stays focused on flag
// binding.
func parseClonePickFlags(args []string) clonePickParsed {
	defaults := clonepick.DefaultFlags()
	flags := defaults
	fs := flag.NewFlagSet("clone-pick", flag.ExitOnError)
	fs.BoolVar(&flags.Ask, constants.FlagClonePickAsk, defaults.Ask,
		constants.FlagDescClonePickAsk)
	fs.StringVar(&flags.Name, constants.FlagClonePickName, defaults.Name,
		constants.FlagDescClonePickName)
	fs.StringVar(&flags.Mode, constants.FlagClonePickMode, defaults.Mode,
		constants.FlagDescClonePickMode)
	fs.StringVar(&flags.Branch, constants.FlagClonePickBranch, defaults.Branch,
		constants.FlagDescClonePickBranch)
	fs.IntVar(&flags.Depth, constants.FlagClonePickDepth, defaults.Depth,
		constants.FlagDescClonePickDepth)
	fs.BoolVar(&flags.Cone, constants.FlagClonePickCone, defaults.Cone,
		constants.FlagDescClonePickCone)
	fs.StringVar(&flags.Dest, constants.FlagClonePickDest, defaults.Dest,
		constants.FlagDescClonePickDest)
	fs.BoolVar(&flags.KeepGit, constants.FlagClonePickKeepGit, defaults.KeepGit,
		constants.FlagDescClonePickKeepGit)
	fs.BoolVar(&flags.DryRun, constants.FlagClonePickDryRun, defaults.DryRun,
		constants.FlagDescClonePickDryRun)
	fs.BoolVar(&flags.Quiet, constants.FlagClonePickQuiet, defaults.Quiet,
		constants.FlagDescClonePickQuiet)
	fs.BoolVar(&flags.Force, constants.FlagClonePickForce, defaults.Force,
		constants.FlagDescClonePickForce)
	fs.StringVar(&flags.Replay, constants.FlagClonePickReplay, defaults.Replay,
		constants.FlagDescClonePickReplay)
	output := fs.String(constants.FlagCloneTermOutput, "",
		constants.FlagDescCloneTermOutput)
	verify := fs.Bool(constants.FlagCloneVerifyCmdFaithful, false,
		constants.FlagDescCloneVerifyCmdFaithful)
	verifyExit := fs.Bool(constants.FlagCloneVerifyCmdFaithfulExitOnMismatch,
		false, constants.FlagDescCloneVerifyCmdFaithfulExitOnMismatch)
	printArgv := fs.Bool(constants.FlagClonePrintArgv, false,
		constants.FlagDescClonePrintArgv)
	noVSCodeSync := fs.Bool(constants.FlagNoVSCodeSync, false,
		constants.FlagDescNoVSCodeSync)

	reordered := reorderFlagsBeforeArgs(args)
	fs.Parse(reordered)

	if fs.NArg() < 1 && len(flags.Replay) == 0 {
		fmt.Fprintln(os.Stderr, constants.MsgClonePickMissingURL)
		os.Exit(2)
	}
	rawURL := ""
	if fs.NArg() >= 1 {
		rawURL = fs.Arg(0)
	}
	rawPaths := ""
	if fs.NArg() >= 2 {
		rawPaths = fs.Arg(1)
	}

	return clonePickParsed{
		RawURL:                          rawURL,
		RawPaths:                        rawPaths,
		Flags:                           flags,
		Output:                          *output,
		VerifyCmdFaithful:               *verify,
		VerifyCmdFaithfulExitOnMismatch: *verifyExit,
		PrintCloneArgv:                  *printArgv,
		NoVSCodeSync:                    *noVSCodeSync,
	}
}

// runClonePickExecute opens the DB (best-effort), runs the
// sparse-checkout, and translates the Result to an exit code.
// replayId is non-zero when the Plan came from --replay; on success
// CreatedAt is bumped so most-recently-replayed sorts to the top.
func runClonePickExecute(plan clonepick.Plan, noVSCodeSync bool, replayId int64) {
	progress := io.Writer(os.Stderr)
	if plan.Quiet {
		progress = io.Discard
	}

	db, dbErr := openDB()
	if dbErr != nil {
		// DB open failure is non-fatal -- clone still proceeds, just
		// without persistence. Per the zero-swallow policy we surface
		// the error to stderr so it isn't silently dropped.
		fmt.Fprintln(os.Stderr, dbErr)
	}

	result := clonepick.Execute(plan, db, progress)
	announceClonePickPersistence(plan, result, replayId, db)

	if result.Status == clonepick.StatusFailed {
		maybeExitOnCmdFaithfulMismatch()
		os.Exit(1)
	}

	// VS Code Project Manager sync. Result.Detail carries the
	// resolved destination path on success; fall back to the plan's
	// DestDir when empty.
	syncClonePickResultToVSCodePM(plan, result, noVSCodeSync)

	if plan.DestDir != "." && plan.DestDir != "" {
		WriteShellHandoff(result.Detail)
	}
	maybeExitOnCmdFaithfulMismatch()
}

// announceClonePickPersistence prints the saved/replayed line and,
// for replays, bumps the CreatedAt column. Split out so the main
// executor stays under the function-length cap.
func announceClonePickPersistence(plan clonepick.Plan, result clonepick.Result, replayId int64, db *store.DB) {
	name := plan.Name
	if len(name) == 0 {
		name = "(unnamed)"
	}
	switch {
	case replayId > 0 && result.Status == clonepick.StatusOK:
		fmt.Fprintf(os.Stderr, constants.MsgClonePickReplayed,
			replayId, plan.RepoCanonicalId, name)
		if !plan.DryRun && db != nil {
			if err := clonepick.TouchAfterReplay(db, replayId, plan.DryRun); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}
	case result.SelectionId > 0:
		fmt.Fprintf(os.Stderr, constants.MsgClonePickSaved,
			result.SelectionId, plan.RepoCanonicalId, name)
	}
}

// syncClonePickResultToVSCodePM registers a successful sparse-checkout
// dest in projects.json. The pick name (when set) wins over the folder
// basename so users see their alias in the Project Manager sidebar.
func syncClonePickResultToVSCodePM(plan clonepick.Plan, result clonepick.Result, skip bool) {
	dest := result.Detail
	if dest == "" {
		dest = plan.DestDir
	}
	abs, err := filepath.Abs(dest)
	if err != nil {
		abs = dest
	}
	name := plan.Name
	if name == "" {
		name = filepath.Base(abs)
	}
	syncSingleClonedRepoToVSCodePM(abs, name, skip)
}
