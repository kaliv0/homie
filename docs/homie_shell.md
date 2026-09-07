## homie shell

Generate a shell integration script

### Synopsis

To enable shell integration execute:

bash:
echo 'source <(homie shell bash)' >> ~/.bashrc

zsh:
echo 'source <(homie shell zsh)' >> ~/.zshrc

Then reload your shell.

(On macOS bash, ensure ~/.bash_profile sources ~/.bashrc.)

```
homie shell [bash|zsh]
```

### Options

```
  -h, --help   help for shell
```

### SEE ALSO

- [homie](homie.md) - Terminal-based clipboard manager
