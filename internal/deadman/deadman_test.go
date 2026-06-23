package deadman

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"cronbot/internal/inventory"
)

var now = time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC)

func cron(repo, path, expr string) inventory.Cron {
	return inventory.Cron{Repo: repo, Path: path, CronExpression: expr,
		WorkflowName: "WF", State: "active", DefaultBranch: "main"}
}

func TestCollapseKeepsFastestCadence(t *testing.T) {
	files := CollapseFiles([]inventory.Cron{
		cron("o/r", ".github/workflows/w.yml", "0 9 * * 1"),    // weekly
		cron("o/r", ".github/workflows/w.yml", "*/10 * * * *"), // every 10 min
	})
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	if files[0].FastestExpr != "*/10 * * * *" {
		t.Errorf("fastest = %q, want '*/10 * * * *'", files[0].FastestExpr)
	}
	if len(files[0].Expressions) != 2 {
		t.Errorf("got %d expressions, want 2", len(files[0].Expressions))
	}
}

func TestRecentDailyIsFiringNotMissed(t *testing.T) {
	files := CollapseFiles([]inventory.Cron{cron("o/r", "w", "0 9 * * *")})
	last := map[string]inventory.RunEvidence{
		"o/r::w": {LastRun: "2026-06-17T09:00:00Z", Actor: "x_EMU", URL: "u"},
	}
	rows := Assess(files, last, now)
	if rows[0].Health != "firing" {
		t.Errorf("health = %q, want firing", rows[0].Health)
	}
	if len(Missed(rows)) != 0 {
		t.Errorf("expected no missed rows")
	}
}

func TestStaleDailyIsMissed(t *testing.T) {
	files := CollapseFiles([]inventory.Cron{cron("o/r", "w", "0 9 * * *")})
	last := map[string]inventory.RunEvidence{"o/r::w": {LastRun: "2026-05-01T09:00:00Z"}}
	rows := Assess(files, last, now)
	if rows[0].Health != "stale" {
		t.Errorf("health = %q, want stale", rows[0].Health)
	}
	if len(Missed(rows)) != 1 {
		t.Errorf("expected 1 missed row")
	}
}

func TestNoRunRecordIsNeverFired(t *testing.T) {
	files := CollapseFiles([]inventory.Cron{cron("o/r", "w", "0 9 * * *")})
	rows := Assess(files, map[string]inventory.RunEvidence{}, now)
	if rows[0].Health != "never_fired" {
		t.Errorf("health = %q, want never_fired", rows[0].Health)
	}
	if rows[0].DaysSince != nil {
		t.Errorf("DaysSince = %v, want nil", *rows[0].DaysSince)
	}
	if len(Missed(rows)) != 1 {
		t.Errorf("expected 1 missed row")
	}
}

func TestMissedSortsNeverFirstThenOverdueDesc(t *testing.T) {
	files := CollapseFiles([]inventory.Cron{
		cron("o/a", "w", "0 9 * * *"),
		cron("o/b", "w", "0 9 * * *"),
		cron("o/c", "w", "0 9 * * *"),
	})
	last := map[string]inventory.RunEvidence{
		"o/a::w": {LastRun: "2026-05-20T09:00:00Z"}, // ~29d
		"o/b::w": {LastRun: "2026-03-01T09:00:00Z"}, // ~109d (worse)
		// o/c never fired
	}
	got := Missed(Assess(files, last, now))
	want := []struct{ repo, health string }{
		{"o/c", "never_fired"},
		{"o/b", "stale"},
		{"o/a", "stale"},
	}
	for i, w := range want {
		if got[i].Repo != w.repo || got[i].Health != w.health {
			t.Errorf("row %d = (%s,%s), want (%s,%s)", i, got[i].Repo, got[i].Health, w.repo, w.health)
		}
	}
}

func TestEmitHandlesEmpty(t *testing.T) {
	var b bytes.Buffer
	Emit(nil, &b)
	if !strings.Contains(b.String(), "No missed/dead crons") {
		t.Errorf("got %q", b.String())
	}
}

func TestEmitListsRows(t *testing.T) {
	files := CollapseFiles([]inventory.Cron{cron("o/r", ".github/workflows/w.yml", "0 9 * * *")})
	rows := Missed(Assess(files, map[string]inventory.RunEvidence{}, now))
	var b bytes.Buffer
	Emit(rows, &b)
	out := b.String()
	if !strings.Contains(out, "never_fired") || !strings.Contains(out, "o/r") {
		t.Errorf("got %q", out)
	}
}
