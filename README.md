![GPM Logo](assets/logo.png)

**Go Process Manager (GPM)** is a lightweight, production-ready, cross-platform process management tool written in Go that simplifies running and monitoring background applications. GPM features a resilient daemon, an intuitive command-line interface with color-coded process list tables, log tailing/filtering capabilities, and native service management support for automatically starting registered processes on system boot (via Windows Services and Linux systemd).

## Installation & Diagnostics

### Step 1: Download the pre-compiled binary
Download the compiled binary for your platform from the GitHub Releases page:
- **[Linux Binary](https://github.com/dennismutuku2005/gpm/releases/latest/download/gpm-linux-amd64)**
- **[Windows Binary](https://github.com/dennismutuku2005/gpm/releases/latest/download/gpm-windows-amd64.exe)**

---

### Step 2: Run the Interactive Installer

The `gpm setup` command automatically copies the binary to your system program folder, registers the executable in your system PATH, and configures the automatic startup service.

#### Windows (Run PowerShell as Administrator):
```powershell
# Run the downloaded binary with the setup command:
.\gpm-windows-amd64.exe setup
```

#### Ubuntu / Linux:
```bash
# Rename the downloaded binary and make it executable:
mv gpm-linux-amd64 gpm
chmod +x gpm

# Run setup with root privileges:
sudo ./gpm setup
```

---

### Step 3: Verify with Diagnostics (`gpm doctor`)

Run `gpm doctor` from any standard terminal session to diagnose and verify the health of your installation:
```bash
gpm doctor
```
It will inspect:
- Current user privileges
- System environment PATH configuration
- Running status of the GPM background daemon
- OS startup integration configuration
- Read/write permissions for configurations and logs

---

### Alternative Method: Install via Go
If you have Go installed on your system, you can install GPM directly with:
```bash
go install github.com/dennismutuku2005/gpm/cmd/gpm@latest
```
