targetScope = 'subscription'

@description('Base name for all resources')
param baseName string = 'f1raceintel'

@description('Azure region')
param location string = 'westus3'

@description('GitHub org/repo for OIDC federation')
param githubRepo string = 'karlkuhnhausen/f1-race-intelligence'

@description('AKS node count')
param aksNodeCount int = 2

@description('AKS node VM size')
param aksNodeVmSize string = 'Standard_B2s'

@description('Monthly budget alert threshold in USD')
param budgetThreshold int = 150

var rgName = 'rg-${baseName}'
var tags = {
  project: 'f1-race-intelligence'
  managedBy: 'bicep'
}

resource rg 'Microsoft.Resources/resourceGroups@2024-03-01' = {
  name: rgName
  location: location
  tags: tags
}

module logAnalytics 'modules/log-analytics.bicep' = {
  scope: rg
  name: 'logAnalytics'
  params: {
    baseName: baseName
    location: location
    tags: tags
  }
}

module acr 'modules/acr.bicep' = {
  scope: rg
  name: 'acr'
  params: {
    baseName: baseName
    location: location
    tags: tags
  }
}

module cosmosDb 'modules/cosmos-db.bicep' = {
  scope: rg
  name: 'cosmosDb'
  params: {
    baseName: baseName
    location: location
    tags: tags
  }
}

module keyVault 'modules/key-vault.bicep' = {
  scope: rg
  name: 'keyVault'
  params: {
    baseName: baseName
    location: location
    tags: tags
  }
}

module aks 'modules/aks.bicep' = {
  scope: rg
  name: 'aks'
  params: {
    baseName: baseName
    location: location
    tags: tags
    nodeCount: aksNodeCount
    nodeVmSize: aksNodeVmSize
    logAnalyticsWorkspaceId: logAnalytics.outputs.workspaceId
  }
}

module identities 'modules/identities.bicep' = {
  scope: rg
  name: 'identities'
  params: {
    baseName: baseName
    location: location
    tags: tags
    aksOidcIssuer: aks.outputs.oidcIssuerUrl
    acrId: acr.outputs.acrId
    cosmosAccountName: cosmosDb.outputs.accountName
    keyVaultName: keyVault.outputs.vaultName
  }
}

module ciIdentity 'modules/ci-identity.bicep' = {
  scope: rg
  name: 'ciIdentity'
  params: {
    baseName: baseName
    location: location
    tags: tags
    githubRepo: githubRepo
    acrId: acr.outputs.acrId
    aksId: aks.outputs.aksId
  }
}

output resourceGroupName string = rg.name
output aksName string = aks.outputs.aksName
output acrLoginServer string = acr.outputs.loginServer
output cosmosEndpoint string = cosmosDb.outputs.endpoint
output keyVaultUri string = keyVault.outputs.vaultUri
output backendIdentityClientId string = identities.outputs.backendClientId
output ciClientId string = ciIdentity.outputs.clientId
output ciTenantId string = ciIdentity.outputs.tenantId
output ciSubscriptionId string = subscription().subscriptionId
