# CloudWatch Log TUI 🪵👀

A Command Line Interface (CLI) tool for viewing AWS CloudWatch Logs with a text-based user interface (TUI).

<img width="1910" height="989" alt="cloudwatch-log-tui" src="https://github.com/user-attachments/assets/3c9c1149-dfbd-46b0-83aa-83da467bb991" />

### 🚀 Features

- Browse and search Log Groups
- Browse Log Streams within a selected Log Group
- View log events interactively
- Vim-like keybindings (`j` / `k`) for intuitive navigation


### 🛠️ Installation


```bash
brew tap ryutaro-asada/cloudwatch-log-tui
brew trust ryutaro-asada/cloudwatch-log-tui
brew install cloudwatch-log-tui
```

Recent Homebrew versions refuse to load formulae from non-official taps until they are trusted.
`brew trust` records this once in `~/.homebrew/trust.json` (or `$XDG_CONFIG_HOME/homebrew/trust.json` if `XDG_CONFIG_HOME` is set).
To trust only this formula instead of the whole tap, run `brew trust --formula ryutaro-asada/cloudwatch-log-tui/cloudwatch-log-tui`.

To upgrade:

```bash
brew update && brew upgrade cloudwatch-log-tui
```

binary is available at [here](https://github.com/ryutaro-asada/cloudwatch-log-tui/releases)

### ▶️ Usage
To run the application:

```bash
cloudwatch-log-tui
```

AWS credentials and region are read from the usual environment variables and AWS config files.

### ⌨️ Keybindings

#### Log Group Panel
| Action                      | Key       |
|-----------------------------|-----------|
| Move Up/Down                | j / k     |
| Select Log Group            | Enter     |
| Search Log Groups           | /         |
| Leave the search field      | Enter / Esc |
| Move to Log Stream Panel    | Tab       |

The search runs as you type. It accepts the characters allowed in log group names (`A-Z a-z 0-9 . - _ / #`).
Moving onto `NextPage ...` / `... PrevPage` loads the next / previous page.

#### Log Stream Panel
| Action                      | Key       |
|-----------------------------|-----------|
| Move Up/Down                | j / k     |
| Add Log Stream and show its events | Enter |
| Move to Log Group Panel     | Tab       |

Each selected stream is marked with `x`; selecting more streams adds them (up to 100).
Selecting `All Log Streams` clears the selection, so events of every stream are shown.
Selecting another log group clears the selection.

#### Log Event Panel
| Action                      | Key    |
|-----------------------------|--------|
| Move to Next Widget         | Tab    |
| Change Option in Dropdown   | j / k  |
| Open Dropdown / Select Option | Enter |
| Close Dropdown              | Esc    |
| Back to Log Group Panel (from a closed dropdown) | Esc |
| Apply Filter Pattern        | Enter  |
| Press Button                | Enter  |

Changing a dropdown reloads the events. The filter pattern is applied when you press Enter in the filter field.
The view shows at most 1000 events.

`Save Button` writes all events of the current query to the file in `Write Output File`
(default: `output.txt` in the current directory), one event per line.
The status line at the bottom shows the absolute path and the number of events when the save is done.

### 📝 Log file

Errors are shown in the status line at the bottom and written to a log file:

- macOS: `~/Library/Caches/cloudwatch-log-tui/cloudwatch-log-tui.log`
- Linux: `$XDG_CACHE_HOME/cloudwatch-log-tui/cloudwatch-log-tui.log` (default `~/.cache/...`)

### 📄 License

MIT License

### 🤝 Contributing

Contributions are welcome! Feel free to open issues or submit pull requests.
