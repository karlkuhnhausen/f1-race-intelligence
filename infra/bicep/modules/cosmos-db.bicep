@description('Base name')
param baseName string
param location string
param tags object

var accountName = 'cosmos-${baseName}'

resource cosmosAccount 'Microsoft.DocumentDB/databaseAccounts@2024-02-15-preview' = {
  name: accountName
  location: location
  tags: tags
  kind: 'GlobalDocumentDB'
  properties: {
    databaseAccountOfferType: 'Standard'
    locations: [
      {
        locationName: location
        failoverPriority: 0
      }
    ]
    capabilities: [
      {
        name: 'EnableServerless'
      }
    ]
    consistencyPolicy: {
      defaultConsistencyLevel: 'Session'
    }
  }
}

resource database 'Microsoft.DocumentDB/databaseAccounts/sqlDatabases@2024-02-15-preview' = {
  parent: cosmosAccount
  name: 'f1raceintelligence'
  properties: {
    resource: {
      id: 'f1raceintelligence'
    }
  }
}

resource meetingsContainer 'Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers@2024-02-15-preview' = {
  parent: database
  name: 'meetings'
  properties: {
    resource: {
      id: 'meetings'
      partitionKey: {
        paths: ['/season']
        kind: 'Hash'
      }
    }
  }
}

resource standingsContainer 'Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers@2024-02-15-preview' = {
  parent: database
  name: 'standings'
  properties: {
    resource: {
      id: 'standings'
      partitionKey: {
        paths: ['/season']
        kind: 'Hash'
      }
    }
  }
}

output accountName string = cosmosAccount.name
output endpoint string = cosmosAccount.properties.documentEndpoint
