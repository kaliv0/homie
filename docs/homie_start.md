## homie start

Start the clipboard manager daemon in the background (spawns `homie run` in a new session).<br>
If a daemon is already running, homie prints a message and exits without starting another one.

```
homie start
```

### Behavior

- `No daemon running`: starts the background daemon.
- `Daemon already running`: prints `homie daemon is already running` and exits successfully.

The daemon coordinates via a pidfile (default `$XDG_RUNTIME_DIR/homie.pid`, with fallbacks — see `homie status`).

### Options

```
  -h, --help   help for start
```

### SEE ALSO

* [homie](homie.md)	 - Terminal-based clipboard manager
* [homie stop](homie_stop.md)	 - Stop clipboard manager
* [homie restart](homie_restart.md)	 - Restart clipboard manager
* [homie status](homie_status.md)	 - Show daemon status
