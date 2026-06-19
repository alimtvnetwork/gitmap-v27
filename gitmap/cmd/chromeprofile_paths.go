// Package cmd — chromeprofile_paths.go: cross-platform Chrome User
// Data directory resolution. See spec/04-generic-cli/40-chrome-profile-copy.md.
package cmd

import (
	"os"
	"path/filepath"
	"runtime"
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

// pathExists reports whether path exists on disk.
func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
