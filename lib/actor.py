"""Scheduled-run actor classification by account durability.

The actor of a scheduled run (last_runs.json `actor`) is the ground truth for
who fires a cron. Its durability decides whether the cron needs re-homing:

  bot / service  -> durable automation; leave it.
  deprovisioned  -> EMU-anonymized (former employee); runs have STOPPED.
  human          -> a personal `*_LinkedIn` handle; dies on departure.
  external       -> a live non-LinkedIn account (mirrored upstream); unowned.
  none           -> no scheduled run on record.

Mirrors scripts/cron_owner_burndown.py so the two stay consistent.
"""

import re

# An EMU handle anonymizes to a long hex hash + `_LinkedIn` on deprovision.
ANON_RE = re.compile(r"^[0-9a-f]{20,}_LinkedIn$")

# Non-App bot actors seen firing crons.
BOTS = {"li-auto-merge", "web-flow"}

# Most-fragile first (handy for risk sorting).
ACTOR_CLASS_ORDER = {"deprovisioned": 0, "none": 1, "external": 2,
                     "human": 3, "service": 4, "bot": 5}

# Classes whose crons should be moved onto a durable account.
_NEEDS_REHOME = {"deprovisioned", "human", "external"}

DISPOSITION = {
    "bot": "leave (bot-owned)",
    "service": "leave (svc bot-owned)",
    "deprovisioned": "URGENT re-home (actor deprovisioned)",
    "human": "re-home (personal account)",
    "external": "re-home (external account)",
    "none": "inert (never fired here)",
}


def is_bot(login):
    return bool(login) and (login.endswith("[bot]") or login in BOTS)


def is_service(login):
    return bool(login) and (login.startswith("svc-") or login.startswith("svc_"))


def actor_class(login):
    """Bucket a scheduled-run actor login by account durability."""
    if not login:
        return "none"
    if is_bot(login):
        return "bot"
    if is_service(login):
        return "service"
    if ANON_RE.match(login):
        return "deprovisioned"
    if login.endswith("_LinkedIn"):
        return "human"
    return "external"


def needs_rehome(cls):
    """True if a cron fired by an actor of this class should be re-homed."""
    return cls in _NEEDS_REHOME


def disposition(cls):
    """Recommended action text for a cron given its run-actor class."""
    return DISPOSITION[cls]
