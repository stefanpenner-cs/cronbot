"""Tests for fix-cron/lib/cron_schedule.py.

Parity cases mirror scripts/cron_owner_burndown.py's cron_interval_days /
firing_label / run_health so the extracted lib stays behaviorally identical.
"""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__)))

import cron_schedule as cs


# --- interval_days ---------------------------------------------------------

def test_interval_daily():
    assert cs.interval_days("0 9 * * *") == 1.0


def test_interval_twice_daily():
    # two hours listed -> half a day between fires
    assert cs.interval_days("0 9,17 * * *") == 0.5


def test_interval_hourly():
    assert cs.interval_days("0 * * * *") == 1 / 24.0


def test_interval_every_n_minutes():
    assert cs.interval_days("*/15 * * * *") == 15 / 1440.0


def test_interval_every_n_hours():
    assert cs.interval_days("0 */6 * * *") == 6 / 24.0


def test_interval_weekly_single_dow():
    assert cs.interval_days("0 9 * * 1") == 7.0


def test_interval_weekday_range():
    # Mon-Fri -> 5 fires per week
    assert cs.interval_days("0 9 * * 1-5") == 7.0 / 5


def test_interval_dow_list():
    assert cs.interval_days("0 9 * * 1,4") == 7.0 / 2


def test_interval_monthly_dom():
    assert cs.interval_days("0 9 1 * *") == 30.4


def test_interval_every_n_days_dom():
    assert cs.interval_days("0 9 */3 * *") == 3.0


def test_interval_bad_input_defaults_one_day():
    assert cs.interval_days("not a cron") == 1.0
    assert cs.interval_days("") == 1.0


# --- firing_label / health -------------------------------------------------

def test_never_when_days_none():
    assert cs.firing_label(None, "0 9 * * *") == "never"
    assert cs.health(None, "0 9 * * *") == "never_fired"


def test_recent_daily_is_firing():
    assert cs.firing_label(2, "0 9 * * *") == "2d ago"
    assert cs.health(2, "0 9 * * *") == "firing"


def test_daily_stale_floor_is_14_days():
    # daily cadence -> 3*1=3 but floored at 14
    assert cs.stale_threshold_days("0 9 * * *") == 14
    assert cs.health(15, "0 9 * * *") == "stale"
    assert cs.health(14, "0 9 * * *") == "firing"


def test_monthly_threshold_uses_three_intervals():
    # monthly cadence 30.4 -> 3*30.4 = 91.2 dominates the 14d floor
    assert cs.stale_threshold_days("0 9 1 * *") == 3 * 30.4
    assert cs.health(60, "0 9 1 * *") == "firing"
    assert cs.health(100, "0 9 1 * *") == "stale"
