package recon

import (
	"testing"
)

func TestExtractRepoAndPath(t *testing.T) {
	cases := []struct {
		desc, wantRepo, wantPath string
	}{
		{"octo-org/foo :: .github/workflows/x.yml (add)", "octo-org/foo", ".github/workflows/x.yml"},
		{"acme/web :: .github/workflows/sub/y.yaml (remove)", "acme/web", ".github/workflows/sub/y.yaml"},
		{"bad description", "", ""},
	}
	for _, c := range cases {
		repo, path := ExtractRepoAndPath(c.desc)
		if repo != c.wantRepo || path != c.wantPath {
			t.Errorf("ExtractRepoAndPath(%q) = (%q, %q), want (%q, %q)", c.desc, repo, path, c.wantRepo, c.wantPath)
		}
	}
}

func TestHasInProgress(t *testing.T) {
	if !HasInProgress([]ReconcileResult{{NewState: "in_progress"}}) {
		t.Fatal("should have in_progress")
	}
	if HasInProgress([]ReconcileResult{{NewState: "success"}, {NewState: "failure"}}) {
		t.Fatal("should not have in_progress")
	}
}

func TestAllComplete(t *testing.T) {
	if !AllComplete([]ReconcileResult{{NewState: "success"}, {NewState: "failure"}}) {
		t.Fatal("should be all complete")
	}
	if AllComplete([]ReconcileResult{{NewState: "success"}, {NewState: "in_progress"}}) {
		t.Fatal("should not be all complete")
	}
}

func TestParseTargetPR(t *testing.T) {
	pr, ok := ParseTargetPR(`[{"number":42,"state":"open","merged":false}]`)
	if !ok || pr.Number != 42 || pr.State != "open" || pr.Merged {
		t.Fatalf("unexpected: %+v ok=%v", pr, ok)
	}
}

func TestParseTargetPREmpty(t *testing.T) {
	_, ok := ParseTargetPR(`[]`)
	if ok {
		t.Fatal("empty should return false")
	}
}

func TestParseDeployments(t *testing.T) {
	deploys := ParseDeployments(`[{"id":1,"environment":"cron-deploy","description":"octo-org/foo :: x.yml (add)"}]`)
	if len(deploys) != 1 || deploys[0].ID != 1 {
		t.Fatalf("unexpected: %+v", deploys)
	}
}