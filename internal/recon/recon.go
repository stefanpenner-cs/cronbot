// Package recon orchestrates the reconcile loop: find in-progress deployments,
// check if their target PRs merged, update deployment status + PR labels.
package recon

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"cronbot/internal/branchname"
)

// Shell is the same interface as deploy.Shell — git + gh commands.
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

// Deployment is a GitHub deployment record relevant to reconcile.
type Deployment struct {
	ID          int64  `json:"id"`
	Environment string `json:"environment"`
	Description string `json:"description"`
}

// DeploymentStatus is the latest status of a deployment.
type DeploymentStatus struct {
	State       string `json:"state"`
	Description string `json:"description"`
}

// TargetPR is a deploy PR in a target repo.
type TargetPR struct {
	Number int  `json:"number"`
	State  string `json:"state"`
	Merged bool   `json:"merged"`
}

// ReconcileResult is the outcome of checking one deployment.
type ReconcileResult struct {
	DeploymentID int64  `json:"deployment_id"`
	TargetRepo   string `json:"target_repo"`
	TargetPR     int    `json:"target_pr"`
	OldState     string `json:"old_state"`
	NewState     string `json:"new_state"`
}

// ParseDeployments parses gh api output for cron-deploy deployments.
func ParseDeployments(jsonOut string) []Deployment {
	var deploys []Deployment
	dec := json.NewDecoder(strings.NewReader(jsonOut))
	dec.Decode(&deploys)
	return deploys
}

// ParseDeploymentStatus parses the first status from a statuses list.
func ParseDeploymentStatus(jsonOut string) DeploymentStatus {
	var statuses []DeploymentStatus
	dec := json.NewDecoder(strings.NewReader(jsonOut))
	dec.Decode(&statuses)
	if len(statuses) > 0 {
		return statuses[0]
	}
	return DeploymentStatus{State: "unknown"}
}

// ParseTargetPR parses gh pr list --json output (first element).
func ParseTargetPR(jsonOut string) (TargetPR, bool) {
	var prs []TargetPR
	dec := json.NewDecoder(strings.NewReader(jsonOut))
	dec.Decode(&prs)
	if len(prs) > 0 {
		return prs[0], true
	}
	return TargetPR{}, false
}

// ExtractRepoAndPath parses a deployment description like
// "octo-org/foo :: .github/workflows/x.yml (add)" into repo and path.
func ExtractRepoAndPath(desc string) (repo, path string) {
	parts := strings.SplitN(desc, " :: ", 2)
	if len(parts) < 2 {
		return "", ""
	}
	repo = parts[0]
	rest := parts[1]
	if i := strings.LastIndex(rest, " ("); i >= 0 {
		path = rest[:i]
	} else {
		path = rest
	}
	return repo, strings.TrimSpace(path)
}

// CheckOne inspects one deployment: finds the target PR, checks if it merged,
// returns the result (caller updates the deployment status + labels).
func CheckOne(shell Shell, repo string, deploy Deployment) ReconcileResult {
	result := ReconcileResult{
		DeploymentID: deploy.ID,
		TargetRepo:   "",
		NewState:     "",
	}

	// Get latest status.
	statusOut, _, _ := shell.Gh("api", fmt.Sprintf("repos/%s/deployments/%d/statuses", repo, deploy.ID), "--jq", ".[0].state")
	result.OldState = strings.TrimSpace(statusOut)
	if result.OldState != "in_progress" {
		result.NewState = result.OldState
		return result
	}

	// Parse target repo + path from description.
	targetRepo, targetPath := ExtractRepoAndPath(deploy.Description)
	result.TargetRepo = targetRepo

	// Compute branch name.
	branch := branchname.Deploy(targetRepo, targetPath)

	// Find the PR in the target repo.
	prOut, _, err := shell.Gh("pr", "list", "--repo", targetRepo, "--head", branch, "--state", "all", "--json", "state,number,merged")
	if err != nil {
		result.NewState = "unknown"
		return result
	}

	pr, found := ParseTargetPR(prOut)
	if !found {
		result.NewState = "unknown"
		return result
	}

	result.TargetPR = pr.Number

	if pr.Merged {
		result.NewState = "success"
	} else if pr.State == "closed" {
		result.NewState = "failure"
	} else {
		result.NewState = "in_progress"
	}

	return result
}

// ReconcileAll checks all in-progress deployments and returns results.
func ReconcileAll(shell Shell, repo string) []ReconcileResult {
	// Get all cron-deploy deployments.
	deployOut, _, _ := shell.Gh("api", fmt.Sprintf("repos/%s/deployments", repo), "--paginate", "--jq", `[.[] | select(.environment=="cron-deploy")]`)
	deploys := ParseDeployments(deployOut)

	var results []ReconcileResult
	for _, d := range deploys {
		result := CheckOne(shell, repo, d)
		if result.OldState == "in_progress" && result.NewState != "in_progress" {
			// Update the deployment status.
			shell.Gh("api", fmt.Sprintf("repos/%s/deployments/%d/statuses", repo, d.ID), "--method", "POST",
				"-f", "state="+result.NewState,
				"-f", "description="+result.NewState)
		}
		results = append(results, result)
	}

	return results
}

// HasInProgress returns true if any result is still in_progress.
func HasInProgress(results []ReconcileResult) bool {
	for _, r := range results {
		if r.NewState == "in_progress" {
			return true
		}
	}
	return false
}

// AllComplete returns true if all results are success or failure.
func AllComplete(results []ReconcileResult) bool {
	for _, r := range results {
		if r.NewState == "in_progress" || r.NewState == "unknown" {
			return false
		}
	}
	return true
}