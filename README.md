![GPM Logo](assets/logo.png)

**Go Process Manager (GPM)** is a lightweight, production-ready, cross-platform process management tool written in Go that simplifies running and monitoring background applications. GPM features a resilient daemon, an intuitive command-line interface with color-coded process list tables, log tailing/filtering capabilities, and native service management support for automatically starting registered processes on system boot (via Windows Services and Linux systemd).

**Language-Agnostic Process Management:** GPM is completely language-agnostic and can manage, monitor, and daemonize **any** background process, script, or executable supported by your operating system. Whether it is a Node.js API, a Go binary, a Python backend, a Java application, or a shell script, GPM can run it.

---

## Windows Installation and Setup

Run PowerShell as **Administrator** and execute:

```powershell
Invoke-WebRequest -Uri "https://github.com/dennismutuku2005/gpm/releases/latest/download/gpm-windows-amd64.exe" -OutFile "$HOME\Downloads\gpm-windows-amd64.exe"
cd $HOME\Downloads
.\gpm-windows-amd64.exe setup
```

*The setup utility copies `gpm.exe` to `C:\Program Files\GPM` (configurable), updates your system PATH, and configures the Windows startup service. Restart your PowerShell session after installation.*

---

## Linux Installation and Setup

### Using the Pre-compiled Binary

1. **Download the Linux binary**:
   ```bash
   curl -L -o gpm-linux-amd64 https://github.com/dennismutuku2005/gpm/releases/latest/download/gpm-linux-amd64
   ```
2. **Make it executable**:
   ```bash
   chmod +x gpm-linux-amd64
   ```
3. **Run the interactive setup with root privileges**:
   ```bash
   sudo ./gpm-linux-amd64 setup
   ```
   * This utility will prompt you to confirm installation. It copies the binary to /usr/local/bin/gpm and registers the daemon automatically using systemd.

---

## Verifying the Installation

To verify that GPM has been installed and configured correctly, open a new terminal or PowerShell window and run the diagnostic tool:

```bash
gpm doctor
```

This will run checks on:
* **Permissions:** Current user privilege status.
* **Environment PATH:** Whether GPM is successfully registered and executable globally.
* **Configuration Store:** Read/write permissions of GPM config folders.
* **Startup Service:** The status of the GPM autostart system service.
* **Daemon Status:** Check if the background GPM daemon is online and responsive.

---

## CLI Commands and Usage

Once installed, use gpm to start and manage background processes:

### Management Commands

| Command | Description | Example |
| :--- | :--- | :--- |
| `gpm start <name>` | Starts and monitors a background process. | `gpm start my-api --cmd "node app.js" --dir "/apps/api"` |
| `gpm list` | Lists all processes managed by GPM. | `gpm list` |
| `gpm status` | Shows GPM daemon statistics and process summary. | `gpm status` |
| `gpm stop <name>` | Gracefully stops a running process. | `gpm stop my-api` |
| `gpm restart <name>` | Restarts a managed process execution. | `gpm restart my-api` |
| `gpm delete <name>` | Stops and removes a process config and its logs. | `gpm delete my-api` |
| `gpm logs <name>` | Display stdout/stderr logs. Supports tail/query. | `gpm logs my-api --lines 50` |
| `gpm enable <name>` | Enables process autostart on system boot. | `gpm enable my-api` |
| `gpm disable <name>` | Disables process autostart on system boot. | `gpm disable my-api` |
| `gpm shutdown` | Gracefully terminates all processes and stops daemon. | `gpm shutdown` |

---

## License

This project is licensed under the GPM Non-Commercial License. See the LICENSE file for details.
