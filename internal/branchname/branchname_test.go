package branchname

import "testing"

func TestDeploy(t *testing.T) {
	cases := []struct {
		repo, path, want string
	}{
		{"octo-org/foo", ".github/workflows/nightly.yml", "cron-bot/deploy-octo-org-foo-github-workflows-nightly-yml"},
		{"acme/web", ".github/workflows/sub/cleanup.yaml", "cron-bot/deploy-acme-web-github-workflows-sub-cleanup-yaml"},
	}
	for _, c := range cases {
		if got := Deploy(c.repo, c.path); got != c.want {
			t.Errorf("Deploy(%q, %q) = %q, want %q", c.repo, c.path, got, c.want)
		}
	}
}

func TestDeployDeterministic(t *testing.T) {
	a := Deploy("octo-org/foo", ".github/workflows/x.yml")
	b := Deploy("octo-org/foo", ".github/workflows/x.yml")
	if a != b {
		t.Fatalf("same input should yield same branch: %q != %q", a, b)
	}
}

func TestDeployDifferentPathsDifferentBranches(t *testing.T) {
	a := Deploy("octo-org/foo", ".github/workflows/x.yml")
	b := Deploy("octo-org/foo", ".github/workflows/y.yml")
	if a == b {
		t.Fatalf("different paths should yield different branches: %q == %q", a, b)
	}
}