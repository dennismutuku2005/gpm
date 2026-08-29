![GPM Logo](assets/logo.png)

**Go Process Manager (GPM)** is a lightweight, production-ready, cross-platform process management tool written in Go that simplifies running and monitoring background applications. GPM features a resilient daemon, an intuitive command-line interface with color-coded process list tables, log tailing/filtering capabilities, and native service management support for automatically starting registered processes on system boot (via Windows Services and Linux systemd).

## Installation

### Method 1: Download Pre-compiled Binaries (No Go Required)
You can download the latest compiled binaries directly from the GitHub Releases page:
- **[Linux Binary](https://github.com/dennismutuku2005/gpm/releases/latest/download/gpm-linux-amd64)**
- **[Windows Binary](https://github.com/dennismutuku2005/gpm/releases/latest/download/gpm-windows-amd64.exe)**

#### Quick Install Command:

##### Ubuntu / Linux:
```bash
# Download the binary
curl -L -o gpm https://github.com/dennismutuku2005/gpm/releases/latest/download/gpm-linux-amd64

# Make executable and move to system PATH
chmod +x gpm
sudo mv gpm /usr/local/bin/

# Install the startup daemon service
sudo gpm startup
```

##### Windows (Run PowerShell as Administrator):
```powershell
# Create installation folder
New-Item -ItemType Directory -Force -Path "C:\Program Files\GPM"

# Download the binary
Invoke-WebRequest -Uri "https://github.com/dennismutuku2005/gpm/releases/latest/download/gpm-windows-amd64.exe" -OutFile "C:\Program Files\GPM\gpm.exe"

# Add to system environment PATH
$MachinePath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::Machine)
if ($MachinePath -split ';' -notcontains "C:\Program Files\GPM") {
    [Environment]::SetEnvironmentVariable("Path", $MachinePath + ";C:\Program Files\GPM", [EnvironmentVariableTarget]::Machine)
}

# Install GPM automatic startup service
& "C:\Program Files\GPM\gpm.exe" startup
```

---

### Method 2: Using `go install` (Requires Go)
```bash
go install github.com/dennismutuku2005/gpm/cmd/gpm@latest
```

### Method 3: From Source (Includes Service Autostart Registration)

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
