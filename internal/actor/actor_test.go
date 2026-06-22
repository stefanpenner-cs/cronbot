package actor

import "testing"

func TestClass(t *testing.T) {
	cases := []struct {
		login string
		want  string
	}{
		{"some-app[bot]", "bot"},
		{"web-flow", "bot"},
		{"svc-sast-dms_EMU", "service"},
		{"svc_foo", "service"},
		{"a1b2c3d4e5f600112233_EMU", "deprovisioned"},
		{"ccarini_EMU", "human"},
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

func TestSetEMUSuffix(t *testing.T) {
	defer SetEMUSuffix("_EMU") // restore the default for other tests
	SetEMUSuffix("_acme")
	if got := Class("jdoe_acme"); got != "human" {
		t.Errorf("with suffix _acme, Class(jdoe_acme) = %q, want human", got)
	}
	if got := Class("jdoe_EMU"); got != "external" {
		t.Errorf("after switching suffix, _EMU is external, got %q", got)
	}
	if got := Class("a1b2c3d4e5f600112233_acme"); got != "deprovisioned" {
		t.Errorf("anon hash + new suffix should be deprovisioned, got %q", got)
	}
}
