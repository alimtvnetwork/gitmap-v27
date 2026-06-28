// Package cmd — workflow_open_pr.go: `gitmap open` opens the current
// repo's GitHub URL in the browser; `gitmap pr` lists open PRs.
package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func runOpen(args []string) {
	owner, repo, err := currentRepoOwnerRepo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: ERROR %v\n", err)
		os.Exit(1)
	}
	sub := ""
	if len(args) > 0 {
		switch args[0] {
		case "--issues", "-i":
			sub = "/issues"
		case "--prs", "-p":
			sub = "/pulls"
		case "--actions", "-a":
			sub = "/actions"
		}
	}
	url := fmt.Sprintf("https://github.com/%s/%s%s", owner, repo, sub)
	fmt.Printf("\033[1;96m▸ open\033[0m %s\n", url)
	openBrowser(url)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

func currentRepoOwnerRepo() (string, string, error) {
	out, err := exec.Command("git", "config", "--get", "remote.origin.url").Output()
	if err != nil {
		return "", "", fmt.Errorf("no origin remote: %w", err)
	}
	u := strings.TrimSpace(string(out))
	u = strings.TrimSuffix(u, ".git")
	u = strings.TrimPrefix(u, "git@github.com:")
	u = strings.TrimPrefix(u, "https://github.com/")
	parts := strings.SplitN(u, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unparseable remote url: %s", u)
	}
	return parts[0], parts[1], nil
}

func runPR(args []string) {
	owner := ""
	if len(args) > 0 {
		owner = args[0]
	} else {
		o, _, err := currentRepoOwnerRepo()
		if err != nil {
			fmt.Fprintln(os.Stderr, "pr: ERROR specify <owner> or run inside a github repo")
			os.Exit(2)
		}
		owner = o
	}
	token := os.Getenv("GITHUB_TOKEN")
	url := fmt.Sprintf("https://api.github.com/search/issues?q=is:pr+is:open+user:%s&per_page=50", owner)
	req, _ := http.NewRequest("GET", url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pr: ERROR %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	var body struct {
		Items []struct {
			Title   string `json:"title"`
			HTMLURL string `json:"html_url"`
			User    struct {
				Login string `json:"login"`
			} `json:"user"`
		} `json:"items"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	fmt.Printf("\033[1;94mOpen PRs for %s\033[0m  (%d)\n", owner, len(body.Items))
	for _, it := range body.Items {
		fmt.Printf("  \033[1;96m%s\033[0m  \033[2;37m@%s\033[0m\n    %s\n",
			it.Title, it.User.Login, it.HTMLURL)
	}
	if token == "" {
		fmt.Println("\n\033[2;37mhint:\033[0m export GITHUB_TOKEN for higher rate limits + private repos")
	}
}

func runBlameStats(args []string) {
	root := "."
	if len(args) > 0 {
		root = args[0]
	}
	out, err := exec.Command("git", "-C", root, "ls-files").Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "blame-stats: ERROR ls-files: %v\n", err)
		os.Exit(1)
	}
	totals := map[string]int{}
	for _, f := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if f == "" {
			continue
		}
		blame, err := exec.Command("git", "-C", root, "blame", "--line-porcelain", f).Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(blame), "\n") {
			if strings.HasPrefix(line, "author ") {
				totals[strings.TrimPrefix(line, "author ")]++
			}
		}
	}
	fmt.Printf("\033[1;94mBlame stats\033[0m  %s\n", root)
	for who, n := range totals {
		fmt.Printf("  %-30s %d\n", who, n)
	}
}
