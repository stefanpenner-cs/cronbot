// Command cron-postrecon reads a reconcile result JSON, posts a completion
// comment on the cronbot PR, and removes the "deploying" label if all done.
// Uses gh CLI.
//
// Usage:
//
//	cron-postrecon --result /tmp/reconcile-result.json --repo owner/cronbot
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"cronbot/internal/prcomment"
	"cronbot/internal/recon"
)

type reconcileResult struct {
	Results []recon.ReconcileResult `json:"results"`
	Summary map[string]int           `json:"summary"`
}

func main() {
	resultPath := flag.String("result", "", "path to reconcile result JSON (required)")
	repo := flag.String("repo", "", "cronbot repo (owner/name)")
	flag.Parse()

	if *resultPath == "" {
		fmt.Fprintln(os.Stderr, "--result is required")
		os.Exit(2)
	}

	b, err := os.ReadFile(*resultPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read result:", err)
		os.Exit(2)
	}

	var result reconcileResult
	if err := json.Unmarshal(b, &result); err != nil {
		fmt.Fprintln(os.Stderr, "parse result:", err)
		os.Exit(2)
	}

	if !recon.AllComplete(result.Results) {
		fmt.Println("pending")
		return
	}

	comment := prcomment.ReconcileDone()

	if *repo != "" {
		// Find deploying PRs and update them.
		out, _ := exec.Command("gh", "pr", "list", "--repo", *repo, "--state", "merged",
			"--label", "deploying", "--json", "number", "--jq", ".[].number").Output()
		flatten := flattenLines(string(out))
		for _, num := range flatten {
			if num == "" {
				continue
			}
			exec.Command("gh", "pr", "edit", num, "--repo", *repo, "--remove-label", "deploying", "--add-label", "deployed").Run()
			exec.Command("gh", "pr", "comment", num, "--repo", *repo, "--body", comment).Run()
		}
	}

	fmt.Println("complete")
}

func flattenLines(s string) []string {
	var lines []string
	current := ""
	for _, c := range s {
		if c == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}