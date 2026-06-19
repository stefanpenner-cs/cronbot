"""Tests for fix-cron/lib/actor.py (TDD).

Classifies a scheduled-run actor login by account durability — the signal that
decides whether a cron needs re-homing. Mirrors the proven logic in
scripts/cron_owner_burndown.py (actor_class / is_bot / is_service / ANON_RE).
"""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__)))

import actor


def test_bot_is_durable():
    assert actor.actor_class("li-dep-eng[bot]") == "bot"
    assert actor.actor_class("li-auto-merge") == "bot"
    assert actor.actor_class("web-flow") == "bot"


def test_service_is_durable():
    assert actor.actor_class("svc-sast-dms_LinkedIn") == "service"
    assert actor.actor_class("svc_foo") == "service"


def test_deprovisioned_is_anonymized_hex():
    assert actor.actor_class("a1b2c3d4e5f600112233_LinkedIn") == "deprovisioned"


def test_human_linkedin_handle():
    assert actor.actor_class("ccarini_LinkedIn") == "human"


def test_external_non_linkedin():
    assert actor.actor_class("octocat") == "external"


def test_none_when_missing():
    assert actor.actor_class(None) == "none"
    assert actor.actor_class("") == "none"


def test_needs_rehome_set():
    assert actor.needs_rehome("deprovisioned")
    assert actor.needs_rehome("human")
    assert actor.needs_rehome("external")
    assert not actor.needs_rehome("bot")
    assert not actor.needs_rehome("service")
    assert not actor.needs_rehome("none")


def test_disposition_text():
    assert "URGENT" in actor.disposition("deprovisioned")
    assert actor.disposition("bot").startswith("leave")
    assert actor.disposition("service").startswith("leave")
