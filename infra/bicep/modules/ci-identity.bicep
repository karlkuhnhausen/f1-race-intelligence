@description('Base name')
param baseName string
param location string
param tags object
param githubRepo string

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

// Role assignments for CI identity (AcrPush, AKS Cluster User) are NOT defined here.
// They are granted by infra/scripts/grant-roles.sh, run manually by an Owner.
// This keeps the CI managed identity at Contributor only and prevents automation
// from being able to grant additional roles to itself or others.

output clientId string = ciIdentity.properties.clientId
output principalId string = ciIdentity.properties.principalId
output tenantId string = subscription().tenantId
