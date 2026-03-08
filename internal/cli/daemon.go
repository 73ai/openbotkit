package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/daemon"
	"github.com/priyanshujain/openbotkit/daemon/service"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the obk background daemon",
}

var daemonRunCmd = &cobra.Command{
	Use:    "run",
	Short:  "Run the daemon process (used internally by the system service)",
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

		ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		d := daemon.New(cfg)
		return d.Run(ctx)
	},
}

var daemonInstallCmd = &cobra.Command{
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

		fmt.Printf("daemon installed (platform: %s)\n", service.DetectPlatform())
		return nil
	},
}

var daemonUninstallCmd = &cobra.Command{
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

		fmt.Println("daemon uninstalled")
		return nil
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the daemon status",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := service.NewManager()
		if err != nil {
			return err
		}

		status, err := mgr.Status()
		if err != nil {
			return fmt.Errorf("check status: %w", err)
		}

		fmt.Printf("daemon: %s\n", status)
		return nil
	},
}

func init() {
	daemonCmd.AddCommand(daemonRunCmd)
	daemonCmd.AddCommand(daemonInstallCmd)
	daemonCmd.AddCommand(daemonUninstallCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
}
