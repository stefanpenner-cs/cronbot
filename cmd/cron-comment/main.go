// Command cron-comment posts a markdown comment to a PR using a named template
// from the prcomment package. Uses gh CLI.
//
// Usage:
//
//	cron-comment --type token-needed --pr-number 42 --repo owner/cronbot
//	cron-comment --type deploy-success --pr-number 42 --repo owner/cronbot
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"cronbot/internal/prcomment"
)

func main() {
	commentType := flag.String("type", "", "comment template: token-needed, deploy-success, deploy-blocked")
	prNumber := flag.String("pr-number", "", "PR number")
	repo := flag.String("repo", "", "owner/name")
	flag.Parse()

	if *commentType == "" || *prNumber == "" || *repo == "" {
		fmt.Fprintln(os.Stderr, "--type, --pr-number and --repo are required")
		os.Exit(2)
	}

	var body string
	switch *commentType {
	case "token-needed":
		body = prcomment.TokenNeeded()
	case "deploy-success":
		body = prcomment.DeploySuccess()
	case "deploy-blocked":
		body = prcomment.DeployBlocked()
	default:
		fmt.Fprintf(os.Stderr, "unknown comment type: %s\n", *commentType)
		os.Exit(2)
	}

	cmd := exec.Command("gh", "pr", "comment", *prNumber, "--repo", *repo, "--body", body)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "gh pr comment:", err)
		os.Exit(2)
	}
}