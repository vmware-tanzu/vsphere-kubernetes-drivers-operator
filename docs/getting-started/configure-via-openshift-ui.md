## Configuring VDO via Openshift Web Console

To configure VDO using OpenShift UI please follow the below steps

#### Pre-requisite

Please ensure you have the following information:

- IP address or FQDN of Vcenter
- Secure Connection - If you choose to establish a secure connection to vcenter, you need to provide a ssl thumbprint
- Login credentials for vcenter
- Datacenter(s) - you can provide a comma separated list of datacenters, datacenters in which the kubernetes nodes are present. This is required by CPI and CSI to manage the cluster


### Step-1
Make sure the VDO is up and running, for this you can go to `Installed Operator` side menu and confirm the status of the operator as Succeeded.

### Step-2
Now that we have installed VDO, we need to setup few pre-requisite to configure VDO.
Let's create a secret to store the credentials with which you would want CSI and CPI to connect to VC. Please note if CPI and CSI use different credentials then create secret for each set of credentials. This will be required to configure cloud and storage providers.
Please refer a sample here.

Create the secret with the credentials for your VC of a type `Source secret`
Main Page --> Right Side Menu --> Workloads --> Create Secret(Source secret type)  
**Make sure you create secret in the `kube-system` namespace**
![](../images/create-secret.png)

### Step-3 
Click on the running operator from Step-1 and see the list of `Provided API's`.
![](../images/provided-apis.png)

### Step-4
Now lets start configuring VDO. 
At first Create an Instance of `Vsphere Cloud Config` resource. This resource represents the information required to connect to vcenter.
In the credentials field please enter the name of the secret configured in step 2.
Please note: if you have configured different secrets for CPI and CSI then please create an instance of vspherecloudconfig resource for each of the secret
![](../images/create-vsphere-cloud-config.png)


### Step-5
Once done with the above step lets now proceed to configure the cloud and storage drivers for vsphere.

You can start by  `Creating Instance` in `VDOConfig` from the Step-3.

**5.a** Configure CSI
![](../images/create-vdoconfig-1.png)

You can configure CSI by adding the `Vsphere Cloud Config` name which we created in Step-4.  
You can add cluster distribution as `Openshift`.  
Configure `Custom Kubelet Path` and `File Volumes` as required.

**5.b** Configure CPI [Optional]
![](../images/create-vdoconfig-2.png)

You can configure zones and regions if required.
You have to add the `Vsphere Cloud Config` name which we created in Step-4. 

Click on `Create` Button to finish configuring VDO.

Once done, you can check the status of these drivers by checking the `Pods` in Workload Section.





