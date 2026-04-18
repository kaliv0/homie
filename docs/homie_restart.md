## homie restart

Stop any running clipboard daemon, then start a new one (same spawn path as `homie start`).

```
homie restart
```

### Behavior

1. Terminate every other process that matches the daemon rule (executable `homie`, `run` as the second argv token).
2. Spawn a new daemon in the background.

If no daemon was running, the stop step is a no-op and a new daemon is still started.

### Options

```
  -h, --help   help for restart
```

### SEE ALSO

* [homie](homie.md)	 - Terminal-based clipboard manager
* [homie start](homie_start.md)	 - Start clipboard manager
* [homie stop](homie_stop.md)	 - Stop clipboard manager
