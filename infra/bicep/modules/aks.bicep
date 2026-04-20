@description('Base name')
param baseName string
param location string
param tags object
param nodeCount int
param nodeVmSize string
param logAnalyticsWorkspaceId string

@description('Subnet ID for AKS nodes.')
param vnetSubnetId string

resource aks 'Microsoft.ContainerService/managedClusters@2024-02-01' = {
  name: 'aks-${baseName}'
  location: location
  tags: tags
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    dnsPrefix: baseName
    kubernetesVersion: '1.33'
    agentPoolProfiles: [
      {
        name: 'system'
        count: nodeCount
        vmSize: nodeVmSize
        mode: 'System'
        osType: 'Linux'
        osSKU: 'AzureLinux'
        vnetSubnetID: vnetSubnetId
      }
    ]
    networkProfile: {
      networkPlugin: 'azure'
      networkPolicy: 'calico'
    }
    oidcIssuerProfile: {
      enabled: true
    }
    securityProfile: {
      workloadIdentity: {
        enabled: true
      }
    }
    addonProfiles: {
      omsagent: {
        enabled: true
        config: {
          logAnalyticsWorkspaceResourceID: logAnalyticsWorkspaceId
        }
      }
    }
  }
}

output aksName string = aks.name
output aksId string = aks.id
output oidcIssuerUrl string = aks.properties.oidcIssuerProfile.issuerURL
