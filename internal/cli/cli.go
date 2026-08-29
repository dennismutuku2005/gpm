package cli

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"

	"github.com/dennismutuku2005/gpm/internal/config"
	"github.com/dennismutuku2005/gpm/internal/ipc"
	"github.com/dennismutuku2005/gpm/internal/logs"
	"github.com/dennismutuku2005/gpm/internal/manager"
	"github.com/dennismutuku2005/gpm/internal/process"
	"github.com/dennismutuku2005/gpm/internal/service"
)

// VersionInfo contains the current version of GPM.
const VersionInfo = "1.0.2"

// Runner handles the execution of GPM CLI commands.
type Runner struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// NewRunner creates a new Runner with stdout and stderr output destinations.
func NewRunner(stdout, stderr io.Writer) *Runner {
	return &Runner{
		Stdin:  os.Stdin,
		Stdout: stdout,
		Stderr: stderr,
	}
}

// Run parses arguments and routes them to the correct subcommand.
func (r *Runner) Run(args []string) int {
	if len(args) < 1 {
		r.PrintHelp()
		return 0
	}

	subCommand := args[0]
	switch subCommand {
	case "help", "-h", "--help":
		r.PrintHelp()
		return 0
	case "version", "-v", "--version":
		r.PrintVersion()
		return 0
	case "list":
		return r.connectAndRun(func(conn net.Conn) int {
			return r.executeList(conn)
		})
	case "status":
		return r.connectAndRun(func(conn net.Conn) int {
			return r.executeStatus(conn)
		})
	case "start":
		return r.handleStartCommand(args[1:])
	case "stop":
		if len(args) < 2 {
			fmt.Fprintln(r.Stderr, color.RedString("Error: process name is required. Usage: gpm stop <name>"))
			return 1
		}
		return r.connectAndRun(func(conn net.Conn) int {
			return r.executeSimpleCmd(conn, "stop", args[1])
		})
	case "restart":
		if len(args) < 2 {
			fmt.Fprintln(r.Stderr, color.RedString("Error: process name is required. Usage: gpm restart <name>"))
			return 1
		}
		return r.connectAndRun(func(conn net.Conn) int {
			return r.executeSimpleCmd(conn, "restart", args[1])
		})
	case "delete":
		if len(args) < 2 {
			fmt.Fprintln(r.Stderr, color.RedString("Error: process name is required. Usage: gpm delete <name>"))
			return 1
		}
		return r.connectAndRun(func(conn net.Conn) int {
			return r.executeSimpleCmd(conn, "delete", args[1])
		})
	case "enable":
		if len(args) < 2 {
			fmt.Fprintln(r.Stderr, color.RedString("Error: process name is required. Usage: gpm enable <name>"))
			return 1
		}
		return r.connectAndRun(func(conn net.Conn) int {
			return r.executeSimpleCmd(conn, "enable", args[1])
		})
	case "disable":
		if len(args) < 2 {
			fmt.Fprintln(r.Stderr, color.RedString("Error: process name is required. Usage: gpm disable <name>"))
			return 1
		}
		return r.connectAndRun(func(conn net.Conn) int {
			return r.executeSimpleCmd(conn, "disable", args[1])
		})
	case "logs":
		return r.handleLogsCommand(args[1:])
	case "startup":
		if len(args) > 1 && args[1] == "status" {
			return r.executeStartupStatus()
		}
		return r.executeStartupInstall()
	case "setup":
		return r.executeSetup(args[1:])
	case "doctor":
		return r.executeDoctor()
	case "shutdown":
		return r.connectAndRun(func(conn net.Conn) int {
			req := &ipc.Request{Command: "shutdown"}
			if err := ipc.WriteRequest(conn, req); err != nil {
				fmt.Fprintln(r.Stderr, color.RedString("Error sending shutdown: %v", err))
				return 1
			}
			_, _ = ipc.ReadResponse(conn)
			fmt.Fprintln(r.Stdout, color.YellowString("GPM daemon is shutting down..."))
			return 0
		})
	default:
		fmt.Fprint(r.Stderr, color.RedString("Error: unknown command %q\n\n", subCommand))
		r.PrintHelp()
		return 1
	}
}

// connectAndRun handles dialing the GPM daemon. If it's offline, it boots it up.
func (r *Runner) connectAndRun(f func(conn net.Conn) int) int {
	cs, err := config.NewConfigStore()
	if err != nil {
		fmt.Fprint(r.Stderr, color.RedString("Error accessing configurations: %v\n", err))
		return 1
	}

	pipeName := ipc.GetIPCPipe(cs.GetConfigDir())
	conn, err := ipc.DialIPC(pipeName)
	if err != nil {
		// Daemon is likely not running. Auto-boot it!
		fmt.Fprintln(r.Stdout, color.CyanString("GPM daemon is offline. Booting daemon background process..."))
		if err := r.startDaemonBackground(); err != nil {
			fmt.Fprint(r.Stderr, color.RedString("Error: failed to start daemon: %v\n", err))
			return 1
		}

		// Retry connecting
		for i := 0; i < 5; i++ {
			time.Sleep(500 * time.Millisecond)
			conn, err = ipc.DialIPC(pipeName)
			if err == nil {
				break
			}
		}

		if err != nil {
			fmt.Fprint(r.Stderr, color.RedString("Error: failed to connect to GPM daemon after start: %v\n", err))
			return 1
		}
	}
	defer conn.Close()

	return f(conn)
}

func (r *Runner) startDaemonBackground() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(execPath, "daemon")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Detach process programmatically
	if err := cmd.Start(); err != nil {
		return err
	}

	return nil
}

// PrintHelp writes the premium help documentation to stdout.
func (r *Runner) PrintHelp() {
	header := color.New(color.FgCyan, color.Bold).SprintFunc()
	cmdColor := color.New(color.FgYellow).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	fmt.Fprint(r.Stdout, header("Go Process Manager (GPM)")+" - Production-ready process management for Go apps.\n\n")
	fmt.Fprintln(r.Stdout, bold("Usage:"))
	fmt.Fprintf(r.Stdout, "  gpm <command> [arguments]\n\n")

	fmt.Fprintln(r.Stdout, bold("Management Commands:"))
	fmt.Fprintf(r.Stdout, "  %-18s %s\n", cmdColor("start <name>"), "Start and monitor a process. Auto-registers if configs are provided.")
	fmt.Fprintf(r.Stdout, "  %-18s %s\n", cmdColor("stop <name>"), "Stop a running process gracefully.")
	fmt.Fprintf(r.Stdout, "  %-18s %s\n", cmdColor("restart <name>"), "Restart a managed process execution.")
	fmt.Fprintf(r.Stdout, "  %-18s %s\n", cmdColor("delete <name>"), "Stop and remove a process configuration and clean its logs.")
	fmt.Fprintf(r.Stdout, "  %-18s %s\n", cmdColor("list"), "List all processes managed by GPM.")
	fmt.Fprintf(r.Stdout, "  %-18s %s\n", cmdColor("status"), "Show GPM daemon stats and process summary.")
	fmt.Fprintf(r.Stdout, "  %-18s %s\n", cmdColor("logs <name>"), "Display stdout/stderr logs. Supports tailing, filtering, and deleting.")
	fmt.Fprintf(r.Stdout, "  %-18s %s\n", cmdColor("enable <name>"), "Enable process autostart on operating system boot.")
	fmt.Fprintf(r.Stdout, "  %-18s %s\n", cmdColor("disable <name>"), "Disable process autostart on operating system boot.")
	fmt.Fprintf(r.Stdout, "  %-18s %s\n", cmdColor("startup"), "Register GPM daemon to start automatically after OS boot.")
	fmt.Fprintf(r.Stdout, "  %-18s %s\n", cmdColor("startup status"), "Show current OS startup service configuration status.")
	fmt.Fprintf(r.Stdout, "  %-18s %s\n", cmdColor("setup"), "Interactively install GPM (user-level or system-wide).")
	fmt.Fprintf(r.Stdout, "  %-18s %s\n", cmdColor("doctor"), "Run system diagnostics to verify GPM environment and status.")
	fmt.Fprintf(r.Stdout, "  %-18s %s\n", cmdColor("shutdown"), "Gracefully terminate all processes and stop GPM daemon.")
	fmt.Fprintf(r.Stdout, "  %-18s %s\n", cmdColor("version"), "Show GPM version.")
	fmt.Fprintf(r.Stdout, "  %-18s %s\n", cmdColor("help"), "Show this help screen.\n\n")

	fmt.Fprintln(r.Stdout, bold("Examples:"))
	fmt.Fprintf(r.Stdout, "  gpm start api --cmd \"C:\\Apps\\api.exe\" --dir \"C:\\Apps\"\n")
	fmt.Fprintf(r.Stdout, "  gpm start api\n")
	fmt.Fprintf(r.Stdout, "  gpm logs api --lines 50\n")
	fmt.Fprintf(r.Stdout, "  gpm logs api --query \"panic\"\n")
}

// PrintVersion writes the version.
func (r *Runner) PrintVersion() {
	fmt.Fprintf(r.Stdout, "gpm version v%s (%s/%s)\n", VersionInfo, runtime.GOOS, runtime.GOARCH)
}

func (r *Runner) executeSimpleCmd(conn net.Conn, cmd, name string) int {
	req := &ipc.Request{
		Command: cmd,
		Name:    name,
	}

	if err := ipc.WriteRequest(conn, req); err != nil {
		fmt.Fprint(r.Stderr, color.RedString("Error writing request: %v\n", err))
		return 1
	}

	resp, err := ipc.ReadResponse(conn)
	if err != nil {
		fmt.Fprint(r.Stderr, color.RedString("Error reading response: %v\n", err))
		return 1
	}

	if !resp.Success {
		fmt.Fprint(r.Stderr, color.RedString("Command failed: %s\n", resp.Error))
		return 1
	}

	successMsg := fmt.Sprintf("Process %q %s command completed successfully.", name, cmd)
	fmt.Fprintln(r.Stdout, color.GreenString(successMsg))
	return 0
}

func (r *Runner) handleStartCommand(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(r.Stderr, color.RedString("Error: process name is required. Usage: gpm start <name> [flags]"))
		return 1
	}

	name := args[0]
	cmdArgs := args[1:]

	// Flags for registering new process
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	cmdFlag := fs.String("cmd", "", "Command or executable path")
	dirFlag := fs.String("dir", "", "Working directory")
	argsFlag := fs.String("args", "", "Command arguments (space separated)")
	delayFlag := fs.Int("restart-delay", 5, "Restart delay in seconds")
	restartsFlag := fs.Int("max-restarts", 10, "Max crash restarts count")
	noAutoRestart := fs.Bool("no-autorestart", false, "Disable auto-restart on crash")

	fs.SetOutput(r.Stderr)
	if err := fs.Parse(cmdArgs); err != nil {
		return 1
	}

	var targetCfg *config.ProcessConfig
	if *cmdFlag != "" {
		// User is configuring a new command
		var envMap map[string]string // Can support parsing environment variables here later

		// Parse arguments if specified
		var parsedArgs []string
		if *argsFlag != "" {
			parsedArgs = strings.Fields(*argsFlag)
		}

		// Resolve working directory to an absolute path
		workingDir := *dirFlag
		if workingDir == "" {
			if wd, err := os.Getwd(); err == nil {
				workingDir = wd
			}
		} else {
			if absDir, err := filepath.Abs(workingDir); err == nil {
				workingDir = absDir
			}
		}

		targetCfg = &config.ProcessConfig{
			Name:         name,
			Command:      *cmdFlag,
			Directory:    workingDir,
			Args:         parsedArgs,
			Environment:  envMap,
			AutoStart:    true,
			AutoRestart:  !*noAutoRestart,
			RestartDelay: *delayFlag,
			MaxRestarts:  *restartsFlag,
		}
	}

	return r.connectAndRun(func(conn net.Conn) int {
		req := &ipc.Request{
			Command:      "start",
			Name:         name,
			TargetConfig: targetCfg,
		}

		if err := ipc.WriteRequest(conn, req); err != nil {
			fmt.Fprint(r.Stderr, color.RedString("Error writing request: %v\n", err))
			return 1
		}

		resp, err := ipc.ReadResponse(conn)
		if err != nil {
			fmt.Fprint(r.Stderr, color.RedString("Error reading response: %v\n", err))
			return 1
		}

		if !resp.Success {
			fmt.Fprint(r.Stderr, color.RedString("Failed to start %q: %s\n", name, resp.Error))
			return 1
		}

		fmt.Fprintln(r.Stdout, color.GreenString("Process %q is online.", name))
		return 0
	})
}

func (r *Runner) executeList(conn net.Conn) int {
	req := &ipc.Request{Command: "list"}
	if err := ipc.WriteRequest(conn, req); err != nil {
		fmt.Fprint(r.Stderr, color.RedString("Error: %v\n", err))
		return 1
	}

	resp, err := ipc.ReadResponse(conn)
	if err != nil {
		fmt.Fprint(r.Stderr, color.RedString("Error: %v\n", err))
		return 1
	}

	if !resp.Success {
		fmt.Fprint(r.Stderr, color.RedString("Error listing: %s\n", resp.Error))
		return 1
	}

	var stats []process.ProcessStats
	if err := json.Unmarshal(resp.Data, &stats); err != nil {
		fmt.Fprint(r.Stderr, color.RedString("Parse error: %v\n", err))
		return 1
	}

	r.printColorfulTable(stats)
	return 0
}

func (r *Runner) executeStatus(conn net.Conn) int {
	req := &ipc.Request{Command: "status"}
	if err := ipc.WriteRequest(conn, req); err != nil {
		fmt.Fprint(r.Stderr, color.RedString("Error: %v\n", err))
		return 1
	}

	resp, err := ipc.ReadResponse(conn)
	if err != nil {
		fmt.Fprint(r.Stderr, color.RedString("Error: %v\n", err))
		return 1
	}

	if !resp.Success {
		fmt.Fprint(r.Stderr, color.RedString("Error status: %s\n", resp.Error))
		return 1
	}

	var data manager.GPMStatusData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		fmt.Fprint(r.Stderr, color.RedString("Parse error: %v\n", err))
		return 1
	}

	daemonColor := color.GreenString("running")
	if !data.DaemonRunning {
		daemonColor = color.RedString("stopped")
	}

	fmt.Fprintln(r.Stdout, color.CyanString("GPM STATUS"))
	fmt.Fprintln(r.Stdout, "----------")
	fmt.Fprintf(r.Stdout, "Daemon:            %s\n", daemonColor)
	fmt.Fprintf(r.Stdout, "Platform:          %s\n", color.WhiteString("%s (%s)", runtime.GOOS, runtime.GOARCH))
	fmt.Fprintf(r.Stdout, "Managed processes: %d\n", data.ManagedProcesses)
	fmt.Fprintf(r.Stdout, "Online:            %s\n", color.GreenString(strconv.Itoa(data.Online)))
	fmt.Fprintf(r.Stdout, "Stopped:           %s\n", color.YellowString(strconv.Itoa(data.Stopped)))
	fmt.Fprintf(r.Stdout, "Failed:            %s\n\n", color.RedString(strconv.Itoa(data.Failed)))

	// The IPC connection is single-use (one request → one response).
	// Open a fresh connection to fetch the process list.
	return r.connectAndRun(func(listConn net.Conn) int {
		return r.executeList(listConn)
	})
}

func (r *Runner) printColorfulTable(stats []process.ProcessStats) {
	if len(stats) == 0 {
		fmt.Fprintln(r.Stdout, color.YellowString("No managed processes found. Start one using: gpm start <name> --cmd <executable>"))
		return
	}

	// Columns and widths
	// NAME (12) | STATUS (10) | PID (8) | UPTIME (10) | RESTARTS (10) | AUTOSTART (10)
	header := color.New(color.FgWhite, color.Bold, color.Underline).SprintFunc()
	fmt.Fprintf(r.Stdout, "%s\n", header("NAME         STATUS     PID      UPTIME     RESTARTS   AUTOSTART "))

	for _, s := range stats {
		namePadded := fmt.Sprintf("%-12s", s.Name)
		
		var statusStr string
		statusPadded := fmt.Sprintf("%-10s", s.Status)
		switch s.Status {
		case process.StatusOnline:
			statusStr = color.GreenString(statusPadded)
		case process.StatusStopped:
			statusStr = color.HiBlackString(statusPadded)
		case process.StatusFailed:
			statusStr = color.RedString(statusPadded)
		case process.StatusRestarting:
			statusStr = color.MagentaString(statusPadded)
		case process.StatusStarting, process.StatusStopping:
			statusStr = color.BlueString(statusPadded)
		default:
			statusStr = color.YellowString(statusPadded)
		}

		pidVal := "-"
		if s.PID > 0 {
			pidVal = strconv.Itoa(s.PID)
		}
		pidStr := color.WhiteString(fmt.Sprintf("%-8s", pidVal))

		uptimeVal := "-"
		if s.Status == process.StatusOnline {
			uptimeVal = r.formatDuration(s.Uptime)
		}
		uptimeStr := color.WhiteString(fmt.Sprintf("%-10s", uptimeVal))

		restartsVal := "-"
		if s.Status == process.StatusOnline || s.Status == process.StatusFailed || s.Restarts > 0 {
			restartsVal = strconv.Itoa(s.Restarts)
		}
		restartsStr := color.WhiteString(fmt.Sprintf("%-10s", restartsVal))

		autoStartVal := "no"
		if s.AutoStart {
			autoStartVal = "yes"
		}
		autoStartPadded := fmt.Sprintf("%-10s", autoStartVal)
		var autoStartStr string
		if s.AutoStart {
			autoStartStr = color.GreenString(autoStartPadded)
		} else {
			autoStartStr = color.RedString(autoStartPadded)
		}

		fmt.Fprintf(r.Stdout, "%s %s %s %s %s %s\n", namePadded, statusStr, pidStr, uptimeStr, restartsStr, autoStartStr)
	}
}

func (r *Runner) formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 24 {
		days := h / 24
		hours := h % 24
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func (r *Runner) handleLogsCommand(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(r.Stderr, color.RedString("Error: process name is required. Usage: gpm logs <name> [flags]"))
		return 1
	}

	name := args[0]
	cmdArgs := args[1:]

	fs := flag.NewFlagSet("logs", flag.ContinueOnError)
	linesFlag := fs.Int("lines", 100, "Tail last N lines of logs")
	queryFlag := fs.String("query", "", "Filter logs by matching substring")
	deleteFlag := fs.Bool("delete", false, "Delete all logs for process")

	// Custom parsing if delete is sub-arg: gpm logs delete <name>
	if name == "delete" {
		if len(cmdArgs) < 1 {
			fmt.Fprintln(r.Stderr, color.RedString("Error: process name required for logs delete. Usage: gpm logs delete <name>"))
			return 1
		}
		name = cmdArgs[0]
		return r.deleteLogsDirectly(name)
	}

	fs.SetOutput(r.Stderr)
	if err := fs.Parse(cmdArgs); err != nil {
		return 1
	}

	if *deleteFlag {
		return r.deleteLogsDirectly(name)
	}

	return r.connectAndRun(func(conn net.Conn) int {
		req := &ipc.Request{
			Command: "logs",
			Name:    name,
			Lines:   *linesFlag,
			Query:   *queryFlag,
		}

		if err := ipc.WriteRequest(conn, req); err != nil {
			fmt.Fprint(r.Stderr, color.RedString("Error writing request: %v\n", err))
			return 1
		}

		resp, err := ipc.ReadResponse(conn)
		if err != nil {
			fmt.Fprint(r.Stderr, color.RedString("Error reading response: %v\n", err))
			return 1
		}

		if !resp.Success {
			fmt.Fprint(r.Stderr, color.RedString("Error retrieving logs: %s\n", resp.Error))
			return 1
		}

		var logsResp manager.LogLinesResponse
		if err := json.Unmarshal(resp.Data, &logsResp); err != nil {
			fmt.Fprint(r.Stderr, color.RedString("Parse error: %v\n", err))
			return 1
		}

		// Print outputs
		if len(logsResp.Stdout) == 0 && len(logsResp.Stderr) == 0 {
			fmt.Fprintln(r.Stdout, color.YellowString("(No logs recorded yet.)"))
			return 0
		}

		fmt.Fprintln(r.Stdout, color.GreenString("=== STDOUT LOGS ==="))
		for _, l := range logsResp.Stdout {
			fmt.Fprintln(r.Stdout, l)
		}

		fmt.Fprintln(r.Stdout, color.RedString("\n=== STDERR LOGS ==="))
		for _, l := range logsResp.Stderr {
			fmt.Fprintln(r.Stdout, l)
		}

		return 0
	})
}

func (r *Runner) deleteLogsDirectly(name string) int {
	cs, err := config.NewConfigStore()
	if err != nil {
		fmt.Fprint(r.Stderr, color.RedString("Error: %v\n", err))
		return 1
	}

	err = logs.DeleteLogs(cs.GetConfigDir(), name)
	if err != nil {
		fmt.Fprint(r.Stderr, color.RedString("Error deleting logs: %v\n", err))
		return 1
	}

	fmt.Fprintln(r.Stdout, color.GreenString("Logs deleted successfully for process %q.", name))
	return 0
}

func (r *Runner) executeStartupInstall() int {
	cs, err := config.NewConfigStore()
	if err != nil {
		fmt.Fprint(r.Stderr, color.RedString("Error: %v\n", err))
		return 1
	}

	sm := service.NewServiceManager()
	fmt.Fprintln(r.Stdout, color.CyanString("Installing GPM startup integration..."))
	
	err = sm.ConfigureStartup(cs.GetConfigDir())
	if err != nil {
		fmt.Fprint(r.Stderr, color.RedString("Error installing startup service:\n%v\n", err))
		return 1
	}

	fmt.Fprintln(r.Stdout, color.GreenString("GPM service installed and configured to start on boot successfully!"))
	return 0
}

func (r *Runner) executeStartupStatus() int {
	sm := service.NewServiceManager()
	status, err := sm.GetStartupStatus()
	if err != nil {
		fmt.Fprint(r.Stderr, color.RedString("Error retrieving startup status: %v\n", err))
		return 1
	}

	fmt.Fprintf(r.Stdout, "GPM Startup Service status: %s\n", color.CyanString(status))
	return 0
}

func (r *Runner) readInput(prompt string) string {
	fmt.Fprint(r.Stdout, prompt)
	reader := bufio.NewReader(r.Stdin)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func (r *Runner) executeSetup(args []string) int {
	fmt.Fprintln(r.Stdout, color.CyanString("=== GPM Setup Utility ==="))

	if isAdmin() {
		fmt.Fprintln(r.Stdout, color.GreenString("Mode: System-wide Installation (Administrator/root)"))
	} else {
		fmt.Fprintln(r.Stdout, color.YellowString("Mode: User-level Installation (No Admin/root required)"))
	}

	confirm := r.readInput("This command will install GPM. Do you want to continue? [y/N]: ")
	confirm = strings.ToLower(confirm)
	if confirm != "y" && confirm != "yes" {
		fmt.Fprintln(r.Stdout, color.YellowString("Setup cancelled by user."))
		return 0
	}

	defaultDir, _ := getInstallDestination("")
	var customDir string
	if runtime.GOOS == "windows" {
		prompt := fmt.Sprintf("Installation directory [%s]: ", defaultDir)
		customDir = r.readInput(prompt)
	}

	installDir, targetPath := getInstallDestination(customDir)

	fmt.Fprintf(r.Stdout, "Creating installation directory: %s...\n", installDir)
	if err := os.MkdirAll(installDir, 0755); err != nil {
		fmt.Fprint(r.Stderr, color.RedString("Error creating directory: %v\n", err))
		return 1
	}

	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprint(r.Stderr, color.RedString("Error resolving current executable: %v\n", err))
		return 1
	}

	cs, err := config.NewConfigStore()
	if err == nil {
		pipeName := ipc.GetIPCPipe(cs.GetConfigDir())
		conn, err := ipc.DialIPC(pipeName)
		if err == nil {
			fmt.Fprintln(r.Stdout, color.YellowString("Existing GPM daemon is running. Shutting down daemon..."))
			req := &ipc.Request{Command: "shutdown"}
			_ = ipc.WriteRequest(conn, req)
			_, _ = ipc.ReadResponse(conn)
			conn.Close()
			time.Sleep(1 * time.Second)
		}
	}

	fmt.Fprintf(r.Stdout, "Copying GPM binary to %s...\n", targetPath)
	if err := r.copyFile(execPath, targetPath); err != nil {
		fmt.Fprint(r.Stderr, color.RedString("Error copying binary: %v\n", err))
		return 1
	}

	fmt.Fprintln(r.Stdout, "Configuring system PATH environment variable...")
	if err := configureSystemPath(installDir); err != nil {
		fmt.Fprint(r.Stderr, color.RedString("Error configuring PATH: %v\n", err))
		return 1
	}

	fmt.Fprintln(r.Stdout, "Registering GPM daemon startup service...")
	if err := r.executeStartupInstallDirectly(); err != nil {
		fmt.Fprint(r.Stderr, color.RedString("Error installing startup service: %v\n", err))
		return 1
	}

	fmt.Fprintln(r.Stdout, color.GreenString("=== GPM Installation Completed Successfully! ==="))
	if runtime.GOOS == "windows" {
		fmt.Fprintln(r.Stdout, color.CyanString("Please restart your terminal/PowerShell session to refresh the environment variables and run 'gpm' globally."))
	} else {
		fmt.Fprintln(r.Stdout, color.CyanString("You can now run 'gpm list' and 'gpm status' globally."))
	}

	return 0
}

func (r *Runner) copyFile(src, dst string) error {
	if _, err := os.Stat(dst); err == nil {
		_ = os.Remove(dst)
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func (r *Runner) executeStartupInstallDirectly() error {
	cs, err := config.NewConfigStore()
	if err != nil {
		return err
	}
	sm := service.NewServiceManager()
	return sm.ConfigureStartup(cs.GetConfigDir())
}

func (r *Runner) executeDoctor() int {
	fmt.Fprintln(r.Stdout, color.CyanString("=== GPM Diagnostics (Doctor) ==="))
	
	allPassed := true

	fmt.Fprint(r.Stdout, "Checking user privileges: ")
	if isAdmin() {
		if runtime.GOOS == "windows" {
			fmt.Fprintln(r.Stdout, color.GreenString("[✓] Running with Administrator privileges"))
		} else {
			fmt.Fprintln(r.Stdout, color.GreenString("[✓] Running with Root (sudo) privileges"))
		}
	} else {
		fmt.Fprintln(r.Stdout, color.GreenString("[✓] Running as standard user (user-level install mode)"))
	}

	fmt.Fprint(r.Stdout, "Checking environment PATH: ")
	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintln(r.Stdout, color.RedString("[✗] Failed to resolve current executable: %v", err))
		allPassed = false
	} else {
		execDir := filepath.Dir(execPath)
		pathEnv := os.Getenv("PATH")
		var separator string
		if runtime.GOOS == "windows" {
			separator = ";"
		} else {
			separator = ":"
		}
		
		inPath := false
		execDirClean := strings.ToLower(filepath.Clean(execDir))
		for _, p := range strings.Split(pathEnv, separator) {
			if strings.ToLower(filepath.Clean(p)) == execDirClean {
				inPath = true
				break
			}
		}
		
		if inPath {
			fmt.Fprintln(r.Stdout, color.GreenString("[✓] GPM folder is in your system PATH (%s)", execDir))
		} else {
			fmt.Fprintln(r.Stdout, color.YellowString("[!] GPM directory (%s) is NOT in your system PATH environment variable.", execDir))
			fmt.Fprintln(r.Stdout, "    (Run 'gpm setup' to automatically register GPM in your system PATH)")
			allPassed = false
		}
	}

	fmt.Fprint(r.Stdout, "Checking GPM Daemon status: ")
	cs, err := config.NewConfigStore()
	if err != nil {
		fmt.Fprintln(r.Stdout, color.RedString("[✗] Failed to open Config Store: %v", err))
		allPassed = false
	} else {
		pipeName := ipc.GetIPCPipe(cs.GetConfigDir())
		conn, err := ipc.DialIPC(pipeName)
		if err != nil {
			fmt.Fprintln(r.Stdout, color.RedString("[✗] GPM daemon is offline (unable to connect to IPC pipe/socket)"))
			fmt.Fprintln(r.Stdout, "    (Try running 'gpm status' or starting a process to auto-boot it)")
			allPassed = false
		} else {
			conn.Close()
			fmt.Fprintln(r.Stdout, color.GreenString("[✓] GPM daemon is online and responsive"))
		}
	}

	fmt.Fprint(r.Stdout, "Checking Startup Service configuration: ")
	sm := service.NewServiceManager()
	status, err := sm.GetStartupStatus()
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "access is denied") {
			fmt.Fprintln(r.Stdout, color.YellowString("[!] GPM auto-start service status could not be queried: Access is denied. (requires Administrator privileges)"))
		} else {
			fmt.Fprintln(r.Stdout, color.RedString("[✗] Failed to query service status: %v", err))
			allPassed = false
		}
	} else {
		if strings.Contains(status, "not_installed") {
			fmt.Fprintln(r.Stdout, color.YellowString("[!] GPM auto-start service is not installed on this system"))
			fmt.Fprintln(r.Stdout, "    (Run 'gpm setup' or 'gpm startup' to register auto-start on system boot)")
			allPassed = false
		} else {
			fmt.Fprintln(r.Stdout, color.GreenString("[✓] Auto-start is configured: %s", status))
		}
	}

	fmt.Fprint(r.Stdout, "Checking Config Directory permissions: ")
	if cs != nil {
		configDir := cs.GetConfigDir()
		testFile := filepath.Join(configDir, ".doctor_write_test")
		err := os.WriteFile(testFile, []byte("test"), 0644)
		if err != nil {
			fmt.Fprintln(r.Stdout, color.RedString("[✗] Configuration directory (%s) is not writable: %v", configDir, err))
			allPassed = false
		} else {
			_ = os.Remove(testFile)
			fmt.Fprintln(r.Stdout, color.GreenString("[✓] Configuration directory is writable: %s", configDir))
		}
	}

	fmt.Fprintln(r.Stdout)
	if allPassed {
		fmt.Fprintln(r.Stdout, color.New(color.FgGreen, color.Bold).Sprint("Result: All diagnostic checks passed! Your GPM installation is healthy."))
		return 0
	} else {
		fmt.Fprintln(r.Stdout, color.New(color.FgYellow, color.Bold).Sprint("Result: Diagnostics completed with warnings/errors. Please review the output above to resolve setup issues."))
		return 1
	}
}
