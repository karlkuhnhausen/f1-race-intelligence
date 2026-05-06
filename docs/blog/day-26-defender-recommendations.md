# Day 26: 657 Unhealthy — Triaging Defender for Cloud on a Side Project

*Posted May 5, 2026 · Karl Kuhnhausen*

---

I opened Defender for Cloud for the first time in a while and saw the number: 657 unhealthy security assessments across my resource group. For a side project running two backend replicas, two frontend pods, and a single Cosmos DB, that number felt absurd. It also felt like something I shouldn't ignore.

The good news: 13 assessments were already healthy. Cosmos DB had a clean bill — private endpoints, RBAC-only auth, firewall rules, disabled public network access. Key Vault had RBAC, soft delete, and firewall enabled. AKS had Defender profile, restricted API server access, and Kubernetes RBAC. The NSGs were in place. All the hardening from Days 8, 9, and 13 was still holding.

The bad news was everything else.

---

## The triage

Thirty-nine unique recommendations across two categories: platform-level configuration gaps and container image vulnerabilities.

The platform recommendations were the smaller set — thirteen items covering things like diagnostic logging, purge protection, private endpoints for ACR and Key Vault, the Azure Policy add-on, encryption at host, and the inevitable "protect your VNet with Azure Firewall" recommendation that Microsoft will surface on any VNet that doesn't have a $900/month firewall in front of it.

The container vulnerabilities were the bulk. Twenty-six recommendations, each flagging a specific package — `openssl`, `musl`, `curl`, `busybox`, `zlib`, `libpng`, `libxpm`, `xz` — across dozens of entities. The high hit counts (75–80 per package) came from Defender scanning every layer of every image in ACR and every running container in AKS, creating a separate assessment entity for each finding per image.

When I grouped them, the picture simplified. The OS-level package vulnerabilities fell into two buckets: packages in *my* images (fixable by rebuilding with a newer base) and packages in *AKS system images* (fixable only by upgrading the cluster).

## What I did today

Two actions:

### 1. Rebuild app images with patched Alpine base

The frontend Dockerfile was using untagged Alpine variants:

```dockerfile
FROM node:20-alpine AS build
# ...
FROM nginx:1.27-alpine
```

These pull whatever Alpine version was current when the image was last built. Pinning to `alpine3.21` forces a fresh pull of the latest patched packages:

```dockerfile
FROM node:20-alpine3.21 AS build
# ...
FROM nginx:1.27-alpine3.21
```

The backend Dockerfile was already using `gcr.io/distroless/static-debian12:nonroot` — the most minimal runtime image available. No shell, no package manager, no libc. Nothing to patch. The Go binary is the only file in the image.

PR [#72](https://github.com/karlkuhnhausen/f1-race-intelligence/pull/72) changed two lines in the frontend Dockerfile. CI passed — lint, test, build all green. Merged to master, which triggered the full pipeline: rebuild both images, push to ACR, Helm deploy to AKS.

This single change should clear about 600 of the 657 findings — every `busybox`, `curl`, `openssl`, `musl`, `zlib`, `libpng`, `libxpm`, and `xz` assessment tied to my app images.

### 2. Upgrade AKS from 1.33.8 to 1.33.10

The remaining package vulnerabilities — `glibc`, `systemd`, `util-linux`, plus Go dependencies like `golang.org/x/crypto`, `google.golang.org/grpc`, and `go.opentelemetry.io/otel/sdk` — lived in AKS system pod images. I can't rebuild those. Microsoft ships them as part of the node image.

The node image was already the latest available (`AKSAzureLinux-V3gen2-202604.13.0`), so a node image upgrade wouldn't help. But the Kubernetes version had available patches: `1.33.9` and `1.33.10`.

```
az aks upgrade \
  --resource-group rg-f1raceintel \
  --name aks-f1raceintel \
  --kubernetes-version 1.33.10 \
  --yes
```

This upgrades the control plane first, then does a rolling node pool replacement. AKS creates a surge node (pool temporarily goes from 2 to 3), drains an old node, reimages it, uncordons, and repeats. Zero downtime — the pods redistribute across healthy nodes the entire time.

The upgrade took about 40 minutes end-to-end, which is longer than usual because `az aks update` calls during the deploy (adding and removing the CI runner IP from the API server allowlist) had to wait for the cluster's provisioning state to settle.

## The deploy sequence

The timing was interesting. I kicked off the AKS upgrade and the PR merge concurrently, which meant the CI/CD pipeline reached the "Deploy to AKS" step while the cluster was still upgrading nodes. The pipeline has two production approval gates (one before push, one before deploy), so the natural pauses let the upgrade finish before Helm tried to roll out new pods.

The `az aks update` step in the deploy — the one that temporarily adds the GitHub Actions runner IP to the API server allowlist — ran immediately after the upgrade completed. That single step took nearly 10 minutes because `az aks update` triggers a full reconciliation of the cluster state, which is heavier than usual right after an upgrade.

Everything landed: images pushed, Helm upgrade completed, pods running the new images on K8s 1.33.10.

## What I'm not doing (and why)

The triage produced a prioritized list. Here's the second half — the items I deferred:

| Recommendation | Why deferred |
|---|---|
| **Key Vault purge protection** | Irreversible once enabled. Want to think about whether I'll ever need to recreate the vault. Low risk to defer. |
| **KV and AKS diagnostic logs** | Pure observability — no security risk from deferring. Will add when I set up a Log Analytics workspace properly. |
| **Azure Policy add-on** | Useful but may flag existing workloads with policy violations that need review. Better as a planned task. |
| **ACR private endpoint + network restrict** | Medium risk of breaking AKS image pulls if misconfigured. Needs careful testing. |
| **Key Vault private endpoint** | Could break CI/CD and local `az` access. Needs to be paired with network rule exceptions. |
| **Encryption at host** | Need to verify `Standard_B2s` SKU supports it. May require node pool recreation. |
| **Azure Firewall** | ~$900/month minimum. Not happening on a side project. NSGs and API server allowlisting are sufficient. |

The constitutional principle is clear — "Security and Secrets Baseline" is Principle #4 — but the constitution also says "Minimal dependencies with written justification" and "Start simple." Not every Defender recommendation maps to a real threat on a single-user side project with no customer data. Azure Firewall protecting a VNet that serves a Formula 1 stats dashboard is a compliance checkbox, not a security decision.

The image rebuild and K8s upgrade were the high-leverage moves. They address real CVEs in real packages that are actually running. The rest goes on the backlog.

---

## What's live now

- **PR #72** — Frontend Dockerfile pinned to Alpine 3.21, merged and deployed
- **AKS** — Upgraded to Kubernetes 1.33.10 with latest node image
- **Defender posture** — Expecting ~600 of 657 unhealthy findings to clear once Defender rescans (typically within 24 hours)
- **App verified** — API returning 200, 22 rounds, both services running 2 replicas on the upgraded cluster

---

[← Day 25: The Sprint Sessions That Showed Nothing — A Three-Bug Cascade](day-25-sprint-session-saga.md)
