package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alimtvnetwork/gitmap-v20/gitmap/constants"
)

// printUsageFooter renders the colorful build-info footer shown at the
// bottom of `gitmap` (no args) and `gitmap help`. It always shows the
// installed version and, when the binary's source repo is reachable,
// the origin URL and the last commit (short SHA · subject · date).
//
// All git invocations are best-effort — failures fall back silently so
// the help screen never errors out.
func printUsageFooter() {
	fmt.Println()
	fmt.Println("  " + constants.ColorMagenta +
		"────────────────────────────────────────────────────────────" +
		constants.ColorReset)

	fmt.Printf("  %s● Version:%s     %sv%s%s\n",
		constants.ColorCyan, constants.ColorReset,
		constants.ColorWhite, constants.Version, constants.ColorReset)

	repoDir := resolveFooterRepoDir()
	if len(repoDir) == 0 {
		return
	}

	if url := captureGit(repoDir, "config", "--get", "remote.origin.url"); len(url) > 0 {
		fmt.Printf("  %s● Repo:%s        %s%s%s\n",
			constants.ColorCyan, constants.ColorReset,
			constants.ColorCyan, url, constants.ColorReset)
	}

	if branch := captureGit(repoDir, "rev-parse", "--abbrev-ref", "HEAD"); len(branch) > 0 {
		fmt.Printf("  %s● Branch:%s      %s%s%s\n",
			constants.ColorCyan, constants.ColorReset,
			constants.ColorGreen, branch, constants.ColorReset)
	}

	if commit := captureGit(repoDir, "log", "-1", "--format=%h · %s · %cr"); len(commit) > 0 {
		fmt.Printf("  %s● Last commit:%s %s%s%s\n",
			constants.ColorCyan, constants.ColorReset,
			constants.ColorYellow, commit, constants.ColorReset)
	}

	fmt.Println()
}

// resolveFooterRepoDir picks the best directory to inspect for the
// footer's git metadata. Preference order:
//  1. The source repo baked into the binary (constants.RepoPath).
//  2. The current working directory if it is inside a git repo.
func resolveFooterRepoDir() string {
	if len(constants.RepoPath) > 0 {
		if _, err := os.Stat(filepath.Join(constants.RepoPath, ".git")); err == nil {
			return constants.RepoPath
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	return cwd
}

// captureGit runs `git <args...>` in dir and returns trimmed stdout, or
// "" on any error. Stderr is discarded so the footer stays quiet when
// the directory is not a git repo.
func captureGit(dir string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}
