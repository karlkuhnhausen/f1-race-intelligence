@description('Base name')
param baseName string
param location string
param tags object
param githubRepo string
param acrId string
param aksId string

// CI/CD identity used by GitHub Actions via OIDC
resource ciIdentity 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: 'id-${baseName}-ci'
  location: location
  tags: tags
}

// Federated credential for GitHub Actions OIDC on main/master branch
resource ciMainFedCred 'Microsoft.ManagedIdentity/userAssignedIdentities/federatedIdentityCredentials@2023-01-31' = {
  parent: ciIdentity
  name: 'github-main'
  properties: {
    issuer: 'https://token.actions.githubusercontent.com'
    subject: 'repo:${githubRepo}:ref:refs/heads/master'
    audiences: ['api://AzureADTokenExchange']
  }
}

// Federated credential for GitHub Actions OIDC on 'infrastructure' environment
resource ciInfraFedCred 'Microsoft.ManagedIdentity/userAssignedIdentities/federatedIdentityCredentials@2023-01-31' = {
  parent: ciIdentity
  name: 'github-infra-env'
  properties: {
    issuer: 'https://token.actions.githubusercontent.com'
    subject: 'repo:${githubRepo}:environment:infrastructure'
    audiences: ['api://AzureADTokenExchange']
  }
}

// Grant CI identity AcrPush on ACR
resource ciAcrPush 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(acrId, ciIdentity.id, 'acrpush')
  scope: resourceGroup()
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '8311e382-0749-4cb8-b61a-304f252e45ec') // AcrPush
    principalId: ciIdentity.properties.principalId
    principalType: 'ServicePrincipal'
  }
}

// Grant CI identity Azure Kubernetes Service Cluster User Role
resource aks 'Microsoft.ContainerService/managedClusters@2024-02-01' existing = {
  name: 'aks-${baseName}'
}

resource ciAksUser 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(aksId, ciIdentity.id, 'aksuser')
  scope: aks
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '4abbcc35-e782-43d8-92c5-2d3f1bd2253f') // Azure Kubernetes Service Cluster User Role
    principalId: ciIdentity.properties.principalId
    principalType: 'ServicePrincipal'
  }
}

output clientId string = ciIdentity.properties.clientId
output tenantId string = subscription().tenantId
