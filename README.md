![GPM Logo](assets/logo.png)

**Go Process Manager (GPM)** is a lightweight, production-ready, cross-platform process management tool written in Go that simplifies running and monitoring background applications. GPM features a resilient daemon, an intuitive command-line interface with color-coded process list tables, log tailing/filtering capabilities, and native service management support for automatically starting registered processes on system boot (via Windows Services and Linux systemd).

---

## 💻 Windows Installation & Setup

### Option A: Using the Pre-compiled Binary (Quickest)

1. **Download the Windows executable**:
   * [Windows Binary (gpm-windows-amd64.exe)](https://github.com/dennismutuku2005/gpm/releases/latest/download/gpm-windows-amd64.exe)
2. **Open PowerShell as Administrator**:
   * Press `Win + X` and select **Terminal (Admin)** or **Windows PowerShell (Admin)**.
3. **Navigate to the folder where you downloaded the binary** (for example, your Downloads folder):
   ```powershell
   cd ~/Downloads
   ```
4. **Run the interactive setup**:
   ```powershell
   .\gpm-windows-amd64.exe setup
   ```
   * *This utility will prompt you to confirm installation and specify the destination folder (defaulting to `C:\Program Files\GPM`). It copies the binary as `gpm.exe`, registers it in your system's `PATH`, and sets up the Windows service startup daemon.*
5. **Restart your PowerShell or Terminal session** to load the updated `PATH`.

### Option B: Build and Install from Source (Requires Go)

1. **Clone the repository**:
   ```powershell
   git clone https://github.com/dennismutuku2005/gpm.git
   cd gpm
   ```
2. **Run the installation script in PowerShell as Administrator**:
   ```powershell
   Set-ExecutionPolicy Bypass -Scope Process -Force
   .\scripts\install-windows.ps1
   ```

---

## 🐧 Linux Installation & Setup

### Option A: Using the Pre-compiled Binary (Quickest)

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
   * *This utility will prompt you to confirm installation. It copies the binary to `/usr/local/bin/gpm` and registers the daemon automatically using systemd.*

### Option B: Build and Install from Source (Requires Go)

1. **Clone the repository**:
   ```bash
   git clone https://github.com/dennismutuku2005/gpm.git
   cd gpm
   ```
2. **Run the installation script**:
   ```bash
   chmod +x scripts/install-linux.sh
   ./scripts/install-linux.sh
   ```

---

## 🩺 Verifying the Installation

To verify that GPM has been installed and configured correctly, open a **new** terminal/PowerShell window and run the diagnostic tool:

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

## 🚀 CLI Commands & Usage

Once installed, use `gpm` to start and manage background processes:

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

### Alternative Go Installation Method
If you have Go installed on your machine and prefer using Go tools:
```bash
go install github.com/dennismutuku2005/gpm/cmd/gpm@latest
```
