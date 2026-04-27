@description('Base name')
param baseName string
param location string
param tags object

// NSG for AKS subnet. Defined explicitly so Azure Policy doesn't create an
// empty default NSG that drops all inbound traffic. Allows HTTP/HTTPS from
// the Internet to the ingress LB; Azure's default Allow rules cover VNet
// traffic and Azure LB health probes.
resource aksNsg 'Microsoft.Network/networkSecurityGroups@2024-01-01' = {
  name: 'nsg-${baseName}-snet-aks'
  location: location
  tags: tags
  properties: {
    securityRules: [
      {
        name: 'allow-http-from-internet'
        properties: {
          priority: 100
          direction: 'Inbound'
          access: 'Allow'
          protocol: 'Tcp'
          sourceAddressPrefix: 'Internet'
          sourcePortRange: '*'
          destinationAddressPrefix: '*'
          destinationPortRanges: [
            '80'
            '443'
          ]
        }
      }
    ]
  }
}

resource servicesNsg 'Microsoft.Network/networkSecurityGroups@2024-01-01' = {
  name: 'nsg-${baseName}-snet-services'
  location: location
  tags: tags
  properties: {
    securityRules: []
  }
}

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
          networkSecurityGroup: {
            id: aksNsg.id
          }
        }
      }
      {
        name: 'snet-services'
        properties: {
          addressPrefix: '10.1.16.0/24'
          networkSecurityGroup: {
            id: servicesNsg.id
          }
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
