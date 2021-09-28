## vdoctl deploy

Deploy vSphere Kubernetes Driver Operator

### Synopsis

This command helps to deploy VDO on the kubernetes cluster targeted by --kubeconfig flag or KUBECONFIG environment variable.
Currently the command supports deployment on vanilla k8s cluster

```
vdoctl deploy --spec <path to spec file> (can be http or file based url's) [flags]
```

### Options

```
  -h, --help          help for deploy
      --spec string   url to vdo deployment spec file
```

### Options inherited from parent commands

```
      --config string       config file (default is $HOME/.vdoctl.yaml)
      --kubeconfig string   points to the kubeconfig file of the target k8s cluster
```

### SEE ALSO

* [vdoctl](vdoctl.md)	 - VDO Command Line

