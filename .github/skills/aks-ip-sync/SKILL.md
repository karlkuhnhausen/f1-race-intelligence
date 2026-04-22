---
name: aks-ip-sync
description: 'Check and update the AKS API server authorized IP ranges when your home IP has changed. Use when: IP address changed, locked out of kubectl, AKS API server rejecting connections, need to update authorized IP ranges, home IP rotated.'
argument-hint: 'Optional: provide a specific IP in CIDR format (e.g. 1.2.3.4/32), otherwise current public IP is detected automatically'
---

# AKS IP Sync

## When to Use
- You can no longer run `kubectl` commands or `helm` and get a connection refused / timeout
- Your ISP has rotated your home IP (common with residential broadband)
- You want to verify your current IP is in the AKS allowlist before starting work
- You are working from a different location (office, travel, VPN)

## Procedure

Run [sync-ip.sh](./scripts/sync-ip.sh) in the terminal. It will:

1. Detect your current public IP via `api.ipify.org`
2. Query the current AKS authorized IP ranges
3. If your IP is already allowed → confirm and exit (no change made)
4. If your IP is missing → update `az aks update` with the new IP and update the `ADMIN_IP_RANGES` GitHub secret

### Quick run
```bash
bash .github/skills/aks-ip-sync/scripts/sync-ip.sh
```

### With an explicit IP (e.g. from a VPN or office)
```bash
bash .github/skills/aks-ip-sync/scripts/sync-ip.sh 203.0.113.42/32
```

## What the script does NOT do
- It does not remove other IPs already in the allowlist — it only adds yours if missing
- It does not touch any cluster workloads or deployed services
- It does not modify any Bicep/IaC files (those default to `[]` intentionally)
