package crondiff

import (
	"strings"
	"testing"

	"cronbot/internal/registry"
)

func entry(repo, path, expr string) registry.Entry {
	return registry.Entry{
		Repo: repo, Path: path, Expr: expr,
		OwnerTeam: "cron-reviewers",
	}
}

func TestDiffEmptyToEmpty(t *testing.T) {
	if got := Diff(nil, nil); len(got) != 0 {
		t.Fatalf("empty diff should be empty, got %#v", got)
	}
}

func TestDiffAdd(t *testing.T) {
	old := []registry.Entry{}
	new := []registry.Entry{entry("octo-org/foo", ".github/workflows/nightly.yml", "0 9 * * *")}
	got := Diff(old, new)
	if len(got) != 1 || got[0].Action != Add {
		t.Fatalf("want 1 add, got %#v", got)
	}
	if got[0].NewExpr != "0 9 * * *" {
		t.Fatalf("new expr: %q", got[0].NewExpr)
	}
}

func TestDiffRemove(t *testing.T) {
	old := []registry.Entry{entry("octo-org/foo", ".github/workflows/nightly.yml", "0 9 * * *")}
	new := []registry.Entry{}
	got := Diff(old, new)
	if len(got) != 1 || got[0].Action != Remove {
		t.Fatalf("want 1 remove, got %#v", got)
	}
	if got[0].OldExpr != "0 9 * * *" {
		t.Fatalf("old expr: %q", got[0].OldExpr)
	}
}

func TestDiffUpdate(t *testing.T) {
	old := []registry.Entry{entry("octo-org/foo", ".github/workflows/nightly.yml", "0 9 * * *")}
	new := []registry.Entry{entry("octo-org/foo", ".github/workflows/nightly.yml", "0 10 * * *")}
	got := Diff(old, new)
	if len(got) != 1 || got[0].Action != Update {
		t.Fatalf("want 1 update, got %#v", got)
	}
	if got[0].OldExpr != "0 9 * * *" || got[0].NewExpr != "0 10 * * *" {
		t.Fatalf("old=%q new=%q", got[0].OldExpr, got[0].NewExpr)
	}
}

func TestDiffUnchangedIsNoChange(t *testing.T) {
	e := entry("octo-org/foo", ".github/workflows/nightly.yml", "0 9 * * *")
	got := Diff([]registry.Entry{e}, []registry.Entry{e})
	if len(got) != 0 {
		t.Fatalf("unchanged should be no change, got %#v", got)
	}
}

func TestDiffMultipleChangesSorted(t *testing.T) {
	old := []registry.Entry{
		entry("a/repo", ".github/workflows/x.yml", "0 9 * * *"),
		entry("b/repo", ".github/workflows/y.yml", "0 10 * * *"),
	}
	new := []registry.Entry{
		entry("a/repo", ".github/workflows/x.yml", "0 11 * * *"), // update
		entry("c/repo", ".github/workflows/z.yml", "0 12 * * *"), // add
	}
	got := Diff(old, new)
	if len(got) != 3 {
		t.Fatalf("want 3 changes, got %d: %#v", len(got), got)
	}
	// adds sort first (alphabetically), then removes, then updates
	if got[0].Action != Add {
		t.Fatalf("first should be add, got %s", got[0].Action)
	}
}

func TestPlanMarkdownEmpty(t *testing.T) {
	if got := PlanMarkdown(nil); !strings.Contains(got, "No registry changes") {
		t.Fatalf("empty plan: %q", got)
	}
}

func TestPlanMarkdownHasAdd(t *testing.T) {
	changes := []Change{{
		Action: Add,
		Repo:   "octo-org/foo",
		Path:   ".github/workflows/nightly.yml",
		NewExpr: "0 9 * * *",
	}}
	md := PlanMarkdown(changes)
	if !strings.Contains(md, "➕ Added") {
		t.Fatalf("missing Added header: %q", md)
	}
	if !strings.Contains(md, "octo-org/foo") {
		t.Fatalf("missing repo: %q", md)
	}
	if !strings.Contains(md, "0 9 * * *") {
		t.Fatalf("missing expr: %q", md)
	}
}

func TestPlanMarkdownHasRemove(t *testing.T) {
	changes := []Change{{
		Action:  Remove,
		Repo:    "octo-org/foo",
		Path:    ".github/workflows/nightly.yml",
		OldExpr: "0 9 * * *",
	}}
	md := PlanMarkdown(changes)
	if !strings.Contains(md, "🗑️ Removed") {
		t.Fatalf("missing Removed header: %q", md)
	}
}

func TestPlanMarkdownHasUpdate(t *testing.T) {
	changes := []Change{{
		Action:  Update,
		Repo:    "octo-org/foo",
		Path:    ".github/workflows/nightly.yml",
		OldExpr: "0 9 * * *",
		NewExpr: "0 10 * * *",
	}}
	md := PlanMarkdown(changes)
	if !strings.Contains(md, "✏️ Changed") {
		t.Fatalf("missing Changed header: %q", md)
	}
	if !strings.Contains(md, "0 9 * * *") || !strings.Contains(md, "0 10 * * *") {
		t.Fatalf("missing exprs: %q", md)
	}
}