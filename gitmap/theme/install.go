// Package theme — install.go: optional os.Stdout / os.Stderr
// interception that pipes every write through Filter for the active
// mode. Skipped entirely for ModeBright so the default code path
// stays a true zero-cost passthrough.
package theme

import (
	"io"
	"os"
	"sync"
)

var (
	installOnce sync.Once
	activeMode  Mode
	origStdout  = os.Stdout
	origStderr  = os.Stderr
	stdoutIsTTY = detectTTY(os.Stdout)
	stderrIsTTY = detectTTY(os.Stderr)
)

// Install resolves the active mode from the environment and, if it is
// not ModeBright, replaces os.Stdout and os.Stderr with pipe-backed
// writers whose reader-side goroutines apply Filter before forwarding
// bytes to the original fds. Safe to call multiple times — runs at
// most once per process.
func Install() {
	installOnce.Do(func() {
		activeMode = Resolve()
		if activeMode == ModeBright {
			return
		}
		os.Stdout = wrap(origStdout, activeMode)
		os.Stderr = wrap(origStderr, activeMode)
	})
}

// Active returns the mode chosen at Install time. Defaults to
// ModeBright when Install has not yet been called.
func Active() Mode {
	return activeMode
}

// IsStdoutTTY reports whether the *original* stdout (before any
// theme pipe interception) is a real terminal. Callers in
// gitmap/render gate ANSI pretty-rendering on this so the
// monochrome / standard pipe wrappers don't break TTY detection.
func IsStdoutTTY() bool { return stdoutIsTTY }

// IsStderrTTY is the stderr counterpart of IsStdoutTTY.
func IsStderrTTY() bool { return stderrIsTTY }

// wrap returns a new *os.File whose write end forwards filtered bytes
// to dst. A goroutine drains the read end for the lifetime of the
// process; the OS reaps the pipe on exit.
func wrap(dst *os.File, mode Mode) *os.File {
	r, w, err := os.Pipe()
	if err != nil {
		// Pipe creation should never fail under normal conditions.
		// Fall back to the original fd so output isn't lost.
		return dst
	}
	go forward(r, dst, mode)

	return w
}

// forward reads from r, applies Filter, and writes to dst until EOF.
func forward(r io.ReadCloser, dst io.Writer, mode Mode) {
	defer func() { _ = r.Close() }()

	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			_, _ = dst.Write(Filter(buf[:n], mode))
		}
		if err != nil {
			return
		}
	}
}

// detectTTY captures the TTY state of a handle before Install
// potentially replaces it with a pipe.
func detectTTY(f *os.File) bool {
	if f == nil {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}

	return (info.Mode() & os.ModeCharDevice) != 0
}
