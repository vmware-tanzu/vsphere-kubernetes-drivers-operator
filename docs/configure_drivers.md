#### Configure Drivers

For configuring CloudProvider (if Required) and StorageProvider drivers, following command can be used.
```shell
vdoctl configure drivers
Do you want to configure CloudProvider? (Y/N) y
```
You then need to provide VC IP. Optionally, secure connection can also be established in which case you'll need to provide SSL thumbprint. 
```shell
Please provide the vcenter IP for configuring CloudProvider 
VC IP 10.10.10.10
Do you want to establish a secure connection? (Y/N) y
SSL Thumbprint █
```

You then need to provide the login credentials and Datacenter(s) for the vSphere Platform.

```shell
Please provide the credentials for configuring CloudProvider
Username user
Password *******
Datacenter(s) Datacenter
```

Above user inputs will be validated for configuration. Upon successful validation, more VCs for CloudProvider can be added in a similar fashion as per the requirements.

After you are done providing the details for all the VCs required, you can proceed to configure zones/regions for CloudProvider if desired. For more info on zones and regions, please refer the guide for zones/regions: [zones/regions](https://cloud-provider-vsphere.sigs.k8s.io/tutorials/deploying_cpi_and_csi_with_multi_dc_vc_aka_zones.html)


```shell
Do you want to configure zones/regions for CloudProvider? (Y/N) y
Zones zonea
Regions region1█
```


This completes the process for CloudProvider configuration. We'll then proceed to configuring StorageProvider.

In case of multiVC CloudProvider configuration, you can select which VC you need to configure StorageProvider for. If only one VC is configured for CloudProvider, Storage Provider will also be configured for the same VC IP.
After you're done providing the VC details, you'll have to provide the VC credentials and Datacenters. 
 
```shell
Please provide the credentials for configuring StorageProvider
Username user
Password *******
Datacenter(s) Datacenter
```

You can now configure File Volumes for StorageProvider if required. 

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
We are now done with configuring VDO. You can check the status for drivers using `vdoctl status` command.