## vdoctl completion powershell

generate the autocompletion script for powershell

### Synopsis


Generate the autocompletion script for powershell.

To load completions in your current shell session:
PS C:\> vdoctl completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.


```
vdoctl completion powershell [flags]
```

### Options

```
  -h, --help              help for powershell
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --config string       config file (default is $HOME/.vdoctl.yaml)
      --kubeconfig string   points to the kubeconfig file of the target k8s cluster
```

### SEE ALSO

* [vdoctl completion](vdoctl_completion.md)	 - generate the autocompletion script for the specified shell

