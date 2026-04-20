#!/usr/bin/env bash
# Configure GitHub Actions secrets for OIDC-based Azure deployment.
# Run after deploy-infra.sh. Requires: gh CLI authenticated.
set -euo pipefail

echo "=== Setting up GitHub Actions OIDC secrets ==="
echo ""

if [[ ! -f /tmp/f1-deploy-output.json ]]; then
  echo "ERROR: Run deploy-infra.sh first to generate deployment outputs."
  exit 1
fi

CI_CLIENT=$(jq -r '.properties.outputs.ciClientId.value' /tmp/f1-deploy-output.json)
CI_TENANT=$(jq -r '.properties.outputs.ciTenantId.value' /tmp/f1-deploy-output.json)
CI_SUB=$(jq -r '.properties.outputs.ciSubscriptionId.value' /tmp/f1-deploy-output.json)
RG=$(jq -r '.properties.outputs.resourceGroupName.value' /tmp/f1-deploy-output.json)
AKS=$(jq -r '.properties.outputs.aksName.value' /tmp/f1-deploy-output.json)
ACR=$(jq -r '.properties.outputs.acrLoginServer.value' /tmp/f1-deploy-output.json)
COSMOS=$(jq -r '.properties.outputs.cosmosEndpoint.value' /tmp/f1-deploy-output.json)
KV=$(jq -r '.properties.outputs.keyVaultUri.value' /tmp/f1-deploy-output.json)
BACKEND_ID=$(jq -r '.properties.outputs.backendIdentityClientId.value' /tmp/f1-deploy-output.json)

echo "Setting AZURE_CLIENT_ID..."
gh secret set AZURE_CLIENT_ID --body "$CI_CLIENT"

echo "Setting AZURE_TENANT_ID..."
gh secret set AZURE_TENANT_ID --body "$CI_TENANT"

echo "Setting AZURE_SUBSCRIPTION_ID..."
gh secret set AZURE_SUBSCRIPTION_ID --body "$CI_SUB"

echo "Setting AKS_RESOURCE_GROUP..."
gh secret set AKS_RESOURCE_GROUP --body "$RG"

echo "Setting AKS_CLUSTER_NAME..."
gh secret set AKS_CLUSTER_NAME --body "$AKS"

echo "Setting ACR_LOGIN_SERVER..."
gh secret set ACR_LOGIN_SERVER --body "$ACR"

echo "Setting COSMOS_ENDPOINT..."
gh secret set COSMOS_ENDPOINT --body "$COSMOS"

echo "Setting KEYVAULT_URI..."
gh secret set KEYVAULT_URI --body "$KV"

echo "Setting BACKEND_IDENTITY_CLIENT_ID..."
gh secret set BACKEND_IDENTITY_CLIENT_ID --body "$BACKEND_ID"

echo ""
echo "=== GitHub secrets configured ==="
echo "Push to master to trigger the full CI/CD pipeline."
