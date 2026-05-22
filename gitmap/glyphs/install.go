// Package glyphs — install.go: optional pipe-wrap that runs every
// stdout / stderr byte through Filter. Skipped entirely in ModeRich
// for true zero-overhead passthrough.
package glyphs

import (
	"io"
	"os"
	"sync"
)

var (
	installOnce sync.Once
	activeMode  Mode
)

// Install resolves the active mode and (when ModeSafe) replaces
// os.Stdout / os.Stderr with pipe-backed writers that filter glyphs.
// Idempotent across calls.
func Install() {
	installOnce.Do(func() {
		activeMode = Resolve()
		if activeMode == ModeRich {
			return
		}
		os.Stdout = wrap(os.Stdout, activeMode)
		os.Stderr = wrap(os.Stderr, activeMode)
	})
}

// Active returns the resolved mode (defaults to ModeRich pre-Install).
func Active() Mode { return activeMode }

// wrap returns a *os.File whose writes are filtered before reaching dst.
func wrap(dst *os.File, mode Mode) *os.File {
	r, w, err := os.Pipe()
	if err != nil {
		return dst
	}
	go forward(r, dst, mode)

	return w
}

// forward streams r → Filter → dst until EOF.
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
