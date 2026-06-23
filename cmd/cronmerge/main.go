// Command cronmerge finds the PR number for the most recent merge to main by
// matching the head SHA. Uses gh CLI.
//
// Usage:
//
//	cronmerge --repo owner/cronbot --head-sha <sha> --output pr-number.txt
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	repo := flag.String("repo", "", "cronbot repo (owner/name)")
	headSHA := flag.String("head-sha", "", "HEAD sha of the merge commit")
	output := flag.String("output", "", "write PR number to this path")
	flag.Parse()

	if *repo == "" || *headSHA == "" {
		fmt.Fprintln(os.Stderr, "--repo and --head-sha are required")
		os.Exit(2)
	}

	out, err := exec.Command("gh", "pr", "list", "--repo", *repo, "--state", "merged",
		"--limit", "5", "--json", "number,mergeCommit", "--jq",
		fmt.Sprintf("[.[] | select(.mergeCommit.oid == \"%s\")] | .[0].number", *headSHA)).Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gh pr list:", err)
		os.Exit(2)
	}

	prNumber := strings.TrimSpace(string(out))
	if prNumber == "" || prNumber == "null" {
		prNumber = "0"
	}

	if *output != "" {
		os.WriteFile(*output, []byte(prNumber), 0o644)
	} else {
		fmt.Println(prNumber)
	}
}