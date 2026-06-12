## homie status

Show whether the clipboard manager daemon is running.

```
homie status
```

### Behavior

- `running (pid N)`: daemon is running; exits 0.
- `not running`: no daemon holds the pidfile lock; exits 1.

### Pidfile location

Resolved in order:

1. `pid_file` in `~/.homierc`
2`$XDG_RUNTIME_DIR/homie.pid`
3`/run/user/$UID/homie.pid`

### Options

```
  -h, --help   help for status
```

### SEE ALSO

* [homie](homie.md)	 - Terminal-based clipboard manager
* [homie start](homie_start.md)	 - Start clipboard manager
* [homie stop](homie_stop.md)	 - Stop clipboard manager
