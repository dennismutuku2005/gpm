![GPM Logo](assets/logo.png)

**Go Process Manager (GPM)** is a lightweight, production-ready, cross-platform process management tool written in Go that simplifies running and monitoring background applications. GPM features a resilient daemon, an intuitive command-line interface with color-coded process list tables, log tailing/filtering capabilities, and native service management support for automatically starting registered processes on system boot (via Windows Services and Linux systemd).

## Installation

### Method 1: Using `go install` (Requires Go)
```bash
go install github.com/dennismutuku2005/gpm/cmd/gpm@latest
```

### Method 2: From Source (Includes Service Autostart Registration)

#### Ubuntu / Linux
1. Clone the repository:
   ```bash
   git clone https://github.com/dennismutuku2005/gpm.git
   cd gpm
   ```
2. Run the installer script:
   ```bash
   chmod +x scripts/install-linux.sh
   ./scripts/install-linux.sh
   ```

#### Windows
1. Clone the repository:
   ```powershell
   git clone https://github.com/dennismutuku2005/gpm.git
   cd gpm
   ```
2. Open PowerShell as **Administrator** and run the installer:
   ```powershell
   Set-ExecutionPolicy Bypass -Scope Process -Force
   .\scripts\install-windows.ps1
   ```
