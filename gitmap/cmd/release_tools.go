// Package cmd — release_tools.go: release-notes, release-dry, tag-rename.
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func runReleaseNotes(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "release-notes: ERROR usage: gitmap release-notes <vN..vM>")
		os.Exit(2)
	}
	rng := args[0]
	if !strings.Contains(rng, "..") {
		fmt.Fprintln(os.Stderr, "release-notes: ERROR range must be <tagA>..<tagB>")
		os.Exit(2)
	}
	out, err := exec.Command("git", "log", "--pretty=format:- %s (%h)", rng).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "release-notes: ERROR git log: %v\n%s", err, out)
		os.Exit(1)
	}
	parts := strings.Split(rng, "..")
	fmt.Printf("## %s — auto-generated from %s\n\n%s\n", parts[1], rng, string(out))
}

func runReleaseDry(args []string) {
	tag := ""
	if len(args) > 0 {
		tag = args[0]
	}
	fmt.Println("\033[1;96m▸ release-dry\033[0m  build + local tag rehearsal (no push)")
	if err := runStep("go build ./...", "go", "build", "./..."); err != nil {
		os.Exit(1)
	}
	if tag != "" {
		if err := runStep("git tag "+tag, "git", "tag", tag); err != nil {
			os.Exit(1)
		}
		fmt.Printf("\033[1;94mnotes for %s\033[0m\n", tag)
		_ = runStep("git log -10 --oneline", "git", "log", "-10", "--oneline")
		fmt.Printf("\n\033[2;37mundo:  \033[0m \033[1;96mgit tag -d %s\033[0m\n", tag)
	}
	fmt.Println("\033[1;92m✓ dry release complete\033[0m  nothing pushed")
}

func runStep(label string, name string, args ...string) error {
	fmt.Printf("  \033[2;37m$\033[0m %s\n", label)
	cmd := exec.Command(name, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func runTagRename(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "tag-rename: ERROR usage: gitmap tag-rename <old> <new>")
		os.Exit(2)
	}
	oldTag, newTag := args[0], args[1]
	steps := [][]string{
		{"git", "tag", newTag, oldTag},
		{"git", "tag", "-d", oldTag},
		{"git", "push", "origin", newTag},
		{"git", "push", "origin", ":refs/tags/" + oldTag},
	}
	for _, s := range steps {
		if err := runStep(strings.Join(s, " "), s[0], s[1:]...); err != nil {
			fmt.Fprintf(os.Stderr, "tag-rename: ERROR step failed: %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Printf("\033[1;92m✓ renamed\033[0m %s → %s (local + origin)\n", oldTag, newTag)
}
