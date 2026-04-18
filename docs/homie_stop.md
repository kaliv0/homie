## homie stop

Stop the clipboard manager daemon.<br>
Only processes that are running homie with the `run` subcommand (the background daemon) are terminated.<br>
Other homie commands—such as `homie start`, `homie history`, or one-off invocations—are not matched or killed.

```
homie stop
```

### Behavior

homie scans running processes, finds matching daemon processes (executable name `homie`, argv with `run` as the second token),<br>
and sends terminate to each match except the current process.

### Options

```
  -h, --help   help for stop
```

### SEE ALSO

* [homie](homie.md)	 - Terminal-based clipboard manager
* [homie start](homie_start.md)	 - Start clipboard manager
* [homie restart](homie_restart.md)	 - Restart clipboard manager
