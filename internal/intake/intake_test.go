package intake

import "testing"

const sampleBody = `### Target repository

octo-org/foo

### Workflow path

.github/workflows/nightly.yml

### Cron expression

0 9 * * *

### Justification

nightly backup job
`

func TestParse(t *testing.T) {
	got := Parse(sampleBody)
	want := CronRequest{
		Repo:          "octo-org/foo",
		Path:          ".github/workflows/nightly.yml",
		Expr:          "0 9 * * *",
		Justification: "nightly backup job",
	}
	if got != want {
		t.Fatalf("Parse:\n got %#v\nwant %#v", got, want)
	}
}

func TestParseNoResponseIsEmpty(t *testing.T) {
	body := "### Justification\n\n_No response_\n"
	if got := Parse(body); got.Justification != "" {
		t.Fatalf("_No response_ should be empty, got %q", got.Justification)
	}
}

func TestValidateGood(t *testing.T) {
	if errs := Parse(sampleBody).Validate(); len(errs) != 0 {
		t.Fatalf("valid request should pass, got %v", errs)
	}
}

func TestValidateBad(t *testing.T) {
	req := CronRequest{
		Repo: "not-a-repo",              // no slash
		Path: ".github/workflows/x.txt", // wrong ext
		Expr: "0 9 * *",                 // 4 fields
	}
	if errs := req.Validate(); len(errs) != 3 {
		t.Fatalf("want 3 errors, got %d: %v", len(errs), errs)
	}
}

func TestValidateAllowsNestedWorkflowPaths(t *testing.T) {
	ok := CronRequest{
		Repo: "octo-org/foo",
		Path: ".github/workflows/sub/nightly.yml", // nested path
		Expr: "0 9 * * *",
	}
	if errs := ok.Validate(); len(errs) != 0 {
		t.Fatalf("nested workflow path should be valid, got %v", errs)
	}
}

func TestValidateRemovalRequiresRepoPathReason(t *testing.T) {
	ok := CronRequest{
		Repo: "octo-org/foo", Path: ".github/workflows/nightly.yml",
		Justification: "service retired",
	}
	if errs := ok.ValidateRemoval(); len(errs) != 0 {
		t.Fatalf("valid removal should pass, got %v", errs)
	}
}

func TestValidateRemovalIgnoresExpr(t *testing.T) {
	r := CronRequest{
		Repo: "octo-org/foo", Path: ".github/workflows/nightly.yml",
		Justification: "service retired",
		// no Expr — removal does not need one
	}
	if errs := r.ValidateRemoval(); len(errs) != 0 {
		t.Fatalf("removal must not require a cron expression, got %v", errs)
	}
}

func TestValidateRemovalReportsMissingFields(t *testing.T) {
	r := CronRequest{Repo: "bad", Path: "nope", Justification: ""}
	errs := r.ValidateRemoval()
	if len(errs) != 3 {
		t.Fatalf("want 3 errors (repo, path, reason), got %d: %v", len(errs), errs)
	}
}

func TestParseReasonAliasesJustification(t *testing.T) {
	body := "### Target repository\n\nacme/web\n\n### Workflow path\n\n.github/workflows/nightly.yml\n\n### Reason\n\nService retired.\n"
	req := Parse(body)
	if req.Justification != "Service retired." {
		t.Fatalf("'Reason' should map to Justification, got %q", req.Justification)
	}
}
