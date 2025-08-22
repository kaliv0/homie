<p align="center">
  <img src="https://github.com/kaliv0/homey/blob/main/assets/homey.jpg?raw=true" width="300" alt="Homey">
</p>

[![Go Reference](https://pkg.go.dev/badge/github.com/kaliv0/homey.svg)](https://pkg.go.dev/github.com/kaliv0/homey)

# Homey
### Terminal-based Clipboard Manager

Supports <i>fuzzy search</i>, <i>multi-select</i> and other adorable <i>chicaneries</i>.<br>

-------------
## Installation
``` bash
$ go install github.com/kaliv0/homey@latest
```

-------------
## Usage
``` go
$ homey start
```
Runs <i>homey</i> in a daemon process to track your clipboard.<br> 
It stores all copied items in a sqlite3 `homey.db` file under <i>\$XDG_CONFIG_HOME/</i> or <i>\$HOME/.config/</i> path.

``` go
$ homey stop
```
Stop the daemon process. You will be able to open the <i>history window</i>,<br> 
search and select (and of course - paste) items from it, 
but <i>homey</i> won't track any new changes in the clipboard.

``` go
$ homey history
```
Opens a preview window of the copied chronology.<br>
(Running with the <i>--limit \<n></i> flag retrieves only the last <i>n</i> items. Default limit value: 20)<br>
<br>
The history window comes with integrated fuzzy_search that checks the loaded records against a desired pattern.<br> 
If nothing is found, <i>homey</i> pulls more (paginated) records from the database.<br>
<br>
After selecting an record and closing the window, <i>homey</i> puts the text inside the clipboard (ready the be pasted wherever needed).<br>
(NB: You can select multiple items by pinning them with the <i>tab</i> key. They will be added to your clipboard buffer as a single string separted by spaces.)<br>
To paste the text directly in your terminal run the `history` command with <i>--paste</i>.<br>

``` go
$ homey clear
```
Deletes all items from the `homey.db` store

-------------
## External configuration

``` go
$ homey shell
```
Generates a shell configuration for your `.bashrc` that will start tha application automatically<br>
as well add extra key bindings for opening the preview window.

``` go
$ homey completion
```
Generates a shell configuration for the `.bash_completion` file that will enable auto_complete for all <i>homey</i> commands<br>

-------------
When you start <i>homey</i> it will automatically stop other running instances of the application if any.<br>
After that it will scan the database  and if there are records above certain limit (default size: 500) it will purge them leaving only a minimum amount (default: 20)
- You can control this behavior creating a <i>.homeyrc</i> config inside your <i>root</i> directory. 
(See [.homeyrc example](https://github.com/kaliv0/homey/blob/main/examples/.homeyrc))
- Using `ttl` strategy will delete the oldest records (specified in the config as <i>ttl: \<days></i>) disregarding the total amount of items in the db.
- To disable entirely the <i>history clean-up</i> phase, put <i>clean_up: false</i> in the `.homeyrc`.

-------------
## Known limitations
Currently <i>homey</i> is design only for `bash`, doesn't work with `tmux` yet (although it's fairly easy to implement).<br>
It is also tested only on Linux/Ubuntu so far.

<p align="center">
  <img src="https://github.com/kaliv0/homey/blob/main/assets/doh.gif?raw=true" width="300" alt="D'OH">
</p>
