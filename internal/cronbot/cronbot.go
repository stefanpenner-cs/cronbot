// Package cronbot is the brain of the intake bot. It turns an approved cron
// request into a concrete provisioning plan: the central-registry entry plus the
// schedule-neutral "bot touch" that, once merged by cron-bot[bot], makes the bot
// the durable actor.
//
// The plan is pure and testable. The actual git/PR/merge is performed by the
// intake workflow using `gh` authenticated as the cron-bot App (the hands).
package cronbot

import (
	"regexp"
	"strings"

	"cronbot/internal/cronequiv"
	"cronbot/internal/intake"
	"cronbot/internal/registry"
	"cronbot/internal/rehome"
)

// OwnerTeam is the single team accountable for every managed cron and the only
// crew allowed to approve and merge cron changes. It is fixed policy, not a
// per-request field.
const OwnerTeam = "cron-reviewers"

// Plan is everything the intake workflow needs to land a cron-bot-owned cron.
type Plan struct {
	Repo          string         `json:"repo"`
	Path          string         `json:"path"`
	OldExpr       string         `json:"old_expr"`
	NewExpr       string         `json:"new_expr"`
	CanRewrite    bool           `json:"can_rewrite"`
	Branch        string         `json:"branch"`
	PRTitle       string         `json:"pr_title"`
	CommitMessage string         `json:"commit_message"`
	MergeMethod   string         `json:"merge_method"`
	Entry         registry.Entry `json:"entry"`
}

// RemovalPlan is everything the intake workflow needs to retire a managed cron:
// the central-registry key to de-register and the bot-authored edit that deletes
// the schedule from the target workflow. Like BuildPlan, it is pure and testable;
// the workflow performs the git/PR/merge as the cron-bot App.
type RemovalPlan struct {
	Repo          string `json:"repo"`
	Path          string `json:"path"`
	RegistryKey   string `json:"registry_key"`
	Branch        string `json:"branch"`
	PRTitle       string `json:"pr_title"`
	CommitMessage string `json:"commit_message"`
	MergeMethod   string `json:"merge_method"`
	Request       string `json:"request"`
}

// BuildRemovalPlan computes the plan to retire an approved cron. De-registering
// is the tested, automated part; the workflow deletes the schedule line in the
// target repo and merges as cron-bot[bot].
func BuildRemovalPlan(req intake.CronRequest, requestURL string) RemovalPlan {
	return RemovalPlan{
		Repo:          req.Repo,
		Path:          req.Path,
		RegistryKey:   registry.Key(req.Repo, req.Path),
		Branch:        "cron-bot/remove-" + slug(req.Repo+"-"+req.Path),
		PRTitle:       "chore(cron): remove managed schedule for " + req.Path,
		CommitMessage: "chore(cron): retire " + req.Path + " cron (de-homed from cron-bot[bot])",
		MergeMethod:   rehome.MergeMethod,
		Request:       requestURL,
	}
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	s = nonSlug.ReplaceAllString(strings.ToLower(s), "-")
	return strings.Trim(s, "-")
}

// BuildPlan computes the provisioning plan for an approved request. The bot's
// edit rewrites the cron to a schedule-equivalent but textually different value
// (via cronequiv); merging that as cron-bot[bot] re-attributes the actor without
// changing when it runs.
func BuildPlan(req intake.CronRequest, requestURL string) Plan {
	newExpr, ok := cronequiv.Rewrite(req.Expr)
	if !ok {
		newExpr = req.Expr
	}
	return Plan{
		Repo:          req.Repo,
		Path:          req.Path,
		OldExpr:       req.Expr,
		NewExpr:       newExpr,
		CanRewrite:    ok,
		Branch:        "cron-bot/" + slug(req.Repo+"-"+req.Path),
		PRTitle:       "chore(cron): cron-bot-owned schedule for " + req.Path,
		CommitMessage: "chore(cron): re-home " + req.Path + " onto cron-bot[bot]",
		MergeMethod:   rehome.MergeMethod,
		Entry: registry.Entry{
			Repo:      req.Repo,
			Path:      req.Path,
			Expr:      req.Expr,
			OwnerTeam: OwnerTeam,
			Request:   requestURL,
		},
	}
}
