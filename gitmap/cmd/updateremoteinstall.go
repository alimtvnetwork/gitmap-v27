package cmd

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/alimtvnetwork/gitmap-v23/gitmap/constants"
)

// runUpdateRemoteInstall replaces the legacy source-rebuild update path
// (v5.50.x and earlier) with a remote-installer flow:
//
//  1. Pick the platform installer URL from the *current* gitmap repo
//     (gitmap-v23) — `install.ps1` on Windows, `install.sh` elsewhere.
//  2. Download it to a temp file.
//  3. Execute it. The downloaded installer itself runs the parallel
//     `-v<N+i>` sibling probe (see spec/07-generic-release/09) so the
//     latest gitmap-vN repo wins automatically — we don't probe here.
//
// Returns true if the install completed (exit 0) so the caller can
// short-circuit any legacy source-rebuild fallback.
func runUpdateRemoteInstall() bool {
	url := remoteInstallerURL()
	fmt.Printf(constants.MsgUpdateRemoteFetch, url)

	scriptPath, err := downloadRemoteInstaller(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, constants.ErrUpdateRemoteDownload, err)
		return false
	}
	defer os.Remove(scriptPath)

	fmt.Printf(constants.MsgUpdateRemoteRun, scriptPath)
	if err := runRemoteInstaller(scriptPath); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, constants.ErrUpdateRemoteRun, err)
		return false
	}

	fmt.Print(constants.MsgUpdateRemoteDone)
	return true
}

// remoteInstallerURL returns the canonical install-script URL for the
// current platform. Hosted at the repo root so the URL matches the
// install snippet pinned in README.md.
func remoteInstallerURL() string {
	if runtime.GOOS == "windows" {
		return constants.UpdateRemoteInstallerPwsh
	}
	return constants.UpdateRemoteInstallerBash
}

// downloadRemoteInstaller fetches url into a platform-appropriate temp
// file (.ps1 on Windows, .sh elsewhere) and returns the path. The
// caller is responsible for removing the file when finished.
func downloadRemoteInstaller(url string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	ext := ".sh"
	if runtime.GOOS == "windows" {
		ext = ".ps1"
	}
	tmp, err := os.CreateTemp("", "gitmap-update-*"+ext)
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		// UTF-8 BOM so older PowerShell hosts parse the script correctly.
		_, _ = tmp.Write([]byte{0xEF, 0xBB, 0xBF})
	}
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}
	tmp.Close()

	if runtime.GOOS != "windows" {
		_ = os.Chmod(tmp.Name(), 0o755)
	}
	return tmp.Name(), nil
}

// runRemoteInstaller exec's the downloaded script with the right shell.
func runRemoteInstaller(scriptPath string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell",
			"-ExecutionPolicy", "Bypass",
			"-NoProfile", "-NoLogo",
			"-File", scriptPath)
	} else {
		shell := "bash"
		if _, err := exec.LookPath(shell); err != nil {
			shell = "sh"
		}
		cmd = exec.Command(shell, scriptPath)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Dir = filepath.Dir(scriptPath)
	return cmd.Run()
}
