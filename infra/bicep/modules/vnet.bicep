@description('Base name')
param baseName string
param location string
param tags object

resource vnet 'Microsoft.Network/virtualNetworks@2024-01-01' = {
  name: 'vnet-${baseName}'
  location: location
  tags: tags
  properties: {
    addressSpace: {
      addressPrefixes: [
        '10.1.0.0/16'
      ]
    }
    subnets: [
      {
        name: 'snet-aks'
        properties: {
          addressPrefix: '10.1.0.0/20'
        }
      }
      {
        name: 'snet-services'
        properties: {
          addressPrefix: '10.1.16.0/24'
        }
      }
    ]
  }
}

// Use resourceId() to construct deterministic subnet IDs that ARM can validate at preflight
// (reference()-based outputs fail AKS preflight when the VNet doesn't exist yet)
output vnetId string = vnet.id
output vnetName string = vnet.name
output aksSubnetId string = resourceId('Microsoft.Network/virtualNetworks/subnets', vnet.name, 'snet-aks')
output servicesSubnetId string = resourceId('Microsoft.Network/virtualNetworks/subnets', vnet.name, 'snet-services')
