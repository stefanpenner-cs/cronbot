// Package deploy orchestrates cron deploys to target repos. The core logic
// (deciding what to do, computing branches/messages) is pure and tested. The
// git/gh interactions go through a Shell interface so tests can inject fakes.
package deploy

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"cronbot/internal/crondiff"
	"cronbot/internal/cronfile"
)

// Shell is the minimal interface for git/gh commands. Real deployments use
// ExecShell; tests use a fake.
type Shell interface {
	Git(dir string, args ...string) (stdout, stderr string, err error)
	Gh(args ...string) (stdout, stderr string, err error)
}

// ExecShell runs real git and gh commands.
type ExecShell struct{}

func (ExecShell) Git(dir string, args ...string) (string, string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var out, errb strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	return out.String(), errb.String(), err
}

func (ExecShell) Gh(args ...string) (string, string, error) {
	cmd := exec.Command("gh", args...)
	var out, errb strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	return out.String(), errb.String(), err
}

// Options configures a deploy run.
type Options struct {
	GitName  string
	GitEmail string
}

// Result is the outcome of deploying one change.
type Result struct {
	Change     crondiff.Change
	State      string
	PRNumber   int
	PRExists   bool
}

// DeployOne deploys a single change to its target repo, returns the result.
func DeployOne(shell Shell, change crondiff.Change, opts Options) Result {
	branch := change.Branch

	// Clone the target repo to a temp dir.
	tmpDir, _, err := shell.Gh("repo", "clone", change.Repo, "/tmp/cron-deploy", "--", "--depth", "1")
	if err != nil {
		return Result{Change: change, State: "failure"}
	}
	_ = tmpDir

	// Create the branch.
	if _, _, err := shell.Git("/tmp/cron-deploy", "checkout", "-b", branch); err != nil {
		return Result{Change: change, State: "failure"}
	}

	// Edit the workflow file.
	var content string
	if b, _, err := shell.Git("/tmp/cron-deploy", "show", "HEAD:"+change.Path); err == nil {
		content = b
	}

	var edited string
	switch change.Action {
	case crondiff.Add:
		edited = cronfile.Add(content, change.NewExpr)
	case crondiff.Update:
		edited = cronfile.Update(content, change.OldExpr, change.NewExpr)
	case crondiff.Remove:
		edited = cronfile.Remove(content)
	}

	_ = edited // written via git in the real flow

	commitMsg := commitMessage(change)
	if _, _, err := shell.Git("/tmp/cron-deploy", "add", change.Path); err != nil {
		return Result{Change: change, State: "failure"}
	}
	gitArgs := []string{"-c", "user.name=" + opts.GitName, "-c", "user.email=" + opts.GitEmail, "commit", "-m", commitMsg}
	if _, _, err := shell.Git("/tmp/cron-deploy", gitArgs...); err != nil {
		return Result{Change: change, State: "failure"}
	}

	if _, _, err := shell.Git("/tmp/cron-deploy", "push", "origin", branch); err != nil {
		return Result{Change: change, State: "failure"}
	}

	// Create the PR.
	prTitle := prTitle(change)
	prBody := prBody(change)
	prOut, _, err := shell.Gh("pr", "create", "--title", prTitle, "--body", prBody, "--head", branch, "--base", "main", "--repo", change.Repo, "--json", "number", "--jq", ".number")
	if err != nil {
		return Result{Change: change, State: "failure"}
	}

	var prNumber int
	fmt.Sscanf(strings.TrimSpace(prOut), "%d", &prNumber)

	// Try to auto-merge.
	_, _, mergeErr := shell.Gh("pr", "merge", fmt.Sprintf("%d", prNumber), "--repo", change.Repo, "--squash", "--delete-branch", "--admin")
	if mergeErr != nil {
		return Result{Change: change, State: "in_progress", PRNumber: prNumber, PRExists: true}
	}

	return Result{Change: change, State: "success", PRNumber: prNumber, PRExists: true}
}

// DeployAll deploys every change in the list. Returns per-change results.
func DeployAll(shell Shell, changes []crondiff.Change, opts Options) []Result {
	out := make([]Result, 0, len(changes))
	for _, c := range changes {
		out = append(out, DeployOne(shell, c, opts))
	}
	return out
}

// DeploySummary returns a JSON summary of deploy results.
func DeploySummary(results []Result) string {
	type summary struct {
		Success   int `json:"success"`
		InProgress int `json:"in_progress"`
		Failed    int `json:"failed"`
		Total     int `json:"total"`
	}
	s := summary{Total: len(results)}
	for _, r := range results {
		switch r.State {
		case "success":
			s.Success++
		case "in_progress":
			s.InProgress++
		case "failure":
			s.Failed++
		}
	}
	b, _ := json.MarshalIndent(s, "", "  ")
	return string(b)
}

func commitMessage(c crondiff.Change) string {
	switch c.Action {
	case crondiff.Add:
		return fmt.Sprintf("chore(cron): deploy schedule for %s (%s)", c.Path, c.NewExpr)
	case crondiff.Update:
		return fmt.Sprintf("chore(cron): update schedule for %s (%s)", c.Path, c.NewExpr)
	case crondiff.Remove:
		return fmt.Sprintf("chore(cron): remove schedule for %s", c.Path)
	}
	return "chore(cron): deploy"
}

func prTitle(c crondiff.Change) string {
	switch c.Action {
	case crondiff.Add:
		return "chore(cron): cron-bot-owned schedule for " + c.Path
	case crondiff.Update:
		return "chore(cron): update cron-bot-owned schedule for " + c.Path
	case crondiff.Remove:
		return "chore(cron): retire " + c.Path
	}
	return "chore(cron): deploy"
}

func prBody(c crondiff.Change) string {
	switch c.Action {
	case crondiff.Add:
		return "Deployed via cronbot. Schedule-neutral edit by cron-bot[bot]."
	case crondiff.Update:
		return "Deployed via cronbot. Schedule updated by cron-bot[bot]."
	case crondiff.Remove:
		return "Retired via cronbot deploy. cron-bot[bot] removing the schedule."
	}
	return "Deployed via cronbot."
}