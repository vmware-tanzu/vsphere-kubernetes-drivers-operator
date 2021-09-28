## vdoctl completion bash

generate the autocompletion script for bash

### Synopsis


Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:
$ source <(vdoctl completion bash)

To load completions for every new session, execute once:
Linux:
  $ vdoctl completion bash > /etc/bash_completion.d/vdoctl
MacOS:
  $ vdoctl completion bash > /usr/local/etc/bash_completion.d/vdoctl

You will need to start a new shell for this setup to take effect.
  

```
vdoctl completion bash
```

### Options

```
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --config string       config file (default is $HOME/.vdoctl.yaml)
      --kubeconfig string   points to the kubeconfig file of the target k8s cluster
```

### SEE ALSO

* [vdoctl completion](vdoctl_completion.md)	 - generate the autocompletion script for the specified shell

