package rehome

import (
	"bytes"
	"strings"
	"testing"

	"cronbot/internal/inventory"
)

func cron(repo, path, expr, state string) inventory.Cron {
	return inventory.Cron{Repo: repo, Path: path, CronExpression: expr, State: state,
		WorkflowName: "WF", DefaultBranch: "master", FirstCronLine: 10}
}

func runs(pairs map[string]string) map[string]inventory.RunEvidence {
	m := map[string]inventory.RunEvidence{}
	for k, actorLogin := range pairs {
		m[k] = inventory.RunEvidence{Actor: actorLogin}
	}
	return m
}

func TestDurableActorsAreSkipped(t *testing.T) {
	crons := []inventory.Cron{cron("o/r", "w", "0 9 * * *", "active")}
	for _, durable := range []string{"svc-foo_EMU", "some-app[bot]"} {
		got := Plan(crons, runs(map[string]string{"o/r::w": durable}))
		if len(got) != 0 {
			t.Errorf("actor %q produced %d entries, want 0", durable, len(got))
		}
	}
}

func TestHumanActorIsPlannedWithEquivalentRewrite(t *testing.T) {
	crons := []inventory.Cron{cron("o/r", "w", "0 9 * * *", "active")}
	got := Plan(crons, runs(map[string]string{"o/r::w": "alice_EMU"}))
	if len(got) != 1 {
		t.Fatalf("got %d entries, want 1", len(got))
	}
	e := got[0]
	if e.ActorClass != "human" {
		t.Errorf("class = %q, want human", e.ActorClass)
	}
	if e.OldExpr != "0 9 * * *" || e.NewExpr != "0 9 * 1-12 *" {
		t.Errorf("rewrite %q -> %q, want '0 9 * * *' -> '0 9 * 1-12 *'", e.OldExpr, e.NewExpr)
	}
	if !e.CanRewrite || e.ReEnable {
		t.Errorf("CanRewrite=%v ReEnable=%v, want true/false", e.CanRewrite, e.ReEnable)
	}
}

func TestDeprovisionedActorIsPlanned(t *testing.T) {
	crons := []inventory.Cron{cron("o/r", "w", "37 6 * * 2", "active")}
	got := Plan(crons, runs(map[string]string{"o/r::w": "a1b2c3d4e5f600112233_EMU"}))
	if got[0].ActorClass != "deprovisioned" {
		t.Errorf("class = %q, want deprovisioned", got[0].ActorClass)
	}
	if !strings.Contains(got[0].Disposition, "URGENT") {
		t.Errorf("disposition = %q, want URGENT", got[0].Disposition)
	}
}

func TestExternalActorIsPlanned(t *testing.T) {
	crons := []inventory.Cron{cron("o/r", "w", "0 9 * * *", "active")}
	got := Plan(crons, runs(map[string]string{"o/r::w": "octocat"}))
	if got[0].ActorClass != "external" {
		t.Errorf("class = %q, want external", got[0].ActorClass)
	}
}

func TestDisabledWorkflowFlagsReEnable(t *testing.T) {
	crons := []inventory.Cron{cron("o/r", "w", "0 9 * * *", "disabled_inactivity")}
	got := Plan(crons, runs(map[string]string{"o/r::w": "alice_EMU"}))
	if !got[0].ReEnable {
		t.Errorf("ReEnable = false, want true")
	}
}

func TestNoRunRecordIsNotPlanned(t *testing.T) {
	crons := []inventory.Cron{cron("o/r", "w", "0 9 * * *", "active")}
	if got := Plan(crons, map[string]inventory.RunEvidence{}); len(got) != 0 {
		t.Errorf("got %d entries, want 0", len(got))
	}
}

func TestSortedDeprovisionedBeforeHuman(t *testing.T) {
	crons := []inventory.Cron{
		cron("o/h", "w", "0 9 * * *", "active"),
		cron("o/d", "w", "0 9 * * *", "active"),
	}
	got := Plan(crons, runs(map[string]string{
		"o/h::w": "alice_EMU",
		"o/d::w": "deadbeefdeadbeefdead_EMU",
	}))
	if got[0].ActorClass != "deprovisioned" || got[1].ActorClass != "human" {
		t.Errorf("order = [%s, %s], want [deprovisioned, human]", got[0].ActorClass, got[1].ActorClass)
	}
}

func TestURLPointsAtCronLine(t *testing.T) {
	crons := []inventory.Cron{cron("o/r", ".github/workflows/w.yml", "0 9 * * *", "active")}
	got := Plan(crons, runs(map[string]string{"o/r::.github/workflows/w.yml": "alice_EMU"}))
	want := "https://github.com/o/r/blob/master/.github/workflows/w.yml#L10"
	if got[0].URL != want {
		t.Errorf("URL = %q, want %q", got[0].URL, want)
	}
}

func TestEmitDryRunAppliesNothing(t *testing.T) {
	crons := []inventory.Cron{cron("o/r", "w", "0 9 * * *", "active")}
	rows := Plan(crons, runs(map[string]string{"o/r::w": "alice_EMU"}))
	var b bytes.Buffer
	Emit(rows, &b)
	out := b.String()
	if !strings.Contains(out, "DRY RUN") || !strings.Contains(out, "nothing applied") {
		t.Errorf("got %q", out)
	}
}
