package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/alimtvnetwork/gitmap-v27/gitmap/constants"
)

// CodingGuidelinesOpts controls a single Coding Guidelines v24 install run.
// The Runner factory is injectable so unit tests can substitute a fake
// *exec.Cmd without shelling out to the network. Zero-value opts are valid
// and default to real exec + os stdio.
type CodingGuidelinesOpts struct {
	WorkingDir string
	Runner     func(name string, args ...string) *exec.Cmd
	Stdout     io.Writer
	Stderr     io.Writer
	Stdin      io.Reader
}

// ErrCGShellNotFound is returned when the host lacks the shell required to
// execute the OS-appropriate installer (PowerShell on Windows; bash+curl
// elsewhere). Callers can detect it with errors.Is and surface an
// actionable copy-paste fallback.
var ErrCGShellNotFound = errors.New("coding-guidelines: required shell not found on PATH")

// RunCodingGuidelinesInstall dispatches to the OS-appropriate installer
// (PowerShell on Windows, bash on Unix) and streams stdout/stderr through
// the provided writers. All failures are logged to opts.Stderr per the
// zero-swallow error policy before being returned.
func RunCodingGuidelinesInstall(opts CodingGuidelinesOpts) error {
	opts = withCGDefaults(opts)
	if runtime.GOOS == "windows" {
		return dispatchCGWindows(opts)
	}

	return dispatchCGUnix(opts)
}

// withCGDefaults fills in real-exec + os stdio for any zero-value fields.
func withCGDefaults(opts CodingGuidelinesOpts) CodingGuidelinesOpts {
	if opts.Runner == nil {
		opts.Runner = exec.Command
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if opts.Stdin == nil {
		opts.Stdin = os.Stdin
	}

	return opts
}

// dispatchCGWindows runs the v24 PowerShell installer via `irm | iex`.
func dispatchCGWindows(opts CodingGuidelinesOpts) error {
	pwsh := resolvePowerShellBinary()
	if pwsh == "" {
		fmt.Fprintf(opts.Stderr, constants.ErrCGShellNotFoundWindows, constants.DefaultCodingGuidelinesURLWindows)
		return ErrCGShellNotFound
	}
	url := constants.DefaultCodingGuidelinesURLWindows
	fmt.Fprintf(opts.Stderr, constants.MsgCGRunningWindows, url)
	script := fmt.Sprintf("irm %s | iex", url)
	cmd := opts.Runner(pwsh, "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)

	return runCGInstaller(cmd, opts, "windows", url)
}

// dispatchCGUnix runs the v24 bash installer via `curl -fsSL | bash`.
func dispatchCGUnix(opts CodingGuidelinesOpts) error {
	if _, err := exec.LookPath("bash"); err != nil {
		fmt.Fprintf(opts.Stderr, constants.ErrCGShellNotFoundUnix, constants.DefaultCodingGuidelinesURLUnix)
		return ErrCGShellNotFound
	}
	if _, err := exec.LookPath("curl"); err != nil {
		fmt.Fprintf(opts.Stderr, constants.ErrCGShellNotFoundUnix, constants.DefaultCodingGuidelinesURLUnix)
		return ErrCGShellNotFound
	}
	url := constants.DefaultCodingGuidelinesURLUnix
	fmt.Fprintf(opts.Stderr, constants.MsgCGRunningUnix, url)
	script := fmt.Sprintf("curl -fsSL %s | bash", url)
	cmd := opts.Runner("bash", "-c", script)

	return runCGInstaller(cmd, opts, runtime.GOOS, url)
}

// runCGInstaller wires stdio + working dir onto the prepared command and
// executes it. Failures are logged to opts.Stderr in the standardized
// format before being wrapped with %w so callers can errors.Is / unwrap.
func runCGInstaller(cmd *exec.Cmd, opts CodingGuidelinesOpts, goos, url string) error {
	cmd.Dir = opts.WorkingDir
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr
	cmd.Stdin = opts.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(opts.Stderr, constants.ErrCGInstallFailed, goos, err)
		return fmt.Errorf("coding-guidelines install (%s, %s): %w", goos, url, err)
	}
	fmt.Fprint(opts.Stderr, constants.MsgCGDone)

	return nil
}
