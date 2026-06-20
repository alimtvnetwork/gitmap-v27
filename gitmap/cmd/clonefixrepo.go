// Package cmd — clonefixrepo.go: entry points for `gitmap clone-fix-repo`
// (alias `cfr`) and `gitmap clone-fix-repo-pub` (alias `cfrp`).
//
// These are convenience pipelines that chain three existing commands
// in one shot:
//
//	cfr  : clone <url>  →  cd <folder>  →  fix-repo --all
//	cfrp : clone <url>  →  cd <folder>  →  fix-repo --all  →  make-public --yes
//
// Implementation strategy: the chained commands (runFixRepo,
// runMakePublic) all call os.Exit at the end, which would terminate
// our parent process before the next step runs. To stay decoupled
// and side-effect-clean, we shell out to our own binary (resolved
// via os.Executable) for the fix-repo and make-public steps after
// invoking executeDirectClone in-process. This also keeps each
// step's exit code, stdout, and stderr semantics intact.
package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alimtvnetwork/gitmap-v26/gitmap/clonenext"
	"github.com/alimtvnetwork/gitmap-v26/gitmap/constants"
	"github.com/alimtvnetwork/gitmap-v26/gitmap/gitutil"
)

// runCloneFixRepo implements `gitmap clone-fix-repo` (alias cfr).
func runCloneFixRepo(args []string) {
	checkHelp(constants.CmdCloneFixRepo, args)
	runCloneFixRepoPipeline(args, false)
}

// runCloneFixRepoPub implements `gitmap clone-fix-repo-pub` (alias cfrp).
func runCloneFixRepoPub(args []string) {
	checkHelp(constants.CmdCloneFixRepoPub, args)
	runCloneFixRepoPipeline(args, true)
}

// runCloneFixRepoPipeline is the shared core. `makePublic` controls
// whether the optional 3rd step (visibility flip) runs.
func runCloneFixRepoPipeline(args []string, makePublic bool) {
	url, folderName, noVSCodeSync, requireVersion, useSSH, useHTTPS, autoYes, dryRun := parseCloneFixRepoArgs(args)
	SetCloneDryRun(dryRun)
	if len(url) == 0 {
		fmt.Fprint(os.Stderr, constants.ErrCloneFixRepoUsage)
		os.Exit(constants.ExitCloneFixRepoBadFlag)
	}

	url = applyCloneFixRepoScheme(url, useSSH, useHTTPS)
	escapeNestedGitRepo()

	// cfr/cfrp DO flatten `-vN` suffixes: the local folder mirrors
	// the repo base name (e.g. macro-ahk-v50 → macro-ahk) so that
	// fix-repo can rewrite version tokens across siblings. If the
	// user passes an explicit folder argument, that wins verbatim.
	if len(folderName) == 0 {
		repoName := repoNameFromURL(url)
		parsed := clonenext.ParseRepoName(repoName)
		if parsed.HasVersion {
			folderName = parsed.BaseName
		} else {
			folderName = repoName
		}
	}
	absPath := resolveCloneTargetFolder(url, folderName)
	url = preferExistingFolderTransport(url, absPath)
	requireOnline()
	executeDirectClone(url, folderName, true, false, "", noVSCodeSync)

	// Dry-run short circuit: nothing was cloned, so the chained
	// chdir + fix-repo + make-public steps have no target to act on.
	if dryRun {
		fmt.Printf("  "+constants.MsgCloneDryRunNoop+"\n  would chain: fix-repo --all%s @ %s\n",
			pubSuffix(makePublic), absPath)
		return
	}



	if err := os.Chdir(absPath); err != nil {
		fmt.Fprintf(os.Stderr, constants.ErrCloneFixRepoChdirFmt, absPath, err)
		os.Exit(constants.ExitCloneFixRepoChdir)
	}

	maybeRunFixRepoStep(absPath, requireVersion)
	if makePublic {
		runChainedGitmapStep([]string{constants.CmdMakePublic, "--" + constants.FlagVisYes})
		// Spec 113 §2.3 — offer to privatize prior public versions.
		runCFRPPriorVersionPrivatize(absPath, autoYes)
	}
	fmt.Printf(constants.MsgCloneFixRepoDone, absPath)
}

// applyCloneFixRepoScheme honours --ssh / --https (and short aliases
// --sh / --ht) by rewriting the URL before the in-process clone runs.
// Mirrors `gitmap clone --ssh` semantics: when both flags are set,
// --ssh wins and a one-line stderr warning is printed. Unrecognised
// URL shapes are returned unchanged so non-URL positionals still flow
// through.
func applyCloneFixRepoScheme(url string, useSSH, useHTTPS bool) string {
	if useSSH && useHTTPS {
		fmt.Fprintln(os.Stderr, "warning: --ssh and --https both set; --ssh wins")
		useHTTPS = false
	}
	if useSSH {
		if converted, ok := ConvertURLToSSH(url); ok {
			if converted != url {
				fmt.Printf("↪ --ssh rewrite: %s → %s\n", url, converted)
			}
			return converted
		}
	}
	if useHTTPS {
		if converted, ok := ConvertURLToHTTPS(url); ok {
			if converted != url {
				fmt.Printf("↪ --https rewrite: %s → %s\n", url, converted)
			}
			return converted
		}
	}
	return url
}

// maybeRunFixRepoStep runs `fix-repo --all` only when the cloned repo
// identity carries a `-vN` suffix. The identity comes from Git remote
// metadata first, not the flattened local folder name.
// `--require-version` restores the strict (exit-4) failure mode for
// CI pipelines that want the old contract.
func maybeRunFixRepoStep(absPath string, requireVersion bool) {
	repoName := resolveCloneFixRepoName(absPath)
	parsed := clonenext.ParseRepoName(repoName)
	if parsed.HasVersion {
		runChainedGitmapStep([]string{constants.CmdFixRepo, "--" + constants.FixRepoFlagAll})

		return
	}
	if requireVersion {
		fmt.Fprintf(os.Stderr, constants.ErrCloneFixRepoNeedVersion, parsed.BaseName)
		os.Exit(constants.ExitCloneFixRepoChainFailed)
	}
	fmt.Printf(constants.MsgCloneFixRepoSkipNoVer, parsed.BaseName)
}

func resolveCloneFixRepoName(absPath string) string {
	remoteURL, err := gitutil.RemoteURL(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, constants.WarnCloneFixRepoRemoteFmt, absPath, err)

		return filepath.Base(absPath)
	}
	repo := repoNameFromURL(remoteURL)
	if len(repo) > 0 {
		return repo
	}
	fmt.Fprintf(os.Stderr, constants.WarnCloneFixRepoRemoteFmt, remoteURL, constants.ErrCloneFixRepoRemoteParse)

	return filepath.Base(absPath)
}

// parseCloneFixRepoArgs returns (url, folderName, noVSCodeSync,
// requireVersion, useSSH, useHTTPS). First non-flag arg is the URL;
// second non-flag is the destination folder. Recognized flags:
// --no-vscode-sync, --require-version, --ssh/-ssh/--sh,
// --https/-https/--ht. Single-dash forms are accepted to match Go's
// stdlib `flag` package behaviour the user expects from `-ssh`.
func parseCloneFixRepoArgs(args []string) (string, string, bool, bool, bool, bool, bool, bool) {
	positional := make([]string, 0, len(args))
	noVSCodeSync := false
	requireVersion := false
	useSSH := false
	useHTTPS := false
	autoYes := false
	dryRun := false
	syncFlag := constants.FlagNoVSCodeSync
	reqFlag := constants.FlagRequireVersion
	for _, a := range args {
		name := strings.TrimLeft(a, "-")
		switch name {
		case syncFlag:
			noVSCodeSync = true
			continue
		case reqFlag:
			requireVersion = true
			continue
		case "ssh", "sh":
			useSSH = true
			continue
		case "https", "ht":
			useHTTPS = true
			continue
		case "y", "yes":
			autoYes = true
			continue
		case constants.FlagCloneDryRun, constants.FlagCloneDryRunShort:
			dryRun = true
			continue
		}
		if len(a) > 0 && a[0] != '-' {
			positional = append(positional, a)
		}
	}
	url := ""
	folder := ""
	if len(positional) > 0 {
		url = positional[0]
	}
	if len(positional) > 1 {
		folder = positional[1]
	}

	return url, folder, noVSCodeSync, requireVersion, useSSH, useHTTPS, autoYes, dryRun
}


// resolveCloneTargetFolder mirrors the folder-naming logic in
// executeDirectClone so we know which directory to cd into after
// the clone step finishes. Versioned URLs auto-flatten to BaseName.
func resolveCloneTargetFolder(url, folderName string) string {
	if len(folderName) == 0 {
		repoName := repoNameFromURL(url)
		parsed := clonenext.ParseRepoName(repoName)
		if parsed.HasVersion {
			folderName = parsed.BaseName
		} else {
			folderName = repoName
		}
	}
	abs, err := filepath.Abs(folderName)
	if err != nil {
		return folderName
	}

	return abs
}

// runChainedGitmapStep re-execs the current gitmap binary with the
// given args, streaming stdin/stdout/stderr through. Any non-zero
// exit propagates immediately so the pipeline halts on first failure.
func runChainedGitmapStep(args []string) {
	bin, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, constants.ErrCloneFixRepoExecFmt, err)
		os.Exit(constants.ExitCloneFixRepoChainFailed)
	}
	cmd := exec.Command(bin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if runErr := cmd.Run(); runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, constants.ErrCloneFixRepoExecFmt, runErr)
		os.Exit(constants.ExitCloneFixRepoChainFailed)
	}
}
