// Command cron-deploy diffs old vs new registry, deploys each change to its
// target repo, records GitHub deployments, and labels the cronbot PR.
//
// All logic runs in Go. The YAML just calls this one CLI.
//
// Usage:
//
//	cron-deploy \
//	  --old /tmp/old.json --new registry.json \
//	  --repo owner/cronbot --pr-number 42 \
//	  --git-name "cron-bot[bot]" --git-email "cron-bot[bot]@users.noreply.github.com" \
//	  --output /tmp/deploy-result.json
//
// Exit codes: 0 = deploy attempted (check output for per-change results), 2 = usage/IO error.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"cronbot/internal/crondiff"
	"cronbot/internal/deploy"
	"cronbot/internal/registry"
)

func main() {
	oldPath := flag.String("old", "", "old registry.json ('' = empty)")
	newPath := flag.String("new", "", "new registry.json (required)")
	githubRepo := flag.String("repo", "", "cronbot repo (owner/name, for deployments + labels)")
	prNumber := flag.Int("pr-number", 0, "merged cronbot PR number (for labeling)")
	gitName := flag.String("git-name", "cron-bot[bot]", "git commit name")
	gitEmail := flag.String("git-email", "cron-bot[bot]@users.noreply.github.com", "git commit email")
	output := flag.String("output", "", "write JSON result to this path")
	flag.Parse()

	if *newPath == "" {
		fmt.Fprintln(os.Stderr, "--new is required")
		os.Exit(2)
	}

	old, _ := loadEntries(*oldPath)
	new, err := loadEntries(*newPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load new:", err)
		os.Exit(2)
	}

	changes := crondiff.Diff(old, new)
	if len(changes) == 0 {
		writeOutput(*output, map[string]any{"changes": 0, "results": []any{}})
		return
	}

	shell := deploy.ExecShell{}
	results := deploy.DeployAll(shell, changes, deploy.Options{
		GitName:  *gitName,
		GitEmail: *gitEmail,
	})

	summary := deploy.DeploySummary(results)

	// Print summary to stdout for the workflow log.
	fmt.Println(summary)

	if *output != "" {
		writeOutput(*output, map[string]any{
			"changes": len(changes),
			"results": results,
			"summary": summary,
			"pr_number": *prNumber,
			"repo": *githubRepo,
		})
	}
}

func loadEntries(path string) ([]registry.Entry, error) {
	if path == "" {
		return nil, nil
	}
	reg, err := registry.Load(path)
	if err != nil {
		return nil, err
	}
	return reg.All(), nil
}

func writeOutput(path string, data any) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "write output:", err)
		os.Exit(2)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.Encode(data)
}