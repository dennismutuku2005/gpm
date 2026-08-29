package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dennismutuku2005/gpm/internal/cli"
	"github.com/dennismutuku2005/gpm/internal/config"
	"github.com/dennismutuku2005/gpm/internal/manager"
	"github.com/dennismutuku2005/gpm/internal/service"
)

func main() {
	// 1. If running as Windows Service, run service daemon immediately
	if service.IsWindowsService() {
		cs, err := config.NewConfigStore()
		if err != nil {
			os.Exit(1)
		}
		m := manager.NewManager(cs)
		if err := service.RunWindowsService(m); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// 2. Parse arguments
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "daemon" {
		// Daemon mode setup
		fs := flag.NewFlagSet("daemon", flag.ExitOnError)
		configDirOpt := fs.String("config-dir", "", "Custom configuration directory")
		_ = fs.Parse(args[1:])

		if *configDirOpt != "" {
			_ = os.Setenv("GPM_CONFIG_DIR", *configDirOpt)
		}

		cs, err := config.NewConfigStore()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize config store: %v\n", err)
			os.Exit(1)
		}

		m := manager.NewManager(cs)
		fmt.Printf("Starting GPM daemon (config: %s)...\n", cs.GetConfigDir())
		if err := m.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start daemon: %v\n", err)
			os.Exit(1)
		}
		m.Wait()
		os.Exit(0)
	}

	// 3. CLI mode
	runner := cli.NewRunner(os.Stdout, os.Stderr)
	os.Exit(runner.Run(args))
}
