// Package cronsched computes a cron's rough cadence and judges staleness.
//
// Behavior mirrors scripts/cron_owner_burndown.py (cron_interval_days /
// firing_label / run_health) so the deadman check and the re-home planner
// share one source of truth.
//
// A cron is "stale" relative to its OWN cadence: it has missed roughly three
// expected fires, floored at 14 days so a sub-daily cron does not trip on a
// few missed hours.
package cronsched

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// StaleFloorDays keeps sub-daily crons from flapping on a few missed fires.
	StaleFloorDays = 14.0
	// StaleIntervals is the missed-fire multiple at which a cron is stale.
	StaleIntervals = 3.0
)

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// IntervalDays returns the rough expected days between fires for a 5-field
// cron expression. Unparseable or non-5-field input falls back to 1.0 day.
func IntervalDays(expr string) float64 {
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return 1.0
	}
	minute, hour, dom, dow := parts[0], parts[1], parts[2], parts[4]

	if dom != "*" && dow == "*" { // specific day-of-month
		if rest, ok := strings.CutPrefix(dom, "*/"); ok && allDigits(rest) {
			n, _ := strconv.Atoi(rest)
			return float64(n) // every N days
		}
		return 30.4 // monthly
	}
	if dow != "*" { // specific weekday(s)
		if strings.Contains(dow, "-") && !strings.Contains(dow, ",") {
			a, b, _ := strings.Cut(dow, "-")
			if allDigits(a) && allDigits(b) {
				ai, _ := strconv.Atoi(a)
				bi, _ := strconv.Atoi(b)
				span := bi - ai
				if span < 0 {
					span = -span
				}
				return 7.0 / float64(max(1, span+1))
			}
		}
		return 7.0 / float64(strings.Count(dow, ",")+1)
	}
	if hour == "*" { // hourly / sub-hourly
		if rest, ok := strings.CutPrefix(minute, "*/"); ok && allDigits(rest) {
			n, _ := strconv.Atoi(rest)
			return float64(n) / 1440.0
		}
		return 1.0 / 24.0
	}
	if rest, ok := strings.CutPrefix(hour, "*/"); ok && allDigits(rest) {
		n, _ := strconv.Atoi(rest)
		return float64(n) / 24.0
	}
	return 1.0 / float64(strings.Count(hour, ",")+1) // daily / N-times-daily
}

// StaleThresholdDays is the days-since-last-fire beyond which a cron with this
// cadence is considered stale.
func StaleThresholdDays(expr string) float64 {
	return max(StaleFloorDays, StaleIntervals*IntervalDays(expr))
}

// FiringLabel returns "never", "stale Nd", or "Nd ago". hasRun is false when
// the workflow has no recorded scheduled run.
func FiringLabel(daysSince int, hasRun bool, expr string) string {
	if !hasRun {
		return "never"
	}
	if float64(daysSince) > StaleThresholdDays(expr) {
		return fmt.Sprintf("stale %dd", daysSince)
	}
	return fmt.Sprintf("%dd ago", daysSince)
}

// RunHealth maps a FiringLabel string to a coarse health bucket.
func RunHealth(firing string) string {
	switch {
	case firing == "never":
		return "never_fired"
	case strings.HasPrefix(firing, "stale"):
		return "stale"
	default:
		return "firing"
	}
}

// Health is a convenience: daysSince + expr -> health bucket in one call.
func Health(daysSince int, hasRun bool, expr string) string {
	return RunHealth(FiringLabel(daysSince, hasRun, expr))
}
