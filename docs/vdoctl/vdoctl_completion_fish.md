## vdoctl completion fish

generate the autocompletion script for fish

### Synopsis


Generate the autocompletion script for the fish shell.

To load completions in your current shell session:
$ vdoctl completion fish | source

To load completions for every new session, execute once:
$ vdoctl completion fish > ~/.config/fish/completions/vdoctl.fish

You will need to start a new shell for this setup to take effect.


```
vdoctl completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --config string       config file (default is $HOME/.vdoctl.yaml)
      --kubeconfig string   points to the kubeconfig file of the target k8s cluster
```

### SEE ALSO

* [vdoctl completion](vdoctl_completion.md)	 - generate the autocompletion script for the specified shell

