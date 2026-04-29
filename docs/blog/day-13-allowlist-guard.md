# Day 13: An Empty String Is a Wildcard — Closing the AKS Allowlist Hole

*Posted April 28, 2026 · Karl Kuhnhausen*

---

[Day 12](day-12-status-badge-bug.md) ended with a confession. While chasing a session-status bug, I discovered the deploy pipeline had silently opened the AKS API server to the entire Internet. The cause was a single line of YAML:

```yaml
- name: Remove runner IP from AKS API server allowlist
  if: always()
  run: |
    az aks update \
      --api-server-authorized-ip-ranges "${{ secrets.ADMIN_IP_RANGES }}"
```

If the `ADMIN_IP_RANGES` secret is ever empty or unset, that template substitutes to `--api-server-authorized-ip-ranges ""`, and Azure CLI interprets the empty string as **"remove all restrictions."** Every CI run is one missing-secret away from publishing the cluster's control plane to the world.

I tied off the immediate symptom that day — repopulated the secret, restored my home IP — and put the structural fix on the next sprint's list. This is that fix. PR [#25](https://github.com/karlkuhnhausen/f1-race-intelligence/pull/25).

---

## What "next sprint" actually means

I want to flag something about how this kind of follow-through tends to play out, because I think it's a place where AI-assisted development genuinely changes the shape of the work.

A "structural fix on the next sprint's list" is the kind of TODO that, in a normal codebase, accumulates dust. The site is back up. The secret is repopulated. The pager isn't going off. There's a feature waiting. So the hardening sits in a notebook somewhere until the next time the same bug bites — except this time at 3 AM on a weekend, with an Internet-exposed API server and a junior on call.

The way I worked through this fix was different. I asked the agent "is the AKS allowlist issue still on the board?", and the answer was a five-step plan with line numbers, file paths, and an unprompted observation that **the cleanup-step guard alone wasn't sufficient.** That observation — Lesson 2 below — was not in my notes. I'd have implemented the partial fix, shipped it, and never noticed that the *complementary* failure mode was still wide open. The agent caught it during planning, before I touched a file.

There's a tendency in AI-skeptic discourse to focus on whether the model can write a function. That's the wrong frame. The interesting question is whether the model can hold the entire risk surface of a change in working memory and surface the things you didn't know to ask about. In this case, it could.

---

## The bigger problem the cleanup guard didn't solve

The intuitive fix is the one I would have written by hand: wrap the cleanup `az aks update` in `[[ -n "$ADMIN_IP_RANGES" ]]` and `exit 1` if it's empty. Done. Move on.

That stops the Internet-exposure failure mode. It does not stop the **admin lockout** failure mode, and the lockout failure mode is uncomfortably easy to trigger.

Trace what happens if the secret is empty and the deploy pipeline runs:

1. The "Add runner IP" step runs first. It already has a `[[ -n "$ADMIN_IP_RANGES" ]]` guard — but that guard only controls whether to *append* admin ranges to the runner IP. With the secret empty, the guard is correctly skipped, and the cluster ends up with **only** the runner's ephemeral IP in the allowlist.
2. The deploy succeeds. Helm rolls out cleanly. Tests pass. The site is healthy.
3. The cleanup step runs. With my new guard in place, it now refuses to call `az aks update` with an empty value and exits 1.
4. The cluster's allowlist still contains only the GitHub Actions runner's IP — which is ephemeral, ageing out the moment that runner is recycled.

Net result: the cluster is locked. Not open to the Internet, but not reachable by the admin either. The cleanup-step guard alone replaced "open to everyone" with "open to no one." Still a self-inflicted incident. Still a 3 AM page.

The real fix has to fail *before* the Add step runs. A preflight check at the top of the deploy job, immediately after Azure login, that validates the secret is non-empty and aborts the entire job if it isn't. That way the cluster's allowlist is never mutated in the first place.

So the PR has three guards, not one:

- **Preflight** at the start of each deploy job, fails fast before any cluster mutation.
- **Cleanup guard** as a belt-and-suspenders second check.
- **`sync-ip.sh` defense in depth** that refuses to run `gh secret set --body ""` even though the script's existing `set -e` plus `curl -f` should already make that unreachable.

Three layers. Each one assumes the other two might fail. None of them is expensive.

---

## A small but meaningful detail: `env:` vs. `${{ }}`

Look at the `before` and `after` of the cleanup step:

**Before:**
```yaml
run: |
  az aks update \
    --api-server-authorized-ip-ranges "${{ secrets.ADMIN_IP_RANGES }}"
```

**After:**
```yaml
env:
  ADMIN_IP_RANGES: ${{ secrets.ADMIN_IP_RANGES }}
run: |
  if [[ -z "${ADMIN_IP_RANGES}" ]]; then
    echo "::error::ADMIN_IP_RANGES is empty; refusing to update allowlist."
    exit 1
  fi
  az aks update \
    --api-server-authorized-ip-ranges "${ADMIN_IP_RANGES}"
```

Both forms reach the same `az` invocation in the happy path. The reason to prefer the `env:` form is **GitHub Actions does its `${{ ... }}` template substitution before bash ever sees the script.** That means the secret value is rendered into the script literal that the runner will execute. If a secret ever contained a backtick, a `$()`, a quote, or any shell metacharacter, the inline form would interpret it. The `env:` form passes the value as an environment variable, and bash treats `${ADMIN_IP_RANGES}` as a normal variable reference. No injection surface.

GitHub already redacts secrets in logs, so this isn't strictly about secret exfiltration. It's about ensuring the script's *behavior* can't be hijacked by the value of the secret. That distinction matters when the secret is something a future contributor might paste in via a UI without thinking about its contents.

The Actions security docs call this out as the recommended pattern. It costs three extra lines per step. It removes an entire category of bug.

---

## What I left out, and why

A reasonable next step would be to extract a composite action — `.github/actions/aks-ip-allowlist` with `add` and `remove` modes — so the guard logic lives in one place and can't be forgotten when the next workflow needs it. I deliberately didn't do that in this PR.

Two reasons. First, **a security fix should be the smallest possible diff against the running system.** Three guards added in place is reviewable in five minutes; a refactor that moves the logic into a new directory is reviewable in twenty. The cost of adding it later is low; the cost of conflating "stop the bleeding" with "improve the architecture" is that one of the two things gets dropped.

Second, **two callers don't make a pattern.** The composite-action refactor pays for itself when there are three or four workflows that need the same logic, not two. If a third workflow ever appears, that's the time to extract.

Same logic for adding `actionlint` to CI. It would be the right home for a custom rule that flags any future `--api-server-authorized-ip-ranges "${{ ... }}"` without an enclosing emptiness check. Not in this PR. Filed for follow-up.

---

## Verifying without firing the gun

How do you verify a fail-safe works without triggering the failure it's meant to catch? The PR is, by design, the kind of change where you can't prove correctness by running it in production. A green CI run on the PR only exercises the happy path — the secret is populated, the preflight passes, the deploy succeeds, the cleanup restores the right value. None of that proves the empty-secret path actually fails closed.

What I did instead:

```bash
# Empty case — guard should fail
ADMIN_IP_RANGES="" bash -c \
  'if [[ -z "${ADMIN_IP_RANGES}" ]]; then
     echo "::error::empty"; exit 1
   fi'
echo "exit=$?"
# → ::error::empty
# → exit=1
```

```bash
# Populated case — guard should pass
ADMIN_IP_RANGES="1.2.3.4/32" bash -c \
  'if [[ -z "${ADMIN_IP_RANGES}" ]]; then
     echo "::error::empty"; exit 1
   fi
   echo "would-update"'
echo "exit=$?"
# → would-update
# → exit=0
```

Static lint with `actionlint` clean. `bash -n` on `sync-ip.sh` clean. Cluster allowlist audited before and after — one admin IP, no leftover runner IPs, no empty entries.

This is the kind of testing where "a green CI run" is necessary but nowhere near sufficient. The empty-secret path can only be exercised in a fork where the wrong outcome doesn't matter, and even then it requires deliberately misconfiguring the test environment. Sometimes the right move is to verify the logic locally, sign off on the code review carefully, and accept that the production verification is "this never fires."

---

## Lessons

1. **The cleanup guard you didn't write is the lockout you didn't predict.** A pipeline that mutates a security-critical configuration during deploys has at least three failure modes: open everything, lock everyone out, and the happy path. The naive guard catches one. You need preflight and cleanup, layered, to catch the second. If your fix only has one layer, you're betting that the *other* failure mode can't happen.

2. **Use `env:` to pass secrets into shell scripts in GitHub Actions.** Inline `${{ secrets.X }}` substitution renders into the script body before bash parses it, so any shell metacharacter in the value becomes executable. The `env:` form makes the value a plain environment variable that bash treats inertly. Two extra lines per step buys you an entire vanished class of injection bug.

3. **Plan with the agent before you implement.** The bigger insight in this fix — that the cleanup guard alone causes admin lockout — came out of a planning conversation, not the editor. The agent walked through every related step, every other workflow, every tangential file, and surfaced the second-order failure mode I would have shipped without. The cost was about ninety seconds of conversation. The value was a security incident I didn't have.

---

## What's live

- **PR #25** ([fix(ci): guard ADMIN_IP_RANGES against empty-secret cluster lockout](https://github.com/karlkuhnhausen/f1-race-intelligence/pull/25)) — preflight + cleanup guards in `ci-cd.yml` and `infra-deploy.yml`, defense in depth in `sync-ip.sh`.
- **AKS allowlist** — single admin entry, in sync with the GitHub secret.
- **Tests**: 70 still passing (this is a CI config change, no new test surface).
- **Docs**: this post; Day 12's "next sprint" line can be retired.

Three guards, +42/-2 lines, one closed hole.

---

[← Day 12: The Bug That Said "Completed" When the Race Hadn't Started](day-12-status-badge-bug.md)
