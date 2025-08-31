# CloudWatch Log TUI 🪵👀

A Command Line Interface (CLI) tool for viewing AWS CloudWatch Logs with a text-based user interface (TUI).

<img width="1910" height="989" alt="cloudwatch-log-tui" src="https://github.com/user-attachments/assets/3c9c1149-dfbd-46b0-83aa-83da467bb991" />

### 🚀 Features

- Browse and filter Log Groups
- Browse and filter Log Streams within a selected Log Group
- View log events interactively
- Vim-like keybindings (`j` / `k`) for intuitive navigation


### 🛠️ Installation


```bash
brew tap ryutaro-asada/cloudwatch-log-tui
brew install cloudwatch-log-tui
```

binary is available at [here](https://github.com/ryutaro-asada/cloudwatch-log-tui/releases)

### ▶️ Usage
To run the application:

```bash
cloudwatch-log-tui
```
### ⌨️ Keybindings

#### Log Group Panel
| Action               | Key       |
|----------------------|-----------|
| Move Up/Down         | j / k     |
| Select Log Group     | Enter     |
| Filter Log Groups    | /         |

#### Log Stream Panel
| Action               | Key       |
|----------------------|-----------|
| Move Up/Down         | j / k     |
| Select Log Stream    | Enter     |
| Filter Log Streams   | /         |

#### Log Event Panel
| Action                    | Key   |
|---------------------------|--------|
| Move to Next Widget       | Tab    |
| Move Up/Down in Dropdown  | j / k  |
| Select Option in Dropdown | Enter  |
| Press Button              | Enter  |

### 📄 License

MIT License

### 🤝 Contributing

Contributions are welcome! Feel free to open issues or submit pull requests.
