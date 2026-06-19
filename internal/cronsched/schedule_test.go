package cronsched

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestIntervalDays(t *testing.T) {
	cases := []struct {
		expr string
		want float64
	}{
		{"0 9 * * *", 1.0},        // daily
		{"0 9,17 * * *", 0.5},     // twice daily
		{"0 * * * *", 1.0 / 24.0}, // hourly
		{"*/15 * * * *", 15.0 / 1440.0},
		{"0 */6 * * *", 6.0 / 24.0},
		{"0 9 * * 1", 7.0},         // weekly
		{"0 9 * * 1-5", 7.0 / 5.0}, // Mon-Fri
		{"0 9 * * 1,4", 7.0 / 2.0},
		{"0 9 1 * *", 30.4},  // monthly
		{"0 9 */3 * *", 3.0}, // every 3 days
		{"not a cron", 1.0},
		{"", 1.0},
	}
	for _, c := range cases {
		if got := IntervalDays(c.expr); !approx(got, c.want) {
			t.Errorf("IntervalDays(%q) = %v, want %v", c.expr, got, c.want)
		}
	}
}

func TestNeverWhenNoRun(t *testing.T) {
	if got := FiringLabel(0, false, "0 9 * * *"); got != "never" {
		t.Errorf("got %q, want never", got)
	}
	if got := Health(0, false, "0 9 * * *"); got != "never_fired" {
		t.Errorf("got %q, want never_fired", got)
	}
}

func TestRecentDailyIsFiring(t *testing.T) {
	if got := FiringLabel(2, true, "0 9 * * *"); got != "2d ago" {
		t.Errorf("got %q, want '2d ago'", got)
	}
	if got := Health(2, true, "0 9 * * *"); got != "firing" {
		t.Errorf("got %q, want firing", got)
	}
}

func TestDailyStaleFloorIs14Days(t *testing.T) {
	if got := StaleThresholdDays("0 9 * * *"); !approx(got, 14) {
		t.Errorf("threshold = %v, want 14", got)
	}
	if got := Health(15, true, "0 9 * * *"); got != "stale" {
		t.Errorf("15d -> %q, want stale", got)
	}
	if got := Health(14, true, "0 9 * * *"); got != "firing" {
		t.Errorf("14d -> %q, want firing", got)
	}
}

func TestMonthlyThresholdUsesThreeIntervals(t *testing.T) {
	if got := StaleThresholdDays("0 9 1 * *"); !approx(got, 3*30.4) {
		t.Errorf("threshold = %v, want %v", got, 3*30.4)
	}
	if got := Health(60, true, "0 9 1 * *"); got != "firing" {
		t.Errorf("60d -> %q, want firing", got)
	}
	if got := Health(100, true, "0 9 1 * *"); got != "stale" {
		t.Errorf("100d -> %q, want stale", got)
	}
}
