## homie stop

Stop the clipboard manager daemon.<br>
Sends SIGTERM to the PID recorded in the daemon pidfile.

```
homie stop
```

### Behavior

homie reads the pidfile, sends SIGTERM to the daemon process, and returns.<br>
If no daemon is running, stop is a no-op.

### Options

```
  -h, --help   help for stop
```

### SEE ALSO

* [homie](homie.md)	 - Terminal-based clipboard manager
* [homie start](homie_start.md)	 - Start clipboard manager
* [homie restart](homie_restart.md)	 - Restart clipboard manager
* [homie status](homie_status.md)	 - Show daemon status
