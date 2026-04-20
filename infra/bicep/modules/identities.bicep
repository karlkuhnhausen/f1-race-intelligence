@description('Base name')
param baseName string
param location string
param tags object
param aksOidcIssuer string
param acrId string
param cosmosAccountName string
param keyVaultName string

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

// Grant backend identity AcrPull on ACR
resource acrPull 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(acrId, backendIdentity.id, 'acrpull')
  scope: resourceGroup()
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '7f951dda-4ed3-4680-a7ca-43fe172d538d') // AcrPull
    principalId: backendIdentity.properties.principalId
    principalType: 'ServicePrincipal'
  }
}

// Grant backend identity Cosmos DB Built-in Data Contributor
resource cosmosAccount 'Microsoft.DocumentDB/databaseAccounts@2024-02-15-preview' existing = {
  name: cosmosAccountName
}

resource cosmosDataContributor 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(cosmosAccount.id, backendIdentity.id, 'cosmoscontributor')
  scope: cosmosAccount
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '00000000-0000-0000-0000-000000000002') // Cosmos DB Built-in Data Contributor
    principalId: backendIdentity.properties.principalId
    principalType: 'ServicePrincipal'
  }
}

// Grant backend identity Key Vault Secrets User
resource kv 'Microsoft.KeyVault/vaults@2023-07-01' existing = {
  name: keyVaultName
}

resource kvSecretsUser 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(kv.id, backendIdentity.id, 'kvsecretsuser')
  scope: kv
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '4633458b-17de-408a-b874-0445c86b69e6') // Key Vault Secrets User
    principalId: backendIdentity.properties.principalId
    principalType: 'ServicePrincipal'
  }
}

output backendClientId string = backendIdentity.properties.clientId
output backendPrincipalId string = backendIdentity.properties.principalId
