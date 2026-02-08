<p align="center">
  <img src="https://github.com/kaliv0/homie/blob/main/assets/homie.jpg?raw=true" width="300" alt="Homie">
</p>

<a href="https://github.com/kaliv0/homie/releases"><img src="https://img.shields.io/github/release/kaliv0/homie.svg" alt="Latest Release"></a>
<a href="https://pkg.go.dev/github.com/kaliv0/homie"><img src="https://pkg.go.dev/badge/github.com/kaliv0/homie.svg"></a>
<a href="https://github.com/kaliv0/homie/actions"><img src="https://github.com/kaliv0/homie/actions/workflows/build.yml/badge.svg?branch=main" alt="Build Status"></a>

# Homie

### Terminal-based Clipboard Manager

Supports <i>fuzzy search</i>, <i>multi-select</i> and other adorable <i>chicaneries</i>.<br>

---

## Installation

```shell
$ go install github.com/kaliv0/homie@latest
```
On `linux` you would also need `xclip` or `xsel` installed as an external dependency

---

## Usage

```shell
$ homie start
```

Runs <i>homie</i> in a daemon process to track your clipboard.<br>
It stores all copied items in a sqlite3 `homie.db` file under <i>\$XDG_CONFIG_HOME/</i> or <i>\$HOME/.config/</i> path.

```shell
$ homie stop
```

Stop the daemon process. You will be able to open the <i>history window</i>,<br>
search and select (and of course - paste) items from it,
but <i>homie</i> won't track any new changes in the clipboard.

```shell
$ homie history
```

Opens a preview window of the copied chronology.<br>
(Running with the <i>--limit \<n></i> flag retrieves only the last <i>n</i> items. Default limit value: 20)<br>
<br>
The history window comes with integrated fuzzy_search that checks the loaded records against a desired pattern.<br>
If nothing is found, <i>homie</i> pulls more (paginated) records from the database.<br>
<br>
After selecting an record and closing the window, <i>homie</i> puts the text inside the clipboard (ready the be pasted wherever needed).<br>
(NB: You can select multiple items by pinning them with the <i>tab</i> key. They will be added to your clipboard buffer as a single string separted by spaces.)<br>
To paste the text directly in your terminal run the `history` command with <i>--paste</i>.<br>

```shell
$ homie clear
```

Deletes all items from the `homie.db` store

---

## External configuration

```shell
$ homie shell
```

Generates a shell configuration for your `.bashrc` that will start the application automatically<br>
as well add extra key bindings for opening the preview window.

```shell
$ homie completion
```

Generates a shell configuration for the `.bash_completion` file that will enable auto_complete for all <i>homie</i> commands<br>

```shell
$ homie tmux
```

Generates a tmux integration script for your `.tmux.conf`. Requires tmux 3.2+ (for `display-popup` support).<br>

---

When you start <i>homie</i> it will automatically stop other running instances of the application if any.<br>
After that it will scan the database and if there are records above certain limit (default size: 500) it will purge them leaving only a minimum amount (default: 20)

- You can control this behavior creating a <i>.homierc</i> config inside your <i>root</i> directory.
  (See [.homierc example](https://github.com/kaliv0/homie/blob/main/examples/.homierc))
- Using `ttl` strategy will delete the oldest records (specified in the config as <i>ttl: \<days></i>) disregarding the total amount of items in the db.
- To disable entirely the <i>history clean-up</i> phase, put <i>clean_up: false</i> in the `.homierc`.

---

<b>Key bindings</b>:
- <i>Ctrl + h</i> (<i>prefix + h</i> if inside a tmux session) - opens clipboard history popup (copies selection to system clipboard)
- <i>Ctrl + p</i> (<i>prefix + p</i>) - opens clipboard history popup and pastes selected item

You can tweak and customize those in your `.bashrc` and `.tmux.conf` files.

---

## Known limitations

Currently <i>homie</i> is designed for `bash` and `tmux` only.<br>

<p align="center">
  <img src="https://github.com/kaliv0/homie/blob/main/assets/doh.gif?raw=true" width="300" alt="D'OH">
</p>
