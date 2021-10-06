## vdoctl update compatibility-matrix

Command to update the Compatibility matrix of Drivers

### Synopsis

This command helps to update the Compatibility matrix of Drivers, 
which in turns help to upgrade/downgrade the versions of CSI & CPI drivers.

```
vdoctl update compatibility-matrix [flags]
```

### Examples

```
vdoctl update matrix https://github.com/demo/demo.yaml
vdoctl update matrix file://var/sample/sample.yaml
```

### Options

```
  -h, --help   help for compatibility-matrix
```

### Options inherited from parent commands

```
      --config string       config file (default is $HOME/.vdoctl.yaml)
      --kubeconfig string   points to the kubeconfig file of the target k8s cluster
```

### SEE ALSO

* [vdoctl update](vdoctl_update.md)	 - Update the VDO Resources

