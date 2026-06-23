package deploy

import (
	"strings"
	"testing"

	"cronbot/internal/crondiff"
)

// fakeShell records calls and scripts responses.
type fakeShell struct {
	gitCalls  []string
	ghCalls   []string
	gitErr    map[string]bool // key = first arg
	ghErr     map[string]bool // key = first arg
	ghStdout  map[string]string
}

func (f *fakeShell) Git(dir string, args ...string) (string, string, error) {
	f.gitCalls = append(f.gitCalls, strings.Join(args, " "))
	key := args[0]
	if f.gitErr[key] {
		return "", "", errFake("git " + key + " failed")
	}
	return "", "", nil
}

func (f *fakeShell) Gh(args ...string) (string, string, error) {
	f.ghCalls = append(f.ghCalls, strings.Join(args, " "))
	key := args[0]
	if f.ghErr[key] {
		return "", "", errFake("gh " + key + " failed")
	}
	return f.ghStdout[key], "", nil
}

type errFake string

func (e errFake) Error() string { return string(e) }

func TestDeployOneAddSuccess(t *testing.T) {
	f := &fakeShell{
		ghStdout: map[string]string{
			"pr": "42",
		},
	}
	change := crondiff.Change{
		Action:  crondiff.Add,
		Repo:    "octo-org/foo",
		Path:    ".github/workflows/x.yml",
		Branch:  "cron-bot/deploy-test",
		NewExpr: "0 9 * * *",
	}
	res := DeployOne(f, change, Options{GitName: "cron-bot[bot]", GitEmail: "cron-bot[bot]@x"})
	if res.State != "success" {
		t.Fatalf("want success, got %s", res.State)
	}
	if res.PRNumber != 42 {
		t.Fatalf("want PR 42, got %d", res.PRNumber)
	}
}

func TestDeployOneMergeBlocked(t *testing.T) {
	// callCountShell returns responses by gh call index:
	// 0: repo clone (success), 1: pr create (success, returns "42"), 2: pr merge (fails)
	f := &callCountShell{
		ghStdoutByCall: []string{"", "42", ""},
		ghErrByCall:    []bool{false, false, true},
	}

	change := crondiff.Change{
		Action:  crondiff.Add,
		Repo:    "octo-org/foo",
		Path:    ".github/workflows/x.yml",
		Branch:  "cron-bot/deploy-test",
		NewExpr: "0 9 * * *",
	}
	res := DeployOne(f, change, Options{})
	if res.State != "in_progress" {
		t.Fatalf("want in_progress, got %s", res.State)
	}
	if res.PRNumber != 42 {
		t.Fatalf("want PR 42, got %d", res.PRNumber)
	}
}

func TestDeployOneCloneFails(t *testing.T) {
	f := &fakeShell{
		ghErr: map[string]bool{"repo": true},
	}
	change := crondiff.Change{
		Action: crondiff.Add,
		Repo:   "octo-org/foo",
		Path:   ".github/workflows/x.yml",
	}
	res := DeployOne(f, change, Options{})
	if res.State != "failure" {
		t.Fatalf("want failure, got %s", res.State)
	}
}

func TestCommitMessage(t *testing.T) {
	tests := []struct {
		action crondiff.Action
		want   string
	}{
		{crondiff.Add, "chore(cron): deploy schedule for p (e)"},
		{crondiff.Update, "chore(cron): update schedule for p (e)"},
		{crondiff.Remove, "chore(cron): remove schedule for p"},
	}
	for _, tc := range tests {
		c := crondiff.Change{Action: tc.action, Path: "p", NewExpr: "e"}
		if got := commitMessage(c); got != tc.want {
			t.Errorf("commitMessage(%s) = %q, want %q", tc.action, got, tc.want)
		}
	}
}

func TestPRTitle(t *testing.T) {
	c := crondiff.Change{Action: crondiff.Remove, Path: "p"}
	if got := prTitle(c); !strings.Contains(got, "retire") {
		t.Fatalf("prTitle for remove should contain 'retire': %q", got)
	}
}

func TestDeploySummary(t *testing.T) {
	results := []Result{
		{State: "success"},
		{State: "success"},
		{State: "in_progress"},
		{State: "failure"},
	}
	summary := DeploySummary(results)
	if !strings.Contains(summary, `"success": 2`) {
		t.Fatalf("summary: %s", summary)
	}
	if !strings.Contains(summary, `"failed": 1`) {
		t.Fatalf("summary: %s", summary)
	}
	if !strings.Contains(summary, `"total": 4`) {
		t.Fatalf("summary: %s", summary)
	}
}

// callCountShell returns responses by call index.
type callCountShell struct {
	gitCalls      int
	ghCalls      int
	ghStdoutByCall []string
	ghErrByCall    []bool
}

func (c *callCountShell) Git(dir string, args ...string) (string, string, error) {
	c.gitCalls++
	return "", "", nil
}

func (c *callCountShell) Gh(args ...string) (string, string, error) {
	idx := c.ghCalls
	c.ghCalls++
	var stdout string
	var err error
	if idx < len(c.ghStdoutByCall) {
		stdout = c.ghStdoutByCall[idx]
	}
	if idx < len(c.ghErrByCall) && c.ghErrByCall[idx] {
		err = errFake("failed")
	}
	return stdout, "", err
}