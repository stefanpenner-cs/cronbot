# Enterprise rollout — li-cron cron policy

How to make crons managed and durable across the enterprise.

## The model

- Humans never merge cron changes directly.
- A required check (the identity gate) blocks any human cron change.
- To get a cron, you file an issue. The crew signs off. li-cron[bot] lands it.
- Because the bot does the merge, the bot owns the cron. It is durable.
- A central registry records every managed cron.

## Where each piece lives

- **The tool + registry** → one shared repo, e.g. `linkedin-actions/cron-policy`.
  Holds the Go module, the issue form, the workflows, and `registry.json`.
- **The identity-gate check** → registered as a required workflow per org.
- **The approval gate** → a GitHub Environment (`cron-approval`) whose required
  reviewers are the crew that releases ci-workflows/ci-actions.

## The files here

- `ISSUE_TEMPLATE/cron-request.yml` — the request form. Field labels match what
  cronbot parses.
- `intake.yml` — the intake workflow. Validates the request, waits for crew
  sign-off (environment gate), updates the registry, and lands the change as
  the li-cron App.
- `required-workflow.yml` — the identity gate. Orgs register THIS as a required
  workflow. It fails any PR that adds/changes a cron unless the author is
  li-cron[bot].
- `org-ruleset.example.json` — the org ruleset that makes the check required.
- `action.yml`, `cron-registry.example.txt` — the older registry-based lint.
  Optional. Keep it if you also want a per-repo allow-list.

## The flow

```
person files "cron-request" issue
   -> intake.yml
        validate : cronbot checks the request, comments back
        provision: WAIT for crew sign-off (cron-approval environment)
                   cronbot updates registry.json
                   li-cron App lands the cron in the target repo (squash merge)
                   close the issue
```

A human who edits a cron directly hits the identity gate and is sent here.

## Rollout steps

1. Publish this module as `linkedin-actions/cron-policy@v1`. Put
   `required-workflow.yml` at `.github/workflows/cron-policy.yml`, the issue
   form at `.github/ISSUE_TEMPLATE/cron-request.yml`, and `intake.yml` at
   `.github/workflows/cron-intake.yml`.
2. Create the `cron-approval` Environment. Set required reviewers = the crew.
3. Create the li-cron GitHub App: contents:write, pull_requests:write,
   workflows:write. Install it org-wide. Give it branch-protection bypass so
   its PRs can merge. Store `LI_CRON_APP_ID` (var) and `LI_CRON_APP_KEY`
   (secret).
4. In each org, create the ruleset from `org-ruleset.example.json` (set
   `repository_id` to the cron-policy repo). `POST /orgs/{org}/rulesets`.
5. Enterprise-wide = apply that ruleset to every org (scriptable loop).

## What is code vs infra

- Code (in this module, tested): cronbot (request -> plan + registry),
  cronguard (the identity gate), the registry catalog.
- Infra (you set up): the li-cron App, the cron-approval Environment, the org
  rulesets.

## Honest limits

- The identity gate uses the PR author as the actor. Good enough; a later pass
  can check the commit author.
- The merge must be squash or merge-commit, never rebase, or the bot does not
  become the actor.
- One landing step in `intake.yml` is a marked TODO: choose to merge the
  developer's existing PR as the App (simplest), or have the App author the
  workflow. Pick one for your environment.
- A repo admin with bypass can still force a cron in. deadman + rehome catch it.
