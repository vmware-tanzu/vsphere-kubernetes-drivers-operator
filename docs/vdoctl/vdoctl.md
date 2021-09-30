## vdoctl

VDO Command Line

### Synopsis

vdoctl is a command line interface for vSphere Kubernetes Drivers Operator.
vdoctl provides the user with basic set of commands required to install and configure VDO.
For example:
vdoctl deploy
vdoctl configure compat
vdoctl configure drivers
vdoctl store creds


### Options

```
      --config string       config file (default is $HOME/.vdoctl.yaml)
  -h, --help                help for vdoctl
      --kubeconfig string   points to the kubeconfig file of the target k8s cluster
```

### SEE ALSO

* [vdoctl configure](vdoctl_configure.md)	 - command to configure VDO
* [vdoctl delete](vdoctl_delete.md)	 - Delete vSphere Kubernetes Driver Operator
* [vdoctl deploy](vdoctl_deploy.md)	 - Deploy vSphere Kubernetes Driver Operator
* [vdoctl store](vdoctl_store.md)	 - A brief description of your command
* [vdoctl update](vdoctl_update.md)	 - Update the VDO Resources

