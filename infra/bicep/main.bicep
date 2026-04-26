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

@description('Authorized IP ranges for the AKS API server (comma-separated CIDRs). Set via ADMIN_IP_RANGES GitHub secret — do not commit real IPs here.')
param aksAuthorizedIPRanges array = []

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

module vnet 'modules/vnet.bicep' = {
  scope: rg
  name: 'vnet'
  params: {
    baseName: baseName
    location: location
    tags: tags
  }
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
    vnetSubnetId: vnet.outputs.aksSubnetId
    authorizedIPRanges: aksAuthorizedIPRanges
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
  }
}

module privateDns 'modules/private-dns.bicep' = {
  scope: rg
  name: 'privateDns'
  params: {
    baseName: baseName
    tags: tags
    vnetId: vnet.outputs.vnetId
  }
}

module privateEndpoints 'modules/private-endpoints.bicep' = {
  scope: rg
  name: 'privateEndpoints'
  params: {
    baseName: baseName
    location: location
    tags: tags
    subnetId: vnet.outputs.servicesSubnetId
    cosmosAccountId: cosmosDb.outputs.accountId
    cosmosPrivateDnsZoneId: privateDns.outputs.cosmosPrivateDnsZoneId
  }
}

output resourceGroupName string = rg.name
output aksName string = aks.outputs.aksName
output acrName string = acr.outputs.acrName
output acrLoginServer string = acr.outputs.loginServer
output cosmosAccountName string = cosmosDb.outputs.accountName
output cosmosEndpoint string = cosmosDb.outputs.endpoint
output keyVaultName string = keyVault.outputs.vaultName
output keyVaultUri string = keyVault.outputs.vaultUri
output backendIdentityClientId string = identities.outputs.backendClientId
output backendPrincipalId string = identities.outputs.backendPrincipalId
output ciClientId string = ciIdentity.outputs.clientId
output ciPrincipalId string = ciIdentity.outputs.principalId
output ciTenantId string = ciIdentity.outputs.tenantId
output ciSubscriptionId string = subscription().subscriptionId
