@description('Base name')
param baseName string
param location string
param tags object

var acrName = replace('acr${baseName}', '-', '')

resource acr 'Microsoft.ContainerRegistry/registries@2023-11-01-preview' = {
  name: acrName
  location: location
  tags: tags
  sku: {
    name: 'Basic'
  }
  properties: {
    adminUserEnabled: false
  }
}

output acrId string = acr.id
output loginServer string = acr.properties.loginServer
output acrName string = acr.name
