## .homierc

Optional YAML configuration for homie.

### Synopsis

Place the file at `~/.homierc`. CLI flags override config where both apply.
A missing file is fine — defaults are used.

Example: [examples/.homierc](../examples/.homierc)

```
~/.homierc
```

Related paths (not set in `.homierc`):

- Database: `$XDG_CONFIG_HOME/homie/homie.db` or `~/.config/homie/homie.db`
- PID file (default): `$XDG_RUNTIME_DIR/homie.pid`

### Options

```
  limit      int     items shown in homie history (default 20); overridden by -l
  clean_up   bool    run history cleanup when the daemon starts (default false)
  ttl        int     keep history this many days; 0 disables ttl cleanup (default 0)
  max_size   int     max stored records for size-based cleanup (default 500)
  tool       string  clipboard tool for homie history on Linux (see below)
  pid_file   string  daemon pidfile path; supports ~ (default: XDG runtime dir)
  verbose    bool    diagnostic messages (default false); overridden by -v
  log_file   string  append logs here, mode 0600 (default ""); overridden by --log-file
```

Set `tool` to one of `xclip`, `xsel`, or `wl-clipboard` for `homie history` and tmux copy-pipe integration.
Required on Linux when copying selected history back to the clipboard.

When `clean_up: true`, entries older than `ttl` days are removed if `ttl > 0`; otherwise the oldest entries are trimmed when count exceeds `max_size`, keeping at most `limit` rows.

Invalid values (`limit` or `max_size` ≤ 0, negative `ttl`) are clamped at startup. Unknown `tool` is cleared with a warning. Malformed YAML fails at load.

### SEE ALSO

* [homie](homie.md)	 - Terminal-based clipboard manager
* [homie history](homie_history.md)	 - List clipboard history
* [homie start](homie_start.md)	 - Start clipboard manager
