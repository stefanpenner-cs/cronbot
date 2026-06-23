// Command cronbranch prints the deterministic deploy branch name for a repo +
// path. Used by the reconcile workflow to find target repo PRs.
//
// Usage:
//
//	cronbranch --repo octo-org/foo --path .github/workflows/nightly.yml
//	# → cron-bot/deploy-octo-org-foo-github-workflows-nightly-yml
package main

import (
	"flag"
	"fmt"

	"cronbot/internal/branchname"
)

func main() {
	repo := flag.String("repo", "", "target repo (owner/name)")
	path := flag.String("path", "", "workflow file path")
	flag.Parse()

	if *repo == "" || *path == "" {
		fmt.Println("usage: cronbranch --repo owner/name --path .github/workflows/foo.yml")
		return
	}

	fmt.Println(branchname.Deploy(*repo, *path))
}