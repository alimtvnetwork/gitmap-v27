// Package cmd — chromeprofile_paths.go: cross-platform Chrome User
// Data directory resolution. See spec/04-generic-cli/40-chrome-profile-copy.md.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// chromeUserDataDir returns the platform-specific Chrome User Data
// root. Honors GITMAP_CHROME_USER_DATA for tests + custom installs.
func chromeUserDataDir() string {
	if override := os.Getenv("GITMAP_CHROME_USER_DATA"); len(override) > 0 {
		return override
	}
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		if local := os.Getenv("LOCALAPPDATA"); len(local) > 0 {
			return filepath.Join(local, "Google", "Chrome", "User Data")
		}
		return filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data")
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Google", "Chrome")
	default:
		return filepath.Join(home, ".config", "google-chrome")
	}
}

// chromeProfilePath joins the user-data root with a named profile dir.
// Accepts both raw names ("Default", "Profile 1") and absolute paths.
func chromeProfilePath(name string) string {
	if filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(chromeUserDataDir(), name)
}

// chromeProfilePathExists reports whether path exists on disk.
func chromeProfilePathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// availableChromeProfileNames returns every profile-shaped subdir
// ("Default" or "Profile N") under Chrome's User Data root. Empty
// slice on read error so callers can render a helpful "found: none"
// hint without aborting.
func availableChromeProfileNames() []string {
	entries, err := os.ReadDir(chromeUserDataDir())
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "Default" || strings.HasPrefix(name, "Profile ") {
			out = append(out, name)
		}
	}
	return out
}

// printAvailableChromeProfiles writes a "did you mean…" stderr block
// listing every profile we can see under the User Data root. Called
// after a not-found error so the user can pick a real name.
func printAvailableChromeProfiles() {
	root := chromeUserDataDir()
	names := availableChromeProfileNames()
	if len(names) == 0 {
		fmt.Fprintf(os.Stderr, "  available profiles under %s: (none found)\n", root)
		return
	}
	fmt.Fprintf(os.Stderr, "  available profiles under %s:\n", root)
	for _, n := range names {
		fmt.Fprintf(os.Stderr, "    - %s\n", n)
	}
}
