# Day 9: The Struggle Bus to a Private Cosmos DB

*Posted April 26, 2026 · Karl Kuhnhausen*

---

The plan for Sunday was simple: Cosmos DB had public network access enabled, the Azure Policy "Cosmos DB accounts should have public network access disabled" was flagging it as non-compliant, and the constitutional principle is clear — **Security and Secrets Baseline. All secrets in Key Vault via Managed Identity. No plaintext anywhere.**

Public Cosmos endpoints are basically plaintext data exposure waiting for a misconfigured firewall rule. Time to lock it down.

What I thought would take an hour took the entire afternoon. Five PRs. Four distinct failure modes. One Defender alert (false alarm). One existential moment of "wait, did I just break the entire site?"

This is the story of how I learned that **a secure private endpoint is the easy part. Everything around it is the hard part.**

---

## The Plan

The architecture goal:

1. Create an Azure Private Endpoint for Cosmos in a dedicated subnet.
2. Disable Cosmos public network access.
3. Recreate AKS into the new VNet so pods can reach the private endpoint over Azure backbone DNS.
4. Verify the backend can still read and write through the private path.

Simple. Four steps. Should be in the bag by lunchtime.

---

## Struggle #1: The Bicep Role Assignment Trap

First problem: the security model itself.

The CI managed identity that GitHub Actions uses to deploy is intentionally restricted to **Contributor** on the resource group. Not Owner. Not User Access Administrator. Just Contributor. Per the [Day 8 security audit](day-8-ops-security-lockdown.md), I deliberately took away its ability to grant role assignments — because if CI can grant itself roles, it can escalate to Owner, and the security boundary is theater.

But the Bicep templates contained role assignment resources:

```bicep
resource backendAcrPullRole 'Microsoft.Authorization/roleAssignments@...' = {
  scope: acr
  properties: {
    roleDefinitionId: subscriptionResourceId('...', acrPullRoleId)
    principalId: backendIdentity.properties.principalId
  }
}
```

CI couldn't deploy this. Every infra deploy failed with `AuthorizationFailed: does not have authorization to perform action 'Microsoft.Authorization/roleAssignments/write'`.

The textbook fix is to grant CI the `User Access Administrator` role. The textbook fix is wrong, because that's exactly the privilege escalation path I closed in Day 8.

The right fix is **separation of concerns**:

- **Bicep deploys infrastructure** (CI, Contributor-only).
- **Role assignments are granted out-of-band by a human Owner** running [`infra/scripts/grant-roles.sh`](../../infra/scripts/grant-roles.sh).

So I extracted every `Microsoft.Authorization/roleAssignments` resource out of Bicep and into a single shell script. The script reads the Bicep deployment outputs (principal IDs, ACR name, AKS name, Key Vault name, Cosmos account name) and runs `az role assignment create` for each binding.

Five role assignments now live in the script:
- backend → `AcrPull` on ACR
- backend → `Key Vault Secrets User` on Key Vault
- backend → `Cosmos DB SQL Data Contributor` on Cosmos
- ci → `AcrPush` on ACR
- ci → `AKS Cluster User` on AKS

The script is idempotent — re-running it is safe, existing bindings are reported but not duplicated.

**Why this matters:** the infrastructure is now reproducible by CI without ever giving CI the power to escalate privileges. Re-running `grant-roles.sh` is a privileged operation performed deliberately by a human Owner. That separation is the whole point of the security boundary.

---

## Struggle #2: Federated Credentials Have a Race Condition

CI now had to authenticate to Azure for **four different GitHub events**:

- `refs/heads/master` (push to master, deploys app)
- `environment:infrastructure` (infra deploy workflow)
- `environment:production` (production env protection)
- `environment:aks-management` (AKS start/stop workflow)

That's four federated credentials on the same managed identity. I added all four to the Bicep module and triggered the deploy.

```
ConflictError: Concurrent write operations on Federated Identity Credentials
are not allowed.
```

Azure rejects parallel federated credential writes on the same identity. They have to be serialized. Bicep's default behavior is to deploy resources in parallel where possible, so all four creds tried to land at the same time.

The fix is explicit `dependsOn` chains:

```bicep
resource ciInfraFedCred '...' = { ... }
resource ciProductionFedCred '...' = {
  dependsOn: [ ciInfraFedCred ]
  ...
}
resource ciAksManagementFedCred '...' = {
  dependsOn: [ ciProductionFedCred ]
  ...
}
```

Each cred dependsOn the previous one. They land sequentially. Bicep deploy succeeds.

This is one of those "mentioned in passing in some StackOverflow answer" gotchas that you only learn by hitting it.

---

## Struggle #3: Pods Can Pull Images, Right? RIGHT?

With Bicep deployed, AKS recreated into the new VNet, role grants applied — I triggered the app deploy. Helm install ran. Then it timed out:

```
Error: UPGRADE FAILED: resource Deployment/f1-race-intelligence/f1-backend
not ready. status: Failed, message: Progress deadline exceeded
```

`kubectl describe pod` showed:

```
Events:
  Warning  Failed  Failed to pull image "acrf1raceintel.azurecr.io/f1-backend:latest":
  failed to authorize: failed to fetch anonymous token:
  unexpected status from GET request to https://acrf1raceintel.azurecr.io/oauth2/token:
  401 Unauthorized
```

But I had granted `AcrPull` to the backend managed identity. I checked. The role assignment was there. The role definition ID was correct. The scope was correct.

I stared at this for ten minutes before the realization hit:

**Image pulls don't go through the workload identity.**

When kubelet pulls a container image, it authenticates with the **AKS kubelet identity** — a completely separate managed identity that Azure auto-creates per agent pool. The backend identity (`f1-backend-identity`) authenticates the Go process *inside* the pod when it calls Cosmos and Key Vault. The kubelet identity authenticates the *node itself* when it pulls images from ACR.

Two identities. Two purposes. The AcrPull I granted was on the wrong identity.

The fix:

```bash
KUBELET_PRINCIPAL=$(az aks show -g "$RG" -n "$AKS_NAME" \
  --query "identityProfile.kubeletidentity.objectId" -o tsv)
az role assignment create --role AcrPull \
  --assignee-object-id "$KUBELET_PRINCIPAL" \
  --scope "$ACR_ID"
```

Or, equivalently, `az aks update --attach-acr <name>` which does the same thing.

Within 30 seconds of granting the role: pods went from `ImagePullBackOff` to `Running 1/1`. Backend logs immediately showed:

```json
{"level":"INFO","msg":"openf1 poll complete","season":2026,"meetings":26}
```

26 meetings written to Cosmos. Through the private endpoint. **The whole point of the day's work was working.**

I added the kubelet AcrPull grant to `grant-roles.sh` so future AKS recreations don't hit this. ([PR #9](https://github.com/karlkuhnhausen/f1-race-intelligence/pull/9))

---

## Struggle #4: Why is the Frontend URL a 404?

Backend pods up. Cosmos working. I opened the frontend URL: `http://f1.20.171.233.61.nip.io/`

```
This site can't be reached.
20.171.233.61 took too long to respond.
```

Right. AKS got recreated. The old public LoadBalancer IP is gone. The new LB has a new IP: `135.234.65.19`.

The Helm `values.yaml` had the old IP baked into the ingress hostnames:

```yaml
ingress:
  host: f1.20.171.233.61.nip.io  # ← dead
```

Updated both `frontend/values.yaml` and `backend/values.yaml` to the new IP and re-deployed. ([PR #10](https://github.com/karlkuhnhausen/f1-race-intelligence/pull/10))

Tried the new URL: `http://f1.135.234.65.19.nip.io/`

```
This page isn't working right now.
f1.135.234.65.19.nip.io didn't send any data.
```

Different error. Different problem.

---

## Struggle #5: Microsoft Doesn't Trust nip.io

A different error message means a different layer is broken. "Didn't send any data" means TCP connected but the server returned nothing — *or* something is intercepting and dropping the connection mid-handshake.

I tested from inside the cluster:

```bash
$ kubectl exec -- curl -H "Host: f1.135.234.65.19.nip.io" http://localhost/
HTTP/1.1 200 OK
<!doctype html>
<html lang="en">
  <head>
    <title>F1 Race Intelligence</title>
```

The app worked perfectly inside the cluster.

I tested from an external service (downforeveryone.com):

```
f1.135.234.65.19.nip.io is up.
```

The site was up from the public Internet. It was just unreachable from my machine.

The culprit: **Microsoft corporate network filters `*.nip.io`**.

[nip.io](https://nip.io) is a free wildcard DNS service that resolves any subdomain like `f1.135.234.65.19.nip.io` to the embedded IP `135.234.65.19`. It's beloved by developers because it lets you serve hostnames without owning a domain. It's also flagged by enterprise security tools as a **DNS rebinding** vector — an attacker can use it to point a victim's browser at internal IPs.

So Microsoft (and most large corporate networks) silently blocks resolution. The browser gets nothing. "Didn't send any data."

The fix: stop using nip.io. Use the **Azure-provided FQDN** instead.

```bash
az network public-ip update -g <mc-rg> \
  -n <pip-name> --dns-name f1raceintel
```

This produces `f1raceintel.westus3.cloudapp.azure.com`. It's free, persistent, and works on any network because it's just a regular Azure DNS A record.

To make this survive AKS recreations, I added an annotation to the ingress-nginx LoadBalancer service:

```yaml
metadata:
  annotations:
    service.beta.kubernetes.io/azure-dns-label-name: f1raceintel
```

The Azure cloud provider for Kubernetes reads this annotation and configures the public IP's DNS label automatically. Even if the cluster is destroyed and recreated, the DNS label sticks.

I also restructured the ingresses. With a single FQDN, the frontend and backend can't both claim path `/`. The split:

- **Frontend ingress**: `path: /` (Prefix) → React app
- **Backend ingress**: `path: /api` (Prefix) and `path: /healthz` (Exact) → Go service

The frontend's nginx already proxies `/api/` to the backend internally, so this works for both browser navigation and direct API calls. ([PR #11](https://github.com/karlkuhnhausen/f1-race-intelligence/pull/11))

---

## Struggle #6: The NSG You Didn't Create Will Kill You

After the FQDN switch, I tried the URL again.

```
f1raceintel.westus3.cloudapp.azure.com didn't send any data.
```

*Again.* Different host. Same error.

I ran an external probe from check-host.net — Spain, Netherlands, Poland — all timed out. So the site that was reachable 30 minutes ago was now unreachable from the entire public Internet. This was the moment I genuinely thought I'd broken everything.

I checked everything I could think of:

- LoadBalancer service has `EXTERNAL-IP: 135.234.65.19` ✓
- LB rules in Azure Network are correct (frontend IP → backend pool, port 80 → port 80) ✓
- LB health probe returns HTTP 200 from the healthy node (`/healthz` on nodePort 32654) ✓
- ingress-nginx pods Running ✓
- Backend and frontend pods Running ✓
- AKS auto-created NSG `aks-agentpool-17186921-nsg` allows Internet → 135.234.65.19 on TCP ✓

Everything was healthy. And nothing reached the cluster from outside.

Then I noticed something I'd missed:

```
$ az network vnet subnet list -g rg-f1raceintel --vnet-name vnet-f1raceintel
[
  {
    "name": "snet-aks",
    "networkSecurityGroup": {
      "id": ".../networkSecurityGroups/vnet-f1raceintel-snet-aks-nsg-westus3"
    }
  }
]
```

The subnet had its **own NSG**. One I didn't create.

Azure Policy in our subscription auto-applies an NSG to every subnet that doesn't have one. The auto-created NSG has **zero rules** — meaning only Azure's default rules apply, including the priority-65500 `DenyAllInBound`.

NSGs at multiple layers (subnet + NIC) are **AND-ed**. Both must allow the traffic. AKS's NIC NSG correctly allowed Internet → LB. The empty subnet NSG silently denied it.

```bash
$ az network nsg show -g rg-f1raceintel -n vnet-f1raceintel-snet-aks-nsg-westus3 \
    --query securityRules
[]
```

Zero custom rules. Default deny wins.

The fix:

```bash
az network nsg rule create -g rg-f1raceintel \
  --nsg-name vnet-f1raceintel-snet-aks-nsg-westus3 \
  --name allow-http-from-internet \
  --priority 100 --direction Inbound --access Allow \
  --protocol Tcp --source-address-prefixes Internet \
  --destination-address-prefixes "*" \
  --destination-port-ranges 80 443
```

Site went from timeouts to **HTTP 200** within five seconds.

I codified the NSG with its allow rule into [`infra/bicep/modules/vnet.bicep`](../../infra/bicep/modules/vnet.bicep) so a future deploy can't be reverted by Azure Policy creating an empty NSG that overrides the explicit one. ([PR #12](https://github.com/karlkuhnhausen/f1-race-intelligence/pull/12))

---

## What Actually Got Built

When I sat down on Sunday afternoon, the goal was: **one private endpoint**.

What got built by Sunday night:

| Component | State |
|---|---|
| Cosmos DB public network access | **Disabled** ✓ |
| Cosmos private endpoint in dedicated subnet | **Created** ✓ |
| Cosmos private DNS zone (`privatelink.documents.azure.com`) | **Linked to VNet** ✓ |
| Backend → Cosmos via private endpoint | **Verified** (`openf1 poll complete, meetings: 26`) |
| AKS in dedicated VNet with proper subnet NSGs | **Recreated** ✓ |
| CI managed identity at Contributor only | **Confirmed** — no privilege escalation path |
| Role grants extracted to manual Owner-only script | **5 role assignments** in `grant-roles.sh` |
| OIDC federated credentials for 4 GitHub contexts | **Serialized via dependsOn** ✓ |
| AKS kubelet identity → AcrPull | **Codified** ✓ |
| Subnet NSG with explicit Internet allow rule | **Codified in Bicep** ✓ |
| Ingress on Azure FQDN, not nip.io | **Working publicly** at `f1raceintel.westus3.cloudapp.azure.com` |
| Frontend + backend ingresses on shared FQDN | **Path-routed** (`/` → frontend, `/api` → backend) |

Six pull requests. Five distinct failure modes. The Azure Policy compliance check is now green.

---

## Lessons

**1. Private endpoints are the easy part.** The hard parts are:
- Resource policy interactions (auto-NSGs, default DenyAllInBound)
- Identity boundaries (kubelet vs workload identity)
- DNS layer assumptions (nip.io vs FQDN)
- Configuration that breaks across resource recreations (LB IPs in Helm values)

**2. Multi-layer NSGs need explicit configuration at every layer.** A subnet NSG and a NIC NSG AND together. An empty NSG is not "no NSG" — it's "deny everything except defaults."

**3. Two managed identities, two purposes.** AKS's kubelet identity authenticates image pulls. The workload identity authenticates SDK calls inside the pod. Granting AcrPull to the wrong one fails silently with `401 Unauthorized` and looks identical to a missing role.

**4. nip.io is unreliable in enterprise networks.** Don't use it for shareable URLs. Azure-provided FQDNs are free, work everywhere, and persist via the `service.beta.kubernetes.io/azure-dns-label-name` annotation.

**5. Helm values that hardcode infrastructure IPs are landmines.** Use FQDNs. Use service references. Never hardcode an LB IP that might change on the next resource recreation.

**6. CI at Contributor only is worth the friction.** Manual role grants by an Owner is annoying once per environment, but the security boundary it preserves is real. Automation that can grant itself privileges has effectively unlimited privileges.

---

## What's Next

The site is live, secure, and Azure Policy compliant. Backend is reading and writing Cosmos through a private endpoint. Manage Identity is the only credential anywhere in the stack. Key Vault secrets are accessed via Managed Identity. Public Cosmos endpoint is permanently disabled.

Next session: back to **Feature 003 Phase 5-7**, the qualifying and practice components that have been waiting in a stash for two days. With the security foundation locked down, I can focus on actual product features again.

The struggle bus stops here. For now.

---

**Live:** http://f1raceintel.westus3.cloudapp.azure.com/
**API:** http://f1raceintel.westus3.cloudapp.azure.com/api/v1/calendar?year=2026
**Healthz:** http://f1raceintel.westus3.cloudapp.azure.com/healthz

[← Day 8: The Security Alert I Got at 5 AM](day-8-ops-security-lockdown.md)
