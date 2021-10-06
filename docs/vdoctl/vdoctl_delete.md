## vdoctl delete

Delete vSphere Kubernetes Driver Operator

### Synopsis

This command deletes the VDO deployment and associated artifacts from the cluster targeted by --kubeconfig flag or KUBECONFIG environment variable.
Currently, the command supports vanilla k8s cluster

```
vdoctl delete [flags]
```

### Options

```
  -h, --help   help for delete
```

### Options inherited from parent commands

```
      --config string       config file (default is $HOME/.vdoctl.yaml)
      --kubeconfig string   points to the kubeconfig file of the target k8s cluster
```

### SEE ALSO

* [vdoctl](vdoctl.md)	 - VDO Command Line

