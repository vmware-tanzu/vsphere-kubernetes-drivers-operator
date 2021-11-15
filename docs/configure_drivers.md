#### Configure Drivers

VDO currently supports configuring CloudProvider (CPI) and StorageProvider(CSI)

##### CloudProvider
CloudProvider is an optional configuration. you can choose to skip this if you are not looking to install [Kubernetes vSphere Cloud Provider](https://github.com/kubernetes/cloud-provider-vsphere/)

If you want to install `Kubernetes vSphere Cloud Provider`, you will be taken through a series of configuration options to configure CPI

- IP address of vcenter
- Secure Connection - If you choose to establish a secure connection to vcenter, you need to provide a ssl thumbprint
- Login credentials for vcenter
- Datacenter(s) - you can provide a comma separated list of datacenters

```shell
Please provide the vcenter IP for configuring CloudProvider 
VC IP 10.10.10.10
Do you want to establish a secure connection? (Y/N) y
SSL Thumbprint █

Please provide the credentials for configuring CloudProvider
Username user
Password *******
Datacenter(s) dc0, dc1
```

Upon successful validation of the above information, you will be asked for the next set of configuration. At this point, you can choose to configure another VC, if you want CPI to work with multiple vcenters
```shell
Do you want to configure another vcenter for CloudProvider? (Y/N) y

VC IP 11.11.11.11
Do you want to establish a secure connection? (Y/N) y
SSL Thumbprint █

Please provide the credentials for configuring CloudProvider
Username user
Password *******
Datacenter(s) dc2, dc3
```

Once done you can choose to configure zones/regions if required. Please note, the tags for zone/regions need to be available in vcenter. Please refer [CPI](https://github.com/kubernetes/cloud-provider-vsphere/blob/master/docs/book/tutorials/deploying_cpi_and_csi_with_multi_dc_vc_aka_zones.md) documentation on how to configure zones/regions 

```shell
Do you want to configure zones/regions for CloudProvider? (Y/N) y
Zones zonea
Regions region1█
```

##### StorageProvider
Configuration of CSI requires you to enter the configuration of vcenter with which you wish CSI to work

If you have configured more than once vcenter for CPI, you would be asked to choose one of the vcenter's to configure CSI.

If not, we will use the same vcenter ip address for CSI and CPI drivers
```shell 
? Please select vcenter for configuring StorageProvider: 
+   
  ▸ 10.10.10.10
    11.11.11.11
```

As before with CPI, you will also need to provide the credentials to connect to vcenter and the datacenter
```shell
Please provide the credentials for configuring StorageProvider
Username user
Password *******
Datacenter(s) Datacenter
```

You can then provide custom Kubelet Path if required as vSphere CSI driver deployment provides an option to specify the path to kubelet.
```shell
Do you wish to provide custom kubelet Path? (Y/N) y
Kubelet Path /var/data/kubelet
```

Additionally, you can choose to configure Net Permissions for File volumes
```shell
Do you wish to configure File Volumes? (Y/N) y
Do you wish to configure vSAN DataStores for File Volumes (Y/N) y
vSAN DataStore Url(s) //ds:///vmfs/volumes/11111
Do you wish to configure Net permissions for File Volumes (Y/N) y
File Volumes IP Subnet 10.20.20.0/24
Use the arrow keys to navigate: ↓ ↑ → ← 
? Permissions type for File Volumes: 
+   
  ▸ READ_ONLY
    READ_WRITE
Allow Root Access? (Y/N) y█
```

To get more info on Net Permissions, please refer [CSI](https://vsphere-csi-driver.sigs.k8s.io/driver-deployment/installation.html#vsphereconf_for_file) document


This completes VDO configuration. You can check the status of drivers using `vdoctl status` command.