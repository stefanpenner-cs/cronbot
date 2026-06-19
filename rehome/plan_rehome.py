#!/usr/bin/env python3
"""Dry-run re-home planner for GHA crons.

A scheduled workflow's actor is whoever last changed the `cron:` VALUE. When
that actor is a departing/departed human, an external account, or already
deprovisioned, the cron is fragile or dead. Re-homing moves it onto a durable
service/bot account by making a schedule-neutral edit to the expression.

This tool is DRY-RUN ONLY: it reads the inventory, picks the crons whose real
run-actor needs re-homing, and emits the exact planned edit for each
(old -> equivalent new expression, plus whether the workflow must be
re-enabled). It applies nothing — no commits, no pushes, no API writes.

Inputs:
  data/cron/linkedin-actions/crons.json      (scripts/cron_inventory.py)
  data/cron/linkedin-actions/last_runs.json  (scripts/cron_last_runs.py)

Usage:
  python3 fix-cron/rehome/plan_rehome.py
  python3 fix-cron/rehome/plan_rehome.py --json-out reports/cron/rehome_plan.json
"""

import argparse
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "lib"))
import actor as actor_lib  # noqa: E402
import cron_equiv  # noqa: E402

ROOT = Path(__file__).resolve().parents[2]
DATA = ROOT / "data" / "cron" / "linkedin-actions"
DEFAULT_CRONS = DATA / "crons.json"
DEFAULT_LAST_RUNS = DATA / "last_runs.json"

# How a re-home edit must land to re-attribute the actor (cron-debugging
# findings): the new actor is the merger on a squash/merge-commit or the pusher
# on a direct push, but the ORIGINAL author on a rebase merge.
MERGE_METHOD = "squash or merge-commit (never rebase-merge)"
TARGET_ACTOR_HINT = "a durable service/bot account (svc-* or li-cron[bot])"


def _run_actor(repo, path, last_runs):
    return (last_runs.get(f"{repo}::{path}") or {}).get("actor")


def plan(crons, last_runs):
    """Return one dry-run plan entry per cron expression that needs re-homing."""
    out = []
    for row in crons:
        run_actor = _run_actor(row["repo"], row["path"], last_runs)
        cls = actor_lib.actor_class(run_actor)
        if not actor_lib.needs_rehome(cls):
            continue
        old_expr = row.get("cron_expression", "")
        new_expr = cron_equiv.rewrite(old_expr)
        state = row.get("state")
        branch = row.get("default_branch") or "master"
        line = row.get("first_cron_line")
        url = f"https://github.com/{row['repo']}/blob/{branch}/{row['path']}"
        if line:
            url += f"#L{line}"
        out.append({
            "repo": row["repo"],
            "path": row["path"],
            "workflow_name": row.get("workflow_name"),
            "state": state,
            "first_cron_line": line,
            "run_actor": run_actor,
            "actor_class": cls,
            "disposition": actor_lib.disposition(cls),
            "old_expr": old_expr,
            "new_expr": new_expr,
            "can_rewrite": new_expr is not None,
            "re_enable": state is not None and state != "active",
            "merge_method": MERGE_METHOD,
            "target_actor": TARGET_ACTOR_HINT,
            "url": url,
        })
    out.sort(key=lambda r: (actor_lib.ACTOR_CLASS_ORDER[r["actor_class"]],
                            r["repo"], r["path"]))
    return out


def emit(rows, stream=None):
    """Console summary of the dry-run plan. Applies nothing."""
    stream = stream if stream is not None else sys.stdout
    if not rows:
        print("No crons need re-homing. \u2705", file=stream)
        return
    print(f"DRY RUN — {len(rows)} crons to re-home (nothing applied), "
          "worst-first:\n", file=stream)
    for r in rows:
        flag = "" if r["can_rewrite"] else "  [NO SAFE REWRITE]"
        reenable = "  +re-enable" if r["re_enable"] else ""
        print(f"[{r['actor_class']}] {r['repo']} :: {r['path']}{flag}{reenable}",
              file=stream)
        print(f"    actor={r['run_actor']}  state={r['state']}", file=stream)
        print(f"    cron: '{r['old_expr']}'  ->  '{r['new_expr']}'", file=stream)
    print(f"\nLand each edit via {MERGE_METHOD} as {TARGET_ACTOR_HINT}.",
          file=stream)


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--crons", type=Path, default=DEFAULT_CRONS)
    ap.add_argument("--last-runs", type=Path, default=DEFAULT_LAST_RUNS)
    ap.add_argument("--json-out", type=Path, help="write the plan as JSON")
    args = ap.parse_args(argv)

    crons = json.loads(args.crons.read_text())
    last_runs = json.loads(args.last_runs.read_text())

    rows = plan(crons, last_runs)
    emit(rows)

    if args.json_out:
        args.json_out.write_text(json.dumps(rows, indent=2))
        print(f"\nwrote {len(rows)} plan rows -> {args.json_out}", file=sys.stderr)


if __name__ == "__main__":
    main()
