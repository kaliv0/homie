## .homierc

Optional YAML configuration for homie.

### Synopsis

Place the file at `~/.homierc`. A missing file is fine — defaults are used.
`homie history --limit` overrides `limit` from this file. `--paste` is CLI-only.

Example: [examples/.homierc](../examples/.homierc)

```
~/.homierc
```

Related paths (not set in `.homierc`):

- Database: `$XDG_CONFIG_HOME/homie/homie.db` or `~/.config/homie/homie.db`
- PID file (default): `$XDG_RUNTIME_DIR/homie.pid`

### Options

```
  limit      int     history items loaded per page (default 20)
  keep       int     rows kept after size-based cleanup (default 20)
  ttl        int     keep history this many days; 0 disables ttl cleanup (default 0)
  threshold  int     max stored records for size-based cleanup (default 500)
  tool       string  clipboard tool for homie history on Linux (see below)
  pid_file   string  daemon pidfile path; supports ~ (default: XDG runtime dir)
  log_file   string  append logs here, mode 0600 (default "")
  verbose    bool    diagnostic messages (default false)
```

Set `tool` to one of `xclip`, `xsel`, or `wl-clipboard` for `homie history` and tmux copy-pipe integration.
Required on Linux when copying selected history back to the clipboard.

On daemon start, entries older than `ttl` days are removed if `ttl > 0` - otherwise the oldest entries are trimmed when count exceeds `threshold`, preserving at most `keep` rows.

Negative `keep`, `threshold`, `ttl`, or `limit` are normalized at startup. Unknown `tool` is cleared with a warning. Malformed YAML fails at load.

### SEE ALSO

- [homie](homie.md) - Terminal-based clipboard manager
- [homie history](homie_history.md) - List clipboard history
- [homie start](homie_start.md) - Start clipboard manager
