# Day 8: The Security Alert I Got at 5 AM — And What I Did About It

*Posted April 22, 2026 · Karl Kuhnhausen*

---

Two days after posting Day 7, I woke up to a Microsoft Defender for Cloud alert.

> **"Suspicious invocation of a high-risk 'Credential Access' operation detected from a GitHub Actions service principal."**

The timestamp was April 20, 2026, 5:48 UTC. The operation: `listClusterUserCredential` — retrieving Kubernetes credentials. The identity: the CI Managed Identity that GitHub Actions uses to deploy to AKS.

My first thought was: *Has someone stolen my CI credentials and gained access to my cluster?*

My second thought: *Or did I do this myself?*

---

## The Investigation

The Defender alert linked to a burst of `listClusterUserCredential` calls — nine of them, all from GitHub Actions runner IPs, all on April 20 between 04:59 and 05:45 UTC. Defender's heuristic flagged the unusual volume and the fact that the runners were ephemeral (different IP addresses on each run).

The Azure Activity Log told the real story. Every single call correlated to a CI/CD pipeline run triggered by a `git push` to `master`. The commit messages: blog navigation link fixes, README updates, an adjacent blog post getting a "next" link added. Nine blog-related commits in quick succession — each one triggering a full pipeline run including the Helm deploy step, which calls `az aks get-credentials` to authenticate against the cluster.

No compromise. No attacker. Just a developer who writes blog posts in batches and pushes them one by one.

The Defender alert was correct by its own logic. Nine calls from nine different runner IPs in 46 minutes *is* unusual behavior for a credential access operation. It just happened to be me. The alert was doing its job — it found a real anomaly and asked me to explain it.

This is actually the best kind of security alert: a false positive that still reveals something worth thinking about. The volume of calls wasn't a problem, but the fact that *every push to master triggers a full Helm deploy* — including fetching cluster credentials — was worth examining. A docs commit shouldn't touch a Kubernetes cluster.

---

## The Audit

Before implementing anything, I did a full audit. If there was any chance of a real compromise, I needed to know the blast radius.

**What could the CI identity actually do?**

The Managed Identity had three role assignments:
- `AcrPush` on the container registry — push Docker images
- `Contributor` on the AKS cluster — deploy Helm charts
- `Key Vault Secrets User` on Key Vault — read application secrets

No subscription-level access. No ability to create or destroy infrastructure outside the scope of what the CI/CD pipeline legitimately needs. The OIDC federation was scoped to `repo:karlkuhnhausen/f1-race-intelligence:ref:refs/heads/master` — only pushes to master could authenticate.

**What would an attacker have been able to do?**

They could have deployed a malicious container image to the AKS cluster. They could have read secrets from Key Vault. They could not have exfiltrated Cosmos DB data directly, modified Azure infrastructure, accessed other subscriptions, or pivoted to anything outside the resource group. Real damage? Yes, possible. Catastrophic? No.

**Did anything look wrong?**

All nine operations matched the CI/CD run timestamps exactly. No off-hours activity. No operations from non-GitHub IPs. No unusual Key Vault secret access. No new role assignments or identity changes. Every log entry was explainable.

Clean audit. No breach.

---

## The Hardening

Even with a clean bill of health, the audit revealed things worth tightening. The alert was a gift: an opportunity to build the security model that a production system should have, rather than the "get it working and harden later" model that most hobby projects run on indefinitely.

Here's what changed.

### GitHub Actions Allowed-Actions List

Before: any action from any publisher could run in the workflow.

After: only actions from `actions/*`, `azure/*`, and `docker/*` are permitted. Third-party actions with supply chain risk (a real and documented attack vector) can't be added without explicitly expanding the allowlist. Default workflow permissions dropped to read-only.

### GitHub Environments with Required Approvals

Three environments now protect deployment operations:

| Environment | Used by | Gate |
|-------------|---------|------|
| `production` | `ci-cd.yml` push and deploy jobs | Required reviewer |
| `aks-management` | `aks-schedule.yml` start/stop | Required reviewer |
| `infrastructure` | `infra-deploy.yml` | Required reviewer |

"Required reviewer" is me. Every deploy waits for a manual approval before the job runs. This adds friction, but it means no one — including an attacker who has compromised the CI identity — can trigger a deploy without my acknowledgment. I see the pending approval notification. I can cancel it if it's not expected.

The `aks-schedule.yml` workflow also got an actor check: if someone manually dispatches it (via `workflow_dispatch`) and the actor isn't my GitHub username, the workflow exits with an error before touching anything.

### OIDC Federation Scoped to Environments

When you add `environment: production` to a GitHub Actions job, the OIDC token's `sub` claim changes. Instead of `repo:owner/repo:ref:refs/heads/master`, it becomes `repo:owner/repo:environment:production`.

Azure validates the `sub` claim against the federated credential definition. If no matching credential exists, the authentication fails.

This is exactly what happened immediately after I configured the environments: the CI pipeline started failing with `AADSTS70021: No matching federated identity record found`. The workflows had been updated to use environments, but Azure didn't have the corresponding federated credentials.

Fix: add two new federated credential definitions to the CI Managed Identity — one for the `production` environment, one for `aks-management`. The Bicep module (`ci-identity.bicep`) was updated to define them as code, so the next infrastructure redeploy creates them automatically.

The lesson here: OIDC subject claims are precise strings. If you add an environment gate to a GitHub Actions job, you must add a matching federated credential in Azure. The two sides of the OIDC handshake have to agree on the subject.

### AKS API Server IP Allowlist

The AKS API server — the Kubernetes control plane endpoint — was publicly accessible to any IP. This is the default for AKS clusters. Anyone who obtained valid kubeconfig credentials could execute `kubectl` commands from anywhere in the world.

After hardening: the API server only accepts connections from my home IP and from GitHub Actions runner IPs (added dynamically at the start of each deploy and removed at the end).

The workflow pattern looks like this:

```yaml
- name: Add runner IP to AKS allowlist
  run: |
    RUNNER_IP=$(curl -s https://api.ipify.org)/32
    CURRENT=$(az aks show ... --query "apiServerAccessProfile.authorizedIpRanges" -o tsv)
    az aks update ... --api-server-authorized-ip-ranges "${CURRENT},${RUNNER_IP}"

# ... Helm deploy steps ...

- name: Remove runner IP from AKS allowlist
  if: always()
  run: |
    # Remove just the runner's IP, preserve admin IP
```

The `if: always()` on the cleanup step is critical — it runs even if the deploy fails, so a crashed deployment doesn't leave a runner IP permanently in the allowlist.

The admin IP (my home IP) is stored as a GitHub secret called `ADMIN_IP_RANGES`. It's never in code. When my IP changes — residential ISPs rotate IPs periodically — I run a script that detects the current IP, compares it to the allowlist, and updates both AKS and the GitHub secret if they diverge.

### Branch Protection on Master

Before: commits could be pushed directly to `master`. A compromised CI token could technically create commits, not just deploy.

After: `master` requires a pull request. No direct pushes. This also means every feature change is reviewed before merge — good hygiene regardless of security concerns. The one wrinkle: solo developers can't approve their own PRs under the default "required approvals" setting. Solution: keep the PR requirement (enforces the branch model) but disable the approval count requirement (avoid blocking yourself from merging your own work).

---

## The Part That Went Wrong Before It Went Right

Security hardening rarely goes cleanly. Here's what broke.

After adding `environment: production` to the CI/CD workflow and merging to master, the next CI run failed immediately:

```
Error: AADSTS70021: No matching federated identity record found for the token's subject claim.
Incoming token subject: repo:karlkuhnhausen/f1-race-intelligence:environment:production
```

The pipeline had been deploying successfully for weeks. Now it couldn't authenticate to Azure at all. The staging-to-production environment gate worked as designed — it changed the OIDC subject claim — but I'd forgotten to create the matching Azure-side federated credentials first.

The fix was a CLI command:

```bash
az identity federated-credential create \
  --identity-name id-f1raceintel-ci \
  --resource-group rg-f1raceintel \
  --name github-env-production \
  --issuer "https://token.actions.githubusercontent.com" \
  --subject "repo:karlkuhnhausen/f1-race-intelligence:environment:production" \
  --audiences "api://AzureADTokenExchange"
```

Then the same for `aks-management`. Two commands, pipeline unblocked. But this is the kind of thing that bites you in a real incident: you're trying to deploy a fix under pressure and authentication breaks because a token claim changed.

**The lesson:** When you add GitHub environments to OIDC-federated workflows, add the Azure federated credentials *first*, before enabling the environments in the workflow. The old `ref:refs/heads/master` credential keeps working until you're ready to switch over.

---

## The Architecture That Resulted

After all the changes, here's the layered security model:

```
GitHub push to master
    → Branch protection: PR required, no direct push
    → GitHub Actions: only approved action publishers
    → GitHub Environments: manual approval gate (you must approve)
    → OIDC authentication: scoped to specific environment subjects
    → Azure Managed Identity: minimal RBAC (AcrPush + K8s Contributor + KV read)
    → AKS API server: allowlisted IPs only (admin + runner, dynamically managed)
    → Key Vault: RBAC authorization, no connection strings in environment variables
```

Each layer catches a different threat. Branch protection catches rogue commits. Environment gates catch automated deploy attempts without human awareness. OIDC scoping means stolen credentials from one environment don't work in another. IP allowlisting means even valid credentials are useless from the wrong network.

No layer is perfect in isolation. Together, they make the attack surface small enough to monitor.

---

## Lessons for Building in Public on Azure

**Defender alerts are conversations, not verdicts.** The alert was technically accurate — the behavior was anomalous. The correct response was to investigate and explain it, not panic. "Is this expected?" is a better first question than "Am I compromised?"

**Security hardening has a sequence dependency.** When you change OIDC subjects by adding environments, the Azure side must be updated *before* the GitHub side. If you do it in the wrong order, you break your pipeline and have to debug it under the same security constraints you just added.

**The blast radius question is worth answering before an incident.** Before this alert, I hadn't explicitly thought through "what could an attacker actually do with my CI identity?" Doing that analysis in a calm moment, before anything is actually wrong, produces clearer thinking than doing it at 5 AM after an alert fires.

**Hobby projects have the same attack surface as production projects.** The AKS API server didn't care that this was a side project. The GitHub Actions runner IPs looked the same to Defender as they would in an enterprise pipeline. If you're running real infrastructure — even a personal project — the security defaults are not enough.

**IP allowlisting on a residential connection requires automation.** Comcast rotates IPs. The AKS allowlist becomes useless the next morning if you don't update it. The aks-ip-sync script (`.github/skills/aks-ip-sync/scripts/sync-ip.sh`) handles this: it detects your current public IP, checks the allowlist, and updates both AKS and the GitHub secret if they diverge. One command, and you're unblocked.

---

## What's Next

The ops lockdown is complete. The immediate next step is writing Phases 5–7 of Feature 003: qualifying and practice result components with proper formatting. Then Feature 004: Design System and Brand Identity — the dashboard currently looks like a developer built it, which is accurate but not flattering.

The Defender alert has been resolved with a note confirming the activity was legitimate CI/CD traffic, not an intrusion.

And the cluster auto-stops at 7 PM Pacific. I approved that workflow run manually.

---

**Source:** https://github.com/karlkuhnhausen/f1-race-intelligence

---

*Previous: [Day 7: Race Results, Rate Limits, and the Branch You Forgot You Were On](day-7-phase-4-and-branch-confusion.md)*
