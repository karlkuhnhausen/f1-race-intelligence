#!/usr/bin/env bash
# Provision Azure infrastructure for F1 Race Intelligence Dashboard.
# Run from repo root. Requires: az CLI authenticated, Bicep CLI installed.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BICEP_DIR="$SCRIPT_DIR/../bicep"

echo "=== Deploying F1 Race Intelligence infrastructure ==="

# Deploy Bicep at subscription scope
az deployment sub create \
  --name "f1raceintel-$(date +%Y%m%d%H%M%S)" \
  --location westus3 \
  --template-file "$BICEP_DIR/main.bicep" \
  --parameters "$BICEP_DIR/main.parameters.json" \
  --output json | tee /tmp/f1-deploy-output.json

echo ""
echo "=== Deployment complete ==="
echo ""

# Extract outputs
RG=$(jq -r '.properties.outputs.resourceGroupName.value' /tmp/f1-deploy-output.json)
AKS=$(jq -r '.properties.outputs.aksName.value' /tmp/f1-deploy-output.json)
ACR=$(jq -r '.properties.outputs.acrLoginServer.value' /tmp/f1-deploy-output.json)
COSMOS=$(jq -r '.properties.outputs.cosmosEndpoint.value' /tmp/f1-deploy-output.json)
KV=$(jq -r '.properties.outputs.keyVaultUri.value' /tmp/f1-deploy-output.json)
BACKEND_ID=$(jq -r '.properties.outputs.backendIdentityClientId.value' /tmp/f1-deploy-output.json)
CI_CLIENT=$(jq -r '.properties.outputs.ciClientId.value' /tmp/f1-deploy-output.json)
CI_TENANT=$(jq -r '.properties.outputs.ciTenantId.value' /tmp/f1-deploy-output.json)
CI_SUB=$(jq -r '.properties.outputs.ciSubscriptionId.value' /tmp/f1-deploy-output.json)

echo "Resource Group:       $RG"
echo "AKS Cluster:          $AKS"
echo "ACR Login Server:     $ACR"
echo "Cosmos DB Endpoint:   $COSMOS"
echo "Key Vault URI:        $KV"
echo "Backend Identity:     $BACKEND_ID"
echo ""
echo "=== GitHub Actions Secrets to configure ==="
echo "AZURE_CLIENT_ID:      $CI_CLIENT"
echo "AZURE_TENANT_ID:      $CI_TENANT"
echo "AZURE_SUBSCRIPTION_ID: $CI_SUB"
echo ""
echo "=== Get AKS credentials ==="
echo "az aks get-credentials --resource-group $RG --name $AKS"
echo ""
echo "=== Attach ACR to AKS ==="
az aks update \
  --resource-group "$RG" \
  --name "$AKS" \
  --attach-acr "$(jq -r '.properties.outputs.acrLoginServer.value' /tmp/f1-deploy-output.json | cut -d. -f1)" \
  --output none

echo "ACR attached to AKS."
echo ""
echo "=== Create namespace ==="
az aks get-credentials --resource-group "$RG" --name "$AKS" --overwrite-existing
kubectl create namespace f1-race-intelligence --dry-run=client -o yaml | kubectl apply -f -

echo ""
echo "=== Infrastructure ready ==="
echo "Next: configure GitHub secrets, then push to master to trigger CI/CD."
