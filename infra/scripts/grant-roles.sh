#!/usr/bin/env bash
# Grant role assignments required by the F1 Race Intelligence stack.
#
# WHY THIS IS A SEPARATE SCRIPT (NOT IN BICEP):
# Role assignment creation requires User Access Administrator or Owner.
# We deliberately keep the CI managed identity at Contributor only, so
# automation cannot escalate privileges. Role grants are a privileged
# operation performed by a human Owner, out-of-band from the Bicep deploy.
#
# Run this script ONCE after the initial `az deployment sub create` and
# anytime new identities or resources are added that need role bindings.
#
# Prerequisites:
#   - Logged in as an account with Owner (or User Access Administrator) on the RG
#   - jq installed
#   - Bicep deployment has produced /tmp/f1-deploy-output.json (from deploy-infra.sh)
#
# Idempotent: re-running is safe; existing assignments are reported but not duplicated.

set -euo pipefail

RG="${RG:-rg-f1raceintel}"
DEPLOY_OUTPUT="${DEPLOY_OUTPUT:-/tmp/f1-deploy-output.json}"

if [[ ! -f "$DEPLOY_OUTPUT" ]]; then
  echo "ERROR: deployment output file not found at $DEPLOY_OUTPUT"
  echo "Run infra/scripts/deploy-infra.sh first."
  exit 1
fi

# Built-in role definition IDs
ROLE_ACR_PULL="7f951dda-4ed3-4680-a7ca-43fe172d538d"
ROLE_ACR_PUSH="8311e382-0749-4cb8-b61a-304f252e45ec"
ROLE_AKS_CLUSTER_USER="4abbcc35-e782-43d8-92c5-2d3f1bd2253f"
ROLE_KV_SECRETS_USER="4633458b-17de-408a-b874-0445c86b69e6"
# Cosmos DB built-in SQL role: Data Contributor
COSMOS_DATA_CONTRIBUTOR="00000000-0000-0000-0000-000000000002"

# Pull values from Bicep deployment outputs
SUB_ID=$(jq -r '.properties.outputs.ciSubscriptionId.value' "$DEPLOY_OUTPUT")
BACKEND_PRINCIPAL=$(jq -r '.properties.outputs.backendPrincipalId.value' "$DEPLOY_OUTPUT")
CI_PRINCIPAL=$(jq -r '.properties.outputs.ciPrincipalId.value' "$DEPLOY_OUTPUT")
ACR_NAME=$(jq -r '.properties.outputs.acrName.value' "$DEPLOY_OUTPUT")
AKS_NAME=$(jq -r '.properties.outputs.aksName.value' "$DEPLOY_OUTPUT")
KV_NAME=$(jq -r '.properties.outputs.keyVaultName.value' "$DEPLOY_OUTPUT")
COSMOS_ACCOUNT=$(jq -r '.properties.outputs.cosmosAccountName.value' "$DEPLOY_OUTPUT")

ACR_ID="/subscriptions/${SUB_ID}/resourceGroups/${RG}/providers/Microsoft.ContainerRegistry/registries/${ACR_NAME}"
AKS_ID="/subscriptions/${SUB_ID}/resourceGroups/${RG}/providers/Microsoft.ContainerService/managedClusters/${AKS_NAME}"
KV_ID="/subscriptions/${SUB_ID}/resourceGroups/${RG}/providers/Microsoft.KeyVault/vaults/${KV_NAME}"
RG_ID="/subscriptions/${SUB_ID}/resourceGroups/${RG}"

echo "=== Granting role assignments (idempotent) ==="

create_assignment() {
  local role="$1" assignee="$2" scope="$3" desc="$4"
  if az role assignment create \
       --role "$role" \
       --assignee-object-id "$assignee" \
       --assignee-principal-type ServicePrincipal \
       --scope "$scope" \
       --only-show-errors >/dev/null 2>&1; then
    echo "  + $desc"
  else
    # Likely already exists; verify
    if az role assignment list --assignee "$assignee" --scope "$scope" --role "$role" --query '[0].id' -o tsv | grep -q .; then
      echo "  = $desc (already exists)"
    else
      echo "  ! $desc FAILED" >&2
      return 1
    fi
  fi
}

# Backend identity grants (runtime app)
create_assignment "$ROLE_ACR_PULL"        "$BACKEND_PRINCIPAL" "$ACR_ID" "backend → AcrPull on ACR"
create_assignment "$ROLE_KV_SECRETS_USER" "$BACKEND_PRINCIPAL" "$KV_ID"  "backend → Key Vault Secrets User"

# CI identity grants (deploy pipeline)
create_assignment "$ROLE_ACR_PUSH"         "$CI_PRINCIPAL" "$ACR_ID" "ci → AcrPush on ACR"
create_assignment "$ROLE_AKS_CLUSTER_USER" "$CI_PRINCIPAL" "$AKS_ID" "ci → AKS Cluster User"

# Cosmos DB SQL role assignment (different API surface — uses az cosmosdb sql role assignment)
echo "=== Granting Cosmos DB SQL Data Contributor ==="
COSMOS_ROLE_ID=$(az cosmosdb sql role definition show \
  --account-name "$COSMOS_ACCOUNT" \
  --resource-group "$RG" \
  --id "$COSMOS_DATA_CONTRIBUTOR" \
  --query id -o tsv)

EXISTING=$(az cosmosdb sql role assignment list \
  --account-name "$COSMOS_ACCOUNT" \
  --resource-group "$RG" \
  --query "[?principalId=='${BACKEND_PRINCIPAL}' && roleDefinitionId=='${COSMOS_ROLE_ID}'].id" \
  -o tsv)

if [[ -z "$EXISTING" ]]; then
  az cosmosdb sql role assignment create \
    --account-name "$COSMOS_ACCOUNT" \
    --resource-group "$RG" \
    --scope "/" \
    --principal-id "$BACKEND_PRINCIPAL" \
    --role-definition-id "$COSMOS_ROLE_ID" \
    --only-show-errors >/dev/null
  echo "  + backend → Cosmos DB Data Contributor"
else
  echo "  = backend → Cosmos DB Data Contributor (already exists)"
fi

echo "=== Done ==="
