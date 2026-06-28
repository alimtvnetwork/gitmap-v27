// Package cmd — additional `gitmap doctor` probes:
// config paths, GitHub token availability, and network/API connectivity.
package cmd

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	doctorGitHubAPIURL    = "https://api.github.com"
	doctorHTTPTimeoutSecs = 5
)

// probeConfigPaths verifies the .gitmap/ working directory is reachable and writable.
func probeConfigPaths() DoctorCheck {
	return DoctorCheck{
		Name:    "config",
		FixHint: "Run gitmap from a writable directory; check perms on .gitmap/",
		Run: func() (bool, string) {
			wd, err := os.Getwd()
			if err != nil {
				return false, err.Error()
			}
			dir := filepath.Join(wd, ".gitmap")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return false, "cannot create " + dir + ": " + err.Error()
			}
			probe := filepath.Join(dir, ".doctor-probe")
			if err := os.WriteFile(probe, []byte("ok"), 0o644); err != nil {
				return false, "not writable: " + dir
			}
			_ = os.Remove(probe)
			return true, "writable: " + dir
		},
	}
}

// probeGitHubToken checks for GITHUB_TOKEN or GH_TOKEN in the environment.
// Missing token is a warning (ok=true with note) so anonymous flows still pass.
func probeGitHubToken() DoctorCheck {
	return DoctorCheck{
		Name:    "gh-token",
		FixHint: "export GITHUB_TOKEN=<pat>  # needed for private repos & higher rate limits",
		Run: func() (bool, string) {
			for _, k := range []string{"GITHUB_TOKEN", "GH_TOKEN"} {
				if v := os.Getenv(k); v != "" {
					return true, k + " set (" + maskToken(v) + ")"
				}
			}
			return false, "no GITHUB_TOKEN / GH_TOKEN in environment"
		},
	}
}

// probeGitHubAPI verifies network + GitHub API reachability with a short timeout.
func probeGitHubAPI() DoctorCheck {
	return DoctorCheck{
		Name:    "gh-api",
		FixHint: "Check internet / proxy / firewall; api.github.com must be reachable",
		Run: func() (bool, string) {
			client := &http.Client{Timeout: doctorHTTPTimeoutSecs * time.Second}
			req, err := http.NewRequest(http.MethodGet, doctorGitHubAPIURL, nil)
			if err != nil {
				return false, err.Error()
			}
			if tok := firstEnv("GITHUB_TOKEN", "GH_TOKEN"); tok != "" {
				req.Header.Set("Authorization", "Bearer "+tok)
			}
			resp, err := client.Do(req)
			if err != nil {
				if _, dnsErr := net.LookupHost("api.github.com"); dnsErr != nil {
					return false, "DNS lookup failed: " + dnsErr.Error()
				}
				return false, err.Error()
			}
			defer resp.Body.Close()
			if resp.StatusCode >= 500 {
				return false, "api.github.com returned " + resp.Status
			}
			return true, "api.github.com reachable (" + resp.Status + ")"
		},
	}
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

func maskToken(t string) string {
	if len(t) <= 8 {
		return "****"
	}
	return t[:4] + "…" + t[len(t)-4:]
}
