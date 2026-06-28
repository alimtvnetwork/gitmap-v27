// Package cmd — `gitmap stale` (st): list local repos with no commits
// in N days, with optional --archive to move them to .gitmap/archive/.
// v6.68.0.
package cmd

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type staleRepo struct {
	path string
	last time.Time
}

// runStale executes `gitmap stale`.
func runStale(args []string) {
	checkHelp("stale", args)
	fs := flag.NewFlagSet("stale", flag.ContinueOnError)
	days := fs.Int("days", 90, "report repos with no commits in the last N days")
	root := fs.String("root", ".", "scan root directory")
	archive := fs.Bool("archive", false, "move stale repos into .gitmap/archive/")
	dryRun := fs.Bool("dry-run", false, "preview archive moves without touching disk")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	repos := scanForRepos(*root)
	cutoff := time.Now().AddDate(0, 0, -*days)
	var stale []staleRepo
	for _, r := range repos {
		t, ok := lastCommitTime(r)
		if !ok {
			continue
		}
		if t.Before(cutoff) {
			stale = append(stale, staleRepo{path: r, last: t})
		}
	}
	sort.Slice(stale, func(i, j int) bool { return stale[i].last.Before(stale[j].last) })
	printStaleTable(stale, *days)
	if *archive {
		archiveStaleRepos(stale, *dryRun)
	}
}

// scanForRepos returns directories under root that contain a .git folder.
func scanForRepos(root string) []string {
	var out []string
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		full := filepath.Join(root, e.Name())
		if isGitRepo(full) {
			out = append(out, full)
		}
	}

	return out
}

// isGitRepo returns true if dir contains a .git directory or file.
func isGitRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))

	return err == nil
}

// lastCommitTime returns the last commit time for repo at dir.
func lastCommitTime(dir string) (time.Time, bool) {
	cmd := exec.Command("git", "-C", dir, "log", "-1", "--format=%ct")
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}, false
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return time.Time{}, false
	}
	var sec int64
	if _, err := fmt.Sscanf(s, "%d", &sec); err != nil {
		return time.Time{}, false
	}

	return time.Unix(sec, 0), true
}

// printStaleTable renders the stale-repo list.
func printStaleTable(stale []staleRepo, days int) {
	if len(stale) == 0 {
		fmt.Fprintf(os.Stdout, "\n  no repos stale beyond %d days\n\n", days)

		return
	}
	fmt.Fprintf(os.Stdout, "\n  \033[36m%d stale repo(s)\033[0m (no commits in %d days)\n\n", len(stale), days)
	now := time.Now()
	for _, s := range stale {
		age := int(now.Sub(s.last).Hours() / 24)
		fmt.Fprintf(os.Stdout, "  \033[33m%4dd\033[0m  %s  (last: %s)\n", age, s.path, s.last.Format("2006-01-02"))
	}
	fmt.Fprintln(os.Stdout, "")
}

// archiveStaleRepos moves repos into .gitmap/archive/<basename>-<ts>/.
func archiveStaleRepos(stale []staleRepo, dryRun bool) {
	if len(stale) == 0 {
		return
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	archiveDir := filepath.Join(".gitmap", "archive", stamp)
	for _, s := range stale {
		dest := filepath.Join(archiveDir, filepath.Base(s.path))
		if dryRun {
			fmt.Fprintf(os.Stdout, "  \033[33mwould archive\033[0m %s -> %s\n", s.path, dest)

			continue
		}
		if err := os.MkdirAll(archiveDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "  archive mkdir: %v\n", err)

			return
		}
		if err := os.Rename(s.path, dest); err != nil {
			fmt.Fprintf(os.Stderr, "  \033[31mfailed\033[0m %s: %v\n", s.path, err)

			continue
		}
		fmt.Fprintf(os.Stdout, "  \033[32marchived\033[0m %s -> %s\n", s.path, dest)
	}
}
