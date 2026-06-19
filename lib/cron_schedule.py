"""Cron cadence + staleness helpers.

Extracted from scripts/cron_owner_burndown.py so the deadman check and the
re-home planner share one source of truth. Behavior is intentionally identical
to the burndown's cron_interval_days / firing_label / run_health.

A cron is judged "stale" relative to its OWN cadence: it has missed roughly
three expected fires (floored at 14 days so a sub-daily cron doesn't trip on a
few missed hours).
"""

# Floor so sub-daily crons don't flap on a few missed intervals.
STALE_FLOOR_DAYS = 14

# Missed-fire multiple at which a cron is considered stale.
STALE_INTERVALS = 3


def interval_days(expr):
    """Rough expected days between fires for a 5-field cron expression.

    Unparseable / non-5-field input falls back to 1.0 day (the burndown's
    default), which is conservative for staleness.
    """
    parts = expr.split()
    if len(parts) != 5:
        return 1.0
    minute, hour, dom, _month, dow = parts
    if dom != "*" and dow == "*":                 # specific day-of-month
        if dom.startswith("*/") and dom[2:].isdigit():
            return float(dom[2:])                 # every N days
        return 30.4                               # monthly
    if dow != "*":                                # specific weekday(s)
        if "-" in dow and "," not in dow:
            a, b = dow.split("-", 1)
            if a.isdigit() and b.isdigit():
                return 7.0 / max(1, abs(int(b) - int(a)) + 1)
        return 7.0 / (dow.count(",") + 1)
    if hour == "*":                               # hourly / sub-hourly
        if minute.startswith("*/") and minute[2:].isdigit():
            return int(minute[2:]) / 1440.0
        return 1 / 24.0
    if hour.startswith("*/") and hour[2:].isdigit():
        return int(hour[2:]) / 24.0
    return 1.0 / (hour.count(",") + 1)            # daily / N-times-daily


def stale_threshold_days(expr):
    """Days-since-last-fire beyond which a cron with this cadence is stale."""
    return max(STALE_FLOOR_DAYS, STALE_INTERVALS * interval_days(expr))


def firing_label(days_since, expr):
    """'never', 'stale Nd' (missed ~3 intervals), or 'Nd ago'.

    days_since is None when the workflow has no recorded scheduled run.
    """
    if days_since is None:
        return "never"
    threshold = stale_threshold_days(expr)
    return f"stale {days_since}d" if days_since > threshold else f"{days_since}d ago"


def run_health(firing):
    """Map a firing_label string to a coarse health bucket."""
    if firing == "never":
        return "never_fired"
    if firing.startswith("stale"):
        return "stale"
    return "firing"


def health(days_since, expr):
    """Convenience: days_since + expr -> health bucket in one call."""
    return run_health(firing_label(days_since, expr))
