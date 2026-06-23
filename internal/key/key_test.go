package key

import "testing"

func TestCron(t *testing.T) {
	if got := Cron("o/r", ".github/workflows/x.yml"); got != "o/r::.github/workflows/x.yml" {
		t.Fatalf("Cron = %q", got)
	}
}
