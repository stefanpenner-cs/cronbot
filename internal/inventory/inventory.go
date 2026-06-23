// Package inventory defines the on-disk shapes produced by the existing cron
// pipeline (scripts/cron_inventory.py, scripts/cron_last_runs.py) and helpers
// to load them.
package inventory

import (
	"encoding/json"
	"os"

	"cronbot/internal/key"
)

// Cron is one row of crons.json (one per cron expression). Only the fields the
// cronbot tools use are modeled; extra JSON keys are ignored.
type Cron struct {
	Repo           string `json:"repo"`
	Path           string `json:"path"`
	CronExpression string `json:"cron_expression"`
	WorkflowName   string `json:"workflow_name"`
	State          string `json:"state"`
	DefaultBranch  string `json:"default_branch"`
	FirstCronLine  int    `json:"first_cron_line"`
}

// RunEvidence is one value of last_runs.json, keyed "repo::path". Error entries
// (which carry only an "error" key) unmarshal to a zero value, i.e. no run.
type RunEvidence struct {
	Actor      string `json:"actor"`
	Conclusion string `json:"conclusion"`
	LastRun    string `json:"last_run"`
	URL        string `json:"url"`
}

// Key is the last_runs.json map key for a workflow file.
func Key(repo, path string) string { return key.Cron(repo, path) }

// LoadCrons reads and parses a crons.json file.
func LoadCrons(path string) ([]Cron, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []Cron
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// LoadLastRuns reads and parses a last_runs.json file.
func LoadLastRuns(path string) (map[string]RunEvidence, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]RunEvidence{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}
