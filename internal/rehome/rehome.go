// Package rehome produces a DRY-RUN plan to move fragile crons onto a durable
// account by making a schedule-neutral edit to each cron expression. It applies
// nothing — no commits, no pushes, no API writes.
package rehome

import (
	"fmt"
	"io"
	"sort"

	"fixcron/internal/actor"
	"fixcron/internal/cronequiv"
	"fixcron/internal/inventory"
)

// How a re-home edit must land to re-attribute the actor (cron-debugging
// findings): the new actor is the merger on a squash/merge-commit or the pusher
// on a direct push, but the ORIGINAL author on a rebase merge.
const (
	MergeMethod     = "squash or merge-commit (never rebase-merge)"
	TargetActorHint = "a durable service/bot account (svc-* or cron-bot[bot])"
)

// Entry is one dry-run plan row for a single cron expression.
type Entry struct {
	Repo          string `json:"repo"`
	Path          string `json:"path"`
	WorkflowName  string `json:"workflow_name"`
	State         string `json:"state"`
	FirstCronLine int    `json:"first_cron_line"`
	RunActor      string `json:"run_actor"`
	ActorClass    string `json:"actor_class"`
	Disposition   string `json:"disposition"`
	OldExpr       string `json:"old_expr"`
	NewExpr       string `json:"new_expr"`
	CanRewrite    bool   `json:"can_rewrite"`
	ReEnable      bool   `json:"re_enable"`
	MergeMethod   string `json:"merge_method"`
	TargetActor   string `json:"target_actor"`
	URL           string `json:"url"`
}

// Plan returns one dry-run entry per cron expression that needs re-homing,
// sorted worst-first (deprovisioned before human).
func Plan(crons []inventory.Cron, lastRuns map[string]inventory.RunEvidence) []Entry {
	var out []Entry
	for _, row := range crons {
		runActor := lastRuns[inventory.Key(row.Repo, row.Path)].Actor
		cls := actor.Class(runActor)
		if !actor.NeedsRehome(cls) {
			continue
		}
		newExpr, ok := cronequiv.Rewrite(row.CronExpression)

		branch := row.DefaultBranch
		if branch == "" {
			branch = "master"
		}
		url := fmt.Sprintf("https://github.com/%s/blob/%s/%s", row.Repo, branch, row.Path)
		if row.FirstCronLine > 0 {
			url += fmt.Sprintf("#L%d", row.FirstCronLine)
		}

		out = append(out, Entry{
			Repo:          row.Repo,
			Path:          row.Path,
			WorkflowName:  row.WorkflowName,
			State:         row.State,
			FirstCronLine: row.FirstCronLine,
			RunActor:      runActor,
			ActorClass:    cls,
			Disposition:   actor.Disposition(cls),
			OldExpr:       row.CronExpression,
			NewExpr:       newExpr,
			CanRewrite:    ok,
			ReEnable:      row.State != "" && row.State != "active",
			MergeMethod:   MergeMethod,
			TargetActor:   TargetActorHint,
			URL:           url,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if oa, ob := actor.ClassOrder[a.ActorClass], actor.ClassOrder[b.ActorClass]; oa != ob {
			return oa < ob
		}
		if a.Repo != b.Repo {
			return a.Repo < b.Repo
		}
		return a.Path < b.Path
	})
	return out
}

// Emit prints a console summary of the dry-run plan. Applies nothing.
func Emit(rows []Entry, w io.Writer) {
	if len(rows) == 0 {
		fmt.Fprintln(w, "No crons need re-homing. \u2705")
		return
	}
	fmt.Fprintf(w, "DRY RUN — %d crons to re-home (nothing applied), worst-first:\n\n", len(rows))
	for _, r := range rows {
		flag := ""
		if !r.CanRewrite {
			flag = "  [NO SAFE REWRITE]"
		}
		reenable := ""
		if r.ReEnable {
			reenable = "  +re-enable"
		}
		fmt.Fprintf(w, "[%s] %s :: %s%s%s\n", r.ActorClass, r.Repo, r.Path, flag, reenable)
		fmt.Fprintf(w, "    actor=%s  state=%s\n", r.RunActor, r.State)
		fmt.Fprintf(w, "    cron: '%s'  ->  '%s'\n", r.OldExpr, r.NewExpr)
	}
	fmt.Fprintf(w, "\nLand each edit via %s as %s.\n", MergeMethod, TargetActorHint)
}
