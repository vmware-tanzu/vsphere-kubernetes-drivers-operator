## vdoctl completion zsh

generate the autocompletion script for zsh

### Synopsis


Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions for every new session, execute once:
# Linux:
$ vdoctl completion zsh > "${fpath[1]}/_vdoctl"
# macOS:
$ vdoctl completion zsh > /usr/local/share/zsh/site-functions/_vdoctl

You will need to start a new shell for this setup to take effect.


```
vdoctl completion zsh [flags]
```

### Options

```
  -h, --help              help for zsh
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --config string       config file (default is $HOME/.vdoctl.yaml)
      --kubeconfig string   points to the kubeconfig file of the target k8s cluster
```

### SEE ALSO

* [vdoctl completion](vdoctl_completion.md)	 - generate the autocompletion script for the specified shell

