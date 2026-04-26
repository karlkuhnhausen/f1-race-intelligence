@description('Base name')
param baseName string
param location string
param tags object
param aksOidcIssuer string

// Backend workload identity used by pods on AKS
resource backendIdentity 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: 'id-${baseName}-backend'
  location: location
  tags: tags
}

// Federated credential so AKS pods can use this identity
resource backendFedCred 'Microsoft.ManagedIdentity/userAssignedIdentities/federatedIdentityCredentials@2023-01-31' = {
  parent: backendIdentity
  name: 'aks-backend'
  properties: {
    issuer: aksOidcIssuer
    subject: 'system:serviceaccount:f1-race-intelligence:f1-backend'
    audiences: ['api://AzureADTokenExchange']
  }
}

// Role assignments for backend identity (AcrPull, Key Vault Secrets User, Cosmos Data Contributor)
// are NOT defined here. They are granted by infra/scripts/grant-roles.sh, run manually by an Owner.
// This keeps the CI managed identity at Contributor only and prevents privilege escalation through
// automation.

output backendClientId string = backendIdentity.properties.clientId
output backendPrincipalId string = backendIdentity.properties.principalId
