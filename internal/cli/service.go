package cli

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/daemon"
	"github.com/73ai/openbotkit/daemon/service"
	"github.com/73ai/openbotkit/internal/platform"
	"github.com/73ai/openbotkit/internal/server"
)

var allServices = []string{"daemon", "server"}

func resolveServices(args []string) ([]string, error) {
	if len(args) == 0 {
		return allServices, nil
	}
	valid := map[string]bool{"daemon": true, "server": true}
	for _, a := range args {
		if !valid[a] {
			return nil, fmt.Errorf("unknown service %q (valid: daemon, server)", a)
		}
	}
	return args, nil
}

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage obk background services (daemon and server)",
}

var serviceRunCmd = &cobra.Command{
	Use:       "run <daemon|server>",
	Short:     "Run a service process (used internally by the system service)",
	Hidden:    true,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"daemon", "server"},
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if err := cfg.RequireSetup(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		ctx, stop := signal.NotifyContext(cmd.Context(), platform.ShutdownSignals...)
		defer stop()

		switch name {
		case "daemon":
			bridge, _ := cmd.Flags().GetBool("bridge")
			if bridge {
				if !cfg.IsRemote() {
					return fmt.Errorf("bridge mode requires remote mode — set 'mode: remote' in config")
				}
				client, err := remoteClient(cfg)
				if err != nil {
					return err
				}
				return daemon.RunBridge(ctx, cfg, client)
			}
			d := daemon.New(cfg)
			return d.Run(ctx)
		case "server":
			addr, _ := cmd.Flags().GetString("addr")
			s := server.New(cfg, addr)
			return s.Run(ctx)
		default:
			return fmt.Errorf("unknown service %q (valid: daemon, server)", name)
		}
	},
}

var serviceInstallCmd = &cobra.Command{
	Use:       "install [daemon|server]",
	Short:     "Install services as system services",
	Example:   "  obk service install\n  obk service install daemon",
	Args:      cobra.MaximumNArgs(1),
	ValidArgs: []string{"daemon", "server"},
	RunE: func(cmd *cobra.Command, args []string) error {
		names, err := resolveServices(args)
		if err != nil {
			return err
		}
		for _, name := range names {
			mgr, err := service.NewManager(name)
			if err != nil {
				return err
			}
			cfg, err := service.DefaultConfig(name, []string{"service", "run", name})
			if err != nil {
				return err
			}
			if err := mgr.Install(cfg); err != nil {
				return fmt.Errorf("install %s: %w", name, err)
			}
			fmt.Printf("%s service installed (platform: %s)\n", name, service.DetectPlatform())
		}
		return nil
	},
}

var serviceUninstallCmd = &cobra.Command{
	Use:       "uninstall [daemon|server]",
	Short:     "Uninstall system services",
	Example:   "  obk service uninstall\n  obk service uninstall daemon\n  obk service uninstall --force",
	Args:      cobra.MaximumNArgs(1),
	ValidArgs: []string{"daemon", "server"},
	RunE: func(cmd *cobra.Command, args []string) error {
		names, err := resolveServices(args)
		if err != nil {
			return err
		}
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			label := strings.Join(names, " and ")
			fmt.Printf("About to uninstall the %s service(s). Continue? (y/N): ", label)
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" && confirm != "Y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}
		for _, name := range names {
			mgr, err := service.NewManager(name)
			if err != nil {
				return err
			}
			if err := mgr.Uninstall(); err != nil {
				return fmt.Errorf("uninstall %s: %w", name, err)
			}
			fmt.Printf("%s service uninstalled\n", name)
		}
		return nil
	},
}

var serviceStartCmd = &cobra.Command{
	Use:       "start [daemon|server]",
	Short:     "Start services",
	Example:   "  obk service start\n  obk service start daemon",
	Args:      cobra.MaximumNArgs(1),
	ValidArgs: []string{"daemon", "server"},
	RunE: func(cmd *cobra.Command, args []string) error {
		names, err := resolveServices(args)
		if err != nil {
			return err
		}
		for _, name := range names {
			mgr, err := service.NewManager(name)
			if err != nil {
				return err
			}
			if err := mgr.Start(); err != nil {
				return fmt.Errorf("start %s: %w", name, err)
			}
			fmt.Printf("%s service started\n", name)
		}
		return nil
	},
}

var serviceStopCmd = &cobra.Command{
	Use:       "stop [daemon|server]",
	Short:     "Stop services",
	Example:   "  obk service stop\n  obk service stop server",
	Args:      cobra.MaximumNArgs(1),
	ValidArgs: []string{"daemon", "server"},
	RunE: func(cmd *cobra.Command, args []string) error {
		names, err := resolveServices(args)
		if err != nil {
			return err
		}
		for _, name := range names {
			mgr, err := service.NewManager(name)
			if err != nil {
				return err
			}
			if err := mgr.Stop(); err != nil {
				return fmt.Errorf("stop %s: %w", name, err)
			}
			fmt.Printf("%s service stopped\n", name)
		}
		return nil
	},
}

var serviceRestartCmd = &cobra.Command{
	Use:       "restart [daemon|server]",
	Short:     "Restart services",
	Example:   "  obk service restart\n  obk service restart daemon",
	Args:      cobra.MaximumNArgs(1),
	ValidArgs: []string{"daemon", "server"},
	RunE: func(cmd *cobra.Command, args []string) error {
		names, err := resolveServices(args)
		if err != nil {
			return err
		}
		for _, name := range names {
			mgr, err := service.NewManager(name)
			if err != nil {
				return err
			}
			if err := mgr.Stop(); err != nil {
				return fmt.Errorf("stop %s: %w", name, err)
			}
			if err := mgr.Start(); err != nil {
				return fmt.Errorf("start %s: %w", name, err)
			}
			fmt.Printf("%s service restarted\n", name)
		}
		return nil
	},
}

var serviceStatusCmd = &cobra.Command{
	Use:       "status [daemon|server]",
	Short:     "Check service status",
	Example:   "  obk service status\n  obk service status daemon",
	Args:      cobra.MaximumNArgs(1),
	ValidArgs: []string{"daemon", "server"},
	RunE: func(cmd *cobra.Command, args []string) error {
		names, err := resolveServices(args)
		if err != nil {
			return err
		}
		for _, name := range names {
			mgr, err := service.NewManager(name)
			if err != nil {
				return err
			}
			status, err := mgr.Status()
			if err != nil {
				return fmt.Errorf("check %s status: %w", name, err)
			}
			fmt.Printf("%s: %s\n", name, status)
		}
		return nil
	},
}

var serviceLogsCmd = &cobra.Command{
	Use:       "logs <daemon|server>",
	Short:     "Show service logs",
	Example:   "  obk service logs daemon\n  obk service logs server --follow",
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"daemon", "server"},
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		valid := map[string]bool{"daemon": true, "server": true}
		if !valid[name] {
			return fmt.Errorf("unknown service %q (valid: daemon, server)", name)
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("get home dir: %w", err)
		}
		logPath := filepath.Join(home, ".obk", name+".log")

		follow, _ := cmd.Flags().GetBool("follow")
		tail, _ := cmd.Flags().GetInt("tail")

		f, err := os.Open(logPath)
		if err != nil {
			return fmt.Errorf("open log file: %w", err)
		}
		defer f.Close()

		if err := printTail(f, tail); err != nil {
			return err
		}
		if !follow {
			return nil
		}

		ctx, stop := signal.NotifyContext(cmd.Context(), platform.ShutdownSignals...)
		defer stop()
		return followFile(ctx, f)
	},
}

func init() {
	serviceRunCmd.Flags().Bool("bridge", false, "run in bridge mode (sync Apple Notes to remote)")
	serviceRunCmd.Flags().String("addr", ":8443", "address to listen on (server only)")
	serviceCmd.AddCommand(serviceRunCmd)
	serviceCmd.AddCommand(serviceInstallCmd)
	serviceUninstallCmd.Flags().Bool("force", false, "Skip confirmation prompt")
	serviceCmd.AddCommand(serviceUninstallCmd)
	serviceCmd.AddCommand(serviceStartCmd)
	serviceCmd.AddCommand(serviceStopCmd)
	serviceCmd.AddCommand(serviceRestartCmd)
	serviceCmd.AddCommand(serviceStatusCmd)
	serviceLogsCmd.Flags().BoolP("follow", "f", false, "follow log output")
	serviceLogsCmd.Flags().Int("tail", 50, "number of lines to show from the end")
	serviceCmd.AddCommand(serviceLogsCmd)
}
