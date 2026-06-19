#!/usr/bin/env python3
"""Deadman / health check for GHA crons.

Compares each cron's expected cadence against the timestamp of its last ACTUAL
`schedule` run and flags the silent failures: crons that never fired here
(`never_fired`) or that have missed roughly three expected fires (`stale`).

Inputs (produced by the existing pipeline):
  data/cron/linkedin-actions/crons.json      (scripts/cron_inventory.py)
  data/cron/linkedin-actions/last_runs.json  (scripts/cron_last_runs.py)

Output: a JSON list + a console table of missed/dead crons, worst-first. The
emit step is intentionally a single function so a Slack / GitHub-issue sink can
be plugged in later without touching the assessment logic.

Usage:
  python3 fix-cron/deadman/check_crons.py
  python3 fix-cron/deadman/check_crons.py --json-out reports/cron/deadman.json
  python3 fix-cron/deadman/check_crons.py --all        # include healthy crons
"""

import argparse
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "lib"))
import cron_schedule as cs  # noqa: E402

ROOT = Path(__file__).resolve().parents[2]
DATA = ROOT / "data" / "cron" / "linkedin-actions"
DEFAULT_CRONS = DATA / "crons.json"
DEFAULT_LAST_RUNS = DATA / "last_runs.json"

MISSED = ("never_fired", "stale")


def collapse_files(crons):
    """Collapse cron rows to one entry per (repo, path).

    A workflow file can hold several `- cron:` lines; last_runs is per file, so
    health is judged against the file's FASTEST cadence (smallest interval),
    matching scripts/cron_owner_burndown.py.
    """
    files = {}
    for row in crons:
        key = (row["repo"], row["path"])
        expr = row.get("cron_expression", "")
        f = files.get(key)
        if f is None:
            files[key] = {
                "repo": row["repo"],
                "path": row["path"],
                "workflow_name": row.get("workflow_name"),
                "state": row.get("state"),
                "default_branch": row.get("default_branch"),
                "expressions": [expr],
                "fastest_expr": expr,
            }
        else:
            f["expressions"].append(expr)
            if cs.interval_days(expr) < cs.interval_days(f["fastest_expr"]):
                f["fastest_expr"] = expr
    return files


def _days_since(last_run_iso, now):
    if not last_run_iso:
        return None
    dt = datetime.fromisoformat(last_run_iso.replace("Z", "+00:00"))
    return (now - dt).days


def assess(files, last_runs, now):
    """Return an assessment dict per file (all files, every health)."""
    out = []
    for f in files.values():
        evidence = last_runs.get(f"{f['repo']}::{f['path']}") or {}
        last_run = evidence.get("last_run")
        days = _days_since(last_run, now)
        expr = f["fastest_expr"]
        out.append({
            "repo": f["repo"],
            "path": f["path"],
            "workflow_name": f["workflow_name"],
            "state": f["state"],
            "expressions": f["expressions"],
            "fastest_expr": expr,
            "interval_days": round(cs.interval_days(expr), 4),
            "last_run": last_run,
            "days_since": days,
            "run_actor": evidence.get("actor"),
            "url": evidence.get("url"),
            "health": cs.health(days, expr),
        })
    return out


def missed(rows):
    """Filter + sort to silently-failing crons, worst-first.

    never_fired before stale; within stale, the longest-overdue first.
    """
    flagged = [r for r in rows if r["health"] in MISSED]
    return sorted(
        flagged,
        key=lambda r: (0 if r["health"] == "never_fired" else 1,
                       -(r["days_since"] if r["days_since"] is not None else 10 ** 9),
                       r["repo"], r["path"]),
    )


def emit(rows, stream=None):
    """Pluggable alert sink. Today: a console table. (Slack/issue later.)"""
    stream = stream if stream is not None else sys.stdout
    if not rows:
        print("No missed/dead crons. \u2705", file=stream)
        return
    print(f"{len(rows)} missed/dead crons (worst-first):\n", file=stream)
    print(f"{'HEALTH':<11} {'LAST RUN':<11} {'STATE':<9} REPO :: PATH", file=stream)
    print("-" * 80, file=stream)
    for r in rows:
        last = r["last_run"][:10] if r["last_run"] else "never"
        print(f"{r['health']:<11} {last:<11} {str(r['state']):<9} "
              f"{r['repo']} :: {r['path']}", file=stream)


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--crons", type=Path, default=DEFAULT_CRONS)
    ap.add_argument("--last-runs", type=Path, default=DEFAULT_LAST_RUNS)
    ap.add_argument("--json-out", type=Path, help="write the report rows as JSON")
    ap.add_argument("--all", action="store_true",
                    help="include healthy crons in the JSON output")
    args = ap.parse_args(argv)

    crons = json.loads(args.crons.read_text())
    last_runs = json.loads(args.last_runs.read_text())
    now = datetime.now(timezone.utc)

    rows = assess(collapse_files(crons), last_runs, now)
    flagged = missed(rows)

    emit(flagged)

    if args.json_out:
        payload = rows if args.all else flagged
        args.json_out.write_text(json.dumps(payload, indent=2))
        print(f"\nwrote {len(payload)} rows -> {args.json_out}", file=sys.stderr)


if __name__ == "__main__":
    main()
