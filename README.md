![GPM Logo](assets/logo.png)

**Go Process Manager (GPM)** is a lightweight, production-ready process management tool written in Go that simplifies running and monitoring background applications **exclusively on Linux (Ubuntu)**. GPM features a resilient daemon, Unix domain socket IPC, an intuitive CLI with color-coded process list tables, log tailing/filtering capabilities, and native systemd service integration for automatically starting registered processes on system boot.

**Language-Agnostic Process Management:** GPM is completely language-agnostic and can manage, monitor, and daemonize **any** background process, script, or executable supported by Ubuntu Linux. Whether it is a Node.js API, a Go binary, a Python backend, a Java application, or a shell script, GPM can run it.

---

## Linux (Ubuntu) Installation and Setup

### One-Line Quick Install

```bash
curl -L -o gpm-linux-amd64 https://github.com/dennismutuku2005/gpm/releases/latest/download/gpm-linux-amd64
chmod +x gpm-linux-amd64
./gpm-linux-amd64 setup -y
```

*Running without `sudo` installs GPM to `~/.local/bin` for the current user. Running with `sudo ./gpm-linux-amd64 setup` installs GPM globally to `/usr/local/bin` and configures systemd autostart on system boot.*

---

## Verifying the Installation

To verify that GPM has been installed and configured correctly on your Ubuntu system, open a terminal and run the diagnostic tool:

```bash
gpm doctor
```

This will run checks on:
* **Permissions:** Current user privilege status (standard user or root/sudo).
* **Environment PATH:** Whether GPM is successfully registered and executable globally.
* **Configuration Store:** Read/write permissions of GPM config folders.
* **Startup Service:** The status of the GPM systemd autostart service.
* **Daemon Status:** Check if the background GPM daemon is online and responsive.

---

## CLI Commands and Usage

Once installed, use `gpm` to start and manage background processes:

### Management Commands

| Command | Description | Example |
| :--- | :--- | :--- |
| `gpm start <name>` | Starts and monitors a background process. | `gpm start my-api --cmd "node app.js" --dir "/var/www/api"` |
| `gpm list` | Lists all processes managed by GPM. | `gpm list` |
| `gpm status` | Shows GPM daemon statistics and process summary. | `gpm status` |
| `gpm stop <name>` | Gracefully stops a running process. | `gpm stop my-api` |
| `gpm restart <name>` | Restarts a managed process execution. | `gpm restart my-api` |
| `gpm delete <name>` | Stops and removes a process config and its logs. | `gpm delete my-api` |
| `gpm logs <name>` | Display stdout/stderr logs. Supports tail/query. | `gpm logs my-api --lines 50` |
| `gpm enable <name>` | Enables process autostart on system boot. | `gpm enable my-api` |
| `gpm disable <name>` | Disables process autostart on system boot. | `gpm disable my-api` |
| `gpm startup` | Installs GPM systemd service for boot autostart. | `sudo gpm startup` |
| `gpm shutdown` | Gracefully terminates all processes and stops daemon. | `gpm shutdown` |

---

## Operating System Requirement

> [!IMPORTANT]
> GPM is designed exclusively for **Linux (Ubuntu)**. Running GPM on Windows or other non-Linux operating systems is not supported.

---

## Development & Building

GPM relies on Linux system APIs (`systemd`, Unix domain sockets, process signals). When developing or building GPM on Windows or macOS, set `GOOS=linux` during compilation:

**PowerShell (Windows):**
```powershell
$env:GOOS="linux"; go build ./...
```

**Bash / Linux / WSL:**
```bash
GOOS=linux go build ./...
```

---


## License

This project is licensed under the GPM Non-Commercial License. See the [LICENSE](LICENSE) file for details.
