package cli

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/daemon"
	"github.com/priyanshujain/openbotkit/daemon/service"
	"github.com/priyanshujain/openbotkit/internal/platform"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage the obk background service",
}

var serviceRunCmd = &cobra.Command{
	Use:    "run",
	Short:  "Run the service process (used internally by the system service)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		d := daemon.New(cfg)
		return d.Run(ctx)
	},
}

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install obk as a system service",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := service.NewManager()
		if err != nil {
			return err
		}

		cfg, err := service.DefaultConfig()
		if err != nil {
			return err
		}

		if err := mgr.Install(cfg); err != nil {
			return fmt.Errorf("install service: %w", err)
		}

		fmt.Printf("service installed (platform: %s)\n", service.DetectPlatform())
		return nil
	},
}

var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the obk system service",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := service.NewManager()
		if err != nil {
			return err
		}

		if err := mgr.Uninstall(); err != nil {
			return fmt.Errorf("uninstall service: %w", err)
		}

		fmt.Println("service uninstalled")
		return nil
	},
}

var serviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the service status",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := service.NewManager()
		if err != nil {
			return err
		}

		status, err := mgr.Status()
		if err != nil {
			return fmt.Errorf("check status: %w", err)
		}

		fmt.Printf("service: %s\n", status)
		return nil
	},
}

var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the obk service",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := service.NewManager()
		if err != nil {
			return err
		}

		if err := mgr.Start(); err != nil {
			return fmt.Errorf("start service: %w", err)
		}

		fmt.Println("service started")
		return nil
	},
}

var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the obk service",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := service.NewManager()
		if err != nil {
			return err
		}

		if err := mgr.Stop(); err != nil {
			return fmt.Errorf("stop service: %w", err)
		}

		fmt.Println("service stopped")
		return nil
	},
}

var serviceRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the obk service",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := service.NewManager()
		if err != nil {
			return err
		}

		if err := mgr.Stop(); err != nil {
			return fmt.Errorf("stop service: %w", err)
		}

		if err := mgr.Start(); err != nil {
			return fmt.Errorf("start service: %w", err)
		}

		fmt.Println("service restarted")
		return nil
	},
}

func init() {
	serviceCmd.AddCommand(serviceRunCmd)
	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceUninstallCmd)
	serviceCmd.AddCommand(serviceStartCmd)
	serviceCmd.AddCommand(serviceStopCmd)
	serviceCmd.AddCommand(serviceRestartCmd)
	serviceCmd.AddCommand(serviceStatusCmd)
}
