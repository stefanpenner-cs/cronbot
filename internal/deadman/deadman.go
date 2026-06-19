// Package deadman flags GHA crons that have silently stopped: ones that never
// fired here (never_fired) or have missed roughly three expected fires (stale).
package deadman

import (
	"fmt"
	"io"
	"sort"
	"time"

	"fixcron/internal/cronsched"
	"fixcron/internal/inventory"
)

// File is a workflow file collapsed from its (possibly several) cron lines.
type File struct {
	Repo          string
	Path          string
	WorkflowName  string
	State         string
	DefaultBranch string
	Expressions   []string
	FastestExpr   string // smallest-interval expression in the file
}

// Assessment is the deadman verdict for one workflow file.
type Assessment struct {
	Repo         string   `json:"repo"`
	Path         string   `json:"path"`
	WorkflowName string   `json:"workflow_name"`
	State        string   `json:"state"`
	Expressions  []string `json:"expressions"`
	FastestExpr  string   `json:"fastest_expr"`
	IntervalDays float64  `json:"interval_days"`
	LastRun      string   `json:"last_run"`
	DaysSince    *int     `json:"days_since"`
	RunActor     string   `json:"run_actor"`
	URL          string   `json:"url"`
	Health       string   `json:"health"`
}

// CollapseFiles collapses cron rows to one entry per (repo, path). A workflow
// file can hold several cron lines; last_runs is per file, so health is judged
// against the file's FASTEST cadence (smallest interval). Order is first-seen.
func CollapseFiles(crons []inventory.Cron) []*File {
	byKey := map[string]*File{}
	var order []*File
	for _, row := range crons {
		key := inventory.Key(row.Repo, row.Path)
		f, ok := byKey[key]
		if !ok {
			f = &File{
				Repo:          row.Repo,
				Path:          row.Path,
				WorkflowName:  row.WorkflowName,
				State:         row.State,
				DefaultBranch: row.DefaultBranch,
				Expressions:   []string{row.CronExpression},
				FastestExpr:   row.CronExpression,
			}
			byKey[key] = f
			order = append(order, f)
			continue
		}
		f.Expressions = append(f.Expressions, row.CronExpression)
		if cronsched.IntervalDays(row.CronExpression) < cronsched.IntervalDays(f.FastestExpr) {
			f.FastestExpr = row.CronExpression
		}
	}
	return order
}

func daysSince(lastRun string, now time.Time) (int, bool) {
	if lastRun == "" {
		return 0, false
	}
	t, err := time.Parse(time.RFC3339, lastRun)
	if err != nil {
		return 0, false
	}
	return int(now.Sub(t).Hours() / 24), true
}

// Assess returns a verdict per file (every health).
func Assess(files []*File, lastRuns map[string]inventory.RunEvidence, now time.Time) []Assessment {
	out := make([]Assessment, 0, len(files))
	for _, f := range files {
		ev := lastRuns[inventory.Key(f.Repo, f.Path)]
		days, hasRun := daysSince(ev.LastRun, now)
		var daysPtr *int
		if hasRun {
			d := days
			daysPtr = &d
		}
		out = append(out, Assessment{
			Repo:         f.Repo,
			Path:         f.Path,
			WorkflowName: f.WorkflowName,
			State:        f.State,
			Expressions:  f.Expressions,
			FastestExpr:  f.FastestExpr,
			IntervalDays: cronsched.IntervalDays(f.FastestExpr),
			LastRun:      ev.LastRun,
			DaysSince:    daysPtr,
			RunActor:     ev.Actor,
			URL:          ev.URL,
			Health:       cronsched.Health(days, hasRun, f.FastestExpr),
		})
	}
	return out
}

func isMissed(h string) bool { return h == "never_fired" || h == "stale" }

// Missed filters to silently-failing crons, worst-first: never_fired before
// stale; within stale, the longest-overdue first.
func Missed(rows []Assessment) []Assessment {
	var flagged []Assessment
	for _, r := range rows {
		if isMissed(r.Health) {
			flagged = append(flagged, r)
		}
	}
	sort.SliceStable(flagged, func(i, j int) bool {
		a, b := flagged[i], flagged[j]
		ra, rb := neverRank(a.Health), neverRank(b.Health)
		if ra != rb {
			return ra < rb
		}
		da, db := overdue(a.DaysSince), overdue(b.DaysSince)
		if da != db {
			return da > db // more overdue first
		}
		if a.Repo != b.Repo {
			return a.Repo < b.Repo
		}
		return a.Path < b.Path
	})
	return flagged
}

func neverRank(h string) int {
	if h == "never_fired" {
		return 0
	}
	return 1
}

func overdue(d *int) int {
	if d == nil {
		return 1 << 30
	}
	return *d
}

// Emit is the pluggable alert sink. Today: a console table. (Slack/issue later.)
func Emit(rows []Assessment, w io.Writer) {
	if len(rows) == 0 {
		fmt.Fprintln(w, "No missed/dead crons. \u2705")
		return
	}
	fmt.Fprintf(w, "%d missed/dead crons (worst-first):\n\n", len(rows))
	fmt.Fprintf(w, "%-11s %-11s %-9s REPO :: PATH\n", "HEALTH", "LAST RUN", "STATE")
	fmt.Fprintln(w, "--------------------------------------------------------------------------------")
	for _, r := range rows {
		last := "never"
		if len(r.LastRun) >= 10 {
			last = r.LastRun[:10]
		}
		fmt.Fprintf(w, "%-11s %-11s %-9s %s :: %s\n", r.Health, last, r.State, r.Repo, r.Path)
	}
}
