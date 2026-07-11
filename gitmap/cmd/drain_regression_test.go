package cmd

// Regression test for the Windows drain fix (v6.74.0).
//
// Bug: when the theme/glyphs pipe wrappers were installed (ModeSafe
// path), the last stdout line written just before the process exited
// could be lost on Windows because the forwarder goroutine was
// descheduled before the runtime flushed the pipe. The CI installer
// smoke test caught this only after every release cut. This test
// reproduces the failure locally without any release dependency by
// forcing GITMAP_GLYPHS=safe (which activates the pipe wrapper) and
// asserting that `gitmap version` still prints its single-line
// output on a clean exit.
//
// Regression guard: if a future change removes the `defer Drain()`
// calls from runDispatch — or introduces a new entry point that
// calls dispatch directly — this test fails on Windows CI with an
// empty stdout capture.

import (
	"os"
	"strings"
	"testing"

	"github.com/alimtvnetwork/gitmap-v26/gitmap/constants"
)

// TestVersion_FlushedOnCleanExit runs `gitmap version` under
// GITMAP_GLYPHS=safe so the pipe wrappers are actually installed
// (the default hermetic env pins them off). Asserts the output is
// non-empty and contains the pinned constants.Version — which is
// exactly the assertion the CI smoke test makes, brought down to
// unit-test scope so it runs on every PR.
func TestVersion_FlushedOnCleanExit(t *testing.T) {
	t.Parallel()
	// GITMAP_GLYPHS=safe activates glyphs.Install()'s pipe-wrap
	// path, which is the path the Windows drain bug lived in.
	// We inherit the rest of hermeticEnv() and override just the
	// glyphs mode by prepending to the returned slice — later
	// duplicates win on the child's os.Environ lookup order for
	// most Go runtimes, but exec passes env verbatim so we must
	// filter out the pinned "rich" value ourselves.
	env := envSwapGlyphs("safe")
	bin := ensureGitmapBinary(t)
	code, stdout, stderr := runGitmapWithEnv(t, bin, []string{"version"}, "", env)
	if code != 0 {
		t.Fatalf("version exited %d\nstdout=%q\nstderr=%q", code, stdout, stderr)
	}
	if len(strings.TrimSpace(stdout)) == 0 {
		t.Fatalf("version produced empty stdout under glyphs=safe — drain regression\nstderr=%q", stderr)
	}
	if !strings.Contains(stdout, constants.Version) {
		t.Fatalf("version stdout %q missing pinned version %q", stdout, constants.Version)
	}
}

// envSwapGlyphs returns hermeticEnv() with GITMAP_GLYPHS forced to
// mode. Keeps the swap logic in one place so future modes are easy
// to test.
func envSwapGlyphs(mode string) []string {
	base := hermeticEnv()
	out := make([]string, 0, len(base)+1)
	for _, kv := range base {
		if strings.HasPrefix(kv, "GITMAP_GLYPHS=") {
			continue
		}
		out = append(out, kv)
	}
	out = append(out, "GITMAP_GLYPHS="+mode)

	return out
}

// runGitmapWithEnv is a thin fork of runGitmap that lets the caller
// override the child's env. Not merged into runGitmap because every
// existing call site relies on the pinned defaults.
func runGitmapWithEnv(t *testing.T, bin string, args []string, stdin string, env []string) (int, string, string) {
	t.Helper()
	// Reuse runGitmap's plumbing by temporarily swapping os.Environ
	// via a closure would be racy under t.Parallel; instead invoke
	// exec directly. Keeps this helper deterministic even when
	// several drain regression tests run in parallel.
	return execCaptured(t, bin, args, stdin, env)
}

// execCaptured is the file-backed capture pattern used by runGitmap,
// factored to accept a caller-supplied env. See runGitmap in
// cliexit_helpers_test.go for the Windows-pipe rationale.
func execCaptured(t *testing.T, bin string, args []string, stdin string, env []string) (int, string, string) {
	t.Helper()
	// Delegate to the existing file-backed helper by shelling out
	// through the standard runGitmap plumbing. We can't reach the
	// internal helpers cheaply without duplicating, so we accept a
	// tiny duplication here in exchange for env-override capability.
	tmp := t.TempDir()
	stdoutPath := tmp + string(os.PathSeparator) + "stdout"
	stderrPath := tmp + string(os.PathSeparator) + "stderr"

	return runExecFileBacked(t, bin, args, stdin, env, stdoutPath, stderrPath)
}
