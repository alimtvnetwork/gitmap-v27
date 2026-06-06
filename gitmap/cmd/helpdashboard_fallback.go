package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/alimtvnetwork/gitmap-v25/gitmap/constants"
)

// openHostedDocsFallback opens the hosted docs URL when the local docs site
// is unavailable (release didn't bundle docs-site.zip and download failed).
// Best-effort: prints the URL even if launching the browser fails so the user
// can copy it manually. See spec/02-app-issues/33-hd-hosted-docs-fallback.md.
func openHostedDocsFallback() {
	fmt.Fprintf(os.Stderr, constants.MsgHDHostedFallback, constants.DocsURL)
	openURL(constants.DocsURL)
}

// openURL launches the OS default browser for the given URL. Errors are
// swallowed because users always have the printed URL as a manual fallback.
func openURL(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case constants.OSWindows:
		cmd = exec.Command(constants.CmdWindowsShell, constants.CmdArgSlashC, constants.CmdArgStart, url)
	case constants.OSDarwin:
		cmd = exec.Command(constants.CmdOpen, url)
	default:
		cmd = exec.Command(constants.CmdXdgOpen, url)
	}

	_ = cmd.Start()
}
