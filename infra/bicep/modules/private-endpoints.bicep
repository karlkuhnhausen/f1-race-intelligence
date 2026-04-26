@description('Base name')
param baseName string
param location string
param tags object
param subnetId string
param cosmosAccountId string
param cosmosPrivateDnsZoneId string

resource cosmosPrivateEndpoint 'Microsoft.Network/privateEndpoints@2024-01-01' = {
  name: 'pe-cosmos-${baseName}'
  location: location
  tags: tags
  properties: {
    subnet: {
      id: subnetId
    }
    privateLinkServiceConnections: [
      {
        name: 'cosmos-${baseName}'
        properties: {
          privateLinkServiceId: cosmosAccountId
          groupIds: [
            'Sql'
          ]
        }
      }
    ]
  }
}

resource dnsZoneGroup 'Microsoft.Network/privateEndpoints/privateDnsZoneGroups@2024-01-01' = {
  parent: cosmosPrivateEndpoint
  name: 'cosmos-dns-group'
  properties: {
    privateDnsZoneConfigs: [
      {
        name: 'privatelink-documents-azure-com'
        properties: {
          privateDnsZoneId: cosmosPrivateDnsZoneId
        }
      }
    ]
  }
}

output privateEndpointId string = cosmosPrivateEndpoint.id
