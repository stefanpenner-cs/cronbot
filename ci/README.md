# Enterprise rollout — cronlint

How to make cronlint an enterprise-wide check.

## Where each piece lives

- **The tool (cronlint)** → one shared repo, e.g. `linkedin-actions/cron-policy`.
  Published once. Every org points at it. Do not copy it per repo.
- **The registry data** → in each repo: `.github/cron-registry.txt`.
  The cron and its registry line change in the same PR, reviewed by the
  repo's team.
- **The enforcement** → an org-level ruleset that requires the workflow.

So: one tool repo, many small registry files, one ruleset per org.

## The three files here

- `action.yml` — the composite action. The actual lint logic.
  Builds and runs `cmd/cronlint` over the calling repo.
- `required-workflow.yml` — a standalone PR workflow. Orgs register THIS as a
  required workflow. GitHub injects it into every targeted repo. The repo adds
  nothing.
- `org-ruleset.example.json` — the org ruleset that makes the workflow required.
- `cron-registry.example.txt` — a sample registry file for a repo.

## Rollout steps

1. Publish this `fix-cron/` module as a repo, e.g. `linkedin-actions/cron-policy`,
   tagged `v1`. Put `required-workflow.yml` at
   `.github/workflows/cron-policy.yml` in that repo.
2. In each org, create the ruleset from `org-ruleset.example.json`:
   - set `repository_id` to the numeric id of the `cron-policy` repo
   - `~ALL` repos, `~DEFAULT_BRANCH` branch
   - `POST /orgs/{org}/rulesets`
3. Repos add `.github/cron-registry.txt` and list their owned crons.
4. New unregistered crons now fail the required check → cannot merge.

## Enterprise-wide note

GitHub has no single switch that forces one workflow on every repo in the
whole enterprise. Required workflows are set per ORG (via rulesets). So
"enterprise-wide" = apply the same ruleset to every org. That is scriptable:
loop the orgs, POST the same ruleset to each. Enterprise rulesets can broadcast
branch rules, but the required-workflow rule is configured at org scope.

## Policy options

- Lenient (default): `ban-all: "false"` — a cron must be registered.
- Strict: `ban-all: "true"` — no crons at all, except allow-listed files.
- Allow-list: pass `allow` globs (e.g. `vendor/**`) for vendored/mirrored
  workflows you do not own.

## Honest limit

This blocks unregistered/new crons. It does NOT set the cron's owner — that is
decided at merge time by who pushes the change. Pair this with a li-cron
bot-merge (use `cronlint --list-touched` to find cron PRs) so the bot is the
actor, and keep `cmd/deadman` + `cmd/rehome` as the backstop.
