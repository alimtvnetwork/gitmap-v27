package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/alimtvnetwork/gitmap-v20/gitmap/constants"
)

// transportFlags holds the parsed --ssh / --https selection plus any
// remaining positional args to forward to the underlying git command.
type transportFlags struct {
	useSSH   bool
	useHTTPS bool
	rest     []string
}

// parseTransportFlags parses the shared --ssh/--https flag pair used
// by `gitmap push` and the `gitmap pull` cwd short-circuit. Both
// single- and double-dash forms are accepted because Go's `flag`
// already treats `-ssh` and `--ssh` identically when registered as a
// single token.
func parseTransportFlags(cmdName string, args []string) transportFlags {
	fs := flag.NewFlagSet(cmdName, flag.ExitOnError)
	sshFlag := fs.Bool("ssh", false, "Rewrite remote.origin.url to SSH (git@host:owner/repo.git) and persist via `git remote set-url`")
	fs.BoolVar(sshFlag, "sh", false, "Short alias for --ssh")
	httpsFlag := fs.Bool("https", false, "Rewrite remote.origin.url to HTTPS (https://host/owner/repo.git) and persist via `git remote set-url`")
	fs.BoolVar(httpsFlag, "ht", false, "Short alias for --https")

	fs.Parse(reorderFlagsBeforeArgs(args))

	return transportFlags{useSSH: *sshFlag, useHTTPS: *httpsFlag, rest: fs.Args()}
}

// runPush is the entry point for `gitmap push`. It short-circuits to
// a plain `git push` in the current working directory, optionally
// rewriting the origin remote to SSH / HTTPS first.
func runPush(args []string) {
	checkHelp(constants.CmdPush, args)
	requireOnline()
	tf := parseTransportFlags(constants.CmdPush, args)

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "getwd: %v\n", err)
		exitWith(1)

		return
	}
	if !isGitRepoCWD() {
		fmt.Fprintln(os.Stderr, "✗ not a git repository (run `gitmap push` inside a repo)")
		exitWith(1)

		return
	}

	if _, _, _, applyErr := ApplyTransportFlag(cwd, tf.useSSH, tf.useHTTPS); applyErr != nil {
		fmt.Fprintf(os.Stderr, "✗ %v\n", applyErr)
		exitWith(1)

		return
	}

	gitArgs := append([]string{"push"}, tf.rest...)
	fmt.Printf("→ Running: git %s (cwd: %s)\n", joinForLog(gitArgs), cwd)
	cmd := exec.Command("git", gitArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if runErr := cmd.Run(); runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitWith(exitErr.ExitCode())

			return
		}
		fmt.Fprintf(os.Stderr, "git push failed: %v\n", runErr)
		exitWith(1)
	}
}

// joinForLog renders argv as a space-separated banner for stdout
// without quoting — sufficient for the simple `git push <args>` form.
func joinForLog(args []string) string {
	out := ""
	for i, a := range args {
		if i > 0 {
			out += " "
		}
		out += a
	}

	return out
}
