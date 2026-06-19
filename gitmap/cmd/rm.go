package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alimtvnetwork/gitmap-v26/gitmap/store"
)

// rmUsage describes the `gitmap rm` command.
const rmUsage = `Usage: gitmap rm <name-or-path> [<name-or-path> ...]
       gitmap remove <name-or-path> [<name-or-path> ...]
       gitmap del <name-or-path> [<name-or-path> ...]

Removes one or more repositories from the gitmap database. Each target
may be a repo slug/name (e.g. "my-repo") or an absolute/relative path
to the repo on disk. The on-disk files are NOT touched.

Examples:
  gitmap rm my-repo
  gitmap remove my-repo
  gitmap del my-repo
  gitmap rm ./projects/foo ../bar
  gitmap rm repo-a repo-b /abs/path/repo-c
`

// runRm handles `gitmap rm <target>...`. Each target is resolved as a
// path first; if no row matches, it is treated as a slug. Missing
// targets print a warning but do not abort the batch.
func runRm(args []string) {
	checkHelp("rm", args)
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, rmUsage)
		os.Exit(1)
	}

	db, err := openDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "rm: open db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	exit := 0
	for _, target := range args {
		if removeOne(db, target) == 0 {
			exit = 1
		}
	}
	os.Exit(exit)
}

// removeOne deletes a single target by path then by slug. Returns the
// number of rows removed (0 = not found, which surfaces as exit 1).
func removeOne(db *store.DB, target string) int64 {
	target = strings.TrimSpace(target)
	if target == "" {
		return 0
	}

	if abs, err := filepath.Abs(target); err == nil {
		if n, err := db.DeleteByPath(abs); err == nil && n > 0 {
			fmt.Printf("removed %d repo by path: %s\n", n, abs)

			return n
		}
	}

	n, err := db.DeleteBySlug(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rm: %s: %v\n", target, err)

		return 0
	}
	if n == 0 {
		fmt.Fprintf(os.Stderr, "rm: no repo matched %q (tried path and slug)\n", target)

		return 0
	}
	fmt.Printf("removed %d repo(s) by name: %s\n", n, target)

	return n
}
