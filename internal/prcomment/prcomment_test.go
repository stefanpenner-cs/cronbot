package prcomment

import (
	"strings"
	"testing"

	"cronbot/internal/crondiff"
)

func TestPRCheckInvalid(t *testing.T) {
	got := PRCheck(false, "", "target repository \"bad\" must be owner/name")
	if !strings.Contains(got, "❌") {
		t.Fatalf("missing ❌: %q", got)
	}
	if !strings.Contains(got, "must be owner/name") {
		t.Fatalf("missing errors: %q", got)
	}
}

func TestPRCheckValid(t *testing.T) {
	got := PRCheck(true, "## 📋 Deploy Plan\n\n### ➕ Added\n- octo-org/foo", "")
	if !strings.Contains(got, "✅") {
		t.Fatalf("missing ✅: %q", got)
	}
	if !strings.Contains(got, "Deploy Plan") {
		t.Fatalf("missing plan: %q", got)
	}
	if !strings.Contains(got, "Merge this PR") {
		t.Fatalf("missing merge hint: %q", got)
	}
}

func TestDeploySuccess(t *testing.T) {
	got := DeploySuccess()
	if !strings.Contains(got, "✅") || !strings.Contains(got, "cron-bot[bot]") {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestDeployBlocked(t *testing.T) {
	got := DeployBlocked()
	if !strings.Contains(got, "🟡") || !strings.Contains(got, "deployments tab") {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestTokenNeeded(t *testing.T) {
	got := TokenNeeded()
	if !strings.Contains(got, "CRON_APP_ID") {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestReconcileDone(t *testing.T) {
	got := ReconcileDone()
	if !strings.Contains(got, "✅") || !strings.Contains(got, "durable") {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestReconcileFailed(t *testing.T) {
	got := ReconcileFailed([]string{"octo-org/foo", "acme/web"})
	if !strings.Contains(got, "❌") {
		t.Fatalf("missing ❌: %q", got)
	}
	if !strings.Contains(got, "octo-org/foo") || !strings.Contains(got, "acme/web") {
		t.Fatalf("missing repos: %q", got)
	}
}

func TestDeployProgress(t *testing.T) {
	changes := []crondiff.Change{
		{Action: crondiff.Add, Repo: "octo-org/foo", Path: ".github/workflows/x.yml"},
		{Action: crondiff.Remove, Repo: "acme/web", Path: ".github/workflows/y.yml"},
	}
	got := DeployProgress(changes)
	if !strings.Contains(got, "🚀") {
		t.Fatalf("missing 🚀: %q", got)
	}
	if !strings.Contains(got, "octo-org/foo") || !strings.Contains(got, "acme/web") {
		t.Fatalf("missing repos: %q", got)
	}
	if !strings.Contains(got, "[ ] Add cron") || !strings.Contains(got, "[ ] Remove cron") {
		t.Fatalf("missing checklist items: %q", got)
	}
}