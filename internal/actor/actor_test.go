package actor

import "testing"

func TestClass(t *testing.T) {
	cases := []struct {
		login string
		want  string
	}{
		{"li-dep-eng[bot]", "bot"},
		{"li-auto-merge", "bot"},
		{"web-flow", "bot"},
		{"svc-sast-dms_LinkedIn", "service"},
		{"svc_foo", "service"},
		{"a1b2c3d4e5f600112233_LinkedIn", "deprovisioned"},
		{"ccarini_LinkedIn", "human"},
		{"octocat", "external"},
		{"", "none"},
	}
	for _, c := range cases {
		if got := Class(c.login); got != c.want {
			t.Errorf("Class(%q) = %q, want %q", c.login, got, c.want)
		}
	}
}

func TestNeedsRehome(t *testing.T) {
	for _, c := range []string{"deprovisioned", "human", "external"} {
		if !NeedsRehome(c) {
			t.Errorf("NeedsRehome(%q) = false, want true", c)
		}
	}
	for _, c := range []string{"bot", "service", "none"} {
		if NeedsRehome(c) {
			t.Errorf("NeedsRehome(%q) = true, want false", c)
		}
	}
}

func TestDisposition(t *testing.T) {
	if got := Disposition("deprovisioned"); got == "" || got[:6] != "URGENT" {
		t.Errorf("deprovisioned disposition = %q, want URGENT...", got)
	}
	if got := Disposition("bot"); got[:5] != "leave" {
		t.Errorf("bot disposition = %q, want leave...", got)
	}
}
