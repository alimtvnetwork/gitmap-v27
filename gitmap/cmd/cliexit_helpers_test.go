package cmd

// Shared subprocess test harness for the `cliexit_*_test.go` suite.
//
// These tests assert the documented exit codes of `gitmap` subcommands
// (success / user-canceled / failure). Asserting *real* exit codes
// requires running the compiled binary out-of-process because the
// production code calls os.Exit directly — there's no in-process
// "return an int" seam to stub.
//
// Wiring overview:
//
//   1. TestMain builds the gitmap binary once into t.TempDir-style
//      shared cache (under os.TempDir) and reuses it for every test.
//   2. runGitmap(t, args, stdin) execs the binary with a hermetic
//      working dir + minimal env, pipes optional stdin, and returns
//      (exitCode, stdout, stderr).
//   3. Per-command files (cliexit_scan_test.go, cliexit_clone_test.go)
//      drive the harness with table-driven cases.
//
// The whole suite skips when `go` is not on PATH (stripped CI images)
// or the build itself fails for sandbox reasons (e.g. cgo disabled
// without a C toolchain). That keeps the larger test matrix green
// while still failing loudly on real exit-code regressions in any
// environment that *can* build the binary.

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// gitmapBinary holds the path to the once-built binary. Populated
// lazily by ensureGitmapBinary so a test run that touches none of
// the cliexit files pays no build cost.
var (
	gitmapBinary    string
	errGitmapBuild  error
	gitmapBuildOnce sync.Once
)

// ensureGitmapBinary builds the gitmap binary the first time it is
// called and caches the result. Returns the absolute path or skips
// the calling test when the build couldn't run (no `go`, sandbox
// blocks compilation, etc.).
func ensureGitmapBinary(t *testing.T) string {
	t.Helper()
	gitmapBuildOnce.Do(buildGitmapBinaryOnce)
	if errGitmapBuild != nil {
		t.Skipf("gitmap binary unavailable for cliexit tests: %v", errGitmapBuild)
	}

	return gitmapBinary
}

// buildGitmapBinaryOnce is invoked under sync.Once so concurrent
// t.Parallel tests share a single artifact.
func buildGitmapBinaryOnce() {
	if _, err := exec.LookPath("go"); err != nil {
		errGitmapBuild = err

		return
	}
	out := filepath.Join(os.TempDir(), gitmapBinaryName())
	// Build from the gitmap module root. The cwd of `go test` is
	// already the package dir (gitmap/cmd) so "../" is the module.
	cmd := exec.Command("go", "build", "-o", out, "./")
	cmd.Dir = ".." // gitmap/ module root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errGitmapBuild = wrapBuildErr(err, &stderr)

		return
	}
	gitmapBinary = out
}

// gitmapBinaryName returns the right artifact name per platform.
func gitmapBinaryName() string {
	name := "gitmap_cliexit_test"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}

	return name
}

// wrapBuildErr produces a single-line error that carries the tail of
// `go build` stderr so a failing CI log shows the real reason.
func wrapBuildErr(err error, stderr *bytes.Buffer) error {
	tail := strings.TrimSpace(stderr.String())
	if tail == "" {
		return err
	}

	return &buildError{cause: err, stderr: tail}
}

// buildError formats the underlying error + stderr tail.
type buildError struct {
	cause  error
	stderr string
}

// Error renders both the cause and the captured stderr so the test
// log contains everything needed to diagnose a build failure.
func (e *buildError) Error() string {
	return e.cause.Error() + ": " + e.stderr
}

// runGitmap executes the prebuilt binary with args + optional stdin
// and returns (exit code, stdout, stderr). Wraps the awkward parts
// of os/exec so call sites stay declarative.
//
// Windows-CI note (v5.47.0): on the GitHub Actions `windows-latest`
// runner under `pwsh -command ". '{0}'"`, when `cmd.Stdout`/`Stderr`
// is set to a `bytes.Buffer`, Go's `os/exec` internally creates an
// `os.Pipe()` and copies bytes in a goroutine. That pipe inherits
// from pwsh's already-redirected console handles and the runner has
// a long-standing bug where the parent end of those pipes reads
// EOF immediately — even though the child writes correctly. See
// actions/runner#382 and the StepCodex pwsh-stdio bug report. The
// documented workaround is to redirect the child to a *file* (real
// fd inheritance, no Go pipe goroutine in between) and read the
// file after the process exits. We use this everywhere now so the
// same code path runs on every OS instead of carving out Windows.
func runGitmap(t *testing.T, args []string, stdin string) (int, string, string) {
	t.Helper()
	bin := ensureGitmapBinary(t)
	cmd := exec.Command(bin, args...)
	cmd.Dir = t.TempDir()
	cmd.Env = hermeticEnv()
	cmd.Stdin = strings.NewReader(stdin)
	stdoutFile := mustCaptureFile(t, "stdout")
	stderrFile := mustCaptureFile(t, "stderr")
	defer func() { _ = stdoutFile.Close() }()
	defer func() { _ = stderrFile.Close() }()
	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile
	err := cmd.Run()

	return extractTestExitCode(err), mustReadAll(t, stdoutFile), mustReadAll(t, stderrFile)
}

// mustCaptureFile opens a fresh temp file for child-process output
// capture. Inheriting a real OS file handle avoids the Go-pipe
// goroutine path inside `os/exec` that pwsh-on-runner mishandles
// (see runGitmap doc comment).
func mustCaptureFile(t *testing.T, label string) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "gitmap-"+label+"-*.log")
	if err != nil {
		t.Fatalf("create %s capture file: %v", label, err)
	}

	return f
}

// mustReadAll seeks to start and reads the entire capture file.
// Tests never need more than a few KB so a single Read pass is
// sufficient; failures are fatal because they always mean a test
// infrastructure bug, not a product regression.
func mustReadAll(t *testing.T, f *os.File) string {
	t.Helper()
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatalf("seek capture file: %v", err)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("read capture file: %v", err)
	}

	return string(b)
}

// hermeticEnv strips variables that could change behavior between
// developer machines and CI (NO_COLOR honored so terminal renderers
// produce stable output if the test ever asserts on stdout).
func hermeticEnv() []string {
	keep := []string{"PATH", "HOME", "USERPROFILE", "SystemRoot", "TEMP", "TMP", "TMPDIR"}
	out := make([]string, 0, len(keep)+2)
	for _, k := range keep {
		if v := os.Getenv(k); v != "" {
			out = append(out, k+"="+v)
		}
	}
	out = append(out, "NO_COLOR=1")

	return out
}

// extractTestExitCode normalizes os/exec's error -> int conversion. A
// non-ExitError (couldn't start, signal, etc.) returns -1 so the
// caller's table assertion fails with a clear "got -1" rather than
// silently passing on an exit-0 default. Distinct from the
// production extractExitCode in regoldens.go which maps to 127.
func extractTestExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}

	return -1
}
