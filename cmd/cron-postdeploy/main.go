// Command cron-postdeploy reads a deploy result JSON, labels the cronbot PR
// (deploying/deployed), and posts the right comment. Uses gh CLI.
//
// Usage:
//
//	cron-postdeploy --result /tmp/deploy-result.json --repo owner/cronbot --pr-number 42
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

type deployResult struct {
	Results []struct {
		State string `json:"state"`
	}
	PRNumber int    `json:"pr_number"`
	Repo     string `json:"repo"`
}

func main() {
	resultPath := flag.String("result", "", "path to deploy result JSON (required)")
	prNumber := flag.String("pr-number", "", "cronbot PR number")
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

	var result deployResult
	if err := json.Unmarshal(b, &result); err != nil {
		fmt.Fprintln(os.Stderr, "parse result:", err)
		os.Exit(2)
	}

	hasInProgress := false
	for _, r := range result.Results {
		if r.State == "in_progress" {
			hasInProgress = true
		}
	}

	var label, comment string
	if hasInProgress {
		label = "deploying"
		comment = prcomment.DeployBlocked()
	} else {
		label = "deployed"
		comment = prcomment.DeploySuccess()
	}

	if *prNumber != "" && *repo != "" {
		exec.Command("gh", "pr", "edit", *prNumber, "--repo", *repo, "--remove-label", "deploying", "--add-label", label).Run()
		exec.Command("gh", "pr", "comment", *prNumber, "--repo", *repo, "--body", comment).Run()
	}

	fmt.Println(label)
	_ = recon.HasInProgress
}