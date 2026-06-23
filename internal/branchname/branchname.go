// Package branchname generates deterministic git branch names for cron deploys.
package branchname

import (
	"strings"
)

// Deploy returns a deterministic branch name for a cron deploy to a target
// repo + path. The same input always yields the same branch, so the reconcile
// job can find the PR later.
func Deploy(repo, path string) string {
	s := repo + path
	s = strings.ToLower(s)
	s = strings.Map(slugChar, s)
	s = strings.Trim(s, "-")
	return "cron-bot/deploy-" + s
}

func slugChar(r rune) rune {
	switch {
	case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		return r
	case r == '/' || r == '.' || r == '_' || r == '-':
		return '-'
	}
	return '-'
}