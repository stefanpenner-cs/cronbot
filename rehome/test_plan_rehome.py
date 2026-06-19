"""Tests for fix-cron/rehome/plan_rehome.py."""

import os
import sys

sys.path.insert(0, os.path.dirname(__file__))

import plan_rehome as pr


def _cron(repo, path, expr, state="active", **kw):
    row = {"repo": repo, "path": path, "cron_expression": expr, "state": state,
           "workflow_name": "WF", "default_branch": "master", "first_cron_line": 10}
    row.update(kw)
    return row


def test_durable_actors_are_skipped():
    crons = [_cron("o/r", "w", "0 9 * * *")]
    for durable in ("svc-foo_LinkedIn", "li-dep-eng[bot]"):
        last = {"o/r::w": {"actor": durable}}
        assert pr.plan(crons, last) == []


def test_human_actor_is_planned_with_equivalent_rewrite():
    crons = [_cron("o/r", "w", "0 9 * * *")]
    last = {"o/r::w": {"actor": "alice_LinkedIn"}}
    rows = pr.plan(crons, last)
    assert len(rows) == 1
    row = rows[0]
    assert row["actor_class"] == "human"
    assert row["old_expr"] == "0 9 * * *"
    assert row["new_expr"] == "0 9 * 1-12 *"     # schedule-neutral edit
    assert row["can_rewrite"] is True
    assert row["re_enable"] is False


def test_deprovisioned_actor_is_planned():
    crons = [_cron("o/r", "w", "37 6 * * 2")]
    last = {"o/r::w": {"actor": "a1b2c3d4e5f600112233_LinkedIn"}}
    rows = pr.plan(crons, last)
    assert rows[0]["actor_class"] == "deprovisioned"
    assert "URGENT" in rows[0]["disposition"]


def test_external_actor_is_planned():
    crons = [_cron("o/r", "w", "0 9 * * *")]
    last = {"o/r::w": {"actor": "octocat"}}
    assert pr.plan(crons, last)[0]["actor_class"] == "external"


def test_disabled_workflow_flags_re_enable():
    crons = [_cron("o/r", "w", "0 9 * * *", state="disabled_inactivity")]
    last = {"o/r::w": {"actor": "alice_LinkedIn"}}
    assert pr.plan(crons, last)[0]["re_enable"] is True


def test_no_run_record_is_not_planned():
    # actor_class('none') -> not a re-home target (we only move proven runners).
    crons = [_cron("o/r", "w", "0 9 * * *")]
    assert pr.plan(crons, {}) == []


def test_sorted_deprovisioned_before_human():
    crons = [
        _cron("o/h", "w", "0 9 * * *"),
        _cron("o/d", "w", "0 9 * * *"),
    ]
    last = {
        "o/h::w": {"actor": "alice_LinkedIn"},
        "o/d::w": {"actor": "deadbeefdeadbeefdead_LinkedIn"},
    }
    classes = [r["actor_class"] for r in pr.plan(crons, last)]
    assert classes == ["deprovisioned", "human"]


def test_url_points_at_cron_line():
    crons = [_cron("o/r", ".github/workflows/w.yml", "0 9 * * *")]
    last = {"o/r::.github/workflows/w.yml": {"actor": "alice_LinkedIn"}}
    url = pr.plan(crons, last)[0]["url"]
    assert url == "https://github.com/o/r/blob/master/.github/workflows/w.yml#L10"


def test_emit_dry_run_applies_nothing(capsys):
    crons = [_cron("o/r", "w", "0 9 * * *")]
    last = {"o/r::w": {"actor": "alice_LinkedIn"}}
    pr.emit(pr.plan(crons, last))
    out = capsys.readouterr().out
    assert "DRY RUN" in out
    assert "nothing applied" in out
