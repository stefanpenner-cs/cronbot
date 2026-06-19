"""Tests for fix-cron/deadman/check_crons.py."""

import os
import sys
from datetime import datetime, timezone

sys.path.insert(0, os.path.dirname(__file__))

import check_crons as cc

NOW = datetime(2026, 6, 18, tzinfo=timezone.utc)


def _cron(repo, path, expr, **kw):
    row = {"repo": repo, "path": path, "cron_expression": expr,
           "workflow_name": "WF", "state": "active", "default_branch": "main"}
    row.update(kw)
    return row


def test_collapse_keeps_fastest_cadence():
    crons = [
        _cron("o/r", ".github/workflows/w.yml", "0 9 * * 1"),    # weekly
        _cron("o/r", ".github/workflows/w.yml", "*/10 * * * *"), # every 10 min
    ]
    files = cc.collapse_files(crons)
    f = files[("o/r", ".github/workflows/w.yml")]
    assert f["fastest_expr"] == "*/10 * * * *"
    assert len(f["expressions"]) == 2


def test_recent_daily_is_firing_not_missed():
    crons = [_cron("o/r", "w", "0 9 * * *")]
    last = {"o/r::w": {"last_run": "2026-06-17T09:00:00Z", "actor": "x_LinkedIn",
                       "url": "u"}}
    rows = cc.assess(cc.collapse_files(crons), last, NOW)
    assert rows[0]["health"] == "firing"
    assert cc.missed(rows) == []


def test_stale_daily_is_missed():
    crons = [_cron("o/r", "w", "0 9 * * *")]
    last = {"o/r::w": {"last_run": "2026-05-01T09:00:00Z"}}  # ~48 days
    rows = cc.assess(cc.collapse_files(crons), last, NOW)
    assert rows[0]["health"] == "stale"
    assert len(cc.missed(rows)) == 1


def test_no_run_record_is_never_fired():
    crons = [_cron("o/r", "w", "0 9 * * *")]
    rows = cc.assess(cc.collapse_files(crons), {}, NOW)
    assert rows[0]["health"] == "never_fired"
    assert rows[0]["last_run"] is None
    assert len(cc.missed(rows)) == 1


def test_missed_sorts_never_first_then_overdue_desc():
    crons = [
        _cron("o/a", "w", "0 9 * * *"),
        _cron("o/b", "w", "0 9 * * *"),
        _cron("o/c", "w", "0 9 * * *"),
    ]
    last = {
        "o/a::w": {"last_run": "2026-05-20T09:00:00Z"},   # ~29d stale
        "o/b::w": {"last_run": "2026-03-01T09:00:00Z"},   # ~109d stale (worse)
        # o/c never fired
    }
    rows = cc.assess(cc.collapse_files(crons), last, NOW)
    order = [(r["repo"], r["health"]) for r in cc.missed(rows)]
    assert order[0] == ("o/c", "never_fired")
    assert order[1] == ("o/b", "stale")   # more overdue before less
    assert order[2] == ("o/a", "stale")


def test_emit_handles_empty(capsys):
    cc.emit([])
    assert "No missed/dead crons" in capsys.readouterr().out


def test_emit_lists_rows(capsys):
    crons = [_cron("o/r", ".github/workflows/w.yml", "0 9 * * *")]
    rows = cc.missed(cc.assess(cc.collapse_files(crons), {}, NOW))
    cc.emit(rows)
    out = capsys.readouterr().out
    assert "never_fired" in out
    assert "o/r" in out
