package completion

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/alimtvnetwork/gitmap-v19/gitmap/constants"
)

// InstallCDFunction writes the gitmap/gcd shell wrapper to user profiles.
func InstallCDFunction(shell string) error {
	snippet := cdSnippet(shell)
	if len(snippet) == 0 {
		return fmt.Errorf(constants.ErrCompUnknownShell, shell)
	}
	if shell == constants.ShellPowerShell {
		return installPowerShellCDFunction(snippet)
	}

	return appendCDFunctions(snippet, cdProfilePaths(shell))
}

func installPowerShellCDFunction(snippet string) error {
	if err := appendCDFunctions(snippet, cdProfilePaths(constants.ShellPowerShell)); err != nil {
		return err
	}
	if runtime.GOOS != constants.OSWindows {
		return nil
	}

	return installPowerShellCommandShim()
}

func installPowerShellCommandShim() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	dir := filepath.Dir(exe)
	body := renderPowerShellCommandShim(dir)

	return os.WriteFile(filepath.Join(dir, constants.PowerShellShimFile), []byte(body), 0o755)
}

func renderPowerShellCommandShim(dir string) string {
	escaped := strings.ReplaceAll(dir,
		constants.PowerShellSingleQuote,
		constants.PowerShellEscapedQuote)

	return fmt.Sprintf(constants.PowerShellShimTemplateFmt, escaped)
}

// cdSnippet returns the gcd function body for the given shell.
func cdSnippet(shell string) string {
	switch shell {
	case constants.ShellPowerShell:
		return constants.CDFuncPowerShell
	case constants.ShellBash:
		return constants.CDFuncBash
	case constants.ShellZsh:
		return constants.CDFuncZsh
	default:
		return ""
	}
}

// cdProfilePaths returns all profile paths to write the cd function to.
func cdProfilePaths(shell string) []string {
	switch shell {
	case constants.ShellPowerShell:
		return resolvePowerShellProfilePaths()
	case constants.ShellBash:
		home, _ := os.UserHomeDir()
		return []string{filepath.Join(home, ".bashrc")}
	default:
		home, _ := os.UserHomeDir()
		return []string{filepath.Join(home, ".zshrc")}
	}
}

// appendCDFunctions appends the managed wrapper to every resolved profile.
func appendCDFunctions(snippet string, profilePaths []string) error {
	for _, profilePath := range profilePaths {
		if err := appendCDFunction(snippet, profilePath); err != nil {
			return err
		}
	}

	return nil
}

// appendCDFunction appends the gitmap command wrapper to the profile if not present.
func appendCDFunction(snippet, profilePath string) error {
	if err := os.MkdirAll(filepath.Dir(profilePath), 0o755); err != nil {
		return fmt.Errorf(constants.ErrCompProfileWrite, profilePath, err)
	}

	existing, err := os.ReadFile(profilePath)
	if err == nil {
		text := string(existing)
		if hasCurrentCDFunction(text) {
			next := replaceCDFunction(text, snippet)
			if next == text {
				fmt.Fprintf(os.Stderr, constants.MsgCDFuncAlready)

				return nil
			}

			if writeErr := os.WriteFile(profilePath, []byte(next), 0o644); writeErr != nil {
				return fmt.Errorf(constants.ErrCompProfileWrite, profilePath, writeErr)
			}
			fmt.Fprintf(os.Stderr, constants.MsgCDFuncInstalled)

			return nil
		}
	}

	f, err := os.OpenFile(profilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf(constants.ErrCompProfileWrite, profilePath, err)
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "\n%s\n%s\n", constants.CDFuncMarker, snippet)
	if err != nil {
		return fmt.Errorf(constants.ErrCompProfileWrite, profilePath, err)
	}

	fmt.Fprintf(os.Stderr, constants.MsgCDFuncInstalled)

	return nil
}

func hasCurrentCDFunction(text string) bool {
	return strings.Contains(text, constants.CDFuncMarker) &&
		strings.Contains(text, constants.CDFuncMarkerEnd)
}

func replaceCDFunction(text, snippet string) string {
	start := strings.Index(text, constants.CDFuncMarker)
	if start < 0 {
		return text
	}
	end := strings.Index(text[start:], constants.CDFuncMarkerEnd)
	if end < 0 {
		return text
	}
	end += start + len(constants.CDFuncMarkerEnd)
	replacement := constants.CDFuncMarker + "\n" + snippet

	return text[:start] + replacement + text[end:]
}
