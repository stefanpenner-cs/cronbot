"""Tests for fix-cron/lib/cron_equiv.py (written first, TDD).

cron_equiv.rewrite(expr) returns a DIFFERENT 5-field cron string that fires at
exactly the same times — the safe "real edit" that re-attributes a scheduled
workflow's actor without changing when it runs.

Key safety rule: cron treats day-of-month and day-of-week as a UNION when BOTH
are restricted. So we must never expand `*` -> full-range on dom while dow is
restricted (or vice-versa); that would silently turn e.g. "Mondays" into "daily".
"""

import os
import sys

import pytest

sys.path.insert(0, os.path.join(os.path.dirname(__file__)))

import cron_equiv as ce


def test_daily_expands_a_star_field():
    # minute/hour are fixed; month `*` is the first always-safe star to expand.
    assert ce.rewrite("0 9 * * *") == "0 9 * 1-12 *"


def test_monthly_dom_expands_month():
    assert ce.rewrite("0 9 1 * *") == "0 9 1 1-12 *"


def test_weekly_dow_expands_month_not_dom():
    # dow is restricted -> dom `*` must NOT be expanded (would become daily).
    out = ce.rewrite("0 9 * * 1")
    assert out == "0 9 * 1-12 1"
    assert out.split()[2] == "*"  # dom untouched


def test_dom_restricted_dow_star_not_expanded():
    # dom restricted -> dow `*` must NOT be expanded (would become daily).
    out = ce.rewrite("0 9 5 6 *")
    assert out.split()[4] == "*"          # dow untouched
    assert out != "0 9 5 6 *"             # but something changed


def test_every_15_min_expands_hour():
    # minute is `*/15` (not a plain star); hour `*` is the first plain star.
    assert ce.rewrite("*/15 * * * *") == "*/15 0-23 * * *"


def test_comma_fallback_reorders_list():
    # No safely-expandable plain star (dow `*` unsafe: dom & month restricted),
    # so fall back to reordering a comma list.
    assert ce.rewrite("0 9,17 1 1 *") == "0 17,9 1 1 *"


def test_fully_specified_uses_single_value_range():
    # No star, no comma -> turn the first single numeric field into N-N.
    assert ce.rewrite("30 4 1 1 0") == "30-30 4 1 1 0"


def test_result_always_differs_and_has_five_fields():
    for expr in ["0 9 * * *", "0 9 1 * *", "0 9 * * 1",
                 "*/15 * * * *", "0 9,17 1 1 *", "30 4 1 1 0"]:
        out = ce.rewrite(expr)
        assert out != expr
        assert len(out.split()) == 5


def test_whitespace_is_normalized():
    assert ce.rewrite("0   9   *  * *") == "0 9 * 1-12 *"


def test_non_five_field_returns_none():
    assert ce.rewrite("not a cron") is None
    assert ce.rewrite("* * * *") is None
    assert ce.rewrite("") is None
