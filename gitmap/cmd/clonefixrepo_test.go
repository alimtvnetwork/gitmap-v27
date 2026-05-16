package cmd

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/alimtvnetwork/gitmap-v19/gitmap/constants"
)

const TestCloneFixRemoteURL = "https://github.com/alimtvnetwork/gitmap-v20.git"

func TestResolveCloneFixRepoNameUsesRemote(t *testing.T) {
	dir := t.TempDir()
	runTestGit(t, dir, "init")
	runTestGit(t, dir, constants.GitConfigCmd, "remote.origin.url", TestCloneFixRemoteURL)

	got := resolveCloneFixRepoName(dir)
	if got != "gitmap-v20" {
		t.Fatalf("resolveCloneFixRepoName() = %q, want gitmap-v20", got)
	}
}

func TestResolveCloneFixRepoNameFallsBackToFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "gitmap")

	got := resolveCloneFixRepoName(dir)
	if got != "gitmap" {
		t.Fatalf("resolveCloneFixRepoName() = %q, want gitmap", got)
	}
}

func runTestGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command(constants.GitBin, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}