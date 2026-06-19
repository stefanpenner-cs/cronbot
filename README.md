# fix-cron — fix GHA cron rot

Two small, tested tools plus this design doc.

They make GitHub Actions crons durable and observable.

Dry-run only. Nothing here writes to any repo.

## The problem

GHA crons rot in four ways:

- They are hard to maintain.
- The cron "actor" is just the last person to merge a cron-syntax change.
- That person leaves the company, and the cron silently stops.
- There is no alerting, no deadman switch, no health check.

## Root cause (already proven)

Source: `github.com/stefanpenner-cs/cron-debugging`, `../who_owns_cron.md`.

- A scheduled workflow runs as the push actor of the last change to a
  `cron:` value.
  - Squash / merge-commit → the merger.
  - Rebase merge → the original commit author.
  - Direct push → the pusher.
- Only a deleted or deprovisioned (EMU-anonymized) actor stops crons.
  Removing the user from the org does not.
- A comment or whitespace edit does not move the actor. The `cron:`
  value itself must change.
- A cron change does not re-enable a disabled workflow.

## The fix: two pillars

### Pillar A — durable ownership (`rehome/`)

Move a fragile cron onto a durable service/bot account.

The trick: edit the `cron:` value in a way that does not change when it
runs. `lib/cron_equiv.py` rewrites the expression to an equal-but-different
string (for example `0 9 * * *` → `0 9 * 1-12 *`). That real edit moves the
actor; the schedule stays identical.

`rehome/plan_rehome.py`:

- Reads the cron inventory and the last actual `schedule` run per file.
- Picks crons whose run-actor needs re-homing
  (deprovisioned, personal, or external).
- Emits the exact planned edit: old → new expression, and whether the
  workflow must be re-enabled.
- Sorts worst-first (deprovisioned before human).
- Dry-run only. No commits, no pushes, no API writes.

To actually move the actor later, land each edit as a durable account
(svc-* or `li-cron[bot]`) via squash or merge-commit. Never rebase-merge —
that keeps the old author.

### Pillar B — observability (`deadman/`)

Catch crons that have silently stopped.

`deadman/check_crons.py`:

- Reads the inventory and the last `schedule` run per file.
- Compares expected cadence (from the cron expression) against days since
  the last real run.
- Flags two failure states:
  - `never_fired` — no scheduled run on record.
  - `stale` — missed roughly three expected fires (floored at 14 days).
- Prints a worst-first table and can write JSON.
- The alert step is one `emit()` function, so a Slack or GitHub-issue sink
  can drop in later.

## Data flow

Both tools reuse the existing pipeline. They fetch nothing new.

```
scripts/cron_inventory.py   ->  data/cron/linkedin-actions/crons.json
scripts/cron_last_runs.py   ->  data/cron/linkedin-actions/last_runs.json
                                        |
                 +----------------------+----------------------+
                 v                                             v
        deadman/check_crons.py                        rehome/plan_rehome.py
        (missed / dead report)                        (dry-run re-home plan)
```

## Layout

```
fix-cron/
  lib/
    cron_schedule.py   cadence + staleness (interval_days, firing_label, health)
    cron_equiv.py      schedule-neutral cron rewriter
    actor.py           run-actor durability classifier
  deadman/
    check_crons.py     deadman / health check
  rehome/
    plan_rehome.py     dry-run re-home planner
```

Each module has a sibling `test_*.py`. Python + pytest (repo standard).

## Run it

```
python3 fix-cron/deadman/check_crons.py
python3 fix-cron/deadman/check_crons.py --json-out reports/cron/deadman.json

python3 fix-cron/rehome/plan_rehome.py
python3 fix-cron/rehome/plan_rehome.py --json-out reports/cron/rehome_plan.json

pytest fix-cron/
```

## Safety

- Re-home is dry-run. It only prints the planned edit.
- The rewrite never changes when a cron fires. It only changes the string.
- Day-of-month and day-of-week are a union when both are set, so the
  rewriter never expands one while the other is restricted.

## Future work (out of scope for this prototype)

- Apply mode: open a draft PR per repo for the re-home edit.
- Prevention: a CI lint that flags new human-owned crons at PR time.
- Real alert sinks: Slack, GitHub issues.
- A cron registry: expected cadence and owning team as the source of truth.
