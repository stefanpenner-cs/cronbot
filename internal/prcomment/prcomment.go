// Package prcomment builds markdown comments for cron-bot PR comments.
// All functions are pure — they take data in and return a markdown string.
package prcomment

import (
	"fmt"
	"strings"

	"cronbot/internal/crondiff"
)

// PRCheck builds the comment for a PR check result.
// valid=false → error comment with validationErrors.
// valid=true  → success comment with the deploy plan.
func PRCheck(valid bool, planMarkdown, validationErrors string) string {
	if !valid {
		return fmt.Sprintf(`## ❌ Registry has errors

Edit the issue to fix these:

`+"```"+`
%s
`+"```"+`

Fix these before merging.`, validationErrors)
	}

	return fmt.Sprintf(`## ✅ Registry is valid

%s

Merge this PR to deploy. The cron-bot App will open a PR in each target repo.`, planMarkdown)
}

// DeployStarted builds the comment when deploy begins.
func DeployStarted(repo string) string {
	return "## 🚀 Deploy in progress\n\nOpening deploy PR in `" + repo + "`..."
}

// DeploySuccess builds the comment when all deploys merged.
func DeploySuccess() string {
	return `## ✅ Deployed

All target repo PRs merged. The cron(s) are now owned by ` + "`cron-bot[bot]`" + `.`
}

// DeployBlocked builds the comment when some target PRs can't auto-merge.
func DeployBlocked() string {
	return `## 🟡 Deploy in progress

One or more target repo PRs are waiting for merge. The reconcile job
checks hourly, or you can trigger it manually from the Actions tab.

Check the [deployments tab](../../deployments) for status.`
}

// DeployFailed builds the comment when a deploy PR creation fails.
func DeployFailed(repo string) string {
	return "## ❌ Deploy failed\n\nCould not create or merge a deploy PR in `" + repo + "`.\n\nCheck the workflow run for details."
}

// TokenNeeded builds the comment when no deploy token is configured.
func TokenNeeded() string {
	return `## ⏳ Deploy skipped — no token configured

Registry is updated, but ` + "`CRON_APP_ID`" + ` or ` + "`DEPLOY_TOKEN`" + ` is not set.

See [ci/README.md](../blob/main/ci/README.md) for setup.`
}

// ReconcileDone builds the comment when reconcile completes all deploys.
func ReconcileDone() string {
	return `## ✅ Deploy complete

All target repo PRs have been merged. The crons are now live and durable.`
}

// ReconcileFailed builds the comment when a target PR was closed without merging.
func ReconcileFailed(repos []string) string {
	var b strings.Builder
	b.WriteString("## ❌ Deploy failed — target PR closed without merge\n\n")
	b.WriteString("The following target repo PRs were closed without merging:\n\n")
	for _, r := range repos {
		b.WriteString(fmt.Sprintf("- `%s`\n", r))
	}
	b.WriteString("\nReopen or recreate the target PR, then trigger reconcile.")
	return b.String()
}

// DeployProgress builds a checklist comment for an in-progress deploy.
func DeployProgress(changes []crondiff.Change) string {
	var b strings.Builder
	b.WriteString("## 🚀 Deploying\n\n")
	for _, c := range changes {
		switch c.Action {
		case crondiff.Add:
			b.WriteString(fmt.Sprintf("- [ ] Add cron to `%s` :: `%s`\n", c.Repo, c.Path))
		case crondiff.Update:
			b.WriteString(fmt.Sprintf("- [ ] Update cron in `%s` :: `%s`\n", c.Repo, c.Path))
		case crondiff.Remove:
			b.WriteString(fmt.Sprintf("- [ ] Remove cron from `%s` :: `%s`\n", c.Repo, c.Path))
		}
	}
	return b.String()
}