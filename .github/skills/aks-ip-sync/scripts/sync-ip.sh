#!/usr/bin/env bash
# sync-ip.sh — Sync your current public IP with the AKS API server authorized IP ranges
# Usage: bash sync-ip.sh [optional-ip-cidr]
# Example: bash sync-ip.sh 203.0.113.42/32

set -euo pipefail

RESOURCE_GROUP="rg-f1raceintel"
CLUSTER_NAME="aks-f1raceintel"
REPO="karlkuhnhausen/f1-race-intelligence"
SECRET_NAME="ADMIN_IP_RANGES"

# ── Determine target IP ────────────────────────────────────────────────────────
if [[ $# -ge 1 && -n "${1:-}" ]]; then
  MY_IP="$1"
  # Append /32 if no prefix length given
  [[ "$MY_IP" != *"/"* ]] && MY_IP="${MY_IP}/32"
  echo "Using provided IP: $MY_IP"
else
  RAW_IP=$(curl -fsSL https://api.ipify.org)
  MY_IP="${RAW_IP}/32"
  echo "Detected current public IP: $MY_IP"
fi

# ── Get current authorized IP ranges ──────────────────────────────────────────
echo ""
echo "Fetching current AKS authorized IP ranges..."
CURRENT_RANGES=$(az aks show \
  --resource-group "$RESOURCE_GROUP" \
  --name "$CLUSTER_NAME" \
  --query "properties.apiServerAccessProfile.authorizedIpRanges" \
  -o tsv 2>/dev/null || echo "")

echo "Current ranges: ${CURRENT_RANGES:-"(none — API server is open to all)"}"

# ── Check if IP is already allowed ────────────────────────────────────────────
if echo "$CURRENT_RANGES" | grep -qF "$MY_IP"; then
  echo ""
  echo "✓ Your IP ($MY_IP) is already in the authorized ranges. No changes needed."
  exit 0
fi

# ── Build new ranges list (keep existing, add new IP) ─────────────────────────
if [[ -z "$CURRENT_RANGES" ]]; then
  NEW_RANGES="$MY_IP"
else
  # Convert tab-separated to comma-separated, then append new IP
  EXISTING_CSV=$(echo "$CURRENT_RANGES" | tr '\t' ',' | tr '\n' ',' | sed 's/,$//')
  NEW_RANGES="${EXISTING_CSV},${MY_IP}"
fi

echo ""
echo "Updating AKS authorized IP ranges to: $NEW_RANGES"
az aks update \
  --resource-group "$RESOURCE_GROUP" \
  --name "$CLUSTER_NAME" \
  --api-server-authorized-ip-ranges "$NEW_RANGES" \
  --only-show-errors

echo ""
echo "✓ AKS API server updated."

# ── Update GitHub secret ───────────────────────────────────────────────────────
# Defense in depth: never push an empty value. An empty ADMIN_IP_RANGES secret
# would cause the CI cleanup step to run `az aks update --api-server-authorized-ip-ranges ""`,
# which Azure interprets as "remove all restrictions" — opening the API server to the Internet.
if [[ -z "$NEW_RANGES" ]]; then
  echo "✗ Refusing to set empty value for '$SECRET_NAME' — aborting." >&2
  exit 1
fi

if command -v gh &>/dev/null; then
  echo "Updating GitHub secret '$SECRET_NAME'..."
  gh secret set "$SECRET_NAME" --body "$NEW_RANGES" --repo "$REPO"
  echo "✓ GitHub secret '$SECRET_NAME' updated to: $NEW_RANGES"
else
  echo "⚠ 'gh' CLI not found — skipping GitHub secret update."
  echo "  Manually update secret '$SECRET_NAME' in repo $REPO to: $NEW_RANGES"
fi

echo ""
echo "Done. You can now run kubectl and helm against $CLUSTER_NAME."
