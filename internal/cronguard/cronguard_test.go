package cronguard

import (
	"bytes"
	"strings"
	"testing"
)

const cronA = "on:\n  schedule:\n    - cron: '0 9 * * *'\n"
const cronB = "on:\n  schedule:\n    - cron: '0 10 * * *'\n"

func TestNewCronByHumanFails(t *testing.T) {
	diffs := []FileDiff{{Path: "w.yml", Base: "", Head: cronA}}
	v := Guard(diffs, "alice_EMU", nil)
	if len(v) != 1 || v[0].Expr != "0 9 * * *" {
		t.Fatalf("human adding a cron should fail, got %#v", v)
	}
}

func TestNewCronByBotPasses(t *testing.T) {
	diffs := []FileDiff{{Path: "w.yml", Base: "", Head: cronA}}
	if v := Guard(diffs, "cron-bot[bot]", nil); len(v) != 0 {
		t.Fatalf("bot adding a cron should pass, got %#v", v)
	}
}

func TestChangedValueByHumanFails(t *testing.T) {
	diffs := []FileDiff{{Path: "w.yml", Base: cronA, Head: cronB}}
	v := Guard(diffs, "alice_EMU", nil)
	// A value change is an add of the new value plus a removal of the old one;
	// both are blocked for a human.
	got := map[string]bool{}
	for _, x := range v {
		got[x.Expr] = true
	}
	if len(v) != 2 || !got["0 9 * * *"] || !got["0 10 * * *"] {
		t.Fatalf("changing a cron value should flag old and new, got %#v", v)
	}
}

func TestRemovedCronByHumanFails(t *testing.T) {
	diffs := []FileDiff{{Path: "w.yml", Base: cronA, Head: "on:\n  push: {}\n"}}
	v := Guard(diffs, "alice_EMU", nil)
	if len(v) != 1 || v[0].Expr != "0 9 * * *" {
		t.Fatalf("a human removing a cron should be blocked, got %#v", v)
	}
	if !strings.Contains(v[0].Message, "cron-removal request") {
		t.Fatalf("removal message should point to the removal request, got %q", v[0].Message)
	}
}

func TestUnchangedCronPasses(t *testing.T) {
	diffs := []FileDiff{{Path: "w.yml", Base: cronA, Head: cronA}}
	if v := Guard(diffs, "alice_EMU", nil); len(v) != 0 {
		t.Fatalf("unchanged cron should pass, got %#v", v)
	}
}

func TestAllowedActorCaseInsensitive(t *testing.T) {
	diffs := []FileDiff{{Path: "w.yml", Base: "", Head: cronA}}
	if v := Guard(diffs, "CRON-BOT[BOT]", nil); len(v) != 0 {
		t.Fatalf("allowed-actor match should be case-insensitive, got %#v", v)
	}
}

func TestRemovingCronByHumanFails(t *testing.T) {
	diffs := []FileDiff{{Path: "w.yml", Base: cronA, Head: ""}}
	if v := Guard(diffs, "alice_EMU", nil); len(v) != 1 {
		t.Fatalf("a human deleting a cron file should be blocked, got %#v", v)
	}
}

func TestRemovingCronByBotPasses(t *testing.T) {
	// The bot retires crons via the gated removal flow, so its deletes must pass.
	diffs := []FileDiff{{Path: "w.yml", Base: cronA, Head: ""}}
	if v := Guard(diffs, "cron-bot[bot]", nil); len(v) != 0 {
		t.Fatalf("the bot removing a cron should pass, got %#v", v)
	}
}

func TestEmitEmptyIsClean(t *testing.T) {
	var b bytes.Buffer
	if n := Emit(nil, &b); n != 0 {
		t.Fatalf("want 0, got %d", n)
	}
	if !strings.Contains(b.String(), "No unauthorized cron changes") {
		t.Fatalf("unexpected output: %q", b.String())
	}
}

func TestEmitReportsViolations(t *testing.T) {
	var b bytes.Buffer
	v := Guard([]FileDiff{{Path: "w.yml", Base: "", Head: cronA}}, "alice_EMU", nil)
	if n := Emit(v, &b); n != 1 {
		t.Fatalf("want 1, got %d", n)
	}
	out := b.String()
	if !strings.Contains(out, "w.yml") || !strings.Contains(out, "0 9 * * *") {
		t.Fatalf("violation detail missing: %q", out)
	}
}
